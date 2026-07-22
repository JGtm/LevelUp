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
  /** Description lisible localisée de la condition du jalon (A9). Vide si le
   *  jalon n'a pas de condition explicite → aucune ligne affichée. */
  condition_fr?: string
  condition_en?: string
  earned: boolean
  earned_at?: string | null
}

export interface MilestonesResponse {
  items: MilestoneItem[]
}

// ── Profile (sections A1/A2/B/C) ─────────────────────────────────────────────

export interface ParticipationAxisValue {
  axis: string  // "combat" | "survival" | "support" | "score" | "objective" | "impact"
  value: number // 0..100
  raw?: number
}

export interface RadarAxisInsight {
  axis: string
  value: number
  message?: string
}

export interface StyleSignature {
  first_kill_count: number
  first_death_count: number
  fkfd_ratio: number
  style_key?: string // "opportunistic_finisher" | "overextended" | "hyper_engaged" | "passive"
}

export interface EngagementSnapshot {
  score: number
  tier: 'low' | 'regular' | 'high' | 'intense'
  matches_per_day_avg: number
  max_gap_days: number
  regularity_coach?: string
}

export interface SkillRatingSnapshot {
  tier_name: string
  tier_name_fr: string
  sub_tier: number
  label: string
  mu: number
  sigma: number
  next_tier_label?: string
  next_tier_mu?: number
  gap_to_next?: number
  progress_ratio?: number
}

export interface LUSRComponentBreakdown {
  name: string
  weight: number
  current_avg: number
  personal_top_20: number
  target_for_tier: number
  trend: number
}

export interface ProgressionLeverage {
  component: string
  leverage_value: number
  narrative_axes: string[]
  coaching_message: string
}

export interface SuggestedChallenge {
  template_id: string
  target_tier: 'normal' | 'heroic' | 'legendary'
  historical_streak: number
  is_arc_step: boolean
  arc_id?: string
  label_fr?: string
  label_en?: string
  description_fr?: string
  description_en?: string
}

export interface PlayerProfile {
  user_id: string
  title_slug: string
  updated_at: string
  has_enough_data: boolean
  matches_analyzed: number
  // A1 — narrative
  dominant_role?: string
  secondary_role?: string
  radar_axes?: ParticipationAxisValue[]
  strengths?: RadarAxisInsight[]
  improvement_areas?: RadarAxisInsight[]
  // A2 — style & discipline
  style_signature: StyleSignature
  engagement_snap: EngagementSnapshot
  // B — LUSR
  skill_rating: SkillRatingSnapshot
  lusr_components?: LUSRComponentBreakdown[]
  mu_trend: { metric: string; slope: number; window: number }
  // C — coaching
  leverages?: ProgressionLeverage[]
  suggested_challenges?: SuggestedChallenge[]
}

// ── Calendrier d'activité (DEC-5/D3) ─────────────────────────────────────────

/** Un jour joué (>= 1 match). Les jours vides sont omis par le backend. */
export interface ActivityDay {
  date: string // jour UTC (YYYY-MM-DD)
  count: number // nb de matchs distincts ce jour-là
}

/** Réponse GET /activity-calendar — miroir de profile.ActivityCalendar. */
export interface ActivityCalendar {
  since: string // jour UTC (YYYY-MM-DD) inclus
  until: string // jour UTC (YYYY-MM-DD) inclus
  days: ActivityDay[] // uniquement les jours avec count > 0, triés ASC
}

// ── Patterns contextuels / comportementaux (phases 1-3) ──────────────────────

export type ContextType = 'by_mode' | 'by_map' | 'by_squad'
export type PatternSignal = 'strength' | 'weakness' | 'neutral'
export type BehaviorType = 'tilt' | 'session_fatigue' | 'engagement_drop' | 'accuracy_plateau' | 'perf_ceiling'
export type PatternSeverity = 'low' | 'medium' | 'high'

export interface ContextualPattern {
  type: ContextType
  key: string
  /** Libellé lisible résolu côté backend (nom de carte pour by_map). Absent
   *  pour by_mode (clé déjà lisible) et by_squad (libellé i18n front). */
  label?: string
  match_count: number
  win_rate: number
  avg_kda: number
  avg_oc: number
  avg_dr: number
  avg_perf?: number
  avg_delta_csr?: number
  avg_delta_lusr?: number
  delta: number
  signal: PatternSignal
}

export interface BehavioralPattern {
  type: BehaviorType
  trigger: string
  evidence: string
  severity: PatternSeverity
  confirmed: boolean
}

export interface PatternLever {
  rank: number
  axis: string
  label: string
  current_val: number
  target_val: number
  horizon: number
  impact: number
  source_pattern: string
}

export interface PatternReport {
  window_size: number
  context_patterns: ContextualPattern[]
  behavior_patterns: BehavioralPattern[]
  levers: PatternLever[]
  computed_at: string
  /** Seuil de matchs par groupe sous lequel afficher « Échantillon faible »
   *  au lieu de Force/Faiblesse (DEC-8). Servi par le backend, jamais codé en
   *  dur côté front. */
  min_matches_for_signal: number
}
