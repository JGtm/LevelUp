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

/**
 * Les réglages du hook, portée `match` par défaut : c'est la lecture inchangée depuis le
 * 16/08 (le calque était DÉJÀ celui de tout le match), et le défaut du tiroir.
 */
function settings(over: Partial<Parameters<typeof useReplayHeatmap>[3]> = {}) {
  return {
    show: true,
    mode: 'presence' as const,
    span: 'match' as const,
    frameRef: { current: 0 },
    ...over,
  }
}

// `renderHook` est appelé DANS chaque test, jamais depuis un intermédiaire : un helper qui
// appellerait le hook serait lui-même vu comme un hook par eslint (rules-of-hooks), et
// appeler un hook dans le rappel d'un hook est interdit. C'est le patron du dépôt
// (useReplaySettings.test.tsx).

describe('useReplayHeatmap — ce qui est cuit', () => {
  it('calque éteint : aucune grille, donc aucun calcul lourd', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [], settings({ show: false })),
    )
    expect(result.current.grid).toBeNull()
  })

  it('calque allumé : la grille est là, et la rampe du thème avec elle', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [], settings()),
    )
    expect(result.current.grid).not.toBeNull()
    expect(result.current.ramp.length).toBeGreaterThan(0)
  })
})

describe('useReplayHeatmap — la lecture réellement servie', () => {
  it('aucune mort localisée : la lecture retombe sur la présence, éliminations indisponible', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(null)], settings({ mode: 'kills' })),
    )
    expect(result.current.killsAvailable).toBe(false)
    expect(result.current.mode).toBe('presence')
    // Et la grille est bien celle de la présence, pas un calque vide.
    expect(result.current.grid?.mode).toBe('presence')
  })

  it('des morts localisées : la lecture demandée est servie, même sans tueur relu', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(18)], settings({ mode: 'kills' })),
    )
    expect(result.current.killsAvailable).toBe(true)
    expect(result.current.mode).toBe('kills')
    expect(result.current.grid?.mode).toBe('kills')
  })

  it('la préférence présence reste la présence, même quand des morts sont localisées', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(doc(), BOUNDS, [killAt(18)], settings()),
    )
    expect(result.current.mode).toBe('presence')
  })
})

/**
 * V2 (2026-08-18) — LA PORTÉE `live` : la carte se remplit au fil de la lecture.
 *
 * CE QUI EST VÉRIFIÉ, ET C'EST LE POINT DÉLICAT : le hook ne recuit PAS à chaque image. Il
 * lit l'image courante dans une référence, l'arrondit au pas de recuisson, et ne rend une
 * nouvelle grille que lorsque ce seau change. Les tests règlent donc la référence AVANT le
 * rendu et comparent deux instants, plutôt que d'espérer un minuteur.
 */
describe('useReplayHeatmap — portée de temps (V2)', () => {
  /** Un joueur en (8, 8) sur les 10 premières images, puis en (30, 30). */
  function docDeuxLieux() {
    const points: ReplayPoint[] = []
    for (let t = 0; t <= 10; t++) points.push({ t, x: 8, y: 8 })
    for (let t = 11; t <= 40; t++) points.push({ t, x: 30, y: 30 })
    return testReplayDoc({
      frameIntervalMs: 100,
      bounds: BOUNDS,
      tracks: [{ slot: 0, team: 0, points }],
    })
  }

  /** Somme des valeurs de la grille : ce que la carte a mesuré en tout. */
  const total = (g: { value: Float32Array } | null) =>
    g === null ? 0 : g.value.reduce((s, v) => s + v, 0)

  it('portée « toute la partie » : la grille ignore l image courante', () => {
    const { result } = renderHook(() =>
      useReplayHeatmap(docDeuxLieux(), BOUNDS, [], settings({ frameRef: { current: 5 } })),
    )
    const tout = renderHook(() =>
      useReplayHeatmap(docDeuxLieux(), BOUNDS, [], settings({ frameRef: { current: 40 } })),
    )
    expect(total(result.current.grid)).toBeCloseTo(total(tout.result.current.grid), 5)
  })

  it('portée « jusqu à l image courante » : tôt dans le film, la carte mesure MOINS', () => {
    const tot = renderHook(() =>
      useReplayHeatmap(docDeuxLieux(), BOUNDS, [], settings({ span: 'live', frameRef: { current: 40 } })),
    )
    const early = renderHook(() =>
      useReplayHeatmap(docDeuxLieux(), BOUNDS, [], settings({ span: 'live', frameRef: { current: 10 } })),
    )
    expect(total(early.result.current.grid)).toBeGreaterThan(0)
    expect(total(early.result.current.grid)).toBeLessThan(total(tot.result.current.grid))
  })
})
