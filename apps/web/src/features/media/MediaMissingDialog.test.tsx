/**
 * Tests — MediaMissingDialog : liste des fichiers introuvables, fusion et oubli en deux temps,
 * ouverture de l'explorateur, FR et EN. Les hooks de requête sont doublés : la coquille
 * (MediaPickerDialog) et la logique de la fenêtre sont seules en jeu.
 */
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useAppShellStore } from '@/stores/appShellStore'
import type { MediaMergeCandidate, MediaMissingItem } from '@/lib/api/types'
import { MediaMissingDialog } from './MediaMissingDialog'
import * as queries from './missingQueries'

vi.mock('./missingQueries', () => ({
  useMissingMedia: vi.fn(),
  useMergeCandidates: vi.fn(),
  useMergeMissingMedia: vi.fn(),
  useForgetMissingMedia: vi.fn(),
  useRevealMedia: vi.fn(),
}))

const ITEMS: MediaMissingItem[] = [
  {
    basename: 'ancien.mp4',
    file_path: '/api/v1/files/ancien.mp4',
    kind: 'clip',
    thumbnail_path: '/api/v1/thumbs/ancien.webp',
    capture_end_utc: '2026-04-26T14:30:00Z',
  },
  { basename: 'autre.png', file_path: '/api/v1/files/autre.png', kind: 'screenshot' },
]

const CANDIDATES: MediaMergeCandidate[] = [
  {
    basename: 'nouveau.mp4',
    file_path: '/api/v1/files/nouveau.mp4',
    kind: 'clip',
    capture_end_utc: '2026-04-26T14:31:00Z',
    delta_seconds: 60,
  },
  { basename: 'voisin.mp4', file_path: '/api/v1/files/voisin.mp4', kind: 'clip', delta_seconds: 600 },
]

const mutations = {
  merge: vi.fn(),
  forget: vi.fn(),
  reveal: vi.fn(),
}

function listState(state: { isLoading?: boolean; isError?: boolean; items?: MediaMissingItem[] }) {
  vi.mocked(queries.useMissingMedia).mockReturnValue({
    data: state.isLoading || state.isError ? undefined : { items: state.items ?? [], root_unavailable: false, bulk_missing_pending: 0 },
    isLoading: state.isLoading ?? false,
    isError: state.isError ?? false,
  } as unknown as ReturnType<typeof queries.useMissingMedia>)
}

function setCapabilities(reveal: boolean) {
  useAppShellStore.setState({
    capabilities: testCapabilities(reveal ? desktopCapabilities() : {}),
  })
}

function renderDialog() {
  return renderWithProviders(<MediaMissingDialog playerSlug="p" onClose={() => {}} />)
}

function clickRow(name: string) {
  fireEvent.click(screen.getByText(name).closest('button') as HTMLButtonElement)
}

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr' })
  setCapabilities(false)
  Object.values(mutations).forEach((m) => m.mockReset())
  listState({ items: ITEMS })
  vi.mocked(queries.useMergeCandidates).mockReturnValue({
    data: { file_path: ITEMS[0].file_path, candidates: CANDIDATES },
    isLoading: false,
    isError: false,
  } as unknown as ReturnType<typeof queries.useMergeCandidates>)
  vi.mocked(queries.useMergeMissingMedia).mockReturnValue({
    mutate: mutations.merge,
    isPending: false,
  } as unknown as ReturnType<typeof queries.useMergeMissingMedia>)
  vi.mocked(queries.useForgetMissingMedia).mockReturnValue({
    mutate: mutations.forget,
    isPending: false,
  } as unknown as ReturnType<typeof queries.useForgetMissingMedia>)
  vi.mocked(queries.useRevealMedia).mockReturnValue({
    mutate: mutations.reveal,
    isPending: false,
  } as unknown as ReturnType<typeof queries.useRevealMedia>)
})

