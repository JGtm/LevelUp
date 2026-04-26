/**
 * i18n FR/EN du système de notifications.
 *
 * Pattern : dictionnaire typé par locale, accessible via getNotificationsText(locale).
 * Aligné avec features/{settings,help,...}/i18n.ts.
 *
 * Les clés notif.<category>.title et notif.<category>.body correspondent aux
 * title_key/body_key reçus du backend. Résolues par format.ts en injectant
 * les params (interpolation simple {name}, {count} sans plurals ICU pour le MVP).
 */
import type { NotificationCategory } from './types'

export type NotificationsLocale = 'fr' | 'en'

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

const FR: NotificationsText = {
  bellAriaLabelEmpty: 'Aucune notification non lue',
  bellAriaLabelWithCount: '{count} notifications non lues',

  dropdownTitle: 'Notifications',
  dropdownUnread: 'Non lues',
  dropdownOlder: 'Plus récentes',
  dropdownEmpty: 'Tout est calme par ici',
  dropdownMarkAllRead: 'Tout marquer comme lu',
  dropdownViewAll: 'Voir tout',
  dropdownErrorLoading: 'Impossible de charger les notifications',

  actionMarkAsRead: 'Marquer comme lu',
  actionMarkAsUnread: 'Marquer comme non lu',
  actionDismiss: 'Ignorer',
  actionView: 'Voir',

  pageTitle: 'Notifications',
  pageFilterAll: 'Toutes',
  pageFilterUnread: 'Non lues uniquement',
  pageGroupToday: "Aujourd'hui",
  pageGroupYesterday: 'Hier',
  pageGroupThisWeek: 'Cette semaine',
  pageGroupOlder: 'Plus ancien',
  pageBulkMarkRead: 'Marquer comme lues ({count})',
  pageBulkDismiss: 'Ignorer ({count})',
  pageLoadMore: 'Charger plus',
  pageEmpty: 'Aucune notification pour le moment',

  settingsTitle: 'Notifications',
  settingsDescription:
    'Gère les notifications in-app, les toasts et les abonnements par catégorie.',
  settingsMaster: 'Activer les notifications',
  settingsMasterDescription:
    "Désactive complètement la cloche et les toasts. Les événements continuent d'être enregistrés en base.",
  settingsToasts: 'Activer les toasts',
  settingsToastsDescription:
    "Affiche les nouvelles notifications en pop-up à l'écran (en plus du dropdown).",
  settingsCategoriesTitle: 'Catégories',
  settingsCategoriesDescription:
    "Choisis pour chaque type d'événement comment tu veux être notifié.",
  settingsDeliveryBoth: 'In-app + toast',
  settingsDeliveryInApp: 'In-app seulement',
  settingsDeliveryToast: 'Toast seulement',
  settingsDeliveryOff: 'Désactivé',
  settingsSubscriptionsTitle: 'Abonnements',
  settingsSubscriptionsDescription:
    'Active ou désactive le suivi automatique pour le joueur courant.',
  settingsRetentionTitle: 'Rétention',
  settingsRetentionLabel: 'Conserver les {n} dernières notifications',
  settingsTestButton: 'Envoyer une notification de test',
  settingsTestSent: 'Notification de test envoyée',

  categoryLabel: {
    app_release: 'Nouveauté app',
    match_synced: 'Match synchronisé',
    media_added: 'Média ajouté',
    objective_assigned: 'Nouvel objectif',
    objective_completed: 'Objectif complété',
    challenge_added: 'Nouveau défi',
    challenge_completed: 'Défi complété',
    season_pass_level: 'Niveau Season Pass',
    sync_error: 'Erreur de synchronisation',
    personal_record: 'Record personnel',
    threshold_crossed: 'Palier franchi',
  },
  categoryDescription: {
    app_release: 'Une nouvelle version de LevelUp est disponible.',
    match_synced: 'Tes nouveaux matchs ont été synchronisés.',
    media_added: 'Une vidéo ou screenshot a été ajouté.',
    objective_assigned: 'Un objectif a été créé automatiquement.',
    objective_completed: 'Tu as terminé un objectif.',
    challenge_added: 'Un nouveau défi est disponible.',
    challenge_completed: 'Tu as terminé un défi.',
    season_pass_level: 'Tu as débloqué un nouveau niveau de Season Pass.',
    sync_error: "La synchronisation a échoué — relance manuelle conseillée.",
    personal_record: 'Tu as battu un record personnel.',
    threshold_crossed: 'Un palier de KD ou winrate a été franchi.',
  },

  templates: {
    'notif.app_release.title': 'Nouvelle version : {version}',
    'notif.app_release.body': 'Découvrez les nouveautés dans le changelog.',
    'notif.match_synced.title': '{count} match(s) synchronisé(s)',
    'notif.match_synced.body': 'Tes statistiques sont à jour.',
    'notif.media_added.title': 'Média ajouté par {actor_name}',
    'notif.media_added.body': '{count} fichier(s) associé(s) à un match.',
    'notif.objective_assigned.title': 'Nouvel objectif',
    'notif.objective_assigned.body': '{name} — récompense {reward}.',
    'notif.objective_completed.title': 'Objectif complété',
    'notif.objective_completed.body': '{name} — bravo !',
    'notif.challenge_added.title': 'Nouveau défi disponible',
    'notif.challenge_added.body': '{name}',
    'notif.challenge_completed.title': 'Défi complété',
    'notif.challenge_completed.body': '{name} — récompense {reward}.',
    'notif.season_pass_level.title': 'Niveau {level} atteint',
    'notif.season_pass_level.body': 'Tu progresses sur le Season Pass.',
    'notif.sync_error.title': 'Erreur de synchronisation',
    'notif.sync_error.body': '{message}',
    'notif.personal_record.title': 'Record personnel',
    'notif.personal_record.body': '{metric}: {value}',
    'notif.threshold_crossed.title': 'Palier franchi',
    'notif.threshold_crossed.body': '{metric} {direction} : {value}',
  },

  relJustNow: 'à l’instant',
  relMinutesAgo: (n) => (n <= 1 ? 'il y a 1 min' : `il y a ${n} min`),
  relHoursAgo: (n) => (n <= 1 ? 'il y a 1 h' : `il y a ${n} h`),
  relDaysAgo: (n) => (n <= 1 ? 'il y a 1 j' : `il y a ${n} j`),
  relOnDate: (iso) => new Date(iso).toLocaleDateString('fr-FR'),
}

