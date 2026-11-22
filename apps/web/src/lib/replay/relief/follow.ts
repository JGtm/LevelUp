/**
 * relief/follow — SUIVRE L'ETAGE DU JOUEUR OUVERT : la coupe prend l'étage de sa piste (§8, D.1 choix 5).
 *
 * Fonction pure : `stepFollow(état, échantillon, contexte)` rend le nouvel état ; la hauteur de coupe
 * courante est `état.cut`. Aucune horloge, aucun React : l'appelant (D.6) donne l'instant lu.
 *
 * POURQUOI LE SOL PORTEUR ET PAS LA HAUTEUR DE LA PISTE (conception §8.2) : la piste porte la hauteur
 * des PIEDS ; un saut ordinaire monte de 0,82 à 0,89 u et dure une seconde, plus haut que la bande de
 * la coupe. Un seuil sur la hauteur seule serait franchi à chaque saut. Le sol porteur (le sol du
 * maillage le plus haut sous les pieds, jusqu'à 0,35 u au-dessus d'eux) ne bouge pas pendant un saut,
 * et devient le sol du bas dès que le joueur franchit le bord d'un étage en tombant.
 *
 * REGLE (§8.4), à chaque instant lu pour le joueur ouvert :
 *   - pas de piste vivante (mort, pas encore réapparu) -> la coupe ne bouge pas ;
 *   - première image d'une vie, ou saut de lecture (clic sur la barre) -> coupe = accroche(cible), sans délai ;
 *   - |cible − coupe| ≤ 0,5 -> rien, la minuterie retombe ;
 *   - sinon la minuterie part ; au bout de 300 ms d'écart ininterrompu, coupe = accroche(cible).
 * où `cible` = le sol porteur ; à défaut le dernier sol porteur connu de cette vie ; à défaut `z`
 * (zéro quand la piste n'en porte pas : `z` est omis côté Go quand il vaut 0).
 *
 * Sans maillage il n'y a pas de coupe : l'état ne bouge pas. Un choix MANUEL d'étage coupe le suivi :
 * c'est l'appelant qui cesse d'appeler `stepFollow` (état de l'interface, pas de cette logique).
 */
import {
  FOLLOW_DELAY_MS,
  FOLLOW_SEEK_GAP_MS,
  FOLLOW_THRESHOLD_U,
} from './constants'
import { bearingFloor } from './floorMesh'
import { snapCut } from './floors'
import type { FloorMesh } from './types'

/** L'état du suivi, immuable : `stepFollow` en rend un neuf. */
export interface FollowState {
  /** La hauteur de coupe courante ; `null` tant que le suivi n'a rien posé. */
  cut: number | null
  /** La vie en cours (numéro de piste) ; `null` quand le joueur n'est pas en vie. */
  lifeKey: string | number | null
  /** Le dernier sol porteur connu de cette vie. */
  lastFloor: number | null
  /** L'instant (ms, temps du rejeu) où l'écart a dépassé le seuil ; `null` si l'écart est sous le seuil. */
  gapSinceMs: number | null
  /** Le dernier instant lu, pour reconnaître un saut de lecture. */
  lastMs: number | null
}

/** L'état de départ : rien posé. */
export const FOLLOW_INITIAL: FollowState = {
  cut: null,
  lifeKey: null,
  lastFloor: null,
  gapSinceMs: null,
  lastMs: null,
}

/** Ce que le suivi lit à un instant : la piste du joueur ouvert. */
export interface FollowSample {
  /** Faux quand le joueur est mort ou pas encore réapparu : la coupe ne bouge pas. */
  alive: boolean
  /** Identifie la vie (numéro de piste) : un autre numéro est une première image d'une vie. */
  lifeKey: string | number
  x: number
  y: number
  /** Hauteur des pieds ; absente = zéro. */
  z?: number | null
  /** L'instant lu, en ms de temps du rejeu. */
  atMs: number
  /** Vrai quand l'utilisateur a sauté dans la lecture (clic sur la barre). */
  seek?: boolean
}

/** Ce que le suivi lit de la carte. */
export interface FollowContext {
  /** Le maillage ; `null` sans maillage (pas de coupe). */
  mesh: FloorMesh | null
  /** Les étages détectés. */
  levels: readonly number[]
}

/**
 * Le sol sous les pieds : le sol porteur, sinon le dernier de la vie, sinon `z`. `floor` est le sol
 * porteur retenu pour la suite de la vie (inchangé quand le point est hors maillage).
 */
function targetOf(
  state: FollowState,
  sample: FollowSample,
  mesh: FloorMesh,
  sameLife: boolean,
): { target: number; floor: number | null } {
  const z = sample.z ?? 0
  const bearing = bearingFloor(mesh, sample.x, sample.y, z)
  const floor = bearing ?? (sameLife ? state.lastFloor : null)
  return { target: floor ?? z, floor }
}

/** Un instant qui recule, ou qui avance de plus que `FOLLOW_SEEK_GAP_MS`, est un saut de lecture. */
function isSeek(state: FollowState, sample: FollowSample): boolean {
  if (sample.seek) return true
  if (state.lastMs === null) return false
  const dt = sample.atMs - state.lastMs
  return dt < 0 || dt > FOLLOW_SEEK_GAP_MS
}

/**
 * stepFollow avance le suivi d'un échantillon. Rend le NOUVEL état ; l'état d'entrée n'est pas modifié.
 * Comparer `suivant.cut` à `précédent.cut` dit si la coupe a changé.
 */
export function stepFollow(state: FollowState, sample: FollowSample, ctx: FollowContext): FollowState {
  if (!sample.alive) {
    return { ...state, lifeKey: null, lastFloor: null, gapSinceMs: null, lastMs: sample.atMs }
  }
  if (ctx.mesh === null) return { ...state, lastMs: sample.atMs }
  const sameLife = state.lifeKey === sample.lifeKey
  const { target, floor } = targetOf(state, sample, ctx.mesh, sameLife)
  const base: FollowState = { ...state, lifeKey: sample.lifeKey, lastFloor: floor, lastMs: sample.atMs }
  if (!sameLife || state.cut === null || isSeek(state, sample)) {
    return { ...base, cut: snapCut(ctx.levels, target), gapSinceMs: null }
  }
  if (Math.abs(target - state.cut) <= FOLLOW_THRESHOLD_U) return { ...base, gapSinceMs: null }
  const since = state.gapSinceMs ?? sample.atMs
  if (sample.atMs - since >= FOLLOW_DELAY_MS) {
    return { ...base, cut: snapCut(ctx.levels, target), gapSinceMs: null }
  }
  return { ...base, gapSinceMs: since }
}