describe('MediaMissingDialog — vue liste', () => {
  it('liste les médias introuvables avec leur nom', () => {
    renderDialog()
    expect(screen.getByText('Fichiers introuvables')).toBeInTheDocument()
    expect(screen.getByText('ancien.mp4')).toBeInTheDocument()
    expect(screen.getByText('autre.png')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Fermer' })).toBeInTheDocument()
  })

  it('état vide', () => {
    listState({ items: [] })
    renderDialog()
    expect(screen.getByText('Aucun fichier introuvable.')).toBeInTheDocument()
  })

  it('état d’erreur', () => {
    listState({ isError: true })
    renderDialog()
    expect(screen.getByText('Erreur de chargement')).toBeInTheDocument()
  })

  it('état de chargement', () => {
    listState({ isLoading: true })
    renderDialog()
    expect(screen.getByText('Chargement…')).toBeInTheDocument()
  })

  it('sans sélection, aucun pied ; avec une sélection, les deux actions', () => {
    renderDialog()
    expect(screen.queryByRole('button', { name: 'Oublier' })).not.toBeInTheDocument()
    clickRow('ancien.mp4')
    expect(screen.getByRole('button', { name: 'Oublier' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Retrouver dans la galerie' })).toBeInTheDocument()
  })

  it('un second clic sur la ligne désélectionne', () => {
    renderDialog()
    clickRow('ancien.mp4')
    clickRow('ancien.mp4')
    expect(screen.queryByRole('button', { name: 'Oublier' })).not.toBeInTheDocument()
  })
})

describe('MediaMissingDialog — fusion en deux temps', () => {
  function openCandidates() {
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Retrouver dans la galerie' }))
  }

  it('retrouver ouvre les fiches candidates de la fiche choisie', () => {
    openCandidates()
    expect(queries.useMergeCandidates).toHaveBeenLastCalledWith('p', '/api/v1/files/ancien.mp4')
    expect(screen.getByText('Médias présents dans la galerie')).toBeInTheDocument()
    expect(screen.getByText('nouveau.mp4')).toBeInTheDocument()
    expect(screen.getByText('voisin.mp4')).toBeInTheDocument()
    expect(screen.getByText('±1 min')).toBeInTheDocument()
    expect(screen.getByText('±10 min')).toBeInTheDocument()
  })

  it('le premier clic sélectionne, rien n’est fondu avant la confirmation', () => {
    openCandidates()
    clickRow('nouveau.mp4')
    expect(screen.getByText('Fondre dans le média choisi ?')).toBeInTheDocument()
    expect(mutations.merge).not.toHaveBeenCalled()
  })

  it('confirmer fond la fiche introuvable dans la fiche choisie, puis revient à la liste', () => {
    mutations.merge.mockImplementation((_req, options) => options?.onSuccess?.())
    openCandidates()
    clickRow('nouveau.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Confirmer' }))
    expect(mutations.merge).toHaveBeenCalledTimes(1)
    expect(mutations.merge.mock.calls[0][0]).toEqual({
      file_path: '/api/v1/files/ancien.mp4',
      target_file_path: '/api/v1/files/nouveau.mp4',
    })
    expect(screen.getByText('Fichiers introuvables')).toBeInTheDocument()
    expect(screen.queryByText('Fondre dans le média choisi ?')).not.toBeInTheDocument()
  })

  it('annuler la sélection ne fond rien', () => {
    openCandidates()
    clickRow('nouveau.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(mutations.merge).not.toHaveBeenCalled()
    expect(screen.queryByText('Fondre dans le média choisi ?')).not.toBeInTheDocument()
  })

  it('le bouton retour revient à la liste', () => {
    openCandidates()
    fireEvent.click(screen.getByRole('button', { name: /Retour/ }))
    expect(screen.getByText('Fichiers introuvables')).toBeInTheDocument()
  })

  it('aucune fiche de même nature : message dédié', () => {
    vi.mocked(queries.useMergeCandidates).mockReturnValue({
      data: { file_path: ITEMS[0].file_path, candidates: [] },
      isLoading: false,
      isError: false,
    } as unknown as ReturnType<typeof queries.useMergeCandidates>)
    openCandidates()
    expect(screen.getByText('Aucun média de même nature dans la galerie.')).toBeInTheDocument()
  })
})

describe('MediaMissingDialog — oubli en deux temps', () => {
  it('le premier clic demande confirmation, le texte dit que l’original n’est pas touché', () => {
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Oublier' }))
    expect(screen.getByText('Oublier ce fichier ?')).toBeInTheDocument()
    expect(screen.getByText(/Le fichier d'origine n'est pas touché/)).toBeInTheDocument()
    expect(mutations.forget).not.toHaveBeenCalled()
  })

  it('confirmer oublie la fiche choisie et ferme la confirmation', () => {
    mutations.forget.mockImplementation((_req, options) => options?.onSuccess?.())
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Oublier' }))
    fireEvent.click(screen.getByRole('button', { name: 'Confirmer' }))
    expect(mutations.forget).toHaveBeenCalledTimes(1)
    expect(mutations.forget.mock.calls[0][0]).toEqual({ file_path: '/api/v1/files/ancien.mp4' })
    expect(screen.queryByText('Oublier ce fichier ?')).not.toBeInTheDocument()
  })

  it('annuler n’oublie rien', () => {
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Oublier' }))
    fireEvent.click(screen.getByRole('button', { name: 'Annuler' }))
    expect(mutations.forget).not.toHaveBeenCalled()
    expect(screen.queryByText('Oublier ce fichier ?')).not.toBeInTheDocument()
  })

  it('la liste vidée après un oubli reste ouverte sur l’état vide', () => {
    const { rerender } = renderDialog()
    listState({ items: [] })
    rerender(<MediaMissingDialog playerSlug="p" onClose={() => {}} />)
    expect(screen.getByText('Fichiers introuvables')).toBeInTheDocument()
    expect(screen.getByText('Aucun fichier introuvable.')).toBeInTheDocument()
  })
})

describe('MediaMissingDialog — explorateur de fichiers', () => {
  it('sans la capacité, aucun bouton « Afficher dans l’explorateur »', () => {
    renderDialog()
    expect(screen.queryByRole('button', { name: "Afficher dans l'explorateur" })).not.toBeInTheDocument()
  })

  it('avec la capacité, un bouton par ligne ouvre l’explorateur sur la fiche de la ligne', () => {
    setCapabilities(true)
    renderDialog()
    const buttons = screen.getAllByRole('button', { name: "Afficher dans l'explorateur" })
    expect(buttons).toHaveLength(ITEMS.length)
    fireEvent.click(buttons[1])
    expect(mutations.reveal).toHaveBeenCalledWith({ file_path: '/api/v1/files/autre.png' })
    // Le bouton ne sélectionne pas la ligne.
    expect(screen.queryByRole('button', { name: 'Oublier' })).not.toBeInTheDocument()
  })
})

describe('MediaMissingDialog — anglais', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'en' })
  })

  it('liste, actions et état vide sans résidu français', () => {
    renderDialog()
    expect(screen.getByText('Missing files')).toBeInTheDocument()
    expect(screen.queryByText('Fichiers introuvables')).not.toBeInTheDocument()
    clickRow('ancien.mp4')
    expect(screen.getByRole('button', { name: 'Find in the gallery' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Forget' })).toBeInTheDocument()
  })

  it('fusion : titre de confirmation en anglais', () => {
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Find in the gallery' }))
    expect(screen.getByText('Media in the gallery')).toBeInTheDocument()
    clickRow('nouveau.mp4')
    expect(screen.getByText('Merge into the chosen media?')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm' })).toBeInTheDocument()
  })

  it('oubli : texte de confirmation en anglais', () => {
    renderDialog()
    clickRow('ancien.mp4')
    fireEvent.click(screen.getByRole('button', { name: 'Forget' }))
    expect(screen.getByText('Forget this file?')).toBeInTheDocument()
    expect(screen.getByText(/The original file is not touched/)).toBeInTheDocument()
  })

  it('état vide en anglais', () => {
    listState({ items: [] })
    renderDialog()
    expect(screen.getByText('No missing file.')).toBeInTheDocument()
  })
})
