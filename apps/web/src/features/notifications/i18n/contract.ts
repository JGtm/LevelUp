/**
 * contract.ts — Le TYPE du dictionnaire des notifications : une seule déclaration, que `fr.ts`,
 * `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies NotificationsText`).
 *
 * Les clés notif.<category>.title et notif.<category>.body correspondent aux
 * title_key/body_key reçus du backend. Résolues par format.ts en injectant
 * les params (interpolation simple {name}, {count} sans plurals ICU pour le MVP).
 */
import type { NotificationCategory } from '../types'

export interface NotificationsText {
  // Cloche & badge
  bellAriaLabelEmpty: string
  bellAriaLabelWithCount: string // "{count} notifications non lues"

  // Dropdown
  dropdownTitle: string
  dropdownUnread: string
  dropdownOlder: string
  dropdownEmpty: string
  dropdownMarkAllRead: string
  dropdownViewAll: string
  dropdownErrorLoading: string

  // Actions item
  actionMarkAsRead: string
  actionMarkAsUnread: string
  actionDismiss: string
  actionView: string

  // Page dédiée
  pageTitle: string
  pageFilterAll: string
  pageFilterUnread: string
  pageGroupToday: string
  pageGroupYesterday: string
  pageGroupThisWeek: string
  pageGroupOlder: string
  pageBulkMarkRead: string
  pageBulkDismiss: string
  pageLoadMore: string
  pageEmpty: string
  /** Profil d'un ami : ses notifications ne sont ni lues ni affichées. */
  privateToOwner: string

  // Settings tab
  settingsTitle: string
  settingsDescription: string
  settingsMaster: string
  settingsMasterDescription: string
  settingsToasts: string
  settingsToastsDescription: string
  settingsCategoriesTitle: string
  settingsCategoriesDescription: string
  settingsDeliveryBoth: string
  settingsDeliveryInApp: string
  settingsDeliveryToast: string
  settingsDeliveryOff: string
  settingsSubscriptionsTitle: string
  settingsSubscriptionsDescription: string
  settingsRetentionTitle: string
  settingsRetentionLabel: string
  settingsTestButton: string
  settingsTestSent: string

  // Catégories — label + description (pour Settings)
  categoryLabel: Record<NotificationCategory, string>
  categoryDescription: Record<NotificationCategory, string>

  // Mapping clé métrique (envoyée par le backend) → libellé localisé.
  // Sert à éviter tout hardcode "KD"/"Winrate" dans les templates.
  metricLabel: Record<string, string>

  // Mapping fenêtre temporelle records V2 → libellé localisé (30d/90d/all_time).
  periodLabel: Record<string, string>

  // Templates de notif (rendus à partir de title_key/body_key + params)
  // Convention : "notif.<category>.title" / ".body"
  // Gardé sous forme de Record<string,string> pour permettre extension future
  // sans modifier le typage.
  templates: Record<string, string>

  // Temps relatif
  relJustNow: string
  relMinutesAgo: (n: number) => string
  relHoursAgo: (n: number) => string
  relDaysAgo: (n: number) => string
  relOnDate: (iso: string) => string
}
