/**
 * weaponRange_logic — LES DÉCISIONS DE LECTURE DE LA SECTION « PORTÉE PAR ARME », hors JSX.
 *
 * Tout ce que la section doit DÉCIDER (a-t-elle de quoi s'afficher, que dit la ligne « sous
 * le seuil ») vit ici, pur et testable. Le composant ne fait que poser des nœuds ; les
 * options ECharts vivent dans `@/components/charts/weaponRangeChart` (partagé avec le
 * Face-à-face et l'Explorer) et dans `_weaponElevationChart.ts`.
 *
 * `resolveWeaponLabel` est parti avec le module de rendu le 2026-09-17 : il ne servait qu'à
 * lui, et le laisser ici aurait fait remonter `components/` vers `features/`.
 */
import type { SynthesisWeaponRange } from '@/lib/api/types'

/**
 * WEAPON_RANGE_MIN_MEASURED — MIROIR de `analysis.WeaponRangeMinMeasured` (Go), sous le
 * seuil duquel un couple (arme, côté) n'est pas publié (décision D9 du plan).
 *
 * POURQUOI UN MIROIR ET PAS UNE VALEUR DU CONTRAT : le seuil n'est pas servi par l'API — le
 * bloc publie les armes RETENUES et la liste NOMMÉE de celles écartées, jamais le seuil
 * lui-même. Le front n'en a besoin que pour ÉCRIRE la phrase « sous le seuil de N mesures » ;
 * il ne s'en sert jamais pour filtrer (ce serait un second seuil, qui divergerait). Si le
 * seuil Go bouge, cette constante devient un libellé faux, jamais un rendu faux : la
 * découverte est consignée au plan (le contrat pourrait le porter).
 */
export const WEAPON_RANGE_MIN_MEASURED = 8

/**
 * hasWeaponRangeRows — la section a-t-elle une ligne à dessiner ?
 *
 * Le service omet DÉJÀ le bloc quand rien n'est mesuré ; cette garde couvre le cas résiduel
 * d'un bloc servi sans arme publiable (toutes sous le seuil). Un graphe à zéro ligne est une
 * carte vide, pas une information.
 */
export function hasWeaponRangeRows(range: SynthesisWeaponRange | null | undefined): boolean {
  return (range?.weapons ?? []).length > 0
}
