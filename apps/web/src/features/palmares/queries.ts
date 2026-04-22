import { useQuery } from '@tanstack/react-query'

import { api } from '@/lib/api/client'
import type { RelationInsight, RelationsPageResponse, SeasonPassPageResponse } from '@/lib/api/types'
import { queryKeys } from '@/lib/query/keys'

interface CareerEncounterRow {
  xuid: string
  gamertag: string
  match_count: number
  as_teammate: number
  as_enemy: number
  avg_kda: number | null
}

interface CareerEncountersApiResponse {
  teammates: CareerEncounterRow[]
  enemies: CareerEncounterRow[]
  total: number
}

function compareNullableDesc(left: number | null | undefined, right: number | null | undefined) {
  return (right ?? Number.NEGATIVE_INFINITY) - (left ?? Number.NEGATIVE_INFINITY)
}

function buildRelationInsight(row: CareerEncounterRow): RelationInsight {
  return {
    xuid: row.xuid,
    gamertag: row.gamertag,
    total_matches: row.match_count,
    teammate_matches: row.as_teammate,
    teammate_wins: 0,
    teammate_win_rate: null,
    enemy_matches: row.as_enemy,
    enemy_wins: 0,
    enemy_win_rate: null,
    avg_kda_with: row.as_teammate > 0 ? row.avg_kda : null,
    avg_kda_against: row.as_enemy > 0 ? row.avg_kda : null,
    last_seen_at: null,
  }
}

function limitInsights(items: RelationInsight[]) {
  return items.slice(0, 10)
}

function mapCareerEncountersToRelations(response: CareerEncountersApiResponse): RelationsPageResponse {
  const byKey = new Map<string, RelationInsight>()
  for (const row of [...response.teammates, ...response.enemies]) {
    const key = row.xuid || row.gamertag
    byKey.set(key, buildRelationInsight(row))
  }

  const allInsights = [...byKey.values()]
  const allies = allInsights.filter((item) => item.teammate_matches > 0)
  const enemies = allInsights.filter((item) => item.enemy_matches > 0)
  const closedCircle = allInsights.filter(
    (item) => item.teammate_matches > 0 && item.enemy_matches > 0,
  )

  return {
    overview: {
      distinct_players: allInsights.length,
      frequent_allies: allies.length,
      repeat_rivals: enemies.length,
      closed_circle: closedCircle.length,
    },
    frequent_allies: limitInsights(
      [...allies].sort(
        (left, right) =>
          right.teammate_matches - left.teammate_matches ||
          right.total_matches - left.total_matches ||
          left.gamertag.localeCompare(right.gamertag),
      ),
    ),
    best_synergies: limitInsights(
      [...allies].sort(
        (left, right) =>
          compareNullableDesc(left.avg_kda_with, right.avg_kda_with) ||
          right.teammate_matches - left.teammate_matches ||
          left.gamertag.localeCompare(right.gamertag),
      ),
    ),
    nemeses: limitInsights(
      [...enemies].sort(
        (left, right) =>
          right.enemy_matches - left.enemy_matches ||
          right.total_matches - left.total_matches ||
          left.gamertag.localeCompare(right.gamertag),
      ),
    ),
    favorite_victims: limitInsights(
      [...enemies].sort(
        (left, right) =>
          compareNullableDesc(left.avg_kda_against, right.avg_kda_against) ||
          right.enemy_matches - left.enemy_matches ||
          left.gamertag.localeCompare(right.gamertag),
      ),
    ),
    closed_circle: limitInsights(
      [...closedCircle].sort(
        (left, right) =>
          right.total_matches - left.total_matches || left.gamertag.localeCompare(right.gamertag),
      ),
    ),
  }
}

export function useSeasonPassPage(playerSlug: string) {
  return useQuery<SeasonPassPageResponse>({
    queryKey: queryKeys.seasonPass(playerSlug),
    queryFn: () =>
      api.get<SeasonPassPageResponse>(`/players/${playerSlug}/pages/palmares/season-pass`),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}

export function useRelationsPage(playerSlug: string) {
  return useQuery<RelationsPageResponse>({
    queryKey: queryKeys.palmaresRelations(playerSlug),
    queryFn: () =>
      api
        .get<CareerEncountersApiResponse>(`/players/${playerSlug}/pages/career/encounters`)
        .then(mapCareerEncountersToRelations),
    enabled: !!playerSlug,
    staleTime: 5 * 60 * 1000,
  })
}
