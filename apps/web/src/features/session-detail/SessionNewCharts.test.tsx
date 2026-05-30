/**
 * Tests des builders d'option ECharts des nouveaux graphes session-detail :
 * participation (miroir), net score cumulé, dumbbell MMR, breakdown modes, donut centré.
 */
import { describe, expect, it } from 'vitest'

import { buildDonutOption } from '@/components/charts/DonutChart'

import { buildSessionParticipationBarsOption } from './SessionParticipationBars'
import { buildSessionNetScoreOption } from './SessionNetScoreArea'
import { buildSessionMmrDumbbellOption } from './SessionMmrDumbbell'
import { buildSessionModeBreakdownOption } from './SessionModeBreakdown'
import { buildSessionDamageOption } from './SessionDamageComposite'
import { buildSessionPlacementOption } from './SessionPlacementBreakdown'

/* eslint-disable @typescript-eslint/no-explicit-any */

describe('buildSessionParticipationBarsOption — miroir gauche/droite', () => {
  const series = [
    { key: 'p', datapoints: [{ label: 'Combat', value: 78 }, { label: 'Support', value: 55 }] },
  ]

  it('axisSide=right : axe catégories à droite, barres vers la gauche (xAxis inversé)', () => {
    const opt = buildSessionParticipationBarsOption(series, { axisSide: 'right', colorToken: 'compare-a' }) as any
    expect(opt.yAxis.position).toBe('right')
    expect(opt.xAxis.inverse).toBe(true)
    expect(opt.xAxis.max).toBe(100)
    expect(opt.series[0].label.position).toBe('left') // valeur côté extérieur (gauche)
  })

  it('axisSide=left : axe catégories à gauche, barres vers la droite (xAxis normal)', () => {
    const opt = buildSessionParticipationBarsOption(series, { axisSide: 'left', colorToken: 'compare-b' }) as any
    expect(opt.yAxis.position).toBe('left')
    expect(opt.xAxis.inverse).toBe(false)
    expect(opt.series[0].label.position).toBe('right')
  })
})

describe('buildSessionNetScoreOption', () => {
  it('courbe en aire + visualMap divergent (pos/neg)', () => {
    const opt = buildSessionNetScoreOption(
      [{ key: 'n', datapoints: [{ label: '#1', cumulative: 3 }, { label: '#2', cumulative: -2 }] }],
      { seriesLabel: 'Net' },
    ) as any
    expect(opt.series[0].type).toBe('line')
    expect(opt.series[0].areaStyle).toBeDefined()
    expect(opt.series[0].data).toEqual([3, -2])
    expect(opt.visualMap.pieces).toHaveLength(2)
    expect(opt.xAxis.boundaryGap).toBe(false)
  })

  it('option vide sans points', () => {
    const opt = buildSessionNetScoreOption([], { seriesLabel: 'Net' }) as any
    expect(opt.series).toBeUndefined()
  })
})

describe('buildSessionMmrDumbbellOption', () => {
  it('1 série custom (liaisons) + 2 scatter (équipe/adverse) + légende', () => {
    const opt = buildSessionMmrDumbbellOption(
      [{ key: 'm', datapoints: [{ label: '#1', team: 1420, enemy: 1505 }] }],
      { teamLabel: 'Équipe', enemyLabel: 'Adverse' },
    ) as any
    expect(opt.series).toHaveLength(3)
    expect(opt.series[0].type).toBe('custom')
    expect(opt.series[1].name).toBe('Équipe')
    expect(opt.series[2].name).toBe('Adverse')
    expect(opt.series[2].symbol).toBe('diamond')
    expect(opt.legend.data).toEqual(['Équipe', 'Adverse'])
    expect(opt.yAxis.inverse).toBe(true) // #1 en haut
  })
})

describe('buildSessionModeBreakdownOption', () => {
  it('barres verticales = comptes par mode', () => {
    const opt = buildSessionModeBreakdownOption(
      [{ key: 'm', datapoints: [{ mode: 'Slayer', count: 5 }, { mode: 'CTF', count: 2 }] }],
      { countLabel: 'matchs' },
    ) as any
    expect(opt.series[0].type).toBe('bar')
    expect(opt.series[0].data.map((d: { value: number }) => d.value)).toEqual([5, 2])
    expect(opt.xAxis.data).toEqual(['Slayer', 'CTF'])
    expect(opt.yAxis.minInterval).toBe(1)
  })
})

describe('buildSessionDamageOption', () => {
  it('2 barres empilées (infligés/subis) horizontales par match', () => {
    const opt = buildSessionDamageOption(
      [{ key: 'd', datapoints: [{ label: '#1', dealt: 3000, taken: 1500 }] }],
      { dealtLabel: 'Infligés', takenLabel: 'Subis' },
    ) as any
    expect(opt.series).toHaveLength(2)
    expect(opt.series[0].stack).toBe('dmg')
    expect(opt.series[1].stack).toBe('dmg')
    expect(opt.series[0].data).toEqual([3000])
    expect(opt.series[1].data).toEqual([1500])
    expect(opt.yAxis.type).toBe('category')
    expect(opt.xAxis.type).toBe('value')
    expect(opt.legend.data).toEqual(['Infligés', 'Subis'])
  })

  it('option vide sans points', () => {
    const opt = buildSessionDamageOption([], { dealtLabel: 'A', takenLabel: 'B' }) as any
    expect(opt.series).toBeUndefined()
  })
})

describe('buildSessionPlacementOption', () => {
  it('barres verticales par placement (#1..#N) + comptes', () => {
    const opt = buildSessionPlacementOption(
      [
        {
          key: 'p',
          datapoints: [
            { placement: 1, count: 3 },
            { placement: 2, count: 1 },
            { placement: 3, count: 0 },
            { placement: 4, count: 2 },
          ],
        },
      ],
      { countLabel: 'matchs' },
    ) as any
    expect(opt.series[0].type).toBe('bar')
    expect(opt.xAxis.data).toEqual(['#1', '#2', '#3', '#4'])
    expect(opt.series[0].data.map((d: { value: number }) => d.value)).toEqual([3, 1, 0, 2])
    expect(opt.yAxis.minInterval).toBe(1)
  })
})

describe('buildDonutOption — label central', () => {
  it('ajoute un titre centré (valeur + libellé) quand centerValue est fourni', () => {
    const opt = buildDonutOption(
      [{ key: 'o', datapoints: [{ name: 'Victoires', value: 6 }, { name: 'Défaites', value: 4 }] }],
      { centerValue: '60 %', centerLabel: 'Victoires' },
    ) as any
    expect(opt.title.text).toBe('60 %')
    expect(opt.title.subtext).toBe('Victoires')
    expect(opt.title.left).toBe('center')
  })

  it('pas de titre central sans centerValue', () => {
    const opt = buildDonutOption(
      [{ key: 'o', datapoints: [{ name: 'A', value: 1 }] }],
      {},
    ) as any
    expect(opt.title).toBeUndefined()
  })
})
