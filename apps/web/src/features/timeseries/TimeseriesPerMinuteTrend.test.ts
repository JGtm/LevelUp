/**
 * Tests du graphe « Stats par minute » (timeseries.14) : cadences par match, courbes de dégâts sur
 * l'axe de droite, absence (pas zéro) des valeurs inconnues.
 */
import { describe, expect, it } from 'vitest'

import type { TimeseriesMatchRow } from '@/lib/api/types'

import { buildPerMinuteTrendOption, type PerMinuteTrendLabels } from './TimeseriesPerMinuteTrend'
import { matchPace, paceHasDamage } from './timeseriesPerMinute.logic'

function row(o: Partial<TimeseriesMatchRow>): TimeseriesMatchRow {
  return { start_time: '2026-01-01T00:00:00Z', kills: 0, deaths: 0, assists: 0, ...o } as TimeseriesMatchRow
}

const LABELS: PerMinuteTrendLabels = {
  kills: 'Frags / min',
  deaths: 'Morts / min',
  assists: 'Assistances / min',
  damageDealt: 'Dégâts infligés / min',
  damageTaken: 'Dégâts subis / min',
  damageAxis: 'Dégâts / min',
  suffix: ' /min',
}

interface Opt {
  series: Array<{ type: string; name: string; data: Array<number | null>; yAxisIndex?: number }>
  yAxis: Array<{ min?: number; max?: number; position?: string; axisLabel: { formatter: (v: number) => string } }>
}

const ROWS = [
  row({ kills: 10, deaths: 5, assists: 2, time_played_seconds: 600, damage_dealt: 3000, damage_taken: 1200 }),
  row({ kills: 3, deaths: 1, assists: 1, time_played_seconds: 180, damage_dealt: 900, damage_taken: 300 }),
  row({ kills: 4, deaths: 4, assists: 0, time_played_seconds: 600 }), // dégâts inconnus
  row({ kills: 7, deaths: 7, assists: 7, damage_dealt: 7777, damage_taken: 7777 }), // durée inconnue
]

describe('matchPace', () => {
  it('rapporte chaque compte aux minutes jouées du match', () => {
    const p = matchPace(ROWS[0], true)
    expect(p.kills).toBeCloseTo(1, 10)
    expect(p.deaths).toBeCloseTo(0.5, 10)
    expect(p.damageDealt).toBeCloseTo(300, 10)
    expect(p.damageTaken).toBeCloseTo(120, 10)
  })
  it('durée inconnue : aucune cadence ; dégât inconnu : absent, pas zéro', () => {
    expect(matchPace(ROWS[3], true)).toEqual({ kills: null, deaths: null, assists: null, damageDealt: null, damageTaken: null })
    const p = matchPace(ROWS[2], true)
    expect(p.kills).toBeCloseTo(0.4, 10)
    expect(p.damageDealt).toBeNull()
    expect(p.damageTaken).toBeNull()
  })
  it('titre sans dégâts subis : absents même avec une valeur rangée', () => {
    expect(matchPace(ROWS[0], false).damageTaken).toBeNull()
  })
})

describe('paceHasDamage', () => {
  it('vrai dès qu un match porte une cadence de dégâts', () => {
    expect(paceHasDamage(ROWS, true)).toBe(true)
  })
  it('faux sans dégâts connus, ou sans durée', () => {
    expect(paceHasDamage([ROWS[2], ROWS[3]], true)).toBe(false)
    expect(paceHasDamage([], true)).toBe(false)
  })
  it('faux quand seuls des dégâts subis existent et que le titre ne les fournit pas', () => {
    expect(paceHasDamage([row({ kills: 1, time_played_seconds: 60, damage_taken: 100 })], false)).toBe(false)
  })
})

describe('buildPerMinuteTrendOption', () => {
  it('sans match : pas d option', () => {
    expect(buildPerMinuteTrendOption([], LABELS, true)).toBeNull()
  })

  it('trois colonnes par match, morts sous le zéro, trou pour une durée inconnue', () => {
    const opt = buildPerMinuteTrendOption(ROWS, LABELS, true) as unknown as Opt
    const [kills, deaths, assists] = opt.series
    expect(kills.type).toBe('bar')
    expect(kills.data).toEqual([1, 1, 0.4, null])
    expect(deaths.data).toEqual([-0.5, -0.33, -0.4, null])
    expect(assists.data).toEqual([0.2, 0.33, 0, null])
  })

  it('deux courbes de dégâts sur l axe de droite ; match sans dégâts = trou', () => {
    const opt = buildPerMinuteTrendOption(ROWS, LABELS, true) as unknown as Opt
    expect(opt.series).toHaveLength(5)
    const [dealt, taken] = opt.series.slice(3)
    expect(dealt.type).toBe('line')
    expect(dealt.yAxisIndex).toBe(1)
    expect(dealt.name).toBe(LABELS.damageDealt)
    expect(dealt.data).toEqual([300, 300, null, null])
    expect(taken.name).toBe(LABELS.damageTaken)
    expect(taken.data).toEqual([120, 100, null, null])
  })

  it('axe de droite aligné sur le zéro de l axe de gauche, sans étiquette négative', () => {
    const opt = buildPerMinuteTrendOption(ROWS, LABELS, true) as unknown as Opt
    expect(opt.yAxis).toHaveLength(2)
    const [l, r] = opt.yAxis
    expect(r.position).toBe('right')
    const share = (a: { min?: number; max?: number }) => -a.min! / (a.max! - a.min!)
    expect(share(l)).toBeCloseTo(share(r), 10)
    expect(r.max!).toBeGreaterThanOrEqual(300)
    expect(r.axisLabel.formatter(r.min!)).toBe('')
  })

  it('titre sans dégâts subis : la courbe « subis » est retirée', () => {
    const opt = buildPerMinuteTrendOption(ROWS, LABELS, false) as unknown as Opt
    expect(opt.series.map((s) => s.name)).toEqual([
      LABELS.kills,
      LABELS.deaths,
      LABELS.assists,
      LABELS.damageDealt,
    ])
  })

  it('aucun dégât connu : ni courbe ni axe de droite', () => {
    const opt = buildPerMinuteTrendOption([ROWS[2]], LABELS, true) as unknown as Opt
    expect(opt.series).toHaveLength(3)
    expect(opt.yAxis).toHaveLength(1)
  })
})
