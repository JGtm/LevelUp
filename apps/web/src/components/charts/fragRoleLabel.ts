/**
 * fragRoleLabel — source UNIQUE du libellé d'un arc de niveau 2 du sunburst
 * « Répartition des frags », partagée par le sunburst (fragSunburstModel) et le
 * « Détails des frags » (fragDetailBreakdown).
 *
 * Deux natures de rôle coexistent au niveau 2 (V73-3.2) :
 *
 *  - les rôles CANONIQUES title-agnostic (precision, automatic, assassination,
 *    grenade_frag…) : clés stables traduites ici via le manifeste `frags.toml` ;
 *  - les ENGINS des classes véhicule/tourelle (h5_vehicle_warthog, h5_turret_gauss…) :
 *    le rôle est un `weapon_key` propre au TITRE, et son libellé est SERVI par l'API
 *    dans `FragRoleEntry.label` — sa source est
 *    config/titles/{slug}/mappings/weapon_names.toml. Recopier ces noms dans le
 *    manifeste web (partagé par tous les titres) dupliquerait ce TOML et le ferait
 *    enfler à chaque titre ajouté ; le libellé servi fait donc foi quand il existe.
 *
 * Garde-rail : fragRoleLabel.test.ts (les deux natures + le repli).
 */
import type { FragRoleEntry } from '@/lib/api/types'

/**
 * Libellé d'affichage d'un rôle : le libellé servi par l'API s'il existe (engins),
 * sinon la traduction de la clé canonique.
 * @param role entrée de niveau 2 de la FragDistribution.
 * @param roleLabel résolveur i18n des clés canoniques (injecté — helper pur).
 */
export function fragRoleDisplayLabel(role: FragRoleEntry, roleLabel: (r: string) => string): string {
  const served = role.label?.trim()
  return served ? served : roleLabel(role.role)
}
