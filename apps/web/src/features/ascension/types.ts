/**
 * Types TypeScript miroir de l'API backend Go pour la couche progression V2.
 *
 * Source : apps/go-api/internal/api/handlers/progression.go (DTOs).
 * Convention : snake_case côté API, identique côté types pour minimiser la
 * conversion (cf. notifications/types.ts).
 */

export type StreakType =
  | 'daily_play'
  | 'daily_perf'
  | 'weekly_play'
  | 'weekly_kda_threshold'

export type StreakStatus = 'active' | 'paused' | 'broken'

export interface Streak {
  id: string
  type: StreakType
  started_at: string // ISO timestamp UTC
  current_length: number
  best_length: number
  last_increment_at?: string | null
  threshold?: number | null
  shields_used: number
  shields_available: number
  status: StreakStatus
  broken_at?: string | null
  pp_multiplier: number // calculé serveur
}

export interface StreaksResponse {
  items: Streak[]
}

export type RecordPeriod = '30d' | '90d' | 'all_time'

export interface PersonalBest {
  metric: string
  period: RecordPeriod
  value: number
  achieved_at?: string | null
  achieved_match_id?: string
  previous_value?: number | null
  previous_achieved_at?: string | null
  updated_at: string
}

export interface RecordHistory {
  id: string
  metric: string
  period: RecordPeriod
  value: number
  achieved_at: string
}

export interface RecordsResponse {
  personal_bests: PersonalBest[]
  history: RecordHistory[]
}

export interface MilestoneItem {
  id: string
  metric: string
  threshold: number
  title_en: string
  title_fr: string
  icon?: string
  condition?: string
  earned: boolean
  earned_at?: string | null
}

export interface MilestonesResponse {
  items: MilestoneItem[]
}
