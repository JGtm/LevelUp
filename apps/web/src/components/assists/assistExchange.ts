/**
 * assistExchange.ts — calculs d'affichage des assistances échangées (purs, testés).
 *
 * PARTS : chaque sens se rapporte à un dénominateur de frags —
 *  - reçues (il t'a assisté) : part de TES frags → `received / my_frags` ;
 *  - données (tu l'as assisté) : part de SES frags → `given / partner_frags`.
 * Elles s'affichent en chiffres et en infobulle, PAS dans la longueur des barres.
 *
 * LONGUEUR DES BARRES (révisée le 2026-09-16 après revue du rendu) : le VOLUME
 * d'assistances, sur une échelle LOGARITHMIQUE commune à toute la page
 * (`volumeMax` = plus gros volume d'un sens parmi toutes les relations). Deux échecs de
 * la version « part rapportée au maximum de la carte » ont motivé ce choix :
 *  - un binôme à 3 assistances remplissait sa demi-barre (part maximale de SA carte) ;
 *  - un fidèle à 1 match mesuré (part élevée sur un petit échantillon) écrasait un fidèle
 *    à 250 assistances.
 * Le log garde l'ordre des volumes tout en laissant visibles les petits échanges :
 * 3 assistances ≈ 23 % d'une demi-barre quand le maximum est 422, 250 ≈ 91 %.
 */
import type { AssistTiers, RelationAssists } from '@/lib/api/types'

import type { AssistTier } from './assistsI18n'

export interface AssistSegment {
  tier: AssistTier
  count: number
  /** Largeur en % de la demi-barre. */
  widthPct: number
}

/** Part `count / frags` (0..1), null si aucun frag. */
export function assistShare(count: number, frags: number): number | null {
  if (!(frags > 0)) return null
  return count / frags
}

export function receivedShare(a: RelationAssists): number | null {
  return assistShare(a.received.total, a.my_frags)
}

export function givenShare(a: RelationAssists): number | null {
  return assistShare(a.given.total, a.partner_frags)
}

/** Plus gros volume d'un sens (reçues ou données) parmi les échanges ; 0 si rien. */
export function assistVolumeMax(list: ReadonlyArray<RelationAssists | null | undefined>): number {
  let max = 0
  for (const a of list) {
    if (!a) continue
    max = Math.max(max, a.received.total, a.given.total)
  }
  return max
}

/** Longueur (0..100 % de la demi-barre) d'un volume, échelle log bornée par `volumeMax`. */
export function assistVolumeLengthPct(total: number, volumeMax: number): number {
  if (!(total > 0) || !(volumeMax > 0)) return 0
  return Math.min(100, (Math.log1p(total) / Math.log1p(volumeMax)) * 100)
}

const TIERS: AssistTier[] = ['low', 'mid', 'high']

/**
 * Découpe une longueur de barre (0..100 %) en segments par tranche, du centre vers
 * l'extérieur (coup de pouce → frag préparé) : chaque tranche prend sa proportion du
 * total. Une assistance sans part mesurée n'entre dans aucune tranche : elle n'est pas
 * dessinée (la somme des segments peut être inférieure à la longueur).
 */
function splitByTier(tiers: AssistTiers, lengthPct: number): AssistSegment[] {
  if (lengthPct === 0 || !(tiers.total > 0)) return []
  return TIERS.map((tier) => ({
    tier,
    count: tiers[tier],
    widthPct: (lengthPct * tiers[tier]) / tiers.total,
  })).filter((s) => s.count > 0)
}

/** Segments d'une demi-barre du papillon : la longueur totale vient du VOLUME (échelle log). */
export function assistSegments(tiers: AssistTiers, volumeMax: number): AssistSegment[] {
  return splitByTier(tiers, assistVolumeLengthPct(tiers.total, volumeMax))
}

/**
 * Segments d'une barre de PART (tuile de match) : la longueur totale est la part
 * `total / frags` des frags assistés, bornée à 100 % ; chaque tranche en prend sa
 * proportion — un segment mesure donc directement `count / frags`. Aucun frag → rien.
 */
export function assistShareSegments(tiers: AssistTiers, frags: number): AssistSegment[] {
  const share = assistShare(tiers.total, frags)
  if (share === null) return []
  return splitByTier(tiers, Math.min(100, share * 100))
}

/** Clé de tri des tableaux : assistances échangées (données + reçues), undefined si non mesuré. */
export function assistSortValue(a: RelationAssists | null | undefined): number | undefined {
  return a ? a.given.total + a.received.total : undefined
}

// ─── Échelle LINÉAIRE commune aux deux sens (encart cible Explorer) ──────────
//
// L'Explorer n'affiche qu'UNE paire : la borne y est le plus gros des DEUX totaux de
// cette paire, sur une échelle linéaire — la longueur est alors la donnée, pas son
// logarithme (63 contre 48 se voit). `assist_volume_max` et `assistVolumeLengthPct`
// restent l'échelle du hub Relations, où plusieurs paires se comparent entre elles.

/** Borne linéaire commune aux deux sens : le plus gros des deux totaux ; 0 si vide. */
export function assistPairBound(a: RelationAssists): number {
  return Math.max(a.received.total, a.given.total)
}

/** Segments d'une barre à l'échelle linéaire commune (0..100 % de la piste). */
export function assistLinearSegments(tiers: AssistTiers, bound: number): AssistSegment[] {
  if (!(bound > 0)) return []
  return splitByTier(tiers, Math.min(100, (tiers.total / bound) * 100))
}

/**
 * Position (0..100 % de la piste) du trait de PARITÉ : la moitié du total des deux
 * sens — l'endroit où les deux barres se rejoindraient si l'échange était équilibré.
 * null quand la borne est nulle ou que le trait sortirait de la piste.
 */
export function assistParityPct(a: RelationAssists, bound: number): number | null {
  if (!(bound > 0)) return null
  const pct = ((a.received.total + a.given.total) / 2 / bound) * 100
  return pct > 0 && pct <= 100 ? pct : null
}
