import type { synthesisManifest } from '@/lib/i18n/generated/synthesis'

/**
 * Clé i18n du libellé « vol à la tire » (hijacks), sélectionnée PAR TITRE : le terme
 * officiel diffère (Halo 5 = « Vol à la tire », Halo Infinite = « Dépositaire » ; EN
 * commun = « Hijacks »). Le compteur « Véhicules détruits » reste commun.
 *
 * Précédent NavL2 (`currentTitleSlug === 'halo_5'`) : le gating par slug est INTERDIT
 * côté Go (adapter/capability) mais toléré côté front pour un simple choix de libellé.
 * Défaut (tout slug ≠ halo_5, y compris bootstrap non chargé) = Halo Infinite.
 *
 * Helper PUR (aucun hook / store) → testable hors rendu.
 */
export function hijacksLabelKey(
  titleSlug: string,
): keyof typeof synthesisManifest {
  return titleSlug === 'halo_5'
    ? 'synthesis.combat_profile.hijacks_h5'
    : 'synthesis.combat_profile.hijacks_infinite'
}
