/**
 * types.ts — Types TS miroir du DTO Go domain.SquadPageV2Response.
 *
 * Conformement au PLAN_SQUAD_GO_PORTAGE Phase 1 chunk S11 : le backend Go
 * expose un payload riche (17 charts + 3 tableaux). Ces types sont la
 * frontiere TS pour consommer ce payload depuis React.
 *
 * Les types charts standard reutilisent les types ChartSeries/ChartPoint
 * deja exposes par les wrappers ECharts (apps/web/src/components/charts/).
 */

import type { ChartSeries } from '@/components/charts/ChartCard'
import type { CombatProfileBlock } from '@/lib/api/types'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import type { RadarSeriesPayload } from '@/components/charts/RadarChart'

/** Outcome d'un match (mirror canonical.Outcome Go). */
export type Outcome = 'win' | 'loss' | 'tie' | 'dnf'

/** Cle i18n pour les libelles. */
export interface AssetReference {
  kind?: string
  id: string
  default_label?: string
  labels?: Record<string, string>
  icon_url?: string
}

/** CapabilityGap : signale une section degradée côté backend. */
export interface CapabilityGap {
  capability_key: string
  reason_code: string
  severity: 'info' | 'warning' | 'blocking'
  message: string
  retryable?: boolean
}

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
  /** Moyenne arithmetique field-by-field des kpis_by_xuid (reference trends ▲/▼). */
  team_avg_kpis?: KPIStats
}

/** ImpactRolesMatrix (S5). */
export interface ImpactRoleCell {
  role_key: string
  label_key: string
  color_token: string
  inverted?: boolean
}

export interface ImpactRolesMatchRow {
  match_id: string
  started_at_utc: string
  main_outcome: Outcome
  roles_by_player: Record<string, ImpactRoleCell[]>
}

export interface ImpactRolesMatrix {
  match_rows: ImpactRolesMatchRow[]
  squad_gamertags: string[]
}

export interface ImpactRankingEntry {
  gamertag: string
  count: number
}

export interface ImpactRanking {
  role_key: string
  label_key: string
  inverted?: boolean
  entries: ImpactRankingEntry[]
}

/** Charts groupes par onglet. Champ nilable = section omise par le backend. */
export interface SquadCharts {
  // Synergies
  map_breakdown_lollipop?: ChartSeries<ChartPointStacked>
  bullet_winrate?: ChartSeries<ChartPointStacked>
  perf_vs_historical?: ChartSeries<ChartPoint2D>
  heatmap_player_map?: ChartSeries<ChartPointHeatmap>
  timeline_multi_player?: ChartSeries<ChartPoint2D>[]
  form_score?: ChartSeries<ChartPoint2D>
  // Cadence + Intensite
  cadence?: ChartSeries<ChartPointStacked>
  intensity_heatmap?: ChartSeries<ChartPointHeatmap>
  // Impact
  impact_matrix?: ImpactRolesMatrix
  impact_ranking?: ImpactRanking[]
  // Contributions
  per_minute_stats?: ChartSeries<ChartPointStacked>
  frags_deaths_combined?: ChartSeries<ChartPointStacked>
  hs_pk_stacked?: ChartSeries<ChartPointStacked>
  killing_spree_max?: ChartSeries<ChartPoint2D>[]
  assists_timeseries?: ChartSeries<ChartPoint2D>[]
  kda_timeseries?: ChartSeries<ChartPoint2D>[]
  accuracy_timeseries?: ChartSeries<ChartPoint2D>[]
  avg_life_timeseries?: ChartSeries<ChartPoint2D>[]
  performance_timeseries?: ChartSeries<ChartPoint2D>[]
  // Radar
  radar?: RadarSeriesPayload[]
}

/** Tableaux + galerie (S9). */
export interface WeaponsTableRow {
  weapon_id: number
  label?: string
  kills_by_xuid: Record<string, number>
  total: number
  is_grenade_melee?: boolean
}

export interface MedalEntry {
  medal_id: number
  count: number
  label?: string
}

export interface MedalsGalleryEntry {
  match_id: string
  medals_by_xuid: Record<string, MedalEntry[]>
}

export interface SquadTables {
  weapons?: WeaponsTableRow[]
  medals?: MedalsGalleryEntry[]
}

/** Match partage (intersection du squad). */
export interface SquadSharedMatch {
  match_id: string
  started_at_utc: string
  map?: AssetReference
  mode?: AssetReference
  playlist?: AssetReference
  outcome: Outcome
  // Le backend expose `players: Record<string, PlayerMatchRow>` mais le front
  // n'a pas besoin du PlayerMatchRow complet ici — on declare opaque.
  players: Record<string, unknown>
}

/** Reponse complete du endpoint /pages/squad/v2. */
export interface SquadPageV2Response {
  main_player: string
  teammates: string[]
  period: string
  shared_matches_count: number
  shared_matches: SquadSharedMatch[]
  header?: SquadHeader
  charts?: SquadCharts
  tables?: SquadTables
  capabilities?: CapabilityGap[]
}

/** Periode de filtrage (mirror temporal.Period Go). */
export type SquadPeriod = 'all' | '2y' | '1y' | '1m' | '1w'
