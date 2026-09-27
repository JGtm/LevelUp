/**
 * habit.logic.ts — LE MODÈLE PUR de « Contrôle des ressources, soirée après soirée » (bloc
 * « Par rapport à d'habitude » de l'onglet Emprise ; lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décision D5 ; maquette de l'onglet,
 * `renderHabChart('habPrises', …)`).
 *
 * Source : `squad_emprise.habit` (Go, lot L4) — les soirées précédentes comparables de la
 * composition (familles de mode jouées ce soir, dix au plus, de la plus ancienne à la plus
 * récente) puis ce soir ; par soirée, notre part des prises de chaque ressource. Le web ne fait
 * que ranger : une soirée par point, ce soir en dernier, et la médiane des soirées précédentes
 * par ressource quand il y en a au moins trois (D5). Pur : aucun React, aucune couleur, aucune
 * chaîne de langue.
 */
import type { SquadEmpriseBlock, SquadEmpriseEvening } from '@/lib/api/types'

import { median } from '../objectif/objectif.logic'
import { RESOURCE_ORDER } from './emprise.logic'

/** La médiane n'est tracée qu'à partir de trois soirées précédentes (D5). */
export const HABIT_MIN_PREVIOUS_FOR_MEDIAN = 3

export interface HabitPoint {
  current: boolean
  startTime: string
  /** Notre part des prises en POURCENT (0..100) par ressource ; null = ressource absente ce soir-là. */
  shares: Record<string, number | null>
}

export type HabitView =
  /** Rien à montrer : pas d'habitude publiée, ou aucune part ce soir. */
  | { kind: 'none' }
  /** Ce soir seul : aucune soirée précédente comparable. */
  | { kind: 'noHistory'; resources: string[]; current: HabitPoint }
  | { kind: 'chart'; resources: string[]; points: HabitPoint[]; medians: Record<string, number | null> }

function toPoint(e: SquadEmpriseEvening, current: boolean): HabitPoint {
  const shares: Record<string, number | null> = {}
  for (const s of e.shares ?? []) shares[s.resource] = s.share * 100
  return { current, startTime: e.start_time, shares }
}

/**
 * buildHabitView — les ressources tracées (celles qui ont une part ce soir ou une soirée
 * précédente, dans l'ordre de l'onglet), les soirées précédentes puis ce soir, et la médiane
 * des précédentes par ressource (null sous trois soirées qui la portent).
 */
export function buildHabitView(block: SquadEmpriseBlock): HabitView {
  const habit = block.habit
  if (!habit) return { kind: 'none' }
  const current = toPoint(habit.current, true)
  const previous = (habit.previous ?? []).map((e) => toPoint(e, false))
  const points = [...previous, current]
  const resources = RESOURCE_ORDER.filter((r) => points.some((p) => p.shares[r] != null))
  if (!resources.some((r) => current.shares[r] != null)) return { kind: 'none' }
  if (previous.length === 0) return { kind: 'noHistory', resources, current }
  const medians: Record<string, number | null> = {}
  for (const r of resources) {
    const past = previous.flatMap((p) => (p.shares[r] == null ? [] : [p.shares[r] as number]))
    medians[r] = past.length >= HABIT_MIN_PREVIOUS_FOR_MEDIAN ? median(past) : null
  }
  return { kind: 'chart', resources, points, medians }
}
