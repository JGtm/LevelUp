/**
 * squadRoster — appariement d'une composition courante à une escouade enregistrée.
 *
 * Identité d'escouade = roster EXACT par xuid des membres HORS joueur COURANT
 * (cf. backend Phase C). Partagé entre SquadFocusStrip (affiche « Gérer » si la
 * compo est déjà une escouade) et useSquadPresets (empêche d'enregistrer un
 * doublon). Source unique → évite la divergence des deux logiques de matching.
 *
 * Clé XUID absolue (fix player-agnostic) : on exclut le viewer par SON xuid, pas
 * par matching de chaîne contre le slug d'URL. Sinon un viewer dont le gamertag
 * diffère du slug n'est jamais retiré du roster (taille qui ne matche pas), et
 * une escouade créée par un AUTRE membre (créateur ≠ viewer) n'est pas reconnue.
 */
import type { SquadMember, SquadWithMembers } from '@/lib/prestige'

/**
 * squadTeammates — membres du roster HORS joueur COURANT, exclusion par XUID
 * absolu. Source UNIQUE de « qui sont mes coéquipiers dans cette escouade »,
 * partagée entre findSquadByRoster (comparaison de rosters) et useSquadPresets
 * (gamertags à charger) pour éviter deux logiques d'exclusion divergentes.
 */
export function squadTeammates(members: SquadMember[], currentPlayerXuid: string): SquadMember[] {
  return members.filter((m) => m.xuid && m.xuid !== currentPlayerXuid)
}

/**
 * Retourne l'escouade dont le roster (hors `currentPlayerXuid`) correspond
 * EXACTEMENT à `selectionXuids`, ou null. Sélection vide → null.
 */
export function findSquadByRoster(
  squads: SquadWithMembers[] | undefined,
  selectionXuids: string[],
  currentPlayerXuid: string,
): SquadWithMembers | null {
  if (!squads || selectionXuids.length === 0) return null
  const sel = new Set(selectionXuids)
  return (
    squads.find((sm) => {
      const others = squadTeammates(sm.members, currentPlayerXuid).map((m) => m.xuid)
      return others.length === sel.size && others.every((x) => sel.has(x))
    }) ?? null
  )
}
