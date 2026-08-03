/**
 * squadEfficiencyChart.test.ts — pistes empilées « écart à la frontière élite ».
 *
 * Couvre les helpers PURS (écart signé, borne symétrique de l'axe partagé, découpe
 * pos/neg avec insertion du croisement, projection des points d'une piste) puis la
 * structure de l'option : une grille / un axe X / un axe Y / trois séries par
 * joueur, étiquettes de piste colorées, axe partagé, dégradation d'un joueur sans
 * données, et valeurs brutes au survol.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import {
  GAP_AXIS_MAX_PCT,
  GAP_AXIS_MIN_PCT,
  buildSquadEfficiencyTracksOption,
  buildTrackPoints,
  eliteGapPercent,
  splitSignedAtZero,
  symmetricGapBound,
  tracksChartHeight,
  type EfficiencyTrackLabels,
} from './squadEfficiencyChart'

function pt(
  order: number,
  fields: Partial<SquadPerformanceSeriesPoint>,
): SquadPerformanceSeriesPoint {
  return {
    match_id: `m${order}`,
    start_time: '2026-04-30T12:00:00Z',
    match_order: order,
    kills: 0,
    deaths: 0,
    assists: 0,
    ...fields,
  } as SquadPerformanceSeriesPoint
}

interface TrackSeries {
  name?: string
  type: string
  xAxisIndex: number
  yAxisIndex: number
  data: Array<{ value: [number, number | null]; player?: string } | [number, number | null]>
  lineStyle?: { color?: string; width?: number }
  areaStyle?: { color: string; opacity: number; origin: number }
  connectNulls?: boolean
}
interface AxisOpt {
  gridIndex: number
  min: number
  max: number
  axisLabel: { show?: boolean; formatter: (v: number) => string }
}
interface GraphicOpt {
  type: string
  top: number
  style: { text: string; fill: string }
}

const COLORS: Record<string, string> = { Me: '#aaa', F1: '#bbb' }
const PLAYERS = ['Me', 'F1']
const OC_P80 = 0.9
const DR_P80 = 1.65

const LABELS: EfficiencyTrackLabels = {
  offensiveMetric: 'Rendement',
  defensiveMetric: 'Résistance',
  eliteBoundary: 'frontière élite',
  damageDealt: 'Dégâts infligés',
  damageTaken: 'Dégâts subis',
  perFrag: '/ frag effectif',
  perDeath: '/ mort',
}

function opts(metric: 'offensive' | 'defensive', eliteP80: number) {
  return { metric, eliteP80, colorByPlayer: COLORS, labels: LABELS } as const
}

describe('eliteGapPercent — écart signé à la frontière élite', () => {
  it('pile sur la frontière → 0 %, au-dessus → positif, en dessous → négatif', () => {
    expect(eliteGapPercent(0.9, OC_P80)).toBe(0)
    expect(eliteGapPercent(1.08, OC_P80)).toBeCloseTo(20, 6)
    expect(eliteGapPercent(0.45, OC_P80)).toBeCloseTo(-50, 6)
  })

  it('les DEUX métriques montent quand le joueur va mieux (DR élevée = positif)', () => {
    expect(eliteGapPercent(1.98, DR_P80)).toBeCloseTo(20, 6)
    expect(eliteGapPercent(0.825, DR_P80)).toBeCloseTo(-50, 6)
  })

  it('indicateur absent ou frontière invalide → null (jamais de division par zéro)', () => {
    expect(eliteGapPercent(null, OC_P80)).toBeNull()
    expect(eliteGapPercent(undefined, OC_P80)).toBeNull()
    expect(eliteGapPercent(Number.NaN, OC_P80)).toBeNull()
    expect(eliteGapPercent(1, 0)).toBeNull()
    expect(eliteGapPercent(1, -1)).toBeNull()
  })
})

describe('symmetricGapBound — borne de l\'axe partagé', () => {
  it('plancher à ±40 % : aucune donnée ou données étroites', () => {
    expect(symmetricGapBound([])).toBe(GAP_AXIS_MIN_PCT)
    expect(symmetricGapBound([12, -5])).toBe(GAP_AXIS_MIN_PCT)
    expect(symmetricGapBound([null, 33, null])).toBe(GAP_AXIS_MIN_PCT)
  })

  it('s\'élargit par pas de 10 quand les données débordent', () => {
    expect(symmetricGapBound([47])).toBe(50)
    expect(symmetricGapBound([-71, 12])).toBe(80)
  })

  it('plafonne à ±150 % (un match aberrant n\'aplatit pas toutes les pistes)', () => {
    expect(symmetricGapBound([-143])).toBe(GAP_AXIS_MAX_PCT)
    expect(symmetricGapBound([1000])).toBe(GAP_AXIS_MAX_PCT)
  })
})

describe('splitSignedAtZero — remplissage vert au-dessus / rouge en dessous', () => {
  it('insère le croisement EXACT quand le signe change', () => {
    const { positive, negative } = splitSignedAtZero([10, -10])
    expect(positive).toEqual([
      [0, 10],
      [0.5, 0],
      [1, 0],
    ])
    expect(negative).toEqual([
      [0, 0],
      [0.5, 0],
      [1, -10],
    ])
  })

  it('croisement pondéré par les amplitudes (−30 → 10 coupe à 75 % du segment)', () => {
    const { positive } = splitSignedAtZero([-30, 10])
    expect(positive).toEqual([
      [0, 0],
      [0.75, 0],
      [1, 10],
    ])
  })

  it('aucun croisement quand le signe ne change pas (0 appartient aux deux)', () => {
    expect(splitSignedAtZero([3, 7]).positive).toEqual([
      [0, 3],
      [1, 7],
    ])
    expect(splitSignedAtZero([0, 5]).positive).toHaveLength(2)
  })

  it('un trou rompt les deux polylignes et interdit un croisement à cheval', () => {
    const { positive, negative } = splitSignedAtZero([-5, null, 5])
    expect(positive).toEqual([
      [0, 0],
      [1, null],
      [2, 5],
    ])
    expect(negative).toEqual([
      [0, -5],
      [1, null],
      [2, 0],
    ])
  })
})

describe('buildTrackPoints — projection d\'une piste sur l\'ordre de match partagé', () => {
  it('offensif : écart depuis rendement_offensif + valeurs brutes (frag EFFECTIF)', () => {
    const points = buildTrackPoints(
      [pt(1, { rendement_offensif: 1.08, damage_dealt: 1800, kills: 8, assists: 3 })],
      3,
      { metric: 'offensive', eliteP80: OC_P80 },
    )
    expect(points[0]).toBeNull()
    expect(points[2]).toBeNull()
    expect(points[1]?.gap).toBeCloseTo(20, 6)
    expect(points[1]?.normalized).toBe(1.08)
    expect(points[1]?.rawDamage).toBe(1800)
    // 8 frags + 3 assistances / 3 = 9 frags effectifs (ADR 0006) → 1800 / 9.
    expect(points[1]?.perEvent).toBeCloseTo(200, 6)
  })

  it('défensif : écart depuis resistance_defensive + dégâts subis par mort', () => {
    const points = buildTrackPoints(
      [pt(0, { resistance_defensive: 1.98, damage_taken: 1200, deaths: 4 })],
      1,
      { metric: 'defensive', eliteP80: DR_P80 },
    )
    expect(points[0]?.gap).toBeCloseTo(20, 6)
    expect(points[0]?.rawDamage).toBe(1200)
    expect(points[0]?.perEvent).toBe(300)
  })

  it('DR à 0 SANS dégât subi = aucun engagement mesuré → pas de point (pas de −100 % faux)', () => {
    const points = buildTrackPoints(
      [pt(0, { resistance_defensive: 0, damage_taken: 0, deaths: 0 })],
      1,
      { metric: 'defensive', eliteP80: DR_P80 },
    )
    expect(points[0]).toBeNull()
  })

  it('DR à 0 AVEC des dégâts subis reste tracée (−100 %, information réelle)', () => {
    const points = buildTrackPoints(
      [pt(0, { resistance_defensive: 0, damage_taken: 500, deaths: 2 })],
      1,
      { metric: 'defensive', eliteP80: DR_P80 },
    )
    expect(points[0]?.gap).toBeCloseTo(-100, 6)
  })

  it('indicateur absent ou match_order hors bornes → ignoré', () => {
    const points = buildTrackPoints(
      [pt(0, { damage_dealt: 900, kills: 4 }), pt(9, { rendement_offensif: 1 })],
      2,
      { metric: 'offensive', eliteP80: OC_P80 },
    )
    expect(points).toEqual([null, null])
  })
})

describe('tracksChartHeight', () => {
  it('croît d\'une piste à l\'autre et garde une hauteur utile à 1 joueur', () => {
    expect(tracksChartHeight(1)).toBe(110)
    expect(tracksChartHeight(2)).toBe(174)
    expect(tracksChartHeight(4)).toBe(302)
    // 0 joueur ne doit pas produire une hauteur négative/nulle.
    expect(tracksChartHeight(0)).toBe(110)
  })
})

describe('buildSquadEfficiencyTracksOption', () => {
  const rows = {
    Me: [
      pt(0, { rendement_offensif: 1.08, damage_dealt: 1800, kills: 8, assists: 3 }),
      pt(1, { rendement_offensif: 0.45, damage_dealt: 1000, kills: 2, assists: 3, map_name: 'Streets' }),
    ],
    F1: [pt(0, { rendement_offensif: 0.9, damage_dealt: 900, kills: 4 })],
  }

  it('vide (aucune donnée) → fond minimal, pas de série', () => {
    const opt = buildSquadEfficiencyTracksOption({}, PLAYERS, opts('offensive', OC_P80))
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('une grille / un axe X / un axe Y / trois séries par joueur, dans l\'ordre donné', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    expect(opt.grid as unknown[]).toHaveLength(2)
    expect(opt.xAxis as unknown[]).toHaveLength(2)
    expect(opt.yAxis as unknown[]).toHaveLength(2)
    const series = opt.series as unknown as TrackSeries[]
    expect(series).toHaveLength(6)
    // Aire positive, aire négative, puis le trait du joueur (dessus).
    expect(series[0].areaStyle?.origin).toBe(0)
    expect(series[1].areaStyle?.origin).toBe(0)
    expect(series[2].name).toBe('Me')
    expect(series[2].lineStyle?.color).toBe('#aaa')
    expect(series[5].name).toBe('F1')
    expect(series[5].lineStyle?.color).toBe('#bbb')
    // Chaque triplet est rattaché à SA piste.
    expect(series.map((s) => s.xAxisIndex)).toEqual([0, 0, 0, 1, 1, 1])
    expect(series.map((s) => s.yAxisIndex)).toEqual([0, 0, 0, 1, 1, 1])
  })

  it('axe symétrique PARTAGÉ : mêmes bornes sur toutes les pistes', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    const yAxes = opt.yAxis as unknown as AxisOpt[]
    // Écarts : +20 %, −50 %, 0 % → borne arrondie à 50.
    expect(yAxes[0].min).toBe(-50)
    expect(yAxes[0].max).toBe(50)
    expect(yAxes[1].min).toBe(yAxes[0].min)
    expect(yAxes[1].max).toBe(yAxes[0].max)
    // Le zéro (= la frontière élite) n'est pas étiqueté : il porte le trait repère.
    expect(yAxes[0].axisLabel.formatter(0)).toBe('')
    expect(yAxes[0].axisLabel.formatter(50)).toBe('+50 %')
  })

  it('étiquette de piste = gamertag à la couleur du joueur, une par piste', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    const graphic = opt.graphic as unknown as GraphicOpt[]
    expect(graphic).toHaveLength(2)
    expect(graphic[0].style).toMatchObject({ text: 'Me', fill: '#aaa' })
    expect(graphic[1].style).toMatchObject({ text: 'F1', fill: '#bbb' })
    expect(graphic[1].top).toBeGreaterThan(graphic[0].top)
  })

  it('libellés de match sur la SEULE dernière piste (une seule frise partagée)', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    const xAxes = opt.xAxis as unknown as AxisOpt[]
    expect(xAxes[0].axisLabel.show).toBe(false)
    expect(xAxes[1].axisLabel.show).toBe(true)
    expect(xAxes[1].axisLabel.formatter(1)).toBe('#2 · Streets')
    expect(xAxes[1].axisLabel.formatter(0)).toBe('#1')
    // Un tick fractionnaire d'ECharts ne doit pas inventer d'étiquette.
    expect(xAxes[1].axisLabel.formatter(0.5)).toBe('')
  })

  it('joueur sans donnée → piste présente, trait entièrement null (pas de trou de mise en page)', () => {
    const opt = buildSquadEfficiencyTracksOption(
      { Me: rows.Me, F1: [] },
      PLAYERS,
      opts('offensive', OC_P80),
    )
    expect(opt.grid as unknown[]).toHaveLength(2)
    const series = opt.series as unknown as TrackSeries[]
    const f1Line = series[5]
    expect(f1Line.name).toBe('F1')
    expect(f1Line.data.map((d) => (Array.isArray(d) ? d[1] : d.value[1]))).toEqual([null, null])
    // Aucune aire ne se referme sur un joueur sans données.
    expect(series[3].data).toEqual([
      [0, null],
      [1, null],
    ])
  })

  it('survol : écart signé + indicateur normalisé + dégâts BRUTS du match', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    const series = opt.series as unknown as TrackSeries[]
    const first = series[2].data[0]
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    const html = formatter([{ data: first }])
    expect(html).toContain('#1')
    expect(html).toContain('Me')
    expect(html).toContain('+20 %')
    expect(html).toContain('Rendement 1.08')
    expect(html).toContain('frontière élite 0.90')
    expect(html).toContain('Dégâts infligés 1800')
    expect(html).toContain('200 / frag effectif')
  })

  it('survol défensif : libellés et unité de la métrique défensive', () => {
    const opt = buildSquadEfficiencyTracksOption(
      { Me: [pt(0, { resistance_defensive: 1.98, damage_taken: 1200, deaths: 4 })] },
      ['Me'],
      opts('defensive', DR_P80),
    )
    const series = opt.series as unknown as TrackSeries[]
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    const html = formatter([{ data: series[2].data[0] }])
    expect(html).toContain('Résistance 1.98')
    expect(html).toContain('frontière élite 1.65')
    expect(html).toContain('Dégâts subis 1200')
    expect(html).toContain('300 / mort')
  })

  it('survol hors point (aires seules) → chaîne vide, jamais de tooltip fantôme', () => {
    const opt = buildSquadEfficiencyTracksOption(rows, PLAYERS, opts('offensive', OC_P80))
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    expect(formatter([{ data: [0.5, 0] }])).toBe('')
    expect(formatter('pas un tableau')).toBe('')
  })
})
