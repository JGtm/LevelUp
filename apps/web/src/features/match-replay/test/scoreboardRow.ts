/**
 * scoreboardRow.ts — UNE ligne de scoreboard minimale pour les tests du rejeu.
 *
 * Seuls le camp et l'identité comptent pour les fiches ; tout le reste est à null ou à une
 * valeur neutre.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'

export function scoreboardRow(
  xuid: string,
  gamertag: string,
  side: string | null,
  over: Partial<MatchScoreboardRow> = {},
): MatchScoreboardRow {
  return {
    xuid,
    gamertag,
    team_side: side ?? undefined,
    is_me: false,
    rank: 1,
    score: 0,
    kills: 1,
    deaths: 1,
    assists: 0,
    ...over,
  }
}
