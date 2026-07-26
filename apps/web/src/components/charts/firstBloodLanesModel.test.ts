/**
 * Tests du modèle pur de FirstBloodLanes : médianes (interpolation + null),
 * tri des lanes, formats de durée / d'écart, totaux « n/total ».
 * Aucun React, aucun ECharts.
 */
import { describe, expect, it } from 'vitest'

import {
  GRID_BOTTOM,
  GRID_TOP,
  LANE_HEIGHT,
  buildFirstBloodLanes,
  firstBloodLanesHeight,
  formatAxisTick,
  formatGapSeconds,
  formatLaneSeconds,
  medianSeconds,
  quantileSorted,
  sanitizeRichText,
  type FirstBloodPlayerSeries,
} from './firstBloodLanesModel'

describe('quantileSorted / medianSeconds', () => {
  it('interpole linéairement entre les deux valeurs encadrantes', () => {
    expect(quantileSorted([10, 20], 0.5)).toBe(15)
    expect(quantileSorted([10, 20, 30, 40], 0.5)).toBe(25)
    // pos = 3 × 0.5 = 1.5 → entre 2 et 8
    expect(quantileSorted([1, 2, 8, 9], 0.5)).toBe(5)
  })

  it('renvoie la valeur exacte sur un effectif impair, null sur vide', () => {
    expect(quantileSorted([10, 20, 30], 0.5)).toBe(20)
    expect(quantileSorted([42], 0.5)).toBe(42)
    expect(quantileSorted([], 0.5)).toBeNull()
  })

  it('trie, ignore les valeurs non finies et arrondit à la seconde', () => {
    expect(medianSeconds([30, 10, 20])).toBe(20)
    expect(medianSeconds([10, 21])).toBe(16) // 15.5 → 16
    expect(medianSeconds([Number.NaN, 10, Number.POSITIVE_INFINITY, 20])).toBe(15)
    expect(medianSeconds([])).toBeNull()
  })
})

describe('buildFirstBloodLanes', () => {
  const data: FirstBloodPlayerSeries[] = [
    {
      player: 'Slow',
      matches: [
        { matchId: 'm1', firstKillSec: 80, firstDeathSec: 40 },
        { matchId: 'm2', firstKillSec: 100, firstDeathSec: 60 },
      ],
    },
    {
      player: 'Fast',
      matches: [
        { matchId: 'm1', firstKillSec: 20, firstDeathSec: 50 },
        { matchId: 'm2', firstKillSec: 40, firstDeathSec: 70 },
      ],
    },
    {
      player: 'Silent',
      matches: [{ matchId: 'm1', firstKillSec: null, firstDeathSec: 33 }],
    },
  ]

  it('trie les lanes par médiane du premier frag croissante, nulles en dernier', () => {
    const lanes = buildFirstBloodLanes(data)
    expect(lanes.map((l) => l.player)).toEqual(['Fast', 'Slow', 'Silent'])
    expect(lanes[0].medianKillSec).toBe(30)
    expect(lanes[1].medianKillSec).toBe(90)
    expect(lanes[2].medianKillSec).toBeNull()
  })

  it("calcule l'écart signé médiane(mort) − médiane(frag)", () => {
    const lanes = buildFirstBloodLanes(data)
    // Fast : méd. mort 60 − méd. frag 30 = +30 (avance)
    expect(lanes[0].gapSec).toBe(30)
    // Slow : méd. mort 50 − méd. frag 90 = −40 (retard)
    expect(lanes[1].gapSec).toBe(-40)
    // Silent : pas de médiane frag → pas d'écart, donc pas de barre
    expect(lanes[2].gapSec).toBeNull()
  })

  it('exclut les null des points et des médianes mais les compte dans le total', () => {
    const lanes = buildFirstBloodLanes([
      {
        player: 'Mixed',
        matches: [
          { matchId: 'a', firstKillSec: 10, firstDeathSec: null },
          { matchId: 'b', firstKillSec: null, firstDeathSec: 90 },
          { matchId: 'c', firstKillSec: 30, firstDeathSec: 70 },
        ],
      },
    ])
    const lane = lanes[0]
    expect(lane.totalMatches).toBe(3)
    expect(lane.kills).toHaveLength(2)
    expect(lane.kills.map((p) => p.matchId)).toEqual(['a', 'c'])
    expect(lane.deaths).toHaveLength(2)
    expect(lane.medianKillSec).toBe(20)
    expect(lane.medianDeathSec).toBe(80)
  })

  it('gère un joueur sans aucun événement exploitable', () => {
    const lanes = buildFirstBloodLanes([
      { player: 'Ghost', matches: [{ matchId: 'a', firstKillSec: null, firstDeathSec: null }] },
    ])
    expect(lanes[0]).toMatchObject({
      totalMatches: 1,
      medianKillSec: null,
      medianDeathSec: null,
      gapSec: null,
    })
    expect(lanes[0].kills).toEqual([])
  })

  it('conserve l’ordre d’entrée entre lanes de même médiane (tri stable)', () => {
    const lanes = buildFirstBloodLanes([
      { player: 'A', matches: [{ matchId: 'x', firstKillSec: 50, firstDeathSec: null }] },
      { player: 'B', matches: [{ matchId: 'y', firstKillSec: 50, firstDeathSec: null }] },
    ])
    expect(lanes.map((l) => l.player)).toEqual(['A', 'B'])
  })
})

describe('formats', () => {
  it('formate les durées : « 45s » sous la minute, « 1m05 » au-delà', () => {
    expect(formatLaneSeconds(0)).toBe('0s')
    expect(formatLaneSeconds(45)).toBe('45s')
    expect(formatLaneSeconds(59.4)).toBe('59s')
    expect(formatLaneSeconds(60)).toBe('1m00')
    expect(formatLaneSeconds(65)).toBe('1m05')
    expect(formatLaneSeconds(64.6)).toBe('1m05')
    expect(formatLaneSeconds(3599)).toBe('59m59')
  })

  it('formate l’écart avec son signe (moins typographique)', () => {
    expect(formatGapSeconds(14)).toBe('+14s')
    expect(formatGapSeconds(0)).toBe('+0s')
    expect(formatGapSeconds(-22)).toBe('−22s')
    expect(formatGapSeconds(-65)).toBe('−1m05')
  })

  it('formate les ticks de l’axe X en minutes', () => {
    expect(formatAxisTick(0)).toBe('0')
    expect(formatAxisTick(60)).toBe('1m')
    expect(formatAxisTick(300)).toBe('5m')
  })

  it('neutralise les métacaractères du texte enrichi ECharts', () => {
    expect(sanitizeRichText('a|b{c}d')).toBe('a b c d')
    expect(sanitizeRichText('ligne1\nligne2')).toBe('ligne1 ligne2')
  })
})

describe('firstBloodLanesHeight', () => {
  it('dérive la hauteur du nombre de lanes (marges du grid incluses)', () => {
    expect(firstBloodLanesHeight(4)).toBe(4 * LANE_HEIGHT + GRID_TOP + GRID_BOTTOM)
    // Aucune lane → une hauteur de bande minimale (état vide lisible).
    expect(firstBloodLanesHeight(0)).toBe(LANE_HEIGHT + GRID_TOP + GRID_BOTTOM)
  })
})
