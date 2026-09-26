/**
 * Tests unitaires — MediaThumbnailCard et son bouton « J'aime » (MediaViewer.tsx).
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { MediaThumbnailCard } from './MediaViewer'
import type { MediaItemRow } from '@/lib/api/types'

// ─── MediaThumbnailCard — fallback "Pas de match associé" ───────────────────

function makeItem(overrides: Partial<MediaItemRow>): MediaItemRow {
  return {
    basename: 'shot.png',
    file_path: '/media/shot.png',
    kind: 'screenshot',
    thumbnail_path: '/media/shot-thumb.png',
    match_id: 'match-1',
    capture_end_utc: '2026-04-26T14:30:00Z',
    match_start_time: null,
    section: 'mine',
    owner_gamertag: 'me',
    map_name: 'Aquarius',
    mode_name: 'Slayer',
    liked: false,
    like_count: 0,
    ...overrides,
  }
}

describe('MediaThumbnailCard — affichage map / fallback match', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche le map_name quand la map est connue', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: 'Aquarius', match_id: 'match-1' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('Aquarius')).toBeInTheDocument()
    expect(screen.queryByText(/Pas de match associé/)).not.toBeInTheDocument()
  })

  it('affiche "Pas de match associé" (FR) quand match_id est null', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: null, match_id: null })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('Pas de match associé')).toBeInTheDocument()
  })

  it('affiche "No associated match" (EN) quand match_id est null et locale=en', () => {
    act(() => {
      useAppShellStore.setState({ locale: 'en' })
    })
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: null, match_id: null })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('No associated match')).toBeInTheDocument()
  })

  it('n\'affiche ni map_name ni fallback quand match_id existe mais map_name est null', () => {
    // Cas typique : médias depuis MatchMediaTab (match_id forcé, map_name absent)
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: null, match_id: 'match-42' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.queryByText(/Pas de match associé/)).not.toBeInTheDocument()
    expect(screen.queryByText(/No associated match/)).not.toBeInTheDocument()
  })

  it('n\'affiche pas "Carte inconnue" sur la page du match courant (currentMatchId === match_id, map null)', () => {
    // Régression : sur la page d'un match, ses propres médias (map_name absent
    // car Q24 ne joint pas match_registry) ne doivent PAS afficher "Carte
    // inconnue" — la map est déjà dans l'en-tête du match.
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: null, match_id: 'match-42' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        currentMatchId="match-42"
      />,
    )
    expect(screen.queryByText(/Carte inconnue/)).not.toBeInTheDocument()
  })

  it('affiche "Carte inconnue" dans la galerie (média d\'un autre match, map null)', () => {
    // Comportement galerie conservé : hors de la page du match (currentMatchId
    // différent), une map non résolue affiche bien le fallback.
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ map_name: null, match_id: 'match-42' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        currentMatchId="other-match"
      />,
    )
    expect(screen.getByText(/Carte inconnue/)).toBeInTheDocument()
  })
})

describe('MediaThumbnailCard — icône "ouvrir le match"', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche un lien (icône) vers la page du match quand item.match_id existe et playerSlug est fourni', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        playerSlug="myGT"
      />,
    )
    const link = screen.getByRole('link', { name: /Ouvrir.*match/ })
    expect(link).toBeInTheDocument()
    // Lien PLEINE PAGE title-scoped (lot 2-C) : titleSlug = défaut store 'halo_infinite'.
    expect(link).toHaveAttribute('href', '/t/halo_infinite/players/myGT/matches/match-42')
  })

  it('masque l\'icône quand currentMatchId === item.match_id', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        playerSlug="myGT"
        currentMatchId="match-42"
      />,
    )
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
  })

  it('masque l\'icône quand playerSlug est absent', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
  })

  it('le clic sur l\'icône ne déclenche pas onOpen (stopPropagation)', () => {
    const onOpen = vi.fn()
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={onOpen}
        playerSlug="myGT"
      />,
    )
    const link = screen.getByRole('link', { name: /Ouvrir.*match/ })
    // preventDefault() pour éviter la navigation jsdom, stopPropagation() est appelé par le composant
    link.addEventListener('click', (e) => e.preventDefault())
    link.click()
    expect(onOpen).not.toHaveBeenCalled()
  })

  it('en locale=en, l\'aria-label passe à "Open"', () => {
    act(() => {
      useAppShellStore.setState({ locale: 'en' })
    })
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        playerSlug="myGT"
      />,
    )
    expect(screen.getByRole('link', { name: /Open.*match/ })).toBeInTheDocument()
  })

  it('si onOpenMatch est fourni, le rendu est un <button> et le callback reçoit match_id (pas de lien <a>)', () => {
    const onOpenMatch = vi.fn()
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        playerSlug="myGT"
        onOpenMatch={onOpenMatch}
      />,
    )
    // Quand onOpenMatch est fourni, on ne doit plus rendre un <a> mais un <button>.
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
    const btn = screen.getByRole('button', { name: /Ouvrir.*match/ })
    btn.click()
    expect(onOpenMatch).toHaveBeenCalledWith('match-42')
  })

  it('le clic sur le <button> onOpenMatch n\'appelle pas onOpen (stopPropagation)', () => {
    const onOpen = vi.fn()
    const onOpenMatch = vi.fn()
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-42', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={onOpen}
        playerSlug="myGT"
        onOpenMatch={onOpenMatch}
      />,
    )
    screen.getByRole('button', { name: /Ouvrir.*match/ }).click()
    expect(onOpen).not.toHaveBeenCalled()
    expect(onOpenMatch).toHaveBeenCalledTimes(1)
  })
})

describe('MediaThumbnailCard — troncature du nom de map', () => {
  it('n\'altère pas un nom de 13 caractères ou moins', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-1', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('Aquarius')).toBeInTheDocument()
  })

  it('tronque les noms > 13 chars en gardant les 12 premiers + "..."', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-1', map_name: 'Behemoth Forge Edition' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('Behemoth For...')).toBeInTheDocument()
    expect(screen.queryByText('Behemoth Forge Edition')).not.toBeInTheDocument()
  })

  it('n\'altère pas un nom de exactement 13 caractères', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-1', map_name: 'Streets-Forge' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    // "Streets-Forge" = 13 chars exactement → pas de troncature
    expect(screen.getByText('Streets-Forge')).toBeInTheDocument()
  })
})

describe('MediaThumbnailCard — lien "+ Associer"', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche "+ Associer" quand match_id est null et onAssociate est fourni', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: null, map_name: null })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        onAssociate={vi.fn()}
      />,
    )
    // L'article wrapper a role="button" — on cherche par texte exact pour cibler le bouton "+ Associer"
    expect(screen.getByText('+ Associer')).toBeInTheDocument()
    // Le fallback italic doit être remplacé par le bouton actionnable
    expect(screen.queryByText(/Pas de match associé/)).not.toBeInTheDocument()
  })

  it('garde le fallback italic "Pas de match associé" si onAssociate non fourni', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: null, map_name: null })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
      />,
    )
    expect(screen.getByText('Pas de match associé')).toBeInTheDocument()
    expect(screen.queryByText('+ Associer')).not.toBeInTheDocument()
  })

  it('cliquer sur "+ Associer" appelle onAssociate avec l\'item, pas onOpen', () => {
    const onAssociate = vi.fn()
    const onOpen = vi.fn()
    const item = makeItem({ match_id: null, map_name: null, file_path: '/orphan.mp4' })
    renderWithProviders(
      <MediaThumbnailCard
        item={item}
        onToggleLike={vi.fn()}
        onOpen={onOpen}
        onAssociate={onAssociate}
      />,
    )
    const button = screen.getByText('+ Associer')
    button.click()
    expect(onAssociate).toHaveBeenCalledTimes(1)
    expect(onAssociate).toHaveBeenCalledWith(expect.objectContaining({ file_path: '/orphan.mp4' }))
    expect(onOpen).not.toHaveBeenCalled()
  })

  it('n\'affiche pas "+ Associer" quand match_id existe (utiliser le lecteur pour réassocier)', () => {
    renderWithProviders(
      <MediaThumbnailCard
        item={makeItem({ match_id: 'match-1', map_name: 'Aquarius' })}
        onToggleLike={vi.fn()}
        onOpen={vi.fn()}
        onAssociate={vi.fn()}
      />,
    )
    expect(screen.queryByText('+ Associer')).not.toBeInTheDocument()
  })
})

// ─── Bouton « J'aime » : compteur + infobulle des personnes ─────────────────

describe("MediaThumbnailCard — infobulle des mentions « J'aime »", () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  function renderCard(overrides: Partial<MediaItemRow>) {
    return renderWithProviders(
      <MediaThumbnailCard item={makeItem(overrides)} onToggleLike={vi.fn()} onOpen={vi.fn()} />,
    )
  }

  it("n'affiche aucune infobulle quand personne n'a aimé", () => {
    const { container } = renderCard({ like_count: 0, total_likers: 0 })
    fireEvent.mouseOver(screen.getByRole('button', { name: 'Aimer' }).parentElement!)
    expect(container.ownerDocument.querySelector('[role="tooltip"]')).toBeNull()
  })

  it('énumère les noms et le reste au survol du cœur (FR)', () => {
    renderCard({ like_count: 5, liked: true, likers: ['Alice', 'Bob'], total_likers: 5 })
    const button = screen.getByRole('button', { name: "Retirer la mention J'aime" })
    fireEvent.mouseOver(button.parentElement!)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aimé par Alice, Bob et 3 autres')
  })

  it('énumère les noms en anglais quand la langue est EN', () => {
    act(() => {
      useAppShellStore.setState({ locale: 'en' })
    })
    renderCard({ like_count: 2, likers: ['Alice', 'Bob'], total_likers: 2 })
    fireEvent.mouseOver(screen.getByRole('button', { name: 'Like' }).parentElement!)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Liked by Alice and Bob')
  })

  it("accorde le singulier quand une seule autre personne n'est pas nommée", () => {
    renderCard({ like_count: 2, likers: ['Alice'], total_likers: 2 })
    fireEvent.mouseOver(screen.getByRole('button', { name: 'Aimer' }).parentElement!)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aimé par Alice et 1 autre')
  })

  it('se rabat sur un décompte quand les noms sont inconnus', () => {
    renderCard({ like_count: 3, total_likers: 3 })
    fireEvent.mouseOver(screen.getByRole('button', { name: 'Aimer' }).parentElement!)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aimé par 3 personnes')
  })

  it("s'ouvre aussi au focus clavier", () => {
    renderCard({ like_count: 1, likers: ['Alice'], total_likers: 1 })
    fireEvent.focus(screen.getByRole('button', { name: 'Aimer' }).parentElement!)
    expect(screen.getByRole('tooltip')).toHaveTextContent('Aimé par Alice')
  })

  it("n'affiche plus la ligne de noms sous la vignette", () => {
    renderCard({ like_count: 5, likers: ['Alice', 'Bob'], total_likers: 5 })
    expect(screen.queryByText(/Alice, Bob/)).not.toBeInTheDocument()
  })
})
