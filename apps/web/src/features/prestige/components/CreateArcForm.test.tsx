/**
 * CreateArcForm — un refus de création s'affiche par son code (FR + EN), jamais par le texte
 * du serveur (messages fixes en anglais, registre sécurité C7-1).
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'

const mockShellState = { locale: 'fr' as 'fr' | 'en' }
vi.mock('@/stores/appShellStore', () => ({
  useAppShellStore: <T,>(selector: (s: typeof mockShellState) => T) => selector(mockShellState),
}))

const mockCreate = { mutate: vi.fn(), isPending: false, error: null as unknown }
vi.mock('../hooks', () => ({
  useCreateArc: () => mockCreate,
}))

import { CreateArcForm } from './CreateArcForm'

describe('CreateArcForm — refus de création', () => {
  beforeEach(() => {
    cleanup()
    mockShellState.locale = 'fr'
    mockCreate.error = null
  })

  it('valeurs refusées : message traduit en français', () => {
    mockCreate.error = { code: 'invalid_input', message: 'invalid input' }
    render(<CreateArcForm userId="u1" titleSlug="halo_infinite" />)
    expect(screen.getByText('Vérifie les valeurs saisies.')).toBeInTheDocument()
    expect(screen.queryByText('invalid input')).not.toBeInTheDocument()
  })

  it('code inconnu : message générique, jamais le texte du serveur', () => {
    mockCreate.error = { code: 'code_inconnu', message: 'SELECT secret' }
    render(<CreateArcForm userId="u1" titleSlug="halo_infinite" />)
    expect(screen.getByText('Création impossible. Réessaie dans un instant.')).toBeInTheDocument()
    expect(screen.queryByText(/SELECT/)).not.toBeInTheDocument()
  })

  it('message en anglais quand la langue est EN', () => {
    mockShellState.locale = 'en'
    mockCreate.error = { code: 'invalid_input', message: 'invalid input' }
    render(<CreateArcForm userId="u1" titleSlug="halo_infinite" />)
    expect(screen.getByText('Check the values entered.')).toBeInTheDocument()
  })
})
