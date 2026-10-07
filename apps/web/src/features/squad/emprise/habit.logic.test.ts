/**
 * habit.logic.test.ts — « Contrôle des ressources, soirée après soirée » (D5) : les soirées de
 * la maquette (cinq précédentes, puis ce soir 60 % / 44 %), la médiane seulement à partir de
 * trois soirées précédentes COMPARABLES, les soirées hors comparaison, et les cas sans historique.
 */
import { describe, expect, it } from 'vitest'

import type { SquadEmpriseBlock } from '@/lib/api/types'

import { EMPRISE_2209, HABIT_2209 } from './emprise.fixtures'
import { EMPRISE_TEXT } from './empriseStrings'
import { buildHabitView } from './habit.logic'

describe('buildHabitView', () => {
  it('22/09 : cinq soirées précédentes puis ce soir, bonus 60 % et armes spéciales 44 % ; médianes 60 et 50', () => {
    const view = buildHabitView(EMPRISE_2209)
    if (view.kind !== 'chart') throw new Error(view.kind)
    expect(view.resources).toEqual(['powerup', 'power_weapon'])
    expect(view.points).toHaveLength(6)
    const tonight = view.points[5]
    expect(tonight.current).toBe(true)
    expect(EMPRISE_TEXT.fr.pctIntFmt(tonight.shares.powerup!)).toBe('60 %')
    expect(EMPRISE_TEXT.fr.pctIntFmt(tonight.shares.power_weapon!)).toBe('44 %')
    expect(view.points.slice(0, 5).map((p) => Math.round(p.shares.powerup! * 10) / 10)).toEqual([58.3, 65.1, 75, 54.2, 60])
    expect(view.medians.powerup).toBeCloseTo(60, 5)
    expect(view.medians.power_weapon).toBeCloseTo(50, 5)
  })

  it('sous trois soirées précédentes, pas de médiane', () => {
    const block: SquadEmpriseBlock = { ...EMPRISE_2209, habit: { ...HABIT_2209, previous: HABIT_2209.previous!.slice(0, 2) } }
    const view = buildHabitView(block)
    if (view.kind !== 'chart') throw new Error(view.kind)
    expect(view.medians).toEqual({ powerup: null, power_weapon: null })
  })

  it('une soirée hors comparaison reste un point, grisé, et sort de la médiane', () => {
    const previous = HABIT_2209.previous!.map((e, i) => (i === 2 ? { ...e, comparable: false, families: ['Bases'] } : e))
    const view = buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, previous } })
    if (view.kind !== 'chart') throw new Error(view.kind)
    expect(view.points).toHaveLength(6)
    expect(view.points.map((p) => p.comparable)).toEqual([true, true, false, true, true, true])
    expect(view.points[2].families).toEqual(['Bases'])
    // Médiane des bonus sur 58,3 / 65,1 / 54,2 / 60 (le 75 % du 27/08 écarté).
    expect(view.medians.powerup).toBeCloseTo((58.333 + 60) / 2, 1)
  })

  it('aucune soirée précédente : ce soir seul, en graphe ; aucune part nulle part : placeholder ; pas d’habitude : rien', () => {
    const alone = buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, previous: [] } })
    if (alone.kind !== 'chart') throw new Error(alone.kind)
    expect(alone.points).toHaveLength(1)
    expect(alone.medians).toEqual({ powerup: null, power_weapon: null })
    expect(buildHabitView({ ...EMPRISE_2209, habit: undefined }).kind).toBe('none')
    const sansPart = { ...HABIT_2209, previous: [], current: { ...HABIT_2209.current, shares: [] } }
    expect(buildHabitView({ ...EMPRISE_2209, habit: sansPart }).kind).toBe('empty')
  })

  it('ce soir sans part, soirées précédentes avec : le graphe reste, sans point ce soir', () => {
    const view = buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, current: { ...HABIT_2209.current, shares: [] } } })
    if (view.kind !== 'chart') throw new Error(view.kind)
    expect(view.points[5].shares).toEqual({})
  })
})
