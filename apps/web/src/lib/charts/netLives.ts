/**
 * netLives — « Balance des dégâts » d'un match, exprimée en VIES du titre :
 * `(dégâts infligés − dégâts subis) / PV-pour-tuer`.
 *
 * Positif = le joueur a infligé plus de « vies » qu'il n'en a concédé (il porte
 * l'équipe) ; négatif = il en a coûté plus qu'il n'en a rapporté.
 *
 * `null` si l'un des deux termes manque / est non-fini (aucun fallback FDA
 * déguisé — décision produit), ou si le barème du titre est invalide.
 */
import { finiteOrNull } from './cumulativeSeries'

export function netLives(
  damageDealt: number | null | undefined,
  damageTaken: number | null | undefined,
  hpToKill: number,
): number | null {
  const dealt = finiteOrNull(damageDealt)
  const taken = finiteOrNull(damageTaken)
  if (dealt == null || taken == null) return null
  if (!Number.isFinite(hpToKill) || hpToKill <= 0) return null
  return (dealt - taken) / hpToKill
}
