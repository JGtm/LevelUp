/**
 * TimeseriesEfficiency.test.tsx — « Rendement & Résistance » solo, en TAUX
 * rapporté à une vie (%).
 *
 * Verrouille le cadre de lecture aligné sur les cartes Escouade : conversion des
 * dégâts bruts en taux (le payload Timeseries ne sert pas les indicateurs
 * canoniques), polarité UNIQUE (vert au-dessus de 100 %, pour les DEUX courbes),
 * fenêtre d'axe FIXE 50…200 % jamais dérivée de la session, absence de dégradé
 * de trait, et contenu du survol (taux + valeur brute + rappel du pivot).
 */
import { describe, expect, it, vi } from 'vitest'
import { cleanup, render, screen } from '@testing-library/react'

import type { TimeseriesMatchRow } from '@/lib/api/types'
import {
  ONE_LIFE_DAMAGE,
  ONE_LIFE_RATE_BOUNDS,
  ONE_LIFE_RATE_PCT,
  ONE_LIFE_ZONE_OPACITY,
} from '@/lib/charts/oneLifeWindow'
import { TimeseriesEfficiency } from './TimeseriesSquadAdapted'

let captured: Record<string, unknown> | null = null
vi.mock('echarts-for-react', () => ({
  default: ({ option }: { option: Record<string, unknown> }) => {
    captured = option
    return <div data-testid="echarts-mock" />
  },
}))

interface MarkAreaItem {
  yAxis: number
  itemStyle?: { color: string; opacity: number }
}
type SeriesDatum = { value: number; perEvent: number | null } | null
interface EfficiencySeries {
  name?: string
  data: SeriesDatum[]
  lineStyle?: { color?: unknown; width?: number; type?: string }
  markArea?: { data: MarkAreaItem[][] }
  markLine?: { data: Array<{ yAxis: number }>; label?: { formatter?: string } }
  areaStyle?: unknown
}

/** Match gréé pour un taux offensif choisi : `dealt` dégâts pour `kills` frags. */
function row(i: number, dealt: number, kills: number): TimeseriesMatchRow {
  return {
    match_id: `m${i}`,
    damage_dealt: dealt,
    damage_taken: 900,
    kills,
    assists: 0,
    deaths: 4,
  } as TimeseriesMatchRow
}

const LABELS = {
  rendementLabel: 'Rendement (%)',
  resistanceLabel: 'Résistance (%)',
  refLabel: '1 vie',
  perFragLabel: '/ frag effectif',
  perDeathLabel: '/ mort',
}

interface CapturedOption {
  series: EfficiencySeries[]
  yAxis: { min: number; max: number; axisLabel: { formatter: (v: number) => string } }
  tooltip: { formatter: (params: unknown) => string }
}

async function optionFor(rows: TimeseriesMatchRow[]) {
  // Deux options comparées DANS le même test : on démonte le rendu précédent
  // (l'auto-cleanup de testing-library n'intervient qu'entre les tests).
  cleanup()
  captured = null
  render(<TimeseriesEfficiency rows={rows} title="Rendement" emptyMessage="vide" {...LABELS} />)
  await screen.findByTestId('echarts-mock')
  return captured as unknown as CapturedOption
}

