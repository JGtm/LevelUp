import type { components } from '@/lib/api/generated'
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
  /** Titre dans la langue de la requête (le serveur choisit, repli anglais). */
  title: string
  icon?: string
  /** Description lisible de la condition du jalon (A9), dans la langue de la requête. Vide si le
   *  jalon n'a pas de condition explicite → aucune ligne affichée. */
  condition?: string
  earned: boolean
  earned_at?: string | null
}

export type MilestonesResponse = components['schemas']['MilestonesResponse']

// ── Profile (sections A1/A2/B/C) ─────────────────────────────────────────────

export type ParticipationAxisValue = components['schemas']['ParticipationAxisValue']

export type RadarAxisInsight = components['schemas']['RadarAxisInsight']

export type StyleSignature = components['schemas']['StyleSignature']

export type EngagementSnapshot = Omit<components['schemas']['EngagementSnapshot'], 'tier'> & {
  tier: 'low' | 'regular' | 'high' | 'intense'
}

export type SkillRatingSnapshot = components['schemas']['SkillRatingSnapshot']

export type LUSRComponentBreakdown = components['schemas']['LUSRComponentBreakdown']

export type ProgressionLeverage = components['schemas']['ProgressionLeverage']

export type SuggestedChallenge = Omit<components['schemas']['SuggestedChallenge'], 'target_tier'> & {
  target_tier: 'normal' | 'heroic' | 'legendary'
}

export type PlayerProfile = components['schemas']['PlayerProfile']

// ── Calendrier d'activité (DEC-5/D3) ─────────────────────────────────────────

/** Un jour joué (>= 1 match). Les jours vides sont omis par le backend. */
export type ActivityDay = components['schemas']['ActivityDay']

/** Réponse GET /activity-calendar — miroir de profile.ActivityCalendar. */
export type ActivityCalendar = components['schemas']['ActivityCalendar']

// ── Patterns contextuels / comportementaux (phases 1-3) ──────────────────────

export type ContextType = 'by_mode' | 'by_map' | 'by_squad'
export type PatternSignal = 'strength' | 'weakness' | 'neutral'
export type BehaviorType = 'tilt' | 'session_fatigue' | 'engagement_drop' | 'accuracy_plateau' | 'perf_ceiling'
export type PatternSeverity = 'low' | 'medium' | 'high'

export type ContextualPattern = Omit<components['schemas']['ContextualPattern'], 'type' | 'signal'> & {
  type: ContextType
  signal: PatternSignal
}

export type BehavioralPattern = Omit<components['schemas']['BehavioralPattern'], 'type' | 'severity'> & {
  type: BehaviorType
  severity: PatternSeverity
}

export interface PatternLever {
  rank: number
  axis: string
  /** Clé brute du contexte visé (F3) : libellé de mode (by_mode), « with_friends »/
   *  « solo » (by_squad), ou GUID de carte (by_map — NON affiché, cf. context_label). */
  context_key?: string
  /** Nom d'asset résolu du contexte visé (by_map : nom de carte, title-agnostic).
   *  Le front l'affiche pour by_map ; jamais le GUID de context_key. */
  context_label?: string
  current_val: number
  target_val: number
  horizon: number
  impact: number
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
