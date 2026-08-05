/**
 * squadEfficiencyChart.test.ts — cartes Rendement / Résistance en courbes
 * superposées, taux « une vie » (%) sur fenêtre FIXE 50…200 %.
 *
 * Couvre la projection des points (taux = indicateur × 100, valeurs brutes au
 * survol) puis la structure de l'option : une grille / une courbe par joueur,
 * bornes d'axe CONSTANTES (le cas « deux sessions d'amplitudes opposées » échoue
 * si quelqu'un réintroduit des bornes calculées des données), zones de lecture,
 * repère « 1 vie », étiquettes de fin de courbe et contenu du survol.
 */
import { describe, it, expect } from 'vitest'

import type { SquadPerformanceSeriesPoint } from '@/lib/api/types'
import { ONE_LIFE_RATE_PCT, ONE_LIFE_ZONE_OPACITY } from '@/lib/charts/oneLifeWindow'
import {
  EFFICIENCY_CHART_HEIGHT,
  RATE_AXIS_MAX_PCT,
  RATE_AXIS_MIN_PCT,
  buildEfficiencyPoints,
  buildSquadEfficiencyOption,
  type EfficiencyChartLabels,
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

interface MarkAreaItem {
  yAxis: number
  itemStyle?: { color: string; opacity: number }
}
interface PlayerSeries {
  name?: string
  type: string
  data: Array<{ value: number } | null>
  lineStyle?: { color?: string; width?: number }
  showSymbol?: boolean
  connectNulls?: boolean
  endLabel?: { show: boolean; formatter: string; color: string }
  markArea?: { data: MarkAreaItem[][] }
  markLine?: {
    data: Array<{ yAxis: number }>
    lineStyle?: { color?: string }
    label?: { show: boolean; formatter: string }
  }
}
interface AxisOpt {
  min?: number
  max?: number
  scale?: boolean
  data?: string[]
  axisLabel: { formatter?: (v: number) => string; interval?: number }
}

const COLORS: Record<string, string> = { Me: '#aaa', F1: '#bbb' }
const PLAYERS = ['Me', 'F1']

const LABELS: EfficiencyChartLabels = {
  offensiveMetric: 'Rendement',
  defensiveMetric: 'Résistance',
  oneLife: '1 vie',
  damageDealt: 'Dégâts infligés',
  damageTaken: 'Dégâts subis',
  perFrag: '/ frag effectif',
  perDeath: '/ mort',
}

function opts(metric: 'offensive' | 'defensive') {
  return { metric, colorByPlayer: COLORS, labels: LABELS } as const
}

describe('buildEfficiencyPoints — taux « une vie » (%) par match', () => {
  it('offensif : taux = rendement_offensif × 100 + valeurs brutes (frag EFFECTIF)', () => {
    const points = buildEfficiencyPoints(
      [pt(1, { rendement_offensif: 1.08, damage_dealt: 1800, kills: 8, assists: 3 })],
      3,
      'offensive',
    )
    expect(points[0]).toBeNull()
    expect(points[2]).toBeNull()
    expect(points[1]?.rate).toBeCloseTo(108, 6)
    expect(points[1]?.rawDamage).toBe(1800)
    // 8 frags + 3 assistances / 3 = 9 frags effectifs (ADR 0006) → 1800 / 9.
    expect(points[1]?.perEvent).toBeCloseTo(200, 6)
  })

  it('défensif : taux = resistance_defensive × 100 + dégâts subis par mort', () => {
    const points = buildEfficiencyPoints(
      [pt(0, { resistance_defensive: 1.65, damage_taken: 1200, deaths: 4 })],
      1,
      'defensive',
    )
    expect(points[0]?.rate).toBeCloseTo(165, 6)
    expect(points[0]?.rawDamage).toBe(1200)
    expect(points[0]?.perEvent).toBe(300)
  })

  it('100 % = exactement une vie (indicateur natif à 1, aucun barème recodé)', () => {
    expect(buildEfficiencyPoints([pt(0, { rendement_offensif: 1 })], 1, 'offensive')[0]?.rate).toBe(
      ONE_LIFE_RATE_PCT,
    )
    expect(
      buildEfficiencyPoints([pt(0, { resistance_defensive: 1, damage_taken: 900 })], 1, 'defensive')[0]
        ?.rate,
    ).toBe(ONE_LIFE_RATE_PCT)
  })

  it('DR à 0 SANS dégât subi = aucun engagement mesuré → pas de point (pas de 0 % faux)', () => {
    const points = buildEfficiencyPoints(
      [pt(0, { resistance_defensive: 0, damage_taken: 0, deaths: 0 })],
      1,
      'defensive',
    )
    expect(points[0]).toBeNull()
  })

  it('DR à 0 AVEC des dégâts subis reste tracée (0 %, information réelle)', () => {
    const points = buildEfficiencyPoints(
      [pt(0, { resistance_defensive: 0, damage_taken: 500, deaths: 2 })],
      1,
      'defensive',
    )
    expect(points[0]?.rate).toBe(0)
  })

  it('indicateur absent ou match_order hors bornes → ignoré', () => {
    const points = buildEfficiencyPoints(
      [pt(0, { damage_dealt: 900, kills: 4 }), pt(9, { rendement_offensif: 1 })],
      2,
      'offensive',
    )
    expect(points).toEqual([null, null])
  })
})

describe('cadre de lecture FIXE', () => {
  it('la fenêtre est 50…200 % autour du pivot 100 % (une vie)', () => {
    expect(ONE_LIFE_RATE_PCT).toBe(100)
    expect(RATE_AXIS_MIN_PCT).toBe(50)
    expect(RATE_AXIS_MAX_PCT).toBe(200)
  })

  it('la hauteur de carte ne dépend PAS du nombre de joueurs (une seule grille)', () => {
    expect(EFFICIENCY_CHART_HEIGHT).toBeGreaterThan(0)
  })
})

describe('buildSquadEfficiencyOption', () => {
  const rows = {
    Me: [
      pt(0, { rendement_offensif: 1.08, damage_dealt: 1800, kills: 8, assists: 3 }),
      pt(1, {
        rendement_offensif: 0.45,
        damage_dealt: 1000,
        kills: 2,
        assists: 3,
        map_name: 'Streets',
      }),
    ],
    F1: [pt(0, { rendement_offensif: 0.9, damage_dealt: 900, kills: 4 })],
  }

  it('vide (aucune donnée) → fond minimal, pas de série', () => {
    const opt = buildSquadEfficiencyOption({}, PLAYERS, opts('offensive'))
    expect(opt.series).toBeUndefined()
    expect(opt.backgroundColor).toBeDefined()
  })

  it('une SEULE grille, un axe X, un axe Y, une courbe par joueur dans l\'ordre donné', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    expect(Array.isArray(opt.grid)).toBe(false)
    expect(Array.isArray(opt.xAxis)).toBe(false)
    expect(Array.isArray(opt.yAxis)).toBe(false)
    const series = opt.series as unknown as PlayerSeries[]
    expect(series).toHaveLength(2)
    expect(series[0].name).toBe('Me')
    expect(series[0].lineStyle?.color).toBe('#aaa')
    expect(series[1].name).toBe('F1')
    expect(series[1].lineStyle?.color).toBe('#bbb')
  })

  it('fenêtre FIXE : deux sessions d\'amplitudes OPPOSÉES donnent les mêmes bornes', () => {
    // Ce cas échoue dès qu'une borne est recalculée depuis les données.
    const tight = buildSquadEfficiencyOption(
      { Me: [pt(0, { rendement_offensif: 1.01 }), pt(1, { rendement_offensif: 1.02 })] },
      ['Me'],
      opts('offensive'),
    )
    const wide = buildSquadEfficiencyOption(
      { Me: [pt(0, { rendement_offensif: 0.05 }), pt(1, { rendement_offensif: 4.2 })] },
      ['Me'],
      opts('offensive'),
    )
    for (const o of [tight, wide]) {
      const y = o.yAxis as unknown as AxisOpt
      expect(y.min).toBe(RATE_AXIS_MIN_PCT)
      expect(y.max).toBe(RATE_AXIS_MAX_PCT)
      // `scale: true` laisserait ECharts recadrer sur les données.
      expect(y.scale).toBeUndefined()
    }
    // Et la carte défensive partage EXACTEMENT le même cadre.
    const defensive = buildSquadEfficiencyOption(
      { Me: [pt(0, { resistance_defensive: 1.9, damage_taken: 900, deaths: 3 })] },
      ['Me'],
      opts('defensive'),
    )
    expect((defensive.yAxis as unknown as AxisOpt).min).toBe(RATE_AXIS_MIN_PCT)
    expect((defensive.yAxis as unknown as AxisOpt).max).toBe(RATE_AXIS_MAX_PCT)
  })

  it('axe Y étiqueté en %', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const y = opt.yAxis as unknown as AxisOpt
    expect(y.axisLabel.formatter?.(100)).toBe('100 %')
    expect(y.axisLabel.formatter?.(50)).toBe('50 %')
  })

  it('repère « 1 vie » à 100 %, rendu UNE seule fois, à la couleur d\'axe du thème', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const series = opt.series as unknown as PlayerSeries[]
    expect(series[0].markLine?.data).toEqual([{ yAxis: ONE_LIFE_RATE_PCT }])
    expect(series[0].markLine?.label?.formatter).toBe('1 vie')
    expect(series[0].markLine?.lineStyle?.color).toBe('#374151') // fallback --border (jsdom)
    expect(series[1].markLine).toBeUndefined()
  })

  it('zones : VERT au-dessus de 100, ROUGE en dessous', () => {
    // Polarité UNIQUE du cadre « une vie », partagée avec la courbe solo
    // (cf. TimeseriesEfficiency.test.tsx) : les indicateurs tracés sont des taux
    // qui montent quand le joueur va mieux, donc le vert est toujours en haut.
    for (const metric of ['offensive', 'defensive'] as const) {
      const src =
        metric === 'offensive'
          ? rows
          : { Me: [pt(0, { resistance_defensive: 1.9, damage_taken: 900, deaths: 3 })] }
      const opt = buildSquadEfficiencyOption(src, metric === 'offensive' ? PLAYERS : ['Me'], opts(metric))
      const zones = (opt.series as unknown as PlayerSeries[])[0].markArea?.data
      expect(zones).toHaveLength(2)
      const [posLow, posHigh] = zones![0]
      const [negLow, negHigh] = zones![1]
      expect(posLow.yAxis).toBe(ONE_LIFE_RATE_PCT)
      expect(posHigh.yAxis).toBe(RATE_AXIS_MAX_PCT)
      expect(negLow.yAxis).toBe(RATE_AXIS_MIN_PCT)
      expect(negHigh.yAxis).toBe(ONE_LIFE_RATE_PCT)
      // La bande HAUTE porte le token favorable, la bande BASSE le défavorable.
      expect(posLow.itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.pos)
      expect(negLow.itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.neg)
      // Le rouge pèse plus à opacité égale : il est volontairement plus discret.
      expect(negLow.itemStyle?.opacity).toBeLessThan(posLow.itemStyle!.opacity)
    }
    // Zones portées par la grille, donc rendues une seule fois.
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    expect((opt.series as unknown as PlayerSeries[])[1].markArea).toBeUndefined()
  })

  it('étiquette de FIN de courbe = gamertag à la couleur du joueur (remplace la légende)', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const series = opt.series as unknown as PlayerSeries[]
    expect(opt.legend).toBeUndefined()
    expect(series[0].endLabel).toMatchObject({ show: true, formatter: 'Me', color: '#aaa' })
    expect(series[1].endLabel).toMatchObject({ show: true, formatter: 'F1', color: '#bbb' })
  })

  it('libellés de match sur l\'axe X (numéro + carte quand elle est connue)', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const x = opt.xAxis as unknown as AxisOpt
    expect(x.data).toEqual(['#1', '#2 · Streets'])
  })

  it('joueur sans donnée → courbe présente, entièrement null (pas de trou de mise en page)', () => {
    const opt = buildSquadEfficiencyOption({ Me: rows.Me, F1: [] }, PLAYERS, opts('offensive'))
    const series = opt.series as unknown as PlayerSeries[]
    expect(series).toHaveLength(2)
    expect(series[1].name).toBe('F1')
    expect(series[1].data).toEqual([null, null])
    // Aucun pont au-dessus d'un match non joué.
    expect(series[0].connectNulls).toBe(false)
  })

  it('survol : taux en % + dégâts BRUTS + dégâts par frag effectif, tous joueurs du match', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const series = opt.series as unknown as PlayerSeries[]
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    const html = formatter([
      { seriesName: 'Me', color: '#aaa', dataIndex: 0, data: series[0].data[0] },
      { seriesName: 'F1', color: '#bbb', dataIndex: 0, data: series[1].data[0] },
    ])
    expect(html).toContain('#1')
    expect(html).toContain('Me')
    expect(html).toContain('108 %')
    expect(html).toContain('Dégâts infligés 1800')
    expect(html).toContain('200 / frag effectif')
    expect(html).toContain('F1')
    expect(html).toContain('90 %')
    // Le pivot est rappelé : c'est la clé de lecture de la carte.
    expect(html).toContain('Rendement — 100 % = 1 vie')
    // Le mot « élite » a disparu de ces cartes.
    expect(html).not.toContain('élite')
  })

  it('survol défensif : libellés et unité de la métrique défensive', () => {
    const opt = buildSquadEfficiencyOption(
      { Me: [pt(0, { resistance_defensive: 1.65, damage_taken: 1200, deaths: 4 })] },
      ['Me'],
      opts('defensive'),
    )
    const series = opt.series as unknown as PlayerSeries[]
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    const html = formatter([
      { seriesName: 'Me', color: '#aaa', dataIndex: 0, data: series[0].data[0] },
    ])
    expect(html).toContain('165 %')
    expect(html).toContain('Dégâts subis 1200')
    expect(html).toContain('300 / mort')
    expect(html).toContain('Résistance — 100 % = 1 vie')
  })

  it('survol hors point → chaîne vide, jamais de tooltip fantôme', () => {
    const opt = buildSquadEfficiencyOption(rows, PLAYERS, opts('offensive'))
    const formatter = (opt.tooltip as unknown as { formatter: (p: unknown) => string }).formatter
    expect(formatter([{ seriesName: 'F1', dataIndex: 1, data: null }])).toBe('')
    expect(formatter('pas un tableau')).toBe('')
  })

  it('joueur sans couleur attribuée → repli sur la couleur d\'axe (jamais de trait invisible)', () => {
    const opt = buildSquadEfficiencyOption(
      { Ghost: [pt(0, { rendement_offensif: 1 })] },
      ['Ghost'],
      { metric: 'offensive', colorByPlayer: {}, labels: LABELS },
    )
    const series = opt.series as unknown as PlayerSeries[]
    expect(series[0].lineStyle?.color).toBe('#9ca3af') // fallback --muted-foreground (jsdom)
  })
})
