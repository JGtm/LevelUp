/**
 * types.ts — Types TS du header Squad V2 (KPIs + score cards).
 *
 * Ne subsiste que le sous-ensemble encore consomme apres la purge post-HistoryTable :
 *   - KPIStats           : SquadLayout (KPIs solo/all-time) + SessionBriefing
 *   - SquadScoreCard     : SessionBriefing (verdict escouade)
 *   - PlayerScoreCard    : SessionBriefing (cartes joueurs)
 *   - SquadHeader        : lib/api/types.ts (champ header de la reponse squad)
 *
 * Les types charts/tableaux du payload V2 riche (SquadPageV2Response, SquadCharts,
 * SquadTables, useSquadV2, ...) ont ete retires : plus aucun consommateur cote web.
 */

import type { CombatProfileBlock } from '@/lib/api/types'

/** KPIs personnels (header). */
export interface KPIStats {
  matches_count: number
  total_play_seconds: number
  avg_match_seconds: number
  kills_per_game: number
  kills_per_minute: number
  deaths_per_game: number
  deaths_per_minute: number
  assists_per_game: number
  assists_per_minute: number
  avg_accuracy: number
  avg_life_seconds: number
  avg_offensive_conversion?: number | null
  avg_defensive_resistance?: number | null
  // PLAN_COMBAT_PROFILE_WIRING Phase 2
  combat_profile?: CombatProfileBlock | null
  outcomes: { wins: number; losses: number; ties: number; dnf: number }
}

export interface SquadScoreCard {
  score: number
  grade: string
  base_avg: number
  bonus_win_rate: number
  bonus_min_kd: number
  bonus_balance: number
  team_win_rate: number
  min_kd: number
  kills_std_dev: number
}

export interface PlayerScoreCard {
  /** xuid du joueur — utilise pour le drill-down click dans <SessionBriefing>. */
  xuid: string
  gamertag: string
  score: number
  label: 'excellent' | 'good' | 'average' | 'poor' | 'bad'
  comparison: 'above' | 'below' | 'near'
  kd_ratio: number
  win_rate: number
  accuracy: number
  kills: number
}

export interface SquadHeader {
  solo_kpis?: KPIStats
  all_time_kpis?: KPIStats
  squad_score?: SquadScoreCard
  player_cards?: PlayerScoreCard[]
  /** KPIs par xuid sur le scope courant (drill-down SessionBriefing). */
  kpis_by_xuid?: Record<string, KPIStats>
  /** Moyenne arithmetique field-by-field des kpis_by_xuid (reference trends). */
  team_avg_kpis?: KPIStats
}
