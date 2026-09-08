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

/**
 * TACTICAL_REPLAY_FRAME_INTERVAL_MS — le pas d'échantillonnage de l'artefact de rejeu, en
 * millisecondes. MÊME VALEUR que `analysis/replay.DefaultFrameIntervalMS` côté Go
 * (source unique, dépôt) : le contrat `TacticalCelluleReponse` ne la publie pas encore par
 * contribution — elle est aujourd'hui CONSTANTE pour tout le parc (aucune variation par
 * titre ou par match observée). Si un jour l'artefact adopte un pas variable, cette
 * constante devra devenir un champ du contrat plutôt qu'une hypothèse côté web.
 */
export const TACTICAL_REPLAY_FRAME_INTERVAL_MS = 100

/**
 * instantToFrame — convertit un instant (millisecondes) en index de frame, sur l'axe que
 * `playbackStore`/`?frame=` du lecteur 2D consomment (`lib/replay/replayLogic.frameToMs` :
 * `frame * frameIntervalMs` = ms écoulées depuis le début du rejeu).
 *
 * CONVERSION MÉCANIQUE SEULE — ELLE NE CORRIGE AUCUN DÉCALAGE D'HORLOGE. Vérifié sur pièces
 * (lot M1, item 1) : `instant_ms` n'est PAS toujours sur le même axe temporel selon la
 * question (cf. la doc de `TacticalContribution.InstantMs` côté Go) —
 *
 *   temps, routes            l'instant est DÉJÀ sur l'horloge du FILM (ms écoulées depuis
 *                            le début du rejeu) : la frame obtenue est EXACTE.
 *   morts, kills, gagne,
 *   isole                    l'instant vient de l'horloge du MATCH (`match_kill_events.
 *                            time_ms`), qui diffère de celle du film d'un décalage PAR
 *                            MATCH (`DeathOffsetMS`, `analysis/replay/lives_export.go`)
 *                            NON PUBLIÉ dans ce contrat. La frame obtenue est donc une
 *                            APPROXIMATION (écart mesuré de 3,6 à 50,8 s sur les films
 *                            témoins, `analysis/replay/origin.go`) — décalage constant sur
 *                            tout le match, pas un bruit aléatoire, mais réel.
 *
 * Décision du lot M1 (découverte consignée, non traitée : `.ai/DECOUVERTES_TACTIQUE_2026-09-07.md`) :
 * le lien est construit pour LES SIX questions (le contrat ne distingue pas la provenance
 * de l'instant), avec cette réserve documentée plutôt qu'un lien manquant pour quatre
 * questions sur six.
 */
export function instantToFrame(
  instantMs: number,
  frameIntervalMs: number = TACTICAL_REPLAY_FRAME_INTERVAL_MS,
): number {
  if (!(frameIntervalMs > 0) || !(instantMs >= 0)) return 0
  return Math.round(instantMs / frameIntervalMs)
}
