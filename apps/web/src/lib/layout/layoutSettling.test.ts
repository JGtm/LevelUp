import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  DOCK_TRANSITION_MS,
  beginLayoutTransition,
  endLayoutTransition,
  isLayoutSettling,
  onLayoutSettled,
} from './layoutSettling'

beforeEach(() => {
  vi.useFakeTimers()
})
afterEach(() => {
  endLayoutTransition()
  vi.useRealTimers()
})

describe('layoutSettling', () => {
  it('au repos : aucune transition', () => {
    expect(isLayoutSettling()).toBe(false)
  })

  it('une transition déclarée dure jusqu’à sa clôture, qui prévient les abonnés une seule fois', () => {
    const listener = vi.fn()
    const off = onLayoutSettled(listener)
    beginLayoutTransition()
    expect(isLayoutSettling()).toBe(true)
    expect(listener).not.toHaveBeenCalled()
    endLayoutTransition()
    expect(isLayoutSettling()).toBe(false)
    expect(listener).toHaveBeenCalledTimes(1)
    endLayoutTransition()
    expect(listener).toHaveBeenCalledTimes(1)
    off()
  })

  it('le délai de sécurité clôt d’office une transition dont la fin n’est pas signalée', () => {
    const listener = vi.fn()
    const off = onLayoutSettled(listener)
    beginLayoutTransition()
    vi.advanceTimersByTime(DOCK_TRANSITION_MS)
    expect(isLayoutSettling()).toBe(true)
    vi.advanceTimersByTime(200)
    expect(isLayoutSettling()).toBe(false)
    expect(listener).toHaveBeenCalledTimes(1)
    off()
  })

  it('une nouvelle déclaration en cours repousse le délai de sécurité', () => {
    beginLayoutTransition()
    vi.advanceTimersByTime(DOCK_TRANSITION_MS)
    beginLayoutTransition()
    vi.advanceTimersByTime(DOCK_TRANSITION_MS)
    expect(isLayoutSettling()).toBe(true)
  })

  it('un abonné désabonné n’est plus prévenu', () => {
    const listener = vi.fn()
    onLayoutSettled(listener)()
    beginLayoutTransition()
    endLayoutTransition()
    expect(listener).not.toHaveBeenCalled()
  })
})
