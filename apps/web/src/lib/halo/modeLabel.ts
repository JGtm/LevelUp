/**
 * Normalisation canonique du libellé de mode de jeu Halo — source UNIQUE côté front.
 *
 * Miroir de `analysis.NormalizeModeLabel` (backend Go). Avant centralisation, cette
 * logique était DUPLIQUÉE à l'identique dans `features/match-view/format.ts` et
 * `components/ui/match-card.tsx` ; la divergence a produit le bug
 * « Slayer on Forest sur Forêt » (un correctif appliqué à une copie, pas l'autre).
 * Tout consommateur (titre match-view, tuiles, etc.) DOIT importer d'ici.
 */

function escapeRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Normalise un libellé de mode brut :
 *  1. Extrait le sous-mode du format `Prefix:Mode` ("Arena:Slayer" → "Slayer")
 *     ou `Mode : Suffixe` espacé ("Assassin : Classé" → "Assassin").
 *  2. Strippe le suffixe map : d'abord `on/sur <mapLabel>` exact si `mapLabel` fourni,
 *     PUIS un filet générique `on/sur <reste>` TOUJOURS appliqué — indispensable quand
 *     le nom de carte collé au mode est dans une autre langue que `mapLabel`
 *     (mode EN "Slayer on Forest" + mapLabel FR "Forêt"), sinon le doublon réapparaît.
 *  3. Strippe les suffixes ` - Forge` / ` - Ranked`.
 *
 * Retourne `null` si le mode est vide ; sinon le libellé normalisé (jamais vide —
 * retombe sur le mode d'origine trimmé si la normalisation vide tout).
 */
export function normalizeModeLabel(
  modeLabel: string | null | undefined,
  mapLabel: string | null | undefined,
): string | null {
  if (!modeLabel) return null
  let normalized = modeLabel.trim()
  if (!normalized) return null

  const spacedSeparatorIndex = normalized.indexOf(' : ')
  if (spacedSeparatorIndex > 0) {
    normalized = normalized.slice(0, spacedSeparatorIndex).trim()
  } else {
    const separatorIndex = normalized.lastIndexOf(':')
    if (separatorIndex >= 0 && separatorIndex < normalized.length - 1) {
      normalized = normalized.slice(separatorIndex + 1).trim()
    }
  }

  if (mapLabel?.trim()) {
    const escapedMap = escapeRegExp(mapLabel.trim())
    normalized = normalized.replace(new RegExp(`\\s+(?:on|sur)\\s+${escapedMap}$`, 'i'), '')
  }
  // Filet générique TOUJOURS appliqué (miroir de l'étape 3 du backend).
  normalized = normalized.replace(/\s+(?:on|sur)\s+.+$/i, '')

  normalized = normalized.replace(/\s*-\s*(?:Forge|Ranked)\b/gi, '').trim()
  return normalized || modeLabel.trim()
}
