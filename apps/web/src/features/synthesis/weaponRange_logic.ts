/**
 * weaponRange_logic — LES DÉCISIONS DE LECTURE DE LA SECTION « PORTÉE PAR ARME », hors JSX.
 *
 * Tout ce que la section doit DÉCIDER (a-t-elle de quoi s'afficher, comment se nomme une
 * arme, que dit la ligne « sous le seuil ») vit ici, pur et testable. Le composant ne fait
 * que poser des nœuds ; les options ECharts vivent dans `_weaponRangeChart.ts` et
 * `_weaponElevationChart.ts`.
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import type { WeaponBelowThreshold, SynthesisWeaponRange } from '@/lib/api/types'

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

/** Une entrée du contrat qui porte un libellé bilingue et une clé d'arme. */
interface LabelledWeapon {
  weapon_key: string
  label?: string
  label_en?: string
}

/**
 * resolveWeaponLabel — le nom affiché d'une arme, dans la locale courante.
 *
 * Repli sur `weapon_key` quand le registre n'a pas résolu le libellé : une clé technique
 * lisible vaut mieux qu'une ligne anonyme, et elle SE VOIT (le trou de registre se signale
 * de lui-même au lieu de se cacher derrière un tiret).
 */
export function resolveWeaponLabel(w: LabelledWeapon, locale: ManifestLocale): string {
  const label = locale === 'en' ? w.label_en : w.label
  return label && label.trim() !== '' ? label : w.weapon_key
}

/**
 * belowThresholdNames — « Hydra (6), Disrupteur (4) », ou `null` si la liste est vide.
 *
 * L'ORDRE VIENT DU BACKEND (effectif décroissant puis clé) et n'est pas rejoué : deux tris
 * du même fait divergeraient. `null` plutôt que chaîne vide : l'appelant doit pouvoir
 * OMETTRE le demi-énoncé, pas afficher « frags : ».
 */
export function belowThresholdNames(
  entries: readonly WeaponBelowThreshold[] | null | undefined,
  locale: ManifestLocale,
  fmtCount: (n: number) => string,
): string | null {
  const list = entries ?? []
  if (list.length === 0) return null
  return list.map((e) => `${resolveWeaponLabel(e, locale)} (${fmtCount(e.measured)})`).join(', ')
}

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
