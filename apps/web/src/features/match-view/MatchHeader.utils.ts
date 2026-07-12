/**
 * MatchHeader — helpers de formattage et label de rang.
 *
 * Découpé depuis MatchHeader.tsx (audit #6 god-file split).
 * Inclut : formatDuration() + cascade nextTierLabel() (anglais canonique +
 * chiffres romains + fallback digit).
 */

export function formatDuration(secs: number): string {
  const m = Math.floor(secs / 60)
  const s = secs % 60
  return s > 0 ? `${m}m ${s}s` : `${m}m`
}

// Sentinelle « en placement » servie par le back (skill.TierLabelPlacement) : une
// valeur MACHINE de tier (rating_value=0, phase de placement), utilisée telle
// quelle comme tier ET comme libellé côté Go. On la LOCALISE à l'affichage
// (« En placement » / « In placement »), aligné sur les cards d'accueil — jamais
// afficher le littéral EN « Placement » brut. Cf. règle « pas de label FR/EN en
// dur côté Go » : la traduction transite ici, côté front.
export const PLACEMENT_TIER_SENTINEL = 'Placement'

// displayTierLabel remplace la sentinelle « Placement » par son libellé localisé
// (passé par l'appelant depuis son bundle i18n) ; tout autre tier est rendu tel
// quel. Retourne null pour une entrée nulle (rien à afficher).
export function displayTierLabel(
  rawLabel: string | null | undefined,
  placementLabel: string,
): string | null {
  if (rawLabel == null) return null
  return rawLabel === PLACEMENT_TIER_SENTINEL ? placementLabel : rawLabel
}

const TIER_ORDER_EN = [
  'Bronze 1', 'Bronze 2', 'Bronze 3', 'Bronze 4', 'Bronze 5', 'Bronze 6',
  'Silver 1', 'Silver 2', 'Silver 3', 'Silver 4', 'Silver 5', 'Silver 6',
  'Gold 1', 'Gold 2', 'Gold 3', 'Gold 4', 'Gold 5', 'Gold 6',
  'Platinum 1', 'Platinum 2', 'Platinum 3', 'Platinum 4', 'Platinum 5', 'Platinum 6',
  'Diamond 1', 'Diamond 2', 'Diamond 3', 'Diamond 4', 'Diamond 5', 'Diamond 6',
  'Onyx',
] as const

// Chiffres romains I–VI dans l'ordre décroissant de longueur pour éviter
// qu'un préfixe court matche avant le suffixe complet (ex: " V" avant " VI").
const ROMAN_NEXT: Record<string, string> = {
  'VI': '',   // tier boundary — pas de "sous-tier 7"
  'V': 'VI', 'IV': 'V', 'III': 'IV', 'II': 'III', 'I': 'II',
}

export function nextTierLabel(current: string | null | undefined): string | null {
  if (!current) return null

  // Lookup direct sur les labels anglais (ex: "Gold 3")
  const idx = TIER_ORDER_EN.indexOf(current as typeof TIER_ORDER_EN[number])
  if (idx !== -1) return idx < TIER_ORDER_EN.length - 1 ? TIER_ORDER_EN[idx + 1] : null

  // Fallback générique : "[nom] [chiffre romain I-VI]" (ex: "Or III" → "Or IV")
  // Vérifié du plus long au plus court pour éviter les faux-positifs.
  for (const rom of ['VI', 'V', 'IV', 'III', 'II', 'I'] as const) {
    if (current.endsWith(' ' + rom)) {
      const next = ROMAN_NEXT[rom]
      if (!next) return null  // sub-tier VI : tier boundary, nom inconnu
      return current.slice(0, -(rom.length + 1)) + ' ' + next
    }
  }

  // Fallback digit "[nom] [1-5]" (ex: "Gold 3" non anglais canonique)
  const m = current.match(/^(.+)\s+([1-5])$/)
  if (m) return `${m[1]} ${parseInt(m[2]) + 1}`

  return null
}