describe('TimeseriesEfficiency — taux « une vie » (%)', () => {
  it('les deux courbes sont des TAUX en % : une vie par frag / par mort = 100 %', async () => {
    // 4 frags pour 900 dégâts = 225 / frag effectif = une vie exactement.
    // 4 morts pour 900 dégâts subis = 225 / mort = une vie exactement.
    const opt = await optionFor([row(0, 900, 4)])
    expect(opt.series[0].data[0]?.value).toBeCloseTo(ONE_LIFE_RATE_PCT, 6)
    expect(opt.series[0].data[0]?.perEvent).toBeCloseTo(ONE_LIFE_DAMAGE, 6)
    expect(opt.series[1].data[0]?.value).toBeCloseTo(ONE_LIFE_RATE_PCT, 6)
    expect(opt.series[1].data[0]?.perEvent).toBe(ONE_LIFE_DAMAGE)
    expect(opt.yAxis.axisLabel.formatter(100)).toBe('100 %')
  })

  it('la polarité s\'inverse à la conversion : dépenser MOINS par frag = taux plus HAUT', async () => {
    // Point clé de la refonte : la courbe offensive était en dégâts bruts (plus
    // bas = mieux), elle est maintenant un taux (plus haut = mieux).
    const opt = await optionFor([row(0, 900, 8), row(1, 900, 2)])
    const efficace = opt.series[0].data[0]!.value // 112,5 dégâts / frag
    const gaspilleur = opt.series[0].data[1]!.value // 450 dégâts / frag
    expect(efficace).toBeGreaterThan(ONE_LIFE_RATE_PCT)
    expect(gaspilleur).toBeLessThan(ONE_LIFE_RATE_PCT)
  })

  it('polarité UNIQUE : vert au-dessus de 100 %, rouge en dessous, zones rendues une fois', async () => {
    const opt = await optionFor([row(0, 1800, 8), row(1, 1000, 3)])
    const zones = opt.series[0].markArea?.data
    expect(zones).toHaveLength(2)
    // Bande haute (100 % → haut de fenêtre) = favorable.
    expect(zones![0][0].yAxis).toBe(ONE_LIFE_RATE_PCT)
    expect(zones![0][0].itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.pos)
    expect(zones![0][1].yAxis).toBe(ONE_LIFE_RATE_BOUNDS.max)
    // Bande basse (bas de fenêtre → 100 %) = défavorable.
    expect(zones![1][0].yAxis).toBe(ONE_LIFE_RATE_BOUNDS.min)
    expect(zones![1][0].itemStyle?.opacity).toBe(ONE_LIFE_ZONE_OPACITY.neg)
    expect(zones![1][1].yAxis).toBe(ONE_LIFE_RATE_PCT)
    // La courbe « Résistance » ne peint pas un second jeu de bandes : les deux
    // courbes partagent désormais la MÊME lecture.
    expect(opt.series[1]?.markArea).toBeUndefined()
  })

  it('cadre de base 50…200 % : deux sessions d\'amplitudes opposées mais DANS la fenêtre, mêmes bornes', async () => {
    // "wide" reste À L'INTÉRIEUR de 50…200 % (60 % / 180 %) : cas nominal, pas
    // de dépassement — cf. le test dédié "hors fenêtre" ci-dessous pour DEC-5.
    const tight = await optionFor([row(0, 900, 4), row(1, 920, 4)])
    const wide = await optionFor([row(0, 750, 2), row(1, 1000, 8)])
    expect(tight.yAxis.min).toBe(ONE_LIFE_RATE_BOUNDS.min)
    expect(tight.yAxis.max).toBe(ONE_LIFE_RATE_BOUNDS.max)
    expect(wide.yAxis.min).toBe(tight.yAxis.min)
    expect(wide.yAxis.max).toBe(tight.yAxis.max)
  })

  it('DEC-5 : un point hors fenêtre 50…200 % élargit l\'axe (jamais en dessous du plancher), zones alignées', async () => {
    // 4000 dégâts pour 1 frag = 5,6 % (bien sous 50) ; 900 dégâts pour 12 frags
    // = 300 % (bien au-dessus de 200). Sans le fix, ces deux points seraient
    // écrêtés par l'axe fixe.
    const opt = await optionFor([row(0, 4000, 1), row(1, 900, 12)])
    expect(opt.yAxis.min).toBeLessThan(ONE_LIFE_RATE_BOUNDS.min)
    expect(opt.yAxis.max).toBeGreaterThan(ONE_LIFE_RATE_BOUNDS.max)
    // Les zones de lecture suivent la même borne élargie (pas de bande orpheline
    // qui s'arrêterait avant le bord réel de l'axe).
    const zones = opt.series[0].markArea?.data
    expect(zones![0][1].yAxis).toBe(opt.yAxis.max)
    expect(zones![1][0].yAxis).toBe(opt.yAxis.min)
  })

  it('repère « 1 vie » tracé à 100 %, plus au barème en dégâts bruts', async () => {
    const opt = await optionFor([row(0, 1800, 8)])
    expect(opt.series[0].markLine?.data).toEqual([{ yAxis: ONE_LIFE_RATE_PCT }])
    expect(opt.series[0].markLine?.label?.formatter).toBe('1 vie')
  })

  it('aucun dégradé de trait : couleur unie, le jugement est porté par les zones', async () => {
    const opt = await optionFor([row(0, 1800, 8)])
    for (const s of opt.series) {
      expect(typeof s.lineStyle?.color).toBe('string')
      expect(s.areaStyle).toBeUndefined()
    }
  })

  it('survol : taux %, valeur brute avec son unité, et rappel du pivot', async () => {
    const opt = await optionFor([row(0, 900, 4)])
    const html = opt.tooltip.formatter([
      { seriesName: LABELS.rendementLabel, marker: '', data: opt.series[0].data[0] },
      { seriesName: LABELS.resistanceLabel, marker: '', data: opt.series[1].data[0] },
    ])
    expect(html).toContain('Rendement (%) : <b>100 %</b>')
    expect(html).toContain('225 / frag effectif')
    expect(html).toContain('225 / mort')
    expect(html).toContain('100 % = 1 vie')
  })

  it('survol hors point → chaîne vide, jamais de tooltip fantôme', async () => {
    const opt = await optionFor([row(0, 900, 4)])
    expect(opt.tooltip.formatter([{ seriesName: LABELS.rendementLabel, data: null }])).toBe('')
    expect(opt.tooltip.formatter('pas un tableau')).toBe('')
  })

  it('match sans frag effectif → point non tracé (jamais de 0 % faux)', async () => {
    const opt = await optionFor([row(0, 900, 0)])
    expect(opt.series[0].data[0]).toBeNull()
  })
})
