/**
 * relief/floors — LES ETAGES EN RANGS, l'accroche de la coupe et la part tirée du maillage.
 *
 * LIBELLES SANS UNITE (D-14) : une unité du jeu vaut environ 3 m, afficher « +2,2 m » serait faux.
 * Les étages sont des RANGS : le rang 0 est « Bas », le rang 1 « Étage 1 »… Cette bibliothèque rend
 * des RANGS (des indices) ; les libellés viennent des dictionnaires i18n de la feature (D.3).
 *
 * ACCROCHE (conception §8.4) : `accroche(v)` = l'étage détecté le plus proche si `|v − étage| ≤ 0,7`,
 * sinon `v` arrondi au quart d'unité. La même fonction sert au choix manuel d'une pastille et au suivi
 * du joueur ouvert.
 */
import { CUT_CACHE_STEP_U, FOLLOW_SNAP_REACH_U } from './constants'
import type { ReliefData } from './types'

/** Un étage détecté : son rang (0 = « Bas ») et sa hauteur de coupe. */
export interface FloorRank {
  rank: number
  z: number
}

/** floorRanks numérote les étages détectés du plus bas au plus haut. */
export function floorRanks(levels: readonly number[]): FloorRank[] {
  return [...levels].sort((a, b) => a - b).map((z, rank) => ({ rank, z }))
}

/** Le rang de l'étage le plus proche de `z`, s'il est à moins de `reach` ; `null` sinon ou sans étage. */
export function nearestRank(levels: readonly number[], z: number, reach: number = FOLLOW_SNAP_REACH_U): number | null {
  const ranks = floorRanks(levels)
  let best: FloorRank | null = null
  for (const r of ranks) {
    if (best === null || Math.abs(r.z - z) < Math.abs(best.z - z)) best = r
  }
  return best !== null && Math.abs(best.z - z) <= reach ? best.rank : null
}

/** La hauteur de coupe d'un rang ; `null` si le rang n'existe pas. */
export function rankHeight(levels: readonly number[], rank: number): number | null {
  return floorRanks(levels)[rank]?.z ?? null
}

/** quantizeCut arrondit une hauteur de coupe au quart d'unité : le pas du cache des coupes et de l'accroche. */
export function quantizeCut(z: number): number {
  // `+ 0` : un arrondi de −0,01 rendrait −0, que `Object.is` distingue de 0.
  return Math.round(z / CUT_CACHE_STEP_U) * CUT_CACHE_STEP_U + 0
}

/** snapCut est l'accroche : l'étage le plus proche à moins de 0,7 u, sinon `v` au quart d'unité. */
export function snapCut(levels: readonly number[], v: number): number {
  const rank = nearestRank(levels, v)
  return rank === null ? quantizeCut(v) : (rankHeight(levels, rank) as number)
}

/** La part de la zone dont la hauteur vient du maillage de navigation, pour l'infobulle du fond Relief (D-16). */
export interface MeshShare {
  /** 0..1. */
  share: number
  /** Pourcentage entier 0..100, tel que l'infobulle l'écrit. */
  percent: number
}

/**
 * meshShareOf rend la part tirée du maillage : `floors.meshShare` du JSON. `null` quand le JSON ne la
 * porte pas, ou pour une carte sans maillage.
 */
export function meshShareOf(data: ReliefData): MeshShare | null {
  if (data.degradations.includes('sans-maillage')) return null
  const fromJson = data.floors.meshShare
  if (fromJson === null) return null
  return { share: fromJson, percent: Math.round(fromJson * 100) }
}
