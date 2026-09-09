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

/**
 * Les trois causes d'un plan vide. `null` = le plan n'est pas vide.
 *
 * ELLES NE SE DISENT PAS PAREIL, et c'est tout l'objet de cette distinction (point 21 des
 * retours utilisateur, lot 3.2) :
 *
 *	aucun-match     le FILTRE ne retient aucun match sur cette carte. Rien n'a été mesuré
 *	                parce que rien n'a été joué dans ce périmètre.
 *	aucune-mesure   des matchs, mais aucun mesurable — film jamais décodé, journal des
 *	                morts illisible. C'est le message historique, et il reste vrai ici.
 *	densite         des matchs MESURÉS, mais trop dispersés : aucune zone n'atteint le
 *	                plancher de 3 matchs distincts, même à la grille la plus grossière.
 *
 * LE DÉFAUT CORRIGÉ : le troisième cas affichait le message du deuxième. Sur Illusion,
 * 38 matchs étaient retenus ET mesurés, et la page répondait « pas assez de matchs
 * mesurés » — un message qui envoie élargir un filtre déjà large, pour un problème qui
 * n'est pas là.
 */
export type TacticalPlanEmptyReason = 'aucun-match' | 'aucune-mesure' | 'densite'

/**
 * planEmptyReason — pourquoi le plan est vide, ou `null` s'il ne l'est pas.
 *
 * La cause se lit sur les DEUX dénominateurs déjà publiés par le contrat
 * (`matchs_filtres`, `matchs_retenus`) et sur le nombre de cellules peintes : aucun champ
 * supplémentaire n'est nécessaire, et aucune règle n'est recalculée côté client — le
 * serveur ne publie que les cellules déjà au-dessus du plancher.
 */
export function planEmptyReason(
  cellulesPeintes: number,
  matchsRetenus: number,
  matchsFiltres: number,
): TacticalPlanEmptyReason | null {
  if (cellulesPeintes > 0) return null
  if (!(matchsFiltres > 0)) return 'aucun-match'
  if (!(matchsRetenus > 0)) return 'aucune-mesure'
  return 'densite'
}

/** planEmptyText — le titre et la description à afficher pour une cause donnée. */
export function planEmptyText(
  t: TacticalText,
  raison: TacticalPlanEmptyReason,
  matchsRetenus: number,
  pasM: number,
): { title: string; description: string } {
  switch (raison) {
    case 'aucun-match':
      return { title: t.planEmptyNoMatchTitle, description: t.planEmptyNoMatchDescription }
    case 'aucune-mesure':
      return { title: t.planEmptyTitle, description: t.planEmptyDescription }
    default:
      return {
        title: t.planEmptyDensityTitle,
        // LE PAS CITÉ EST CELUI QUE LA LECTURE A RETENU : quand aucune densité ne suffit,
        // c'est le plus grossier essayé, et le dire évite qu'on croie le plan calculé
        // à 0,5 m.
        description: t.planEmptyDensityDescription(matchsRetenus, TACTICAL_CELL_FLOOR, pasM),
      }
  }
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
 *
 * L'ADRESSE RENDUE EST ANCRÉE SUR L'ORIGINE DU MONDE (`floor(x / pas)`), jamais sur
 * `min_x` : c'est la convention du serveur (`analysis/tactical.Grille.Cellule`), donc
 * celle de `CelluleTactique.col/lig` et de la requête de détail de cellule. Une adresse
 * relative aux bornes ne coïncidait avec celle du serveur que sur une carte calée
 * exactement sur (0, 0) — et `trouveCellule` ne retrouvait alors plus rien.
 *
 * `pasM` EST LE PAS PUBLIÉ PAR LA LECTURE (`TacticalRaster.pas_m`), pas une constante :
 * depuis le pas adaptatif (lot 3.2), la même carte peut se lire à 0,5, 1 ou 2 m.
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
    col: Math.floor(worldX / pasM),
    row: Math.floor(worldY / pasM),
  }
}

/**
 * planCanvasView — la projection monde -> canvas du calque de chaleur : ce que
 * `drawTacticalHeatmap` attend (`TacticalLayerView`).
 *
 * POURQUOI `topLeftWorld` N'EST PAS (0, 0). Le peintre place une cellule à
 * `topLeftWorld.x + col × pas × scale`, et `col` est l'adresse SERVEUR — ancrée sur
 * l'origine du monde. Le canvas, lui, commence à `min_x`. L'origine du monde tombe donc
 * à `−min_x × scale` pixels du bord gauche, et c'est cette valeur-là qu'il faut passer :
 * avec (0, 0), tout le calque était décalé de `min_x` mètres, c'est-à-dire entièrement
 * hors du canvas sur une carte dont les coordonnées ne partent pas de zéro.
 *
 * L'ÉCHELLE EST UNIFORME (px par mètre, lue sur X) : le conteneur est mis à l'aspect-ratio
 * du monde, donc la même échelle vaut sur les deux axes.
 */
export function planCanvasView(
  bornes: BornesMonde,
  canvasWidth: number,
): { topLeftWorld: { x: number; y: number }; scale: number } | null {
  const largeurMonde = bornes.max_x - bornes.min_x
  if (!bornes.valide || !(largeurMonde > 0) || !(canvasWidth > 0)) return null
  const scale = canvasWidth / largeurMonde
  // `0 - v` plutôt que `-v` : sur des bornes calées à l'origine, `-0` est un pixel comme
  // les autres pour le canvas, mais il se compare mal (et se lit mal au débogage).
  return { topLeftWorld: { x: 0 - bornes.min_x * scale, y: 0 - bornes.min_y * scale }, scale }
}

/** trouveCellule — la cellule serveur à (col, row), ou `null` si jamais atteinte. */
export function trouveCellule(
  cellules: readonly CelluleTactique[],
  col: number,
  row: number,
): CelluleTactique | null {
  return cellules.find((c) => c.col === col && c.lig === row) ?? null
}

// TACTICAL_REPLAY_FRAME_INTERVAL_MS / instantToFrame ont vécu ici (lot M1, « voir dans le
// rejeu ») : une conversion instant -> frame MÉCANIQUE, sans correction du décalage
// d'horloge match/film pour quatre questions sur six. Retirées le 2026-09-08 (lot M1b,
// décision utilisateur ferme « corriger le décalage ») — mortes : `TacticalCellCard` ne
// pré-calcule plus de frame, il construit `?t=&clock=` et laisse la ROUTE du rejeu
// convertir une fois le document (et son calage) chargé
// (`lib/replay/replayLogic.resolveTacticalReplayInstant` + `msToFrames`).
