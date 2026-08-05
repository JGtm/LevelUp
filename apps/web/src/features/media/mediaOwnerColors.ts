/**
 * Couleurs de pill « propriétaire » des médias — extraites de MediaViewer.tsx
 * pour que le module de composant n'exporte que des composants
 * (react-refresh/only-export-components).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MediaItemRow } from '@/lib/api/types'
import { SQUAD_TEAMMATE_COLOR_TOKENS } from '@/features/squad/colors'

// Source unique (pas de copie locale) : les pills « propriétaire » reprennent
// l'ordre d'assignation des coéquipiers de la page Escouade.
const OWNER_TAG_TOKENS = SQUAD_TEAMMATE_COLOR_TOKENS

// Fallback hash-based pour les callers sans playerColorMap.
export function ownerTagColor(gamertag: string): string {
  let hash = 0
  for (const ch of gamertag) hash = (hash * 31 + ch.charCodeAt(0)) & 0xffff
  return tokenCssVar(OWNER_TAG_TOKENS[hash % OWNER_TAG_TOKENS.length])
}

/**
 * Construit un mapping gamertag→cssVar en ordre d'apparition dans les items,
 * cohérent avec l'ordre d'assignation de la page Escouade.
 * `currentRef` est exclu du mapping (ses médias n'affichent pas de pill).
 */
export function buildOwnerColorMap(
  items: MediaItemRow[],
  currentRef: string | null | undefined,
): Record<string, string> {
  const map: Record<string, string> = {}
  let slot = 0
  for (const item of items) {
    const owner = item.owner_gamertag
    if (!owner) continue
    if (currentRef && owner.toLowerCase() === currentRef.toLowerCase()) continue
    if (!map[owner]) {
      map[owner] = tokenCssVar(OWNER_TAG_TOKENS[slot % OWNER_TAG_TOKENS.length])
      slot++
    }
  }
  return map
}
