/**
 * Tests — MediaMatchPicker : i18n complet FR+EN de la popup de réassociation
 * (GH2-B7). Le dictionnaire i18n-modals.ts (matchPicker) existait mais n'était
 * PAS câblé : la popup était FR en dur et le message d'erreur empruntait une
 * clé leaderboard. Ces tests comparent les deux locales sur le chemin touché.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { MediaMatchPicker } from './MediaMatchPicker'
import * as queries from './queries'

vi.mock('./queries', () => ({
  useMediaMatchCandidates: vi.fn(),
  useAssociateMediaToMatch: vi.fn(() => ({ mutate: vi.fn(), isPending: false })),
}))

const mockCandidates = vi.mocked(queries.useMediaMatchCandidates)

function mockQueryState(state: { isLoading?: boolean; isError?: boolean; data?: unknown }) {
  mockCandidates.mockReturnValue({
    data: state.data,
    isLoading: state.isLoading ?? false,
    isError: state.isError ?? false,
  } as ReturnType<typeof queries.useMediaMatchCandidates>)
}

function renderPicker() {
  return renderWithProviders(
    <MediaMatchPicker playerSlug="p" filePath="/clips/x.mp4" onClose={() => {}} />,
  )
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  mockCandidates.mockReset()
})

describe('MediaMatchPicker — i18n FR', () => {
  it('titre, fenêtre et état vide en français', () => {
    mockQueryState({ data: { candidates: [], capture_utc: null } })
    renderPicker()
    expect(screen.getByText('Réassocier ce média')).toBeInTheDocument()
    expect(screen.getByText('Fenêtre :')).toBeInTheDocument()
    expect(screen.getByText(/Aucun match trouvé dans cette fenêtre/)).toBeInTheDocument()
    expect(screen.getByText('0 match trouvé')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fermer' })).toBeInTheDocument()
  })

  it("message d'erreur de chargement en français (clé dédiée, pas de brut)", () => {
    mockQueryState({ isError: true })
    renderPicker()
    expect(screen.getByText('Erreur de chargement')).toBeInTheDocument()
  })
})

describe('MediaMatchPicker — i18n EN (GH2-B7)', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'en' })
  })

  it('titre, fenêtre et état vide en anglais — aucun résidu FR', () => {
    mockQueryState({ data: { candidates: [], capture_utc: null } })
    renderPicker()
    expect(screen.getByText('Reassociate this media')).toBeInTheDocument()
    expect(screen.getByText('Window:')).toBeInTheDocument()
    expect(screen.getByText(/No match found in this window/)).toBeInTheDocument()
    expect(screen.getByText('0 match found')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Close' })).toBeInTheDocument()
    // Aucun libellé FR résiduel.
    expect(screen.queryByText('Réassocier ce média')).not.toBeInTheDocument()
    expect(screen.queryByText('Fenêtre :')).not.toBeInTheDocument()
  })

  it("message d'erreur de chargement en anglais", () => {
    mockQueryState({ isError: true })
    renderPicker()
    expect(screen.getByText('Loading error')).toBeInTheDocument()
  })

  it("variante Associer (média sans match) : titre EN dédié", () => {
    mockQueryState({ data: { candidates: [], capture_utc: null } })
    renderWithProviders(
      <MediaMatchPicker
        playerSlug="p"
        filePath="/clips/x.mp4"
        onClose={() => {}}
        hasCurrentMatch={false}
      />,
    )
    expect(screen.getByText('Associate this media')).toBeInTheDocument()
  })
})
