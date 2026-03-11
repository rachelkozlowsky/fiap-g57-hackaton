import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import LoginPage from './LoginPage'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const real = await importOriginal()
  return { ...real, useNavigate: () => mockNavigate }
})

vi.mock('../components/Navbar', () => ({ default: () => <div>Navbar</div> }))

const mockLogin = vi.fn()
vi.mock('../context/AuthContext', () => ({
  useAuth: () => ({ login: mockLogin }),
}))

const renderPage = () => render(<MemoryRouter><LoginPage /></MemoryRouter>)

describe('LoginPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders form fields', () => {
    renderPage()
    expect(screen.getByPlaceholderText('seu@email.com')).toBeDefined()
    expect(screen.getByPlaceholderText('••••••••')).toBeDefined()
    expect(screen.getByText('Bem-vindo de volta')).toBeDefined()
  })

  it('updates email and password fields', () => {
    renderPage()
    const email = screen.getByPlaceholderText('seu@email.com')
    const password = screen.getByPlaceholderText('••••••••')
    fireEvent.change(email, { target: { value: 'test@test.com' } })
    fireEvent.change(password, { target: { value: '123456' } })
    expect(email.value).toBe('test@test.com')
    expect(password.value).toBe('123456')
  })

  it('navigates to / on successful login', async () => {
    mockLogin.mockResolvedValueOnce({ success: true })
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('seu@email.com'), { target: { value: 'a@b.com' } })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), { target: { value: '123456' } })
    await act(async () => fireEvent.submit(screen.getByRole('button', { name: /entrar/i })))
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
  })

  it('shows error on failed login', async () => {
    mockLogin.mockResolvedValueOnce({ success: false, message: 'Credenciais inválidas' })
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('seu@email.com'), { target: { value: 'a@b.com' } })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), { target: { value: 'wrong' } })
    await act(async () => fireEvent.submit(screen.getByRole('button', { name: /entrar/i })))
    await waitFor(() => expect(screen.getByText('Credenciais inválidas')).toBeDefined())
  })

  it('renders link to register page', () => {
    renderPage()
    expect(screen.getByText('Registre-se')).toBeDefined()
  })
})
