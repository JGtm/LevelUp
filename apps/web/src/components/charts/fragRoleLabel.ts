/**
 * fragRoleLabel — source UNIQUE du libellé d'un arc de niveau 2 du sunburst
 * « Répartition des frags », partagée par le sunburst (fragSunburstModel) et le
 * « Détails des frags » (fragDetailBreakdown).
 *
 * Deux natures de rôle coexistent au niveau 2 (V73-3.2) :
 *
 *  - les rôles CANONIQUES title-agnostic (precision, automatic, assassination,
 *    grenade_frag…) : clés stables traduites ici via le manifeste `frags.toml` ;
 *  - les OBJETS des classes véhicule/tourelle/équipement/environnement
 *    (h5_vehicle_warthog, h5_turret_gauss, hinf_coil_plasma…) : le rôle est un
 *    `weapon_key` propre au TITRE, et son libellé est SERVI par l'API dans
 *    `FragRoleEntry.label` (FR) / `.label_en` (EN) — sa source est
 *    config/titles/{slug}/mappings/weapon_names.toml. Recopier ces noms dans le
 *    manifeste web (partagé par tous les titres) dupliquerait ce TOML et le ferait
 *    enfler à chaque titre ajouté ; le libellé servi fait donc foi quand il existe.
 *
 * Choix de LOCALE (D2, 2026-08-29) : `label` reste FR-first côté API (compat historique),
 * `label_en` EN-first — ce module choisit lequel afficher SELON LA LOCALE courante, avec
 * repli croisé si le libellé préféré est vide (mieux vaut un nom dans l'autre langue
 * qu'aucun nom).
 *
 * Repli final (D3, 2026-08-29) : si NI label NI label_en n'est servi (objet jamais seedé
 * dans weapon_names.toml) ET que la clé canonique n'existe pas dans le manifeste
 * (`roleLabel` renvoie alors la clé BRUTE — contrat documenté de `formatMessage`,
 * lib/i18n/format.ts), on retombe sur un libellé générique (`frags.role.generic_object`).
 * EXIGENCE ABSOLUE : aucun chemin ne doit jamais afficher une clé `frags.role.*` brute.
 *
 * Garde-rail : fragRoleLabel.test.ts (les deux natures, le choix de locale, le repli
 * croisé, et le repli générique — jamais une clé brute).
 */
import type { FragRoleEntry } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

/** Clé de repli (manifeste frags.toml) quand aucun libellé — servi ou canonique — n'existe. */
const GENERIC_ROLE_KEY = 'generic_object'

/**
 * Reconstruit la clé de manifeste attendue pour un rôle CANONIQUE (frags.role.<role>).
 * Sert à détecter que `roleLabel` n'a trouvé AUCUNE traduction : formatMessage renvoie la
 * clé elle-même quand elle est absente du manifeste (contrat documenté, lib/i18n/format.ts)
 * — signal fiable pour les rôles qui sont en réalité un weapon_key d'objet, jamais une clé
 * canonique.
 */
function canonicalRoleKey(role: string): string {
  return `frags.role.${role}`
}

/**
 * Libellé d'affichage d'un rôle, résolu dans cet ordre :
 *   1. le libellé SERVI par l'API dans la locale courante (label_en en EN, label en FR) ;
 *   2. à défaut, l'AUTRE libellé servi (repli croisé) ;
 *   3. à défaut, la traduction de la clé canonique (rôles precision/automatic/…) ;
 *   4. à défaut (rien n'est servi, aucune clé canonique), un libellé générique — jamais
 *      la clé i18n brute.
 * @param role entrée de niveau 2 de la FragDistribution.
 * @param locale locale d'affichage courante.
 * @param roleLabel résolveur i18n des clés canoniques (injecté — helper pur).
 */
export function fragRoleDisplayLabel(
  role: FragRoleEntry,
  locale: Locale,
  roleLabel: (r: string) => string,
): string {
  const preferred = (locale === 'en' ? role.label_en : role.label)?.trim()
  const crossFallback = (locale === 'en' ? role.label : role.label_en)?.trim()
  const served = preferred || crossFallback
  if (served) return served
  const translated = roleLabel(role.role)
  if (translated && translated !== canonicalRoleKey(role.role)) return translated
  return roleLabel(GENERIC_ROLE_KEY)
}
