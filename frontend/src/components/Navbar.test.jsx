import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import Navbar from './Navbar'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const real = await importOriginal()
  return { ...real, useNavigate: () => mockNavigate }
})

const mockLogout = vi.fn()

vi.mock('../context/AuthContext', () => ({
  useAuth: vi.fn(),
}))

import { useAuth } from '../context/AuthContext'

describe('Navbar', () => {
  it('renders login and register links when no user', () => {
    useAuth.mockReturnValue({ user: null, logout: mockLogout })

    render(<MemoryRouter><Navbar /></MemoryRouter>)

    expect(screen.getByText('Login')).toBeDefined()
    expect(screen.getByText('Começar Agora')).toBeDefined()
  })

  it('renders user name and email when logged in', () => {
    useAuth.mockReturnValue({
      user: { name: 'Alice', email: 'alice@example.com' },
      logout: mockLogout,
    })

    render(<MemoryRouter><Navbar /></MemoryRouter>)

    expect(screen.getByText('Alice')).toBeDefined()
    expect(screen.getByText('alice@example.com')).toBeDefined()
    expect(screen.getByText('Upload')).toBeDefined()
  })

  it('calls logout and navigates to /login on logout button click', () => {
    useAuth.mockReturnValue({
      user: { name: 'Alice', email: 'alice@example.com' },
      logout: mockLogout,
    })

    render(<MemoryRouter><Navbar /></MemoryRouter>)

    const logoutBtn = screen.getByRole('button')
    fireEvent.click(logoutBtn)

    expect(mockLogout).toHaveBeenCalled()
    expect(mockNavigate).toHaveBeenCalledWith('/login')
  })

  it('renders g57 brand link', () => {
    useAuth.mockReturnValue({ user: null, logout: mockLogout })

    render(<MemoryRouter><Navbar /></MemoryRouter>)

    expect(screen.getByText('g57')).toBeDefined()
  })
})
