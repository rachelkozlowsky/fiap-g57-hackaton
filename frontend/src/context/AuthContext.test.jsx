import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, act, waitFor } from '@testing-library/react'
import { AuthProvider, useAuth } from './AuthContext'

vi.mock('../services/api', () => ({
  default: {
    post: vi.fn(),
    interceptors: {
      request: { use: vi.fn() },
      response: { use: vi.fn() },
    },
  },
}))

vi.mock('jwt-decode', () => ({
  jwtDecode: vi.fn(),
}))

import api from '../services/api'
import { jwtDecode } from 'jwt-decode'

const TestConsumer = () => {
  const { user, loading } = useAuth()
  if (loading) return <div>loading</div>
  return <div>{user ? `user:${user.name}` : 'no-user'}</div>
}

describe('AuthContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders children when not loading', async () => {
    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    )
    await waitFor(() => expect(screen.queryByText('loading')).toBeNull())
    expect(screen.getByText('no-user')).toBeDefined()
  })

  it('restores user from valid token in localStorage', async () => {
    localStorage.setItem('token', 'valid-token')
    jwtDecode.mockReturnValue({ exp: Date.now() / 1000 + 3600, name: 'Alice', email: 'a@b.com' })

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    )

    await waitFor(() => expect(screen.getByText('user:Alice')).toBeDefined())
  })

  it('logs out when token is expired', async () => {
    localStorage.setItem('token', 'expired-token')
    jwtDecode.mockReturnValue({ exp: Date.now() / 1000 - 100 })

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    )

    await waitFor(() => expect(screen.getByText('no-user')).toBeDefined())
    expect(localStorage.getItem('token')).toBeNull()
  })

  it('calls logout on invalid token', async () => {
    localStorage.setItem('token', 'bad-token')
    jwtDecode.mockImplementation(() => { throw new Error('invalid') })

    render(
      <AuthProvider>
        <TestConsumer />
      </AuthProvider>
    )

    await waitFor(() => expect(screen.getByText('no-user')).toBeDefined())
  })

  it('login success sets user', async () => {
    api.post.mockResolvedValueOnce({
      data: { access_token: 'tok', user: { name: 'Bob', email: 'b@c.com' } },
    })

    const LoginTrigger = () => {
      const { login, user } = useAuth()
      return (
        <>
          <button onClick={() => login('b@c.com', '123')}>login</button>
          <div>{user ? `user:${user.name}` : 'no-user'}</div>
        </>
      )
    }

    render(<AuthProvider><LoginTrigger /></AuthProvider>)
    await waitFor(() => screen.getByText('no-user'))

    await act(async () => screen.getByText('login').click())
    await waitFor(() => expect(screen.getByText('user:Bob')).toBeDefined())
    expect(localStorage.getItem('token')).toBe('tok')
  })

  it('login failure returns error message', async () => {
    api.post.mockRejectedValueOnce({ response: { data: { error: 'wrong credentials' } } })

    let result
    const LoginTrigger = () => {
      const { login } = useAuth()
      return <button onClick={async () => { result = await login('x@y.com', 'bad') }}>login</button>
    }

    render(<AuthProvider><LoginTrigger /></AuthProvider>)
    await waitFor(() => screen.getByText('login'))
    await act(async () => screen.getByText('login').click())

    expect(result).toEqual({ success: false, message: 'wrong credentials' })
  })

  it('login failure with no response returns fallback message', async () => {
    api.post.mockRejectedValueOnce(new Error('network'))

    let result
    const LoginTrigger = () => {
      const { login } = useAuth()
      return <button onClick={async () => { result = await login('x@y.com', 'bad') }}>login</button>
    }

    render(<AuthProvider><LoginTrigger /></AuthProvider>)
    await waitFor(() => screen.getByText('login'))
    await act(async () => screen.getByText('login').click())

    expect(result).toEqual({ success: false, message: 'Login failed' })
  })

  it('register success returns success', async () => {
    api.post.mockResolvedValueOnce({})

    let result
    const Trigger = () => {
      const { register } = useAuth()
      return <button onClick={async () => { result = await register('Bob', 'b@c.com', '123') }}>reg</button>
    }

    render(<AuthProvider><Trigger /></AuthProvider>)
    await waitFor(() => screen.getByText('reg'))
    await act(async () => screen.getByText('reg').click())

    expect(result).toEqual({ success: true })
  })

  it('register failure returns error', async () => {
    api.post.mockRejectedValueOnce({ response: { data: { error: 'email taken' } } })

    let result
    const Trigger = () => {
      const { register } = useAuth()
      return <button onClick={async () => { result = await register('Bob', 'b@c.com', '123') }}>reg</button>
    }

    render(<AuthProvider><Trigger /></AuthProvider>)
    await waitFor(() => screen.getByText('reg'))
    await act(async () => screen.getByText('reg').click())

    expect(result).toEqual({ success: false, message: 'email taken' })
  })

  it('logout clears user and localStorage', async () => {
    localStorage.setItem('token', 'tok')
    jwtDecode.mockReturnValue({ exp: Date.now() / 1000 + 3600, name: 'Alice' })

    const Trigger = () => {
      const { logout, user } = useAuth()
      return (
        <>
          <button onClick={logout}>logout</button>
          <div>{user ? 'has-user' : 'no-user'}</div>
        </>
      )
    }

    render(<AuthProvider><Trigger /></AuthProvider>)
    await waitFor(() => screen.getByText('has-user'))
    await act(async () => screen.getByText('logout').click())

    expect(screen.getByText('no-user')).toBeDefined()
    expect(localStorage.getItem('token')).toBeNull()
  })
})
