import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { act } from 'react'

import { renderWithProviders } from '@/test/render-utils'

import { TopProgressBar } from './TopProgressBar'

let mockIsFetching = 0
let mockPathname = '/home'

vi.mock('@tanstack/react-query', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-query')>()
  return {
    ...actual,
    useIsFetching: () => mockIsFetching,
  }
})

vi.mock('@tanstack/react-router', () => ({
  useRouterState: (opts: { select: (s: { location: { pathname: string } }) => string }) =>
    opts.select({ location: { pathname: mockPathname } }),
}))

// Doivent correspondre aux constantes du composant.
const SETTLE_MS = 150
const MIN_VISIBLE_MS = 450
const FADE_OUT_MS = 250
const MAX_VISIBLE_MS = 8000
const INITIAL_PROGRESS = 30
const MAX_TRICKLE_PROGRESS = 99

function findBarFill(container: HTMLElement): HTMLElement | null {
  return container.querySelector('div[aria-hidden] > div') as HTMLElement | null
}

function getWidth(container: HTMLElement): number {
  const fill = findBarFill(container)
  if (!fill) return 0
  return parseFloat(fill.style.width)
}

describe('TopProgressBar', () => {
  beforeEach(() => {
    mockIsFetching = 0
    mockPathname = '/home'
    vi.useFakeTimers()
    // Math.random déterministe pour rendre le trickle prédictible (incrément × 0.75).
    vi.spyOn(Math, 'random').mockReturnValue(0.5)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it("ne rend rien quand aucune query n'est en cours et pathname stable", () => {
    const { container } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()
  })

  it('apparaît à INITIAL_PROGRESS quand une query pending démarre', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    const fill = findBarFill(container)
    expect(fill).not.toBeNull()
    expect(fill!.style.width).toBe(`${INITIAL_PROGRESS}%`)
    expect(fill!.style.opacity).toBe('1')
  })

  it('apparaît immédiatement sur changement de pathname (sans query en cours)', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    mockPathname = '/career'
    rerender(<TopProgressBar />)

    const fill = findBarFill(container)
    expect(fill).not.toBeNull()
    expect(fill!.style.width).toBe(`${INITIAL_PROGRESS}%`)
    expect(fill!.style.opacity).toBe('1')
  })

  it('progresse continuellement via le trickle tant que ça fetch', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    expect(getWidth(container)).toBe(INITIAL_PROGRESS)

    act(() => {
      vi.advanceTimersByTime(200)
    })
    const w1 = getWidth(container)
    expect(w1).toBeGreaterThan(INITIAL_PROGRESS)

    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    const w2 = getWidth(container)
    expect(w2).toBeGreaterThan(w1)
    expect(w2).toBeLessThanOrEqual(MAX_TRICKLE_PROGRESS)
  })

  it('plafonne à MAX_TRICKLE_PROGRESS et ne dépasse pas avant la complétion', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    // Avancer juste sous le timeout max.
    act(() => {
      vi.advanceTimersByTime(MAX_VISIBLE_MS - 1)
    })
    expect(getWidth(container)).toBeLessThanOrEqual(MAX_TRICKLE_PROGRESS)
    expect(getWidth(container)).toBeGreaterThanOrEqual(INITIAL_PROGRESS)
  })

  it('complète à 100 % quand pendingCount repasse à 0 (MIN_VISIBLE_MS respecté)', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    // Fetch instantané
    mockIsFetching = 0
    rerender(<TopProgressBar />)

    // SETTLE_MS (150) + completionDelay (MIN_VISIBLE_MS - SETTLE_MS = 300) = 450 ms
    expect(getWidth(container)).toBe(INITIAL_PROGRESS)

    act(() => {
      vi.advanceTimersByTime(MIN_VISIBLE_MS)
    })
    expect(getWidth(container)).toBe(100)
    expect(findBarFill(container)!.style.opacity).toBe('1')
  })

  it('complète après SETTLE_MS quand le fetch a duré > MIN_VISIBLE_MS', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    // Fetch long → on dépasse MIN_VISIBLE_MS
    act(() => {
      vi.advanceTimersByTime(1_000)
    })
    expect(getWidth(container)).toBeGreaterThan(INITIAL_PROGRESS)
    expect(getWidth(container)).toBeLessThanOrEqual(MAX_TRICKLE_PROGRESS)

    mockIsFetching = 0
    rerender(<TopProgressBar />)

    act(() => {
      vi.advanceTimersByTime(SETTLE_MS + 1)
    })
    expect(getWidth(container)).toBe(100)
  })

  it('disparaît du DOM après le fade-out', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    mockIsFetching = 0
    rerender(<TopProgressBar />)

    // SETTLE_MS (absorbé) + MIN_VISIBLE_MS (450) + FADE_OUT_MS (250) = 700 ms
    act(() => {
      vi.advanceTimersByTime(MIN_VISIBLE_MS + FADE_OUT_MS)
    })
    rerender(<TopProgressBar />)
    expect(container.firstChild).toBeNull()
  })

  it('reste affichée si plusieurs queries se chevauchent', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    mockIsFetching = 2
    rerender(<TopProgressBar />)
    mockIsFetching = 1
    rerender(<TopProgressBar />)
    expect(findBarFill(container)).not.toBeNull()

    act(() => {
      vi.advanceTimersByTime(2_000)
    })
    expect(getWidth(container)).toBeGreaterThan(INITIAL_PROGRESS)
    expect(getWidth(container)).toBeLessThanOrEqual(MAX_TRICKLE_PROGRESS)
  })

  it('redéclenche proprement après une 1re séquence terminée', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    mockIsFetching = 0
    rerender(<TopProgressBar />)
    act(() => {
      vi.advanceTimersByTime(MIN_VISIBLE_MS + FADE_OUT_MS)
    })
    rerender(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    // 2e séquence
    mockIsFetching = 1
    rerender(<TopProgressBar />)
    expect(getWidth(container)).toBe(INITIAL_PROGRESS)
  })

  it('utilise la couleur du thème via bg-sidebar-primary', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    mockIsFetching = 1
    rerender(<TopProgressBar />)

    const fill = findBarFill(container)
    expect(fill!.className).toContain('bg-sidebar-primary')
  })

  it('rend un wrapper de 4 px (h-1) — visible mais discret', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    mockIsFetching = 1
    rerender(<TopProgressBar />)

    const wrapper = container.querySelector('div[aria-hidden]') as HTMLElement
    expect(wrapper.className).toContain('h-1')
  })

  // --- Régressions : bugs identifiés au diagnostic du 2026-05-04 -------------

  it('se ferme sur changement de pathname même quand pendingCount reste à 0 (régression)', () => {
    // Avant fix : startBar() était déclenché par le pathname change mais completeBar()
    // n'était appelé que sur transition pendingCount > 0 → 0. Si la nouvelle page n'a
    // pas de queries pending (ou que des queries déjà en cache), la barre restait coincée
    // indéfiniment.
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    // Navigation vers une page sans aucune query pending.
    mockPathname = '/career'
    rerender(<TopProgressBar />)
    expect(getWidth(container)).toBe(INITIAL_PROGRESS)

    // Settle (150) puis completionDelay = MIN_VISIBLE_MS - 150 = 300, puis fade-out 250.
    act(() => {
      vi.advanceTimersByTime(MIN_VISIBLE_MS + FADE_OUT_MS)
    })
    rerender(<TopProgressBar />)
    expect(container.firstChild).toBeNull()
  })

  it('force la complétion après MAX_VISIBLE_MS si la query reste pending (filet de sécurité)', () => {
    // Avant fix : aucun timeout maximum. Si une query restait coincée en pending
    // (timeout réseau, fetch sans résolution), la barre ne se refermait jamais.
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    // Avant le timeout max : barre toujours visible et sous le plafond.
    act(() => {
      vi.advanceTimersByTime(MAX_VISIBLE_MS - 100)
    })
    expect(getWidth(container)).toBeLessThanOrEqual(MAX_TRICKLE_PROGRESS)

    // Le timeout max fire, completeBar est appelé. completionDelay = 0 car elapsed > MIN_VISIBLE_MS.
    act(() => {
      vi.advanceTimersByTime(200)
    })
    expect(getWidth(container)).toBe(100)
  })

  it('annule un settle en cours si une nouvelle navigation arrive pendant la fenêtre de grâce', () => {
    // Edge case : naviguer A → B (settle planifié à 150 ms) puis B → C avant 150 ms.
    // La barre doit rester active, pas se fermer prématurément.
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockPathname = '/career'
    rerender(<TopProgressBar />)
    expect(getWidth(container)).toBe(INITIAL_PROGRESS)

    // À 100 ms (avant SETTLE_MS), nouvelle navigation.
    act(() => {
      vi.advanceTimersByTime(100)
    })
    mockPathname = '/synthesis'
    rerender(<TopProgressBar />)

    // 60 ms plus tard : si le settle initial avait fire à 150 ms, on serait en train
    // de compléter. On vérifie qu'on est toujours actif et pas à 100 %.
    act(() => {
      vi.advanceTimersByTime(60)
    })
    expect(findBarFill(container)).not.toBeNull()
    expect(getWidth(container)).toBeLessThan(100)
  })
})
