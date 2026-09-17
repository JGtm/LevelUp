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
 * Segments d'une demi-barre, du centre vers l'extérieur (coup de pouce → frag préparé).
 * La longueur totale vient du volume ; chaque tranche en prend sa proportion. Une
 * assistance sans part mesurée n'entre dans aucune tranche : elle n'est pas dessinée.
 */
export function assistSegments(tiers: AssistTiers, volumeMax: number): AssistSegment[] {
  const length = assistVolumeLengthPct(tiers.total, volumeMax)
  if (length === 0) return []
  return TIERS.map((tier) => ({
    tier,
    count: tiers[tier],
    widthPct: (length * tiers[tier]) / tiers.total,
  })).filter((s) => s.count > 0)
}

/** Clé de tri des tableaux : assistances échangées (données + reçues), undefined si non mesuré. */
export function assistSortValue(a: RelationAssists | null | undefined): number | undefined {
  return a ? a.given.total + a.received.total : undefined
}
