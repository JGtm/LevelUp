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

// Durée de grâce avant complétion (doit correspondre à SETTLE_MS dans le composant).
const SETTLE_MS = 150

function findBarFill(container: HTMLElement): HTMLElement | null {
  return container.querySelector('div[aria-hidden] > div') as HTMLElement | null
}

describe('TopProgressBar', () => {
  beforeEach(() => {
    mockIsFetching = 0
    mockPathname = '/home'
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it("ne rend rien quand aucune query n'est en cours et pathname stable", () => {
    const { container } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()
  })

  it('apparaît à 30 % quand une query pending démarre', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    const fill = findBarFill(container)
    expect(fill).not.toBeNull()
    expect(fill!.style.width).toBe('30%')
    expect(fill!.style.opacity).toBe('1')
  })

  it('apparaît immédiatement sur changement de pathname (sans query en cours)', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    mockPathname = '/career'
    rerender(<TopProgressBar />)

    const fill = findBarFill(container)
    expect(fill).not.toBeNull()
    expect(fill!.style.width).toBe('30%')
    expect(fill!.style.opacity).toBe('1')
  })

  it('progresse 30 → 70 → 85 % par paliers tant que ça fetch', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    expect(findBarFill(container)!.style.width).toBe('30%')

    act(() => {
      vi.advanceTimersByTime(200)
    })
    expect(findBarFill(container)!.style.width).toBe('70%')

    act(() => {
      vi.advanceTimersByTime(600) // 200 + 600 = 800 ms
    })
    expect(findBarFill(container)!.style.width).toBe('85%')
  })

  it('reste à 85 % indéfiniment si le fetch ne se termine pas', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)

    act(() => {
      vi.advanceTimersByTime(5_000)
    })
    expect(findBarFill(container)!.style.width).toBe('85%')
  })

  it('complète à 100 % quand pendingCount repasse à 0 (MIN_VISIBLE_MS respecté)', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    // Fetch instantané
    mockIsFetching = 0
    rerender(<TopProgressBar />)

    // Pas encore à 100 % : settle (150 ms) + MIN_VISIBLE_MS (450 ms) = 450 ms
    // (le settle est absorbé par le calcul elapsed → completionDelay)
    expect(findBarFill(container)!.style.width).toBe('30%')

    act(() => {
      vi.advanceTimersByTime(450)
    })
    expect(findBarFill(container)!.style.width).toBe('100%')
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
    expect(findBarFill(container)!.style.width).toBe('85%')

    mockIsFetching = 0
    rerender(<TopProgressBar />)

    // SETTLE_MS + 1 pour capturer le setTimeout(fn,0) schedulé par completeBar()
    // au moment exact où le settle timer fire.
    act(() => {
      vi.advanceTimersByTime(SETTLE_MS + 1)
    })
    expect(findBarFill(container)!.style.width).toBe('100%')
  })

  it('disparaît du DOM après le fade-out', () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    mockIsFetching = 1
    rerender(<TopProgressBar />)
    mockIsFetching = 0
    rerender(<TopProgressBar />)

    // Settle (absorbé) + MIN_VISIBLE_MS (450) + FADE_OUT_MS (250) = 700 ms
    act(() => {
      vi.advanceTimersByTime(700)
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
    expect(findBarFill(container)!.style.width).toBe('85%')
  })

  it("redéclenche proprement après une 1re séquence terminée", () => {
    const { container, rerender } = renderWithProviders(<TopProgressBar />)

    // 1re séquence
    mockIsFetching = 1
    rerender(<TopProgressBar />)
    mockIsFetching = 0
    rerender(<TopProgressBar />)
    act(() => {
      vi.advanceTimersByTime(700)
    })
    rerender(<TopProgressBar />)
    expect(container.firstChild).toBeNull()

    // 2e séquence
    mockIsFetching = 1
    rerender(<TopProgressBar />)
    expect(findBarFill(container)!.style.width).toBe('30%')
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
})
