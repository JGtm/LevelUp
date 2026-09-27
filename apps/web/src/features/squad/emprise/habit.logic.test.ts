/**
 * habit.logic.test.ts — « Contrôle des ressources, soirée après soirée » (D5) : les soirées de
 * la maquette (cinq précédentes, puis ce soir 60 % / 44 %), la médiane seulement à partir de
 * trois soirées précédentes, et les cas sans historique.
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

  it('aucune soirée précédente : ce soir seul ; pas d’habitude publiée : rien', () => {
    const alone = buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, previous: [] } })
    expect(alone.kind).toBe('noHistory')
    expect(buildHabitView({ ...EMPRISE_2209, habit: undefined }).kind).toBe('none')
    expect(buildHabitView({ ...EMPRISE_2209, habit: { ...HABIT_2209, current: { ...HABIT_2209.current, shares: [] } } }).kind).toBe('none')
  })
})
