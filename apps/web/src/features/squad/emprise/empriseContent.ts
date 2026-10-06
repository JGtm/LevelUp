/**
 * empriseContent.ts — CE QUE L'ONGLET EMPRISE A À MONTRER (lot L5 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, décision D10). Un seul prédicat, lu par la page
 * (bloc par bloc, état vide) ET par la barre d'onglets de l'Escouade (l'onglet se masque sans
 * rien à montrer) : les deux ne peuvent pas diverger.
 *
 * Sans film (Halo 5 : le titre ne publie pas de résumé d'usage, raison `film_unsupported` posée
 * côté Go par capability), le bloc ne porte que la feuille de match : les frags aux armes
 * spéciales. L'onglet ne garde alors que « Frags obtenus avec les ressources », barre épaisse
 * seule — la grille match par match exige au moins une ligne lue au film (la seule ligne des
 * frags ne fait pas une grille). Aucun branchement sur le titre : tout se lit dans le bloc.
 */
import type { SquadEmpriseBlock, SquadEmprisePlacement } from '@/lib/api/types'

import { buildControlRows, buildMatchGrid, buildPickupSheets, type MatchGrid, type PickupSheets } from './emprise.logic'
import { buildHabitView, type HabitView } from './habit.logic'
import { buildProductionRows, buildYieldRows, type ProductionRow, type YieldRow } from './production.logic'

/** La grille a-t-elle au moins une ligne lue au film (synthèse ou objet) ? */
export function gridHasFilmRows(grid: MatchGrid | null): boolean {
  return grid != null && grid.columns.length > 0 && grid.sections.some((s) => s.summary != null || s.items.length > 0)
}

/** Les fiches ont-elles au moins une ligne ? */
export function sheetsHaveLines(sheets: PickupSheets | null): boolean {
  return sheets != null && sheets.sections.some((s) => s.lines.length > 0)
}

export interface EmpriseSections {
  bilan: boolean
  roles: boolean
  carte: boolean
  prendre: boolean
  placement: boolean
  habitude: boolean
}

/** Le bloc « Groupés ou isolés » a-t-il au moins une vie mesurée à tracer ? (Sans portée de radar ou sans film : absent.) */
export function placementHasLives(placement: SquadEmprisePlacement | null | undefined): boolean {
  return placement != null && (placement.players ?? []).some((p) => p.lives_measured > 0)
}

/** Les blocs à rendre, depuis les modèles des cartes. */
export function empriseSections(m: {
  controlRows: unknown[]
  sheets: PickupSheets | null
  grid: MatchGrid | null
  production: ProductionRow[]
  yieldRows: YieldRow[]
  habit: HabitView
  placement: SquadEmprisePlacement | null | undefined
}): EmpriseSections {
  return {
    bilan: m.controlRows.length > 0,
    roles: sheetsHaveLines(m.sheets),
    carte: gridHasFilmRows(m.grid),
    prendre: m.production.length > 0 || m.yieldRows.length > 0,
    placement: placementHasLives(m.placement),
    habitude: m.habit.kind !== 'none',
  }
}

/** L'onglet a-t-il quelque chose à montrer ? (Faux sans bloc.) */
export function empriseHasContent(block: SquadEmpriseBlock | null | undefined): boolean {
  if (!block) return false
  const s = empriseSections({
    controlRows: buildControlRows(block),
    sheets: buildPickupSheets(block, (o) => o.key),
    grid: buildMatchGrid(block, new Map()),
    production: buildProductionRows(block),
    yieldRows: buildYieldRows(block),
    habit: buildHabitView(block),
    placement: block.placement,
  })
  return Object.values(s).some(Boolean)
}
