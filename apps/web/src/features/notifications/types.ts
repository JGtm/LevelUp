/**
 * Types TypeScript miroir de l'API backend Go (internal/notifications/types.go).
 *
 * Convention :
 * - Le serveur renvoie snake_case (cf. tags JSON Go), on garde tel quel
 *   pour minimiser les conversions.
 * - title_key/body_key sont des clés i18n côté serveur, résolues par
 *   format.ts à partir des params + locale (cf. plan i18n V7).
 */

export type NotificationCategory =
  | 'app_release'
  | 'match_synced'
  | 'media_added'
  | 'media_liked'
  | 'objective_assigned'
  | 'objective_completed'
  | 'challenge_added'
  | 'challenge_completed'
  | 'season_pass_level'      // déprécié 2026-05-16 — remplacé par career_rank + battlepass_completed
  | 'sync_error'
  | 'personal_record'
  | 'threshold_crossed'
  | 'friend_added'           // §6 Squad/Sessions overhaul
  | 'friend_sync_completed'  // §6 Squad/Sessions overhaul
  | 'data_health_warning'    // émis par le scheduler data_health (audit DB périodique)
  | 'career_rank'            // 2026-05-16 — rang Halo lifetime
  | 'skill_tier'             // 2026-05-16 — CSR/LUSR unifié
  | 'battlepass_completed'   // 2026-05-16 — track BP atteint son rang max
  | 'citation_tier'          // 2026-05-16 — palier franchi sur une commendation
  | 'citation_mastery'       // 2026-05-16 — commendation à 100 %
  // 2026-05-18 — couche 3 du plan PROGRESSION_TRACKING (V2 Ascension), coach proactif.
  | 'record_near_miss'       // PB courant à moins de 5 % du record
  | 'milestone_unlocked'     // milestone débloqué
  | 'milestone_near_miss'    // valeur à moins de 10 % du seuil milestone
  | 'lusr_tier_approach'     // μ LUSR à moins de 10 pts du prochain sub-tier
  | 'streak_milestone'       // palier de streak atteint (4/8/15/30 j)
  | 'comeback_welcome'       // reprise après pause > 5 j
  | 'trend_consolidate'      // 2026-06-09 — coach soft-négatif : axe en baisse « à consolider » (neutre)
  | 'title_ready'            // 2026-06-23 — MT-19/axe E : titre fraîchement activé prêt (1er sync)
  | 'rival_encounter'        // 2026-07-17 — lot relations-E : nouveau duel contre un top rival post-sync

export const ALL_CATEGORIES: NotificationCategory[] = [
  'app_release',
  'match_synced',
  'media_added',
  'media_liked',
  'objective_assigned',
  'objective_completed',
  'challenge_added',
  'challenge_completed',
  'season_pass_level',
  'sync_error',
  'personal_record',
  'threshold_crossed',
  'friend_added',
  'friend_sync_completed',
  'data_health_warning',
  'career_rank',
  'skill_tier',
  'battlepass_completed',
  'citation_tier',
  'citation_mastery',
  // Progression V2 — coach proactif.
  'record_near_miss',
  'milestone_unlocked',
  'milestone_near_miss',
  'lusr_tier_approach',
  'streak_milestone',
  'comeback_welcome',
  'trend_consolidate',
  // MT-19 / axe E.
  'title_ready',
  // Relations-E — rival croisé post-sync.
  'rival_encounter',
]

export type NotificationSeverity = 'info' | 'success' | 'warn' | 'error'

export type NotificationDelivery = 'both' | 'inapp' | 'toast' | 'off'

export interface NotificationActor {
  xuid: string
  name: string
}

export interface Notification {
  id: number
  category: NotificationCategory
  severity: NotificationSeverity
  title_key: string
  body_key?: string
  params?: Record<string, unknown>
  target_route?: string
  target_search?: Record<string, unknown>
  actor?: NotificationActor
  source: string
  created_at: string // ISO timestamp UTC
  read_at?: string | null
}

export interface NotificationListResult {
  items: Notification[]
  next_cursor?: number | null
}

export interface UnreadCount {
  count: number
  /** Non-lues de severity != 'info' — alimente le badge cloche (DP6). */
  badge_count: number
  by_category: Record<string, number>
}

export interface NotificationPreference {
  category: NotificationCategory
  enabled: boolean
  delivery: NotificationDelivery
}

export interface NotificationListFilter {
  unread_only?: boolean
  category?: NotificationCategory
  limit?: number
  before_id?: number
}

export interface MarkResult {
  updated: number
}
