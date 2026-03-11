import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import App from './App'

vi.mock('./context/AuthContext', async (importOriginal) => {
  const real = await importOriginal()
  return {
    ...real,
    AuthProvider: ({ children }) => <div>{children}</div>,
    useAuth: vi.fn(() => ({ user: null, loading: false })),
  }
})

vi.mock('./pages/LoginPage', () => ({ default: () => <div>LoginPage</div> }))
vi.mock('./pages/RegisterPage', () => ({ default: () => <div>RegisterPage</div> }))
vi.mock('./pages/DashboardPage', () => ({ default: () => <div>DashboardPage</div> }))
vi.mock('./pages/UploadPage', () => ({ default: () => <div>UploadPage</div> }))

import { useAuth } from './context/AuthContext'

describe('App routing', () => {
  it('renders LoginPage at /login', async () => {
    window.history.pushState({}, '', '/login')
    render(<App />)
    await waitFor(() => expect(screen.getByText('LoginPage')).toBeDefined())
  })

  it('renders RegisterPage at /register', async () => {
    window.history.pushState({}, '', '/register')
    render(<App />)
    await waitFor(() => expect(screen.getByText('RegisterPage')).toBeDefined())
  })

  it('redirects to /login when not authenticated at /', async () => {
    useAuth.mockReturnValue({ user: null, loading: false })
    window.history.pushState({}, '', '/')
    render(<App />)
    await waitFor(() => expect(screen.getByText('LoginPage')).toBeDefined())
  })

  it('shows loading indicator when loading is true', async () => {
    useAuth.mockReturnValue({ user: null, loading: true })
    window.history.pushState({}, '', '/')
    render(<App />)
    await waitFor(() => expect(screen.getByText('Carregando...')).toBeDefined())
  })

  it('renders DashboardPage when user is authenticated at /', async () => {
    useAuth.mockReturnValue({ user: { name: 'Alice' }, loading: false })
    window.history.pushState({}, '', '/')
    render(<App />)
    await waitFor(() => expect(screen.getByText('DashboardPage')).toBeDefined())
  })
})
