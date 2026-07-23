/**
 * Tests CoverFlowModal — vérifient que le lecteur reste stable sur l'item courant
 * même quand l'array `items` change (mutations like, réassociation, refetch).
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { act, fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { CoverFlowModal } from './CoverFlowModal'
import type { MediaItemRow } from '@/lib/api/types'

function makeItem(overrides: Partial<MediaItemRow>): MediaItemRow {
  return {
    basename: 'clip.mp4',
    file_path: '/media/clip.mp4',
    kind: 'clip',
    thumbnail_path: null,
    match_id: 'match-1',
    capture_end_utc: '2026-04-26T14:30:00Z',
    match_start_time: null,
    section: 'mine',
    owner_gamertag: 'me',
    map_name: 'Aquarius',
    mode_name: 'Slayer',
    liked: false,
    like_count: 0,
    likers: undefined,
    total_likers: undefined,
    ...overrides,
  }
}

const ITEM_A = makeItem({ basename: 'A.mp4', file_path: '/media/A.mp4', map_name: 'MapA' })
const ITEM_B = makeItem({ basename: 'B.mp4', file_path: '/media/B.mp4', map_name: 'MapB' })
const ITEM_C = makeItem({ basename: 'C.mp4', file_path: '/media/C.mp4', map_name: 'MapC' })

describe('CoverFlowModal — gestion erreur vidéo (MIME, codec, etc)', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche un message d\'erreur si la vidéo échoue à se charger (MIME 4)', () => {
    const { container } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    const video = container.querySelector('video')
    expect(video).toBeInTheDocument()

    // Simuler une erreur MIME (code 4 = MEDIA_ERR_SRC_NOT_SUPPORTED)
    Object.defineProperty(video, 'error', {
      value: { code: 4 },
      configurable: true,
    })
    act(() => {
      fireEvent.error(video!)
    })

    expect(screen.getByText(/Lecture impossible/)).toBeInTheDocument()
    expect(screen.getByText(/Format vidéo non supporté/)).toBeInTheDocument()
  })

  it('affiche un message spécifique pour erreur de décodage (code 3)', () => {
    const { container } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    const video = container.querySelector('video')!
    Object.defineProperty(video, 'error', { value: { code: 3 }, configurable: true })
    act(() => {
      fireEvent.error(video)
    })
    expect(screen.getByText(/Erreur de décodage/)).toBeInTheDocument()
  })

  it('affiche le basename de la vidéo en erreur pour aider l\'utilisateur', () => {
    const itemNamed = makeItem({ basename: 'broken-clip.mkv', file_path: '/x.mkv' })
    const { container } = renderWithProviders(
      <CoverFlowModal
        items={[itemNamed]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    const video = container.querySelector('video')!
    Object.defineProperty(video, 'error', { value: { code: 4 }, configurable: true })
    act(() => {
      fireEvent.error(video)
    })
    expect(screen.getByText('broken-clip.mkv')).toBeInTheDocument()
  })
})

describe('CoverFlowModal — stabilité de l\'item courant', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche l\'item au startIndex à l\'ouverture', () => {
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    // Le heading affiche map du current item (B)
    expect(screen.getByText(/MapB/)).toBeInTheDocument()
  })

  it('reste sur le même item quand les données du courant sont mises à jour (like)', () => {
    const onToggleLike = vi.fn()
    const { rerender } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={onToggleLike}
      />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Simuler la mutation: l'item B a maintenant liked=true (nouvel objet)
    const ITEM_B_LIKED = { ...ITEM_B, liked: true, like_count: 1 }
    rerender(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B_LIKED, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={onToggleLike}
      />,
    )
    // Toujours sur B, pas sur A ou C
    expect(screen.getByText(/MapB/)).toBeInTheDocument()
    expect(screen.queryByText(/MapA/)).not.toBeInTheDocument()
    expect(screen.queryByText(/MapC/)).not.toBeInTheDocument()
  })

  it('reste sur le même item si l\'array est réordonné (refetch après mutation)', () => {
    const { rerender } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Refetch retourne les items dans un ordre différent (B, C, A)
    rerender(
      <CoverFlowModal
        items={[ITEM_B, ITEM_C, ITEM_A]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    // Doit toujours afficher B (qui est maintenant à l'index 0 dans le nouvel array)
    expect(screen.getByText(/MapB/)).toBeInTheDocument()
  })

  it('reste sur le même item si réassociation modifie map_name (mais pas file_path)', () => {
    const { rerender } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Réassociation : B a maintenant map_name='NewMap'
    const ITEM_B_REASSOC = { ...ITEM_B, map_name: 'NewMap', match_id: 'match-99' }
    rerender(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B_REASSOC, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    // Doit afficher la nouvelle map (NewMap) car c'est toujours l'item B (même file_path)
    expect(screen.getByText(/NewMap/)).toBeInTheDocument()
    expect(screen.queryByText(/MapA/)).not.toBeInTheDocument()
    expect(screen.queryByText(/MapC/)).not.toBeInTheDocument()
  })

  it('garde la dernière position connue si l\'item disparaît temporairement', () => {
    const { rerender } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Pendant un refetch, l'array peut être temporairement vide ou amputé.
    // L'item B disparaît. Le composant doit gracefully gérer (pas de crash, pas de saut visuel brusque).
    rerender(
      <CoverFlowModal
        items={[ITEM_A, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    // Pas de crash, on affiche ce qu'on peut (l'item à la dernière position connue clampée)
    // C'est l'item C (index 1 dans le nouvel array)
    expect(screen.queryByText(/MapB/)).not.toBeInTheDocument()
  })

  it('change d\'item quand l\'utilisateur navigue avec les flèches (next)', () => {
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapA/)).toBeInTheDocument()

    // Flèche droite → item suivant
    fireEvent.keyDown(window, { key: 'ArrowRight' })

    expect(screen.getByText(/MapB/)).toBeInTheDocument()
  })

  it('appelle onToggleLike avec l\'item courant quand on clique sur le bouton like', () => {
    const onToggleLike = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={onToggleLike}
      />,
    )

    const likeButton = screen.getByRole('button', { name: /Liker|Retirer le like/ })
    fireEvent.click(likeButton)

    expect(onToggleLike).toHaveBeenCalledTimes(1)
    expect(onToggleLike).toHaveBeenCalledWith(expect.objectContaining({ file_path: ITEM_B.file_path }))
  })

  it('séquence complète mutation like (3 rerenders consécutifs) : reste sur item B', () => {
    const onToggleLike = vi.fn()
    const baseProps = {
      startIndex: 1,
      onClose: vi.fn(),
      onToggleLike,
    }

    // Render initial : items [A, B, C], on est sur B
    const { rerender } = renderWithProviders(
      <CoverFlowModal items={[ITEM_A, ITEM_B, ITEM_C]} {...baseProps} />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // RERENDER 1 — onMutate : item B est cloné avec liked=true
    const B_OPTIMISTIC = { ...ITEM_B, liked: true, like_count: 1 }
    rerender(<CoverFlowModal items={[ITEM_A, B_OPTIMISTIC, ITEM_C]} {...baseProps} />)
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // RERENDER 2 — onSuccess : item B avec données serveur (likers, total_likers)
    const B_FROM_SERVER = { ...ITEM_B, liked: true, like_count: 1, likers: ['me'], total_likers: 1 }
    rerender(<CoverFlowModal items={[ITEM_A, B_FROM_SERVER, ITEM_C]} {...baseProps} />)
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // RERENDER 3 — onSettled invalidate puis refetch : nouvel array complet
    // (potentiellement réordonné, ou avec items "frais" du backend)
    const A_FRESH = { ...ITEM_A }
    const B_FRESH = { ...ITEM_B, liked: true, like_count: 1 }
    const C_FRESH = { ...ITEM_C }
    rerender(<CoverFlowModal items={[A_FRESH, B_FRESH, C_FRESH]} {...baseProps} />)

    // CRUCIAL : après tout ce ballet, on doit toujours être sur B
    expect(screen.getByText(/MapB/)).toBeInTheDocument()
    expect(screen.queryByText(/MapA · /)).not.toBeInTheDocument()
    expect(screen.queryByText(/MapC · /)).not.toBeInTheDocument()
  })

  it('refetch réordonne items après like : reste sur B même si l\'index change', () => {
    const baseProps = {
      startIndex: 1,
      onClose: vi.fn(),
      onToggleLike: vi.fn(),
    }
    const { rerender } = renderWithProviders(
      <CoverFlowModal items={[ITEM_A, ITEM_B, ITEM_C]} {...baseProps} />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Refetch retourne items dans un ordre TOTALEMENT différent : [B, C, A]
    // (B a été remonté car récemment liké, par exemple)
    const B_REORDERED = { ...ITEM_B, liked: true, like_count: 1 }
    rerender(
      <CoverFlowModal items={[B_REORDERED, ITEM_C, ITEM_A]} {...baseProps} />,
    )

    // On doit toujours afficher B (qui est à index 0 maintenant)
    expect(screen.getByText(/MapB/)).toBeInTheDocument()
  })

  it('refetch retourne moins d\'items mais B est toujours dedans : reste sur B', () => {
    const baseProps = {
      startIndex: 1,
      onClose: vi.fn(),
      onToggleLike: vi.fn(),
    }
    const { rerender } = renderWithProviders(
      <CoverFlowModal items={[ITEM_A, ITEM_B, ITEM_C]} {...baseProps} />,
    )
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Refetch enlève A (par exemple parce qu'il a été supprimé) : [B, C]
    rerender(
      <CoverFlowModal items={[ITEM_B, ITEM_C]} {...baseProps} />,
    )

    expect(screen.getByText(/MapB/)).toBeInTheDocument()
  })

  it('appelle onReassociate avec l\'item courant', () => {
    const onReassociate = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={1}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onReassociate={onReassociate}
      />,
    )

    const reassocButton = screen.getByRole('button', { name: /Réassocier/ })
    fireEvent.click(reassocButton)

    expect(onReassociate).toHaveBeenCalledTimes(1)
    expect(onReassociate).toHaveBeenCalledWith(expect.objectContaining({ file_path: ITEM_B.file_path }))
  })

  it('affiche "Associer" (pas "Réassocier") quand le média n\'a pas de match associé', () => {
    const ITEM_NO_MATCH = makeItem({ file_path: '/media/orphan.mp4', match_id: null, map_name: null })
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_NO_MATCH]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        onReassociate={vi.fn()}
      />,
    )

    expect(screen.getByRole('button', { name: 'Associer' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Réassocier' })).not.toBeInTheDocument()
  })
})

describe('CoverFlowModal — icône "ouvrir le match" header', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('affiche l\'icône cliquable vers la page du match quand l\'item a un match_id et playerSlug est fourni', () => {
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        playerSlug="myGT"
      />,
    )
    const link = screen.getByRole('link', { name: /Ouvrir.*match/ })
    expect(link).toBeInTheDocument()
    // Lien PLEINE PAGE title-scoped (lot 2-C) : titleSlug = défaut store 'halo_infinite'.
    expect(link).toHaveAttribute('href', `/t/halo_infinite/players/myGT/matches/${ITEM_A.match_id}`)
  })

  it('masque l\'icône quand currentMatchId === item.match_id', () => {
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        playerSlug="myGT"
        currentMatchId={ITEM_A.match_id ?? undefined}
      />,
    )
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
  })

  it('masque l\'icône quand item n\'a pas de match associé', () => {
    const ITEM_NO_MATCH = makeItem({ file_path: '/orphan.mp4', match_id: null })
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_NO_MATCH]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        playerSlug="myGT"
      />,
    )
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
  })

  it('le href suit la navigation entre items du carrousel', () => {
    const ITEM_X = makeItem({ file_path: '/x.mp4', match_id: 'match-X' })
    const ITEM_Y = makeItem({ file_path: '/y.mp4', match_id: 'match-Y' })
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_X, ITEM_Y]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        playerSlug="GT"
      />,
    )
    // Lien PLEINE PAGE title-scoped (lot 2-C) : titleSlug = défaut store 'halo_infinite'.
    expect(screen.getByRole('link', { name: /Ouvrir.*match/ })).toHaveAttribute('href', '/t/halo_infinite/players/GT/matches/match-X')

    fireEvent.keyDown(window, { key: 'ArrowRight' })

    expect(screen.getByRole('link', { name: /Ouvrir.*match/ })).toHaveAttribute('href', '/t/halo_infinite/players/GT/matches/match-Y')
  })

  it('si onOpenMatch est fourni, le rendu est un <button> qui appelle le callback avec le match_id courant', () => {
    const ITEM_X = makeItem({ file_path: '/x.mp4', match_id: 'match-X' })
    const ITEM_Y = makeItem({ file_path: '/y.mp4', match_id: 'match-Y' })
    const onOpenMatch = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_X, ITEM_Y]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        playerSlug="GT"
        onOpenMatch={onOpenMatch}
      />,
    )
    // Avec onOpenMatch, plus de <a> mais un <button>.
    expect(screen.queryByRole('link', { name: /Ouvrir.*match/ })).not.toBeInTheDocument()
    screen.getByRole('button', { name: /Ouvrir.*match/ }).click()
    expect(onOpenMatch).toHaveBeenLastCalledWith('match-X')

    fireEvent.keyDown(window, { key: 'ArrowRight' })

    screen.getByRole('button', { name: /Ouvrir.*match/ }).click()
    expect(onOpenMatch).toHaveBeenLastCalledWith('match-Y')
    expect(onOpenMatch).toHaveBeenCalledTimes(2)
  })
})

// ─── Bouton autoChain — visible uniquement si onToggleAutoChain fourni ─────

describe('CoverFlowModal — bouton autoChain', () => {
  afterEach(() => {
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  it('ne rend PAS le bouton enchaînement si onToggleAutoChain est absent', () => {
    // Cas régression : home rail / match tab oubliaient de passer la prop,
    // l'utilisateur n'avait plus le bouton ni l'autoplay.
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
      />,
    )
    expect(screen.queryByRole('button', { name: /Enchaîner|Chain/i })).not.toBeInTheDocument()
  })

  it('rend le bouton enchaînement quand onToggleAutoChain est fourni', () => {
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        autoChain={false}
        onToggleAutoChain={vi.fn()}
      />,
    )
    expect(screen.getByRole('button', { name: /Enchaîner|Chain/i })).toBeInTheDocument()
  })

  it('appelle onToggleAutoChain au clic sur le bouton', () => {
    const onToggleAutoChain = vi.fn()
    renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        autoChain={false}
        onToggleAutoChain={onToggleAutoChain}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: /Enchaîner|Chain/i }))
    expect(onToggleAutoChain).toHaveBeenCalledTimes(1)
  })
})

// ─── Enchaînement automatique EFFECTIF (autoChain) ─────────────────────────
// Régression juin 2026 : (A) `canAdvanceFurther = !canNext || hasNextPage` (le `!`
// désactivait l'enchaînement pour tout item non-dernier) ; (B) `handleVideoEnded`
// mémoïsé (useCallback) figeait un `navigate` périmé → blocage dès le 2e clip.
// Les tests "bouton autoChain" ne couvraient que le rendu/clic, jamais l'avance.

describe('CoverFlowModal — enchaînement automatique effectif', () => {
  afterEach(() => {
    vi.useRealTimers()
    act(() => {
      useAppShellStore.setState({ locale: 'fr' })
    })
  })

  // La vidéo centrale est la seule avec l'attribut `controls` (controls={isCenter}) ;
  // après une avance, ce n'est plus le premier <video> du DOM.
  function centerVideo(container: HTMLElement): HTMLVideoElement {
    return container.querySelector('video[controls]') as HTMLVideoElement
  }

  it('enchaîne les clips à la fin de chaque vidéo, jusqu\'au dernier inclus', () => {
    vi.useFakeTimers()
    const { container } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        autoChain
        onToggleAutoChain={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapA/)).toBeInTheDocument()

    // Fin de A → enchaîne sur B
    act(() => {
      fireEvent.ended(centerVideo(container))
    })
    act(() => {
      vi.advanceTimersByTime(500) // ANIM_MS : libère animatingRef
    })
    expect(screen.getByText(/MapB/)).toBeInTheDocument()

    // Fin de B → C (échouait avec le useCallback périmé : re-navigation vers B)
    act(() => {
      fireEvent.ended(centerVideo(container))
    })
    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(screen.getByText(/MapC/)).toBeInTheDocument()

    // Fin de C : dernier item, pas de page suivante → reste sur C
    act(() => {
      fireEvent.ended(centerVideo(container))
    })
    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(screen.getByText(/MapC/)).toBeInTheDocument()
  })

  it('n\'enchaîne pas quand autoChain est désactivé', () => {
    vi.useFakeTimers()
    const { container } = renderWithProviders(
      <CoverFlowModal
        items={[ITEM_A, ITEM_B, ITEM_C]}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        autoChain={false}
        onToggleAutoChain={vi.fn()}
      />,
    )
    expect(screen.getByText(/MapA/)).toBeInTheDocument()

    act(() => {
      fireEvent.ended(container.querySelector('video') as HTMLVideoElement)
    })
    act(() => {
      vi.advanceTimersByTime(500)
    })
    expect(screen.getByText(/MapA/)).toBeInTheDocument()
  })

  it('enchaîne les images après le délai d\'auto-avance, puis s\'arrête sur la dernière', () => {
    vi.useFakeTimers()
    const imgs = [
      makeItem({ kind: 'image', basename: 'A.png', file_path: '/i/A.png', map_name: 'ImgA' }),
      makeItem({ kind: 'image', basename: 'B.png', file_path: '/i/B.png', map_name: 'ImgB' }),
      makeItem({ kind: 'image', basename: 'C.png', file_path: '/i/C.png', map_name: 'ImgC' }),
    ]
    renderWithProviders(
      <CoverFlowModal
        items={imgs}
        startIndex={0}
        onClose={vi.fn()}
        onToggleLike={vi.fn()}
        autoChain
        onToggleAutoChain={vi.fn()}
      />,
    )
    expect(screen.getByText(/ImgA/)).toBeInTheDocument()

    // Délai image (7000ms) + ANIM_MS (500ms) pour libérer animatingRef.
    act(() => {
      vi.advanceTimersByTime(7500)
    })
    expect(screen.getByText(/ImgB/)).toBeInTheDocument()

    act(() => {
      vi.advanceTimersByTime(7500)
    })
    expect(screen.getByText(/ImgC/)).toBeInTheDocument()

    // Dernière image → plus de timer d'auto-avance (canAdvanceFurther = false)
    act(() => {
      vi.advanceTimersByTime(7500)
    })
    expect(screen.getByText(/ImgC/)).toBeInTheDocument()
  })
})
