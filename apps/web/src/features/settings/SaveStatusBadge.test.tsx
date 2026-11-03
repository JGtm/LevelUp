/**
 * Tests — retour d'enregistrement des Paramètres (badge flottant, motif commun à tous les onglets).
 */
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest'
import { act, render, screen } from '@testing-library/react'
import { useAppShellStore } from '@/stores/appShellStore'
import { SaveStatusBadge } from './SaveStatusBadge'
import { flashSaveStatus, useSaveFeedbackStore } from './saveFeedbackStore'

describe('SaveStatusBadge', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    useAppShellStore.setState({ locale: 'fr' })
  })
  afterEach(() => {
    useSaveFeedbackStore.getState().clear()
    vi.useRealTimers()
  })

  it('ne montre rien au repos', () => {
    render(<SaveStatusBadge />)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('confirme un enregistrement 2 s', () => {
    render(<SaveStatusBadge />)
    act(() => flashSaveStatus('saved'))
    expect(screen.getByRole('status')).toHaveTextContent('✓ Enregistré')
    act(() => void vi.advanceTimersByTime(1999))
    expect(screen.getByRole('status')).toBeInTheDocument()
    act(() => void vi.advanceTimersByTime(2))
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it("garde l'échec 4 s, dans la langue de la page", () => {
    useAppShellStore.setState({ locale: 'en' })
    render(<SaveStatusBadge />)
    act(() => flashSaveStatus('error'))
    expect(screen.getByRole('alert')).toHaveTextContent('✗ Save failed')
    act(() => void vi.advanceTimersByTime(3999))
    expect(screen.getByRole('alert')).toBeInTheDocument()
    act(() => void vi.advanceTimersByTime(2))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('un nouvel enregistrement remplace le badge et relance le délai', () => {
    render(<SaveStatusBadge />)
    act(() => flashSaveStatus('error'))
    act(() => void vi.advanceTimersByTime(1500))
    act(() => flashSaveStatus('saved'))
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
    act(() => void vi.advanceTimersByTime(1999))
    expect(screen.getByRole('status')).toBeInTheDocument()
  })
})
