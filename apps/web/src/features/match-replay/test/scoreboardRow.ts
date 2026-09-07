/**
 * scoreboardRow.ts — UNE ligne de scoreboard minimale pour les tests du rejeu.
 *
 * Seuls le camp et l'identité comptent pour les fiches ; tout le reste est à null ou à une
 * valeur neutre. Née le 2026-09-06 (lot fiches compactes) quand la fixation DOM 4v4 et la
 * mesure de perf en réclamaient chacune une : `ReplayTeams.test.tsx` garde sa copie `sbRow`
 * (règle « ≤ 2 copies », CLAUDE.md n° 6 — ce test-là ne se retouche pas, cf. I3 du plan), ce
 * helper est la seconde et dernière.
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
    team_side: side,
    is_me: false,
    rank: 1,
    score: 0,
    kills: 1,
    deaths: 1,
    assists: 0,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'n/a',
    ...over,
  }
}
