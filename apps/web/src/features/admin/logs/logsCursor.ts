/**
 * logsCursor — logique pure du curseur arrière « charger plus » du viewer de
 * logs (pas de React : testable seul). Le serveur renvoie next_offset (offset
 * octet du début de la ligne la plus ancienne de la page) + has_more ; ce
 * curseur alimente le pageParam `before` de la page suivante, plus ancienne.
 */
import type { AdminLogTail } from '@/lib/api/types'

/**
 * Retourne l'offset à passer en `before` pour charger la tranche plus ancienne,
 * ou undefined s'il n'y a plus rien à charger (début de fichier atteint). Un
 * next_offset absent/0 est traité comme « plus de page » (garde-fou : un 0
 * repartirait de la fin et boucclerait).
 */
export function nextLogCursor(page: AdminLogTail): number | undefined {
  if (!page.has_more) return undefined
  const offset = page.next_offset ?? 0
  return offset > 0 ? offset : undefined
}