const EN: NotificationsText = {
  bellAriaLabelEmpty: 'No unread notifications',
  bellAriaLabelWithCount: '{count} unread notifications',

  dropdownTitle: 'Notifications',
  dropdownUnread: 'Unread',
  dropdownOlder: 'Recent',
  dropdownEmpty: "You're all caught up",
  dropdownMarkAllRead: 'Mark all as read',
  dropdownViewAll: 'View all',
  dropdownErrorLoading: 'Failed to load notifications',

  actionMarkAsRead: 'Mark as read',
  actionMarkAsUnread: 'Mark as unread',
  actionDismiss: 'Dismiss',
  actionView: 'View',

  pageTitle: 'Notifications',
  pageFilterAll: 'All',
  pageFilterUnread: 'Unread only',
  pageGroupToday: 'Today',
  pageGroupYesterday: 'Yesterday',
  pageGroupThisWeek: 'This week',
  pageGroupOlder: 'Older',
  pageBulkMarkRead: 'Mark as read ({count})',
  pageBulkDismiss: 'Dismiss ({count})',
  pageLoadMore: 'Load more',
  pageEmpty: 'No notifications yet',

  settingsTitle: 'Notifications',
  settingsDescription:
    'Manage in-app notifications, toasts and per-category subscriptions.',
  settingsMaster: 'Enable notifications',
  settingsMasterDescription:
    'Fully disables bell and toasts. Events still get persisted server-side.',
  settingsToasts: 'Enable toasts',
  settingsToastsDescription:
    'Show new notifications as on-screen pop-ups (in addition to the dropdown).',
  settingsCategoriesTitle: 'Categories',
  settingsCategoriesDescription:
    'Pick how you want to be notified for each event type.',
  settingsDeliveryBoth: 'In-app + toast',
  settingsDeliveryInApp: 'In-app only',
  settingsDeliveryToast: 'Toast only',
  settingsDeliveryOff: 'Off',
  settingsSubscriptionsTitle: 'Subscriptions',
  settingsSubscriptionsDescription:
    'Toggle automatic tracking for the current player.',
  settingsRetentionTitle: 'Retention',
  settingsRetentionLabel: 'Keep last {n} notifications',
  settingsTestButton: 'Send test notification',
  settingsTestSent: 'Test notification sent',

  categoryLabel: {
    app_release: 'App release',
    match_synced: 'Match synced',
    media_added: 'Media added',
    objective_assigned: 'New objective',
    objective_completed: 'Objective completed',
    challenge_added: 'New challenge',
    challenge_completed: 'Challenge completed',
    season_pass_level: 'Season Pass level',
    sync_error: 'Sync error',
    personal_record: 'Personal record',
    threshold_crossed: 'Threshold crossed',
  },
  categoryDescription: {
    app_release: 'A new LevelUp version is available.',
    match_synced: 'New matches were synchronized.',
    media_added: 'A video or screenshot was added.',
    objective_assigned: 'An objective was auto-assigned to you.',
    objective_completed: 'You completed an objective.',
    challenge_added: 'A new challenge is available.',
    challenge_completed: 'You completed a challenge.',
    season_pass_level: 'You reached a new Season Pass level.',
    sync_error: 'The sync failed — manual retry recommended.',
    personal_record: 'You broke a personal record.',
    threshold_crossed: 'A KD or winrate threshold was crossed.',
  },

  templates: {
    'notif.app_release.title': 'New version: {version}',
    'notif.app_release.body': 'See what changed in the changelog.',
    'notif.match_synced.title': '{count} match(es) synced',
    'notif.match_synced.body': 'Your stats are up to date.',
    'notif.media_added.title': 'Media added by {actor_name}',
    'notif.media_added.body': '{count} file(s) linked to a match.',
    'notif.objective_assigned.title': 'New objective',
    'notif.objective_assigned.body': '{name} — reward {reward}.',
    'notif.objective_completed.title': 'Objective completed',
    'notif.objective_completed.body': '{name} — nice!',
    'notif.challenge_added.title': 'New challenge available',
    'notif.challenge_added.body': '{name}',
    'notif.challenge_completed.title': 'Challenge completed',
    'notif.challenge_completed.body': '{name} — reward {reward}.',
    'notif.season_pass_level.title': 'Level {level} reached',
    'notif.season_pass_level.body': 'Your Season Pass progresses.',
    'notif.sync_error.title': 'Sync error',
    'notif.sync_error.body': '{message}',
    'notif.personal_record.title': 'Personal record',
    'notif.personal_record.body': '{metric}: {value}',
    'notif.threshold_crossed.title': 'Threshold crossed',
    'notif.threshold_crossed.body': '{metric} {direction}: {value}',
  },

  relJustNow: 'just now',
  relMinutesAgo: (n) => (n <= 1 ? '1 min ago' : `${n} min ago`),
  relHoursAgo: (n) => (n <= 1 ? '1 h ago' : `${n} h ago`),
  relDaysAgo: (n) => (n <= 1 ? '1 d ago' : `${n} d ago`),
  relOnDate: (iso) => new Date(iso).toLocaleDateString('en-US'),
}

const DICTS: Record<NotificationsLocale, NotificationsText> = { fr: FR, en: EN }

export function getNotificationsText(locale: NotificationsLocale): NotificationsText {
  return DICTS[locale] ?? FR
}
