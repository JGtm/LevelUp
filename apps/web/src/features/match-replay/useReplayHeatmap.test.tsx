/**
 * useReplayHeatmap.test.tsx — les trois arbitrages du hook : ce qui est cuit (et quand),
 * la lecture réellement servie quand la préférence n'a rien à mesurer, et la rampe du
 * thème (vide quand le thème ne la donne pas — on ne peint alors pas plutôt que d'inventer).
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import type { ReplayBounds, ReplayPoint } from '@/lib/api/types'

// resolveToken -> hex réel, sinon la rampe retomberait à vide et masquerait les assertions
// de mode (même patron que MatchTugOfWarChart.test.tsx).
vi.mock('@/lib/accessibility/resolveToken', () => ({
  resolveToken: (token: string) => (token.endsWith('-low') ? '#1e3a5f' : '#60a5fa'),
}))

import type { KillFxEntry } from './killFx'
import { testReplayDoc } from './test/testDoc'
import { useReplayHeatmap } from './useReplayHeatmap'

const BOUNDS: ReplayBounds = { minX: 0, minY: 0, maxX: 40, maxY: 40 }

function doc() {
  const points: ReplayPoint[] = []
  for (let t = 0; t <= 20; t++) points.push({ t, x: 20, y: 20 })
  return testReplayDoc({
    frameIntervalMs: 100,
    bounds: BOUNDS,
    tracks: [{ slot: 0, team: 0, points }],
  })
}

/** Une mort du calque d'effets : `deathX` porte le lieu, `vx` l'orientation. */
function killAt(x: number | null): KillFxEntry {
  return {
    frame: 10, x: 0, y: 0, vx: null, vy: null, dist: null,
    fam: 'plain', slot: 0, seed: 1,
    deathX: x, deathY: x === null ? null : 12,
  }
}

// `renderHook` est appelé DANS chaque test, jamais depuis un intermédiaire : un helper qui
// appellerait le hook serait lui-même vu comme un hook par eslint (rules-of-hooks), et
// appeler un hook dans le rappel d'un hook est interdit. C'est le patron du dépôt
// (useReplaySettings.test.tsx).

describe('useReplayHeatmap — ce qui est cuit', () => {
  it('calque éteint : aucune grille, donc aucun calcul lourd', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [], { show: false, mode: 'presence' }),
    )
    expect(result.current.grid).toBeNull()
  })

  it('calque allumé : la grille est là, et la rampe du thème avec elle', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [], { show: true, mode: 'presence' }),
    )
    expect(result.current.grid).not.toBeNull()
    expect(result.current.ramp.length).toBeGreaterThan(0)
  })
})

describe('useReplayHeatmap — la lecture réellement servie', () => {
  it('aucune mort localisée : la lecture retombe sur la présence, éliminations indisponible', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(null)], { show: true, mode: 'kills' }),
    )
    expect(result.current.killsAvailable).toBe(false)
    expect(result.current.mode).toBe('presence')
    // Et la grille est bien celle de la présence, pas un calque vide.
    expect(result.current.grid?.mode).toBe('presence')
  })

  it('des morts localisées : la lecture demandée est servie, même sans tueur relu', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(18)], { show: true, mode: 'kills' }),
    )
    expect(result.current.killsAvailable).toBe(true)
    expect(result.current.mode).toBe('kills')
    expect(result.current.grid?.mode).toBe('kills')
  })

  it('la préférence présence reste la présence, même quand des morts sont localisées', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(18)], { show: true, mode: 'presence' }),
    )
    expect(result.current.mode).toBe('presence')
  })
})
