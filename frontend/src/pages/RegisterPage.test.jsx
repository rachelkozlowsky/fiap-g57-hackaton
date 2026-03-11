import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import RegisterPage from './RegisterPage'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const real = await importOriginal()
  return { ...real, useNavigate: () => mockNavigate }
})

vi.mock('../components/Navbar', () => ({ default: () => <div>Navbar</div> }))

const mockRegister = vi.fn()
const mockLogin = vi.fn()
vi.mock('../context/AuthContext', () => ({
  useAuth: () => ({ register: mockRegister, login: mockLogin }),
}))

const renderPage = () => render(<MemoryRouter><RegisterPage /></MemoryRouter>)

describe('RegisterPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders all form fields', () => {
    renderPage()
    expect(screen.getByPlaceholderText('Seu nome')).toBeDefined()
    expect(screen.getByPlaceholderText('seu@email.com')).toBeDefined()
    expect(screen.getByPlaceholderText('••••••••')).toBeDefined()
    expect(screen.getByText('Criar Conta')).toBeDefined()
  })

  it('updates form fields', () => {
    renderPage()
    const name = screen.getByPlaceholderText('Seu nome')
    fireEvent.change(name, { target: { value: 'Bob' } })
    expect(name.value).toBe('Bob')
  })

  it('calls register then login and navigates on success', async () => {
    mockRegister.mockResolvedValueOnce({ success: true })
    mockLogin.mockResolvedValueOnce({ success: true })
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('Seu nome'), { target: { value: 'Bob' } })
    fireEvent.change(screen.getByPlaceholderText('seu@email.com'), { target: { value: 'b@c.com' } })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), { target: { value: '123456' } })
    await act(async () => fireEvent.submit(screen.getByRole('button', { name: 'Registrar' })))
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
    expect(mockRegister).toHaveBeenCalledWith('Bob', 'b@c.com', '123456')
    expect(mockLogin).toHaveBeenCalledWith('b@c.com', '123456')
  })

  it('shows error message on registration failure', async () => {
    mockRegister.mockResolvedValueOnce({ success: false, message: 'Email já cadastrado' })
    renderPage()
    fireEvent.change(screen.getByPlaceholderText('Seu nome'), { target: { value: 'Bob' } })
    fireEvent.change(screen.getByPlaceholderText('seu@email.com'), { target: { value: 'b@c.com' } })
    fireEvent.change(screen.getByPlaceholderText('••••••••'), { target: { value: '123456' } })
    await act(async () => fireEvent.submit(screen.getByRole('button', { name: 'Registrar' })))
    await waitFor(() => expect(screen.getByText('Email já cadastrado')).toBeDefined())
    expect(mockNavigate).not.toHaveBeenCalled()
  })

  it('renders link to login page', () => {
    renderPage()
    expect(screen.getByText(/Já tem uma conta/i)).toBeDefined()
  })
})
