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
  | 'season_pass_level'
  | 'sync_error'
  | 'personal_record'
  | 'threshold_crossed'
  | 'friend_added'           // §6 Squad/Sessions overhaul
  | 'friend_sync_completed'  // §6 Squad/Sessions overhaul

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
