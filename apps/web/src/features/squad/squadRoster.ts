/**
 * squadRoster — appariement d'une composition courante à une escouade enregistrée.
 *
 * Identité d'escouade = roster EXACT par xuid des membres HORS joueur principal
 * (cf. backend Phase C). Partagé entre SquadFocusStrip (affiche « Gérer » si la
 * compo est déjà une escouade) et useSquadPresets (empêche d'enregistrer un
 * doublon). Source unique → évite la divergence des deux logiques de matching.
 */
import type { SquadWithMembers } from '@/lib/prestige'

/**
 * Retourne l'escouade dont le roster (hors `playerSlug`) correspond EXACTEMENT à
 * `selectionXuids`, ou null. Sélection vide → null.
 */
export function findSquadByRoster(
  squads: SquadWithMembers[] | undefined,
  selectionXuids: string[],
  playerSlug: string,
): SquadWithMembers | null {
  if (!squads || selectionXuids.length === 0) return null
  const sel = new Set(selectionXuids)
  // Comparaison de slug insensible à la casse : le slug d'URL et le user_id
  // persisté (db_profiles) peuvent différer de casse → sinon le créateur n'est
  // pas exclu, la taille ne matche pas, et l'escouade enregistrée n'est jamais
  // retrouvée (encart bloqué sur « Enregistrer » + risque de doublon append-only).
  const slug = playerSlug.toLowerCase()
  return (
    squads.find((sm) => {
      const others = sm.members
        .filter((m) => (m.user_id ?? '').toLowerCase() !== slug)
        .map((m) => m.xuid)
      return others.length === sel.size && others.every((x) => sel.has(x))
    }) ?? null
  )
}
