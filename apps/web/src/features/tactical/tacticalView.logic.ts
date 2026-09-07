/**
 * tacticalView.logic — la logique PURE de la vue d'analyse tactique (Phase 5, items
 * 5.2-5.6). Rien ne dépend de React ni du DOM : titre de page, messages de statut,
 * unité par question, projection du raster serveur en grille de peinture (`heatPaint`)
 * et position (col, row) d'un clic sur le canvas. Les composants ne font que rendre ce
 * que ces fonctions décident (règle du dépôt : pas de logique métier dans un composant).
 */
import type { BornesMonde, CelluleTactique, EchelleTactique } from '@/lib/api/types'

import { buildTacticalGrid, type TacticalGrid } from '@/lib/replay/heatPaint'
import type { TacticalText } from './i18n'

/** Les six lectures offertes par la barre d'outils — même vocabulaire que le contrat
 *  (`TacticalRasterBody.question`). */
export type TacticalQuestion = 'morts' | 'kills' | 'gagne' | 'temps' | 'routes' | 'isole'

/** L'axe « qui » — même vocabulaire que le contrat (`TacticalRasterBody.qui`). */
export type TacticalQui = 'moi' | 'escouade' | 'adv'

/**
 * Plancher de mesure par cellule : 3 matchs distincts. MÊME SEUIL QUE LE SERVEUR
 * (calibration mesurée de `mappos-build`, `.ai/PLAN_TACTIQUE_2026-09-06.md` §6,
 * "Plancher par cellule") — affiché, jamais recalculé : le serveur ne publie que les
 * cellules déjà au-dessus.
 */
export const TACTICAL_CELL_FLOOR = 3

/** Questions qui exigent l'artefact de rejeu (position datée), pas seulement le journal
 *  des morts — même liste que la doc du contrat (`TacticalRasterBody.question`). */
const QUESTIONS_ARTEFACT_REJEU: ReadonlySet<TacticalQuestion> = new Set(['temps', 'routes'])

/** pageTitle — « Plan de <carte> — <question> », le titre H2 de la vue. */
export function pageTitle(t: TacticalText, mapName: string, question: TacticalQuestion): string {
  const libelle = t.analysisQuestions.find((q) => q.id === question)?.label ?? question
  return t.analysisPageTitle(mapName, libelle)
}

/** unitForQuestion — l'unité affichée en légende du plan et sur la cellule sélectionnée. */
export function unitForQuestion(t: TacticalText, question: TacticalQuestion): string {
  return t.units[question]
}

/** sourceForQuestion — la provenance de la mesure, affichée au pied du plan. */
export function sourceForQuestion(t: TacticalText, question: TacticalQuestion): string {
  return QUESTIONS_ARTEFACT_REJEU.has(question) ? t.sourceReplay : t.sourceJournal
}

/**
 * statusMessages — les bandeaux « en attente » / « non disponible » au-dessus du plan.
 * LES DEUX PEUVENT COEXISTER (des matchs en cours de cuisson ET d'autres jamais
 * cuisables) : ce ne sont pas des échecs de la lecture, ce sont des dénominateurs qui
 * varient. Aucun message quand les deux compteurs sont à zéro.
 */
export function statusMessages(
  t: TacticalText,
  matchsEnAttente: number,
  matchsNonCuisables: number,
): string[] {
  const messages: string[] = []
  if (matchsEnAttente > 0) messages.push(t.statusPending(matchsEnAttente))
  if (matchsNonCuisables > 0) messages.push(t.statusUnavailable(matchsNonCuisables))
  return messages
}

/** ratioSafe — une proportion 0..1, jamais une division par zéro. */
export function ratioSafe(numerateur: number, denominateur: number): number {
  if (!(denominateur > 0)) return 0
  return numerateur / denominateur
}

/**
 * tacticalGridFromRaster — la grille de peinture (`heatPaint.TacticalGrid`) dérivée du
 * raster serveur : bornes + pas de grille + cellules + échelle p50/p95. `null` si les
 * bornes ne sont pas exploitables (aucun point localisé pour cette question).
 */
export function tacticalGridFromRaster(
  cellules: readonly CelluleTactique[],
  bornes: BornesMonde,
  pasM: number,
  echelle: EchelleTactique,
): TacticalGrid | null {
  if (!bornes.valide || !(pasM > 0)) return null
  const nx = Math.max(1, Math.ceil((bornes.max_x - bornes.min_x) / pasM))
  const ny = Math.max(1, Math.ceil((bornes.max_y - bornes.min_y) / pasM))
  const cells = cellules.map((c) => ({ col: c.col, row: c.lig, value: c.valeur }))
  return buildTacticalGrid(
    cells,
    { cell: pasM, nx, ny, minX: bornes.min_x, minY: bornes.min_y },
    { lo: echelle.p50, hi: echelle.p95 },
    echelle.n_cellules,
  )
}

/**
 * cellFromClick — la cellule (col, row) sous un clic sur le canvas.
 *
 * Le canvas peint le monde [min_x, max_x] × [min_y, max_y] EXACTEMENT sur toute sa
 * surface (aucune marge, aucun pan) : c'est la même convention que `TacticalPlanCard`
 * utilise pour peindre le fond ET la heatmap, donc le clic s'inverse par une simple
 * règle de trois. `null` si les bornes sont invalides, le canvas est vide, ou le clic
 * tombe hors de sa surface.
 */
export function cellFromClick(
  clickX: number,
  clickY: number,
  canvasWidth: number,
  canvasHeight: number,
  bornes: BornesMonde,
  pasM: number,
): { col: number; row: number } | null {
  if (!bornes.valide || !(pasM > 0) || canvasWidth <= 0 || canvasHeight <= 0) return null
  if (clickX < 0 || clickY < 0 || clickX > canvasWidth || clickY > canvasHeight) return null
  const worldX = bornes.min_x + (clickX / canvasWidth) * (bornes.max_x - bornes.min_x)
  const worldY = bornes.min_y + (clickY / canvasHeight) * (bornes.max_y - bornes.min_y)
  return {
    col: Math.floor((worldX - bornes.min_x) / pasM),
    row: Math.floor((worldY - bornes.min_y) / pasM),
  }
}

/** trouveCellule — la cellule serveur à (col, row), ou `null` si jamais atteinte. */
export function trouveCellule(
  cellules: readonly CelluleTactique[],
  col: number,
  row: number,
): CelluleTactique | null {
  return cellules.find((c) => c.col === col && c.lig === row) ?? null
}
