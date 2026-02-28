import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import ConfirmationModal from './ConfirmationModal'

describe('ConfirmationModal', () => {
  it('renders nothing when isOpen is false', () => {
    const { container } = render(
      <ConfirmationModal isOpen={false} onClose={vi.fn()} onConfirm={vi.fn()} title="Test" message="msg" />
    )
    expect(container.firstChild).toBeNull()
  })

  it('renders title and message when open', () => {
    render(
      <ConfirmationModal isOpen={true} onClose={vi.fn()} onConfirm={vi.fn()} title="Delete?" message="Are you sure?" />
    )
    expect(screen.getByText('Delete?')).toBeDefined()
    expect(screen.getByText('Are you sure?')).toBeDefined()
  })

  it('calls onClose when cancel button is clicked', () => {
    const onClose = vi.fn()
    render(
      <ConfirmationModal isOpen={true} onClose={onClose} onConfirm={vi.fn()} title="Title" message="msg" />
    )
    fireEvent.click(screen.getByText('Cancelar'))
    expect(onClose).toHaveBeenCalled()
  })

  it('calls onConfirm when confirm button is clicked', () => {
    const onConfirm = vi.fn()
    render(
      <ConfirmationModal isOpen={true} onClose={vi.fn()} onConfirm={onConfirm} title="Title" message="msg" />
    )
    fireEvent.click(screen.getByText('Confirmar'))
    expect(onConfirm).toHaveBeenCalled()
  })

  it('disables buttons when isLoading is true', () => {
    render(
      <ConfirmationModal isOpen={true} onClose={vi.fn()} onConfirm={vi.fn()} title="T" message="m" isLoading={true} />
    )
    const buttons = screen.getAllByRole('button')
    buttons.forEach(btn => expect(btn.disabled).toBe(true))
  })
})
