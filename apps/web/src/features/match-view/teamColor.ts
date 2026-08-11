/**
 * teamColor.ts — LA COULEUR D'IDENTITÉ D'UNE ÉQUIPE, et il n'y en a qu'une.
 *
 * Extraite de `MatchKillFeed.tsx` quand le rejeu 2D a eu besoin de la même cascade : deux
 * cascades divergentes, ce sont deux couleurs pour la même équipe sur deux pages du même
 * match. Le kill feed de la carte « Dominance » et celui du rejeu appellent désormais cette
 * fonction, et rien d'autre.
 *
 * L'ORDRE DE LA CASCADE reproduit celui de l'en-tête du scoreboard (`MatchScoreboard.tsx`) :
 * couleur fournie par le backend (`team_color`, Halo 5) d'abord, sinon la couleur officielle
 * par `team_id` (Halo Infinite : Eagle bleu, Cobra rouge...), sinon le token sémantique
 * allié/ennemi — surchargeable par les réglages d'accessibilité.
 *
 * Aucun hex n'est écrit ici : ils vivent dans `lib/halo/`, référentiel de couleurs de jeu au
 * même titre que `rarity.ts`.
 */
import { tokenCssVar } from '@/lib/accessibility'
import { parseTeamSideID, resolveTeamColorFromID } from '@/lib/halo/teamNames'
import type { MatchScoreboardRow } from '@/lib/api/types'

/** Résout la couleur d'identité d'une équipe, à partir de son id et de son camp. */
export type TeamColorResolver = (teamID: number | null, ally: boolean) => string

/**
 * teamColorResolver construit le résolveur pour un scoreboard donné.
 *
 * Le scoreboard est indexé UNE fois : un feed de 80 kills ne doit pas le reparcourir à
 * chaque ligne.
 */
export function teamColorResolver(
  scoreboard: MatchScoreboardRow[] | null | undefined,
): TeamColorResolver {
  const backendByTeamID = new Map<number, string>()
  for (const r of scoreboard ?? []) {
    const id = parseTeamSideID(r.team_side)
    if (id != null && r.team_color && !backendByTeamID.has(id)) {
      backendByTeamID.set(id, r.team_color)
    }
  }
  return (teamID, ally) => {
    if (teamID != null) {
      const backend = backendByTeamID.get(teamID)
      if (backend) return backend
      const official = resolveTeamColorFromID(teamID)
      if (official) return official
    }
    return tokenCssVar(ally ? 'team-ally' : 'team-enemy')
  }
}
