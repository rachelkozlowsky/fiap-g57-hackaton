import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import UploadPage from './UploadPage'

const mockNavigate = vi.fn()
vi.mock('react-router-dom', async (importOriginal) => {
  const real = await importOriginal()
  return { ...real, useNavigate: () => mockNavigate }
})

vi.mock('../components/Navbar', () => ({ default: () => <div>Navbar</div> }))

vi.mock('../services/api', () => ({
  default: {
    post: vi.fn(),
    interceptors: { request: { use: vi.fn() }, response: { use: vi.fn() } },
  },
}))

import api from '../services/api'

const renderPage = () => render(<MemoryRouter><UploadPage /></MemoryRouter>)

describe('UploadPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders upload area and title', () => {
    renderPage()
    expect(screen.getByText('Upload de Vídeo')).toBeDefined()
    expect(screen.getByText('MP4, AVI, MKV (Max 500MB)')).toBeDefined()
  })

  it('shows error when submitting without file', async () => {
    renderPage()
    // button is disabled when no file; submit the form directly
    await act(async () =>
      fireEvent.submit(screen.getByRole('button', { name: /cancelar/i }).closest('form'))
    )
    expect(screen.getByText('Selecione um arquivo de vídeo')).toBeDefined()
  })

  it('shows file name and size after selecting a file', () => {
    renderPage()
    const file = new File(['content'], 'video.mp4', { type: 'video/mp4' })
    const input = document.getElementById('video-input')
    fireEvent.change(input, { target: { files: [file] } })
    expect(screen.getByText('video.mp4')).toBeDefined()
  })

  it('navigates to / after successful upload', async () => {
    api.post.mockResolvedValueOnce({})
    renderPage()
    const file = new File(['content'], 'video.mp4', { type: 'video/mp4' })
    const input = document.getElementById('video-input')
    fireEvent.change(input, { target: { files: [file] } })
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /fazer upload/i }))
    )
    await waitFor(() => expect(mockNavigate).toHaveBeenCalledWith('/'))
  })

  it('shows error on upload failure with API message', async () => {
    api.post.mockRejectedValueOnce({ response: { data: { message: 'Arquivo muito grande' } } })
    renderPage()
    const file = new File(['content'], 'video.mp4', { type: 'video/mp4' })
    const input = document.getElementById('video-input')
    fireEvent.change(input, { target: { files: [file] } })
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /fazer upload/i }))
    )
    await waitFor(() => expect(screen.getByText('Arquivo muito grande')).toBeDefined())
  })

  it('shows fallback error when no API message', async () => {
    api.post.mockRejectedValueOnce(new Error('network'))
    renderPage()
    const file = new File(['content'], 'video.mp4', { type: 'video/mp4' })
    const input = document.getElementById('video-input')
    fireEvent.change(input, { target: { files: [file] } })
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: /fazer upload/i }))
    )
    await waitFor(() => expect(screen.getByText('Erro ao fazer upload do vídeo')).toBeDefined())
  })

  it('navigates to / on Cancelar button click', () => {
    renderPage()
    fireEvent.click(screen.getByRole('button', { name: /cancelar/i }))
    expect(mockNavigate).toHaveBeenCalledWith('/')
  })
})
