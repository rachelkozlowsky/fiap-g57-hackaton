import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import DashboardPage from './DashboardPage'

vi.mock('../components/Navbar', () => ({ default: () => <div>Navbar</div> }))
vi.mock('../components/ConfirmationModal', () => ({
  default: ({ isOpen, onClose, onConfirm, isLoading }) =>
    isOpen ? (
      <div>
        <button onClick={onClose} disabled={isLoading}>cancel-modal</button>
        <button onClick={onConfirm} disabled={isLoading}>confirm-modal</button>
      </div>
    ) : null,
}))

vi.mock('../services/api', () => ({
  default: {
    get: vi.fn(),
    delete: vi.fn(),
    interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
  },
}))

import api from '../services/api'

const renderPage = () => render(<MemoryRouter><DashboardPage /></MemoryRouter>)

describe('DashboardPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows loading state initially', () => {
    api.get.mockReturnValue(new Promise(() => {}))
    renderPage()
    expect(screen.getByText('Carregando...')).toBeDefined()
  })

  it('shows empty state when no videos', async () => {
    api.get.mockResolvedValue({ data: [] })
    renderPage()
    await waitFor(() => expect(screen.getByText('Nenhum vídeo encontrado')).toBeDefined())
  })

  it('shows error when API fails', async () => {
    api.get.mockRejectedValue(new Error('network error'))
    renderPage()
    await waitFor(() =>
      expect(screen.getByText('Não foi possível carregar a lista de vídeos.')).toBeDefined()
    )
  })

  it('renders video list', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '1', original_name: 'video.mp4', status: 'completed', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => expect(screen.getByText('video.mp4')).toBeDefined())
    expect(screen.getByText('Concluído')).toBeDefined()
    expect(screen.getByText('Download')).toBeDefined()
  })

  it('renders processing status badge', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '2', original_name: 'proc.mp4', status: 'processing', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => expect(screen.getByText('Processando')).toBeDefined())
  })

  it('renders failed status badge', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '3', original_name: 'fail.mp4', status: 'failed', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => expect(screen.getByText('Falha')).toBeDefined())
  })

  it('renders pending status badge for unknown status', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '4', original_name: 'pend.mp4', status: 'pending', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => expect(screen.getByText('Aguardando')).toBeDefined())
  })

  it('opens delete modal on trash button click', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '1', original_name: 'video.mp4', status: 'pending', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => screen.getByText('video.mp4'))
    // trash button is the last button (after 'Atualizar')
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[buttons.length - 1])
    expect(screen.getByText('confirm-modal')).toBeDefined()
  })

  it('closes modal on cancel', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '1', original_name: 'video.mp4', status: 'pending', created_at: new Date().toISOString() }],
    })
    renderPage()
    await waitFor(() => screen.getByText('video.mp4'))
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[buttons.length - 1])
    await waitFor(() => screen.getByText('cancel-modal'))
    fireEvent.click(screen.getByText('cancel-modal'))
    expect(screen.queryByText('confirm-modal')).toBeNull()
  })

  it('deletes video on confirm', async () => {
    api.get.mockResolvedValue({
      data: [{ id: '1', original_name: 'video.mp4', status: 'pending', created_at: new Date().toISOString() }],
    })
    api.delete.mockResolvedValue({})
    renderPage()
    await waitFor(() => screen.getByText('video.mp4'))
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[buttons.length - 1])
    await waitFor(() => screen.getByText('confirm-modal'))
    await act(async () => fireEvent.click(screen.getByText('confirm-modal')))
    expect(api.delete).toHaveBeenCalledWith('/videos/1')
  })

  it('shows alert on delete failure', async () => {
    const alertMock = vi.spyOn(window, 'alert').mockImplementation(() => {})
    api.get.mockResolvedValue({
      data: [{ id: '1', original_name: 'video.mp4', status: 'pending', created_at: new Date().toISOString() }],
    })
    api.delete.mockRejectedValue(new Error('fail'))
    renderPage()
    await waitFor(() => screen.getByText('video.mp4'))
    const buttons = screen.getAllByRole('button')
    fireEvent.click(buttons[buttons.length - 1])
    await waitFor(() => screen.getByText('confirm-modal'))
    await act(async () => fireEvent.click(screen.getByText('confirm-modal')))
    await waitFor(() => expect(alertMock).toHaveBeenCalledWith('Erro ao excluir vídeo'))
    alertMock.mockRestore()
  })

  it('polls videos periodically via setInterval', async () => {
    const setIntervalSpy = vi.spyOn(global, 'setInterval')
    api.get.mockResolvedValue({ data: [] })
    renderPage()
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1))
    expect(setIntervalSpy).toHaveBeenCalledWith(expect.any(Function), 5000)
    setIntervalSpy.mockRestore()
  })

  it('fetches videos on Atualizar button click', async () => {
    api.get.mockResolvedValue({ data: [] })
    renderPage()
    await waitFor(() => expect(api.get).toHaveBeenCalledTimes(1))
    await act(async () => fireEvent.click(screen.getByText('Atualizar')))
    expect(api.get).toHaveBeenCalledTimes(2)
  })
})
