/**
 * focusFilter.ts — CE QUE « SEUL » ET « MASQUER » FONT À CHAQUE CALQUE QUI N'EST PAS UN PION.
 *
 * LA RÈGLE (utilisateur, 2026-11-15) : tout calque affiché suit Seul et Masquer pour ce qui appartient
 * à un joueur. Seul n'affiche que ce qui est à lui ; Masquer retire ce qui est à lui. Ce qui n'appartient
 * à personne (véhicule vide, drapeau à sa base, zone, socle d'arme) reste affiché : c'est le décor du
 * match. Un élément dont l'attribution ne se lit pas reste affiché lui aussi — rien n'est deviné.
 *
 * UNE SEULE RÈGLE, ÉCRITE UNE FOIS : `ownershipShown` (playerPanelLogic.ts). Ce module ne dit que QUELLES
 * sont les parties de chaque élément, champ du document par champ du document ; chaque calque reçoit un
 * prédicat par genre d'élément et n'embarque aucune règle. Les calques de pions (pistes, tirs, traces des
 * vies, projectiles) gardent `shownAtSlot` : ils portent déjà un slot et une image dans la vie de leur auteur.
 */
import { killFxShown, type PlayerFocus } from './playerPanelLogic'
import type { KillFxEntry } from './killFx'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'

/** Un prédicat par genre d'élément : vrai = l'élément se dessine. */
export interface FocusFilter {
  /** Un effet de mort : tueur et victime. */
  killFx: (e: KillFxEntry) => boolean
}

/**
 * buildFocusFilter rend les prédicats du panneau de joueur, ou `null` quand rien n'est filtré (ni Seul
 * ni Masquer) : les calques gardent alors exactement leur dessin d'avant.
 */
export function buildFocusFilter(
  _doc: ReplayDocumentReady,
  focus: PlayerFocus | null,
): FocusFilter | null {
  const killFx = killFxShown(focus)
  return killFx ? { killFx } : null
}
