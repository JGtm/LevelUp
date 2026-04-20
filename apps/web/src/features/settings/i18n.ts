export type SettingsLocale = 'fr' | 'en'

export interface SettingsText {
  // Page
  pageTitle: string
  pageSubtitle: string
  savedStatus: string
  errorStatus: string
  loading: string

  // Onglets
  tabGeneral: string
  tabSync: string
  tabLab: string
  tabUsers: string

  // Sync manuelle
  manualSyncTitle: string
  manualSyncButton: string
  manualSyncRunning: string
  manualSyncDescription: string

  // Instance
  instanceLabel: string
  instanceTitle: string
  instanceDescription: string
  openLabButton: string

  // Utilisateurs
  usersTitle: string
  usersDescription: string
  openUsersButton: string

  // Interface
  interfaceTitle: string
  langLabel: string
  langFr: string
  langEn: string
  timezoneLabel: string
  showRecords: string
  normalizeModeLabels: string
  excludeBTB: string
  refreshClearsCaches: string

  // Discord
  discordTitle: string
  discordEnabled: string
  discordNotifySync: string
  discordNotifyBackfill: string
  discordNotifyNewMedia: string
  discordNoWebhook: string

  // Médias
  mediaTitle: string
  mediaWatcherEnabled: string
  mediaToleranceLabel: string
  mediaNoBaseDir: string

  // Synchronisation SPNKr
  spnkrTitle: string
  spnkrAutoSync: string
  spnkrAutoSyncInterval: string
  spnkrAutoSyncIntervalUnit: string
  spnkrAutoSyncIntervalMinutes: string
  spnkrAutoSyncIntervalMinutesUnit: string
  spnkrRefreshWithBackfill: string

  // Détection de présence
  watcherTitle: string
  watcherPresenceEnabled: string
  watcherPresenceDescription: string

  // Backfill
  backfillTitle: string
  backfillMedals: string
  backfillSkill: string
  backfillAliases: string
  backfillPersonalScores: string
  backfillPerfScores: string
  backfillLUSR: string
  backfillEvents: string
  backfillWeapons: string
}

const FR_TEXT: SettingsText = {
  pageTitle: 'Paramètres',
  pageSubtitle: "Configuration de l'application",
  savedStatus: '✓ Enregistré',
  errorStatus: '✗ Erreur lors de la sauvegarde',
  loading: 'Chargement des paramètres…',

  tabGeneral: 'Général',
  tabSync: 'Synchronisation',
  tabLab: 'Lab',
  tabUsers: 'Utilisateurs',

  manualSyncTitle: 'Synchronisation manuelle',
  manualSyncButton: '↻ Synchroniser tous les joueurs',
  manualSyncRunning: 'Synchronisation en cours…',
  manualSyncDescription: 'Lance une synchronisation delta immédiate pour tous les joueurs configurés.',

  instanceLabel: 'Instance',
  instanceTitle: 'Lab interne',
  instanceDescription:
    "Ouvrir l'explorateur interne des métadonnées Waypoint, du diff OpenAPI et des diagnostics locaux.",
  openLabButton: 'Ouvrir le Lab',

  usersTitle: 'Gestion des utilisateurs',
  usersDescription:
    'Gérer les comptes utilisateurs, les rôles et les codes d\u2019invitation.',
  openUsersButton: 'Gérer les utilisateurs',

  interfaceTitle: 'Interface',
  langLabel: 'Langue',
  langFr: 'Français',
  langEn: 'English',
  timezoneLabel: 'Fuseau horaire',
  showRecords: 'Afficher les records',
  normalizeModeLabels: 'Normaliser les libellés de modes',
  excludeBTB: 'Exclure BTB du classement carrière',
  refreshClearsCaches: "Vider les caches à l'actualisation",

  discordTitle: 'Notifications Discord',
  discordEnabled: 'Activer les notifications',
  discordNotifySync: 'Notifier à la synchronisation',
  discordNotifyBackfill: 'Notifier au backfill',
  discordNotifyNewMedia: 'Notifier pour les nouveaux médias',
  discordNoWebhook:
    "Les notifications Discord sont activées mais aucun webhook URL n'est configuré.",

  mediaTitle: 'Médias',
  mediaWatcherEnabled: 'Surveillance automatique des médias',
  mediaToleranceLabel: 'Tolérance association (min)',
  mediaNoBaseDir: "La surveillance des médias est activée mais aucun dossier source n'est défini.",

  spnkrTitle: 'Synchronisation périodique',
  spnkrAutoSync: 'Synchronisation automatique',
  spnkrAutoSyncInterval: 'Intervalle (heures)',
  spnkrAutoSyncIntervalUnit: 'h',
  spnkrAutoSyncIntervalMinutes: 'Intervalle (minutes)',
  spnkrAutoSyncIntervalMinutesUnit: 'min',
  spnkrRefreshWithBackfill: 'Lancer un backfill après chaque synchronisation',

  watcherTitle: 'Détection de présence',
  watcherPresenceEnabled: 'Détection automatique via Xbox RTA',
  watcherPresenceDescription:
    'Détecte automatiquement quand vous lancez ou quittez Halo Infinite pour déclencher une synchronisation. Nécessite un jeton XSTS valide.',

  backfillTitle: 'Données à inclure dans le backfill',
  backfillMedals: 'Médailles',
  backfillSkill: 'Classement (CSR/MMR)',
  backfillAliases: 'Alias gamertag',
  backfillPersonalScores: 'Scores personnels',
  backfillPerfScores: 'Scores performance',
  backfillLUSR: 'LUSR',
  backfillEvents: 'Événements',
  backfillWeapons: 'Armes',
}

const EN_TEXT: SettingsText = {
  pageTitle: 'Settings',
  pageSubtitle: 'Application configuration',
  savedStatus: '✓ Saved',
  errorStatus: '✗ Save failed',
  loading: 'Loading settings…',

  tabGeneral: 'General',
  tabSync: 'Synchronisation',
  tabLab: 'Lab',
  tabUsers: 'Users',

  manualSyncTitle: 'Manual synchronisation',
  manualSyncButton: '↻ Synchronise all players',
  manualSyncRunning: 'Synchronisation running…',
  manualSyncDescription: 'Trigger an immediate delta sync for all configured players.',

  instanceLabel: 'Instance',
  instanceTitle: 'Internal lab',
  instanceDescription:
    'Open the internal explorer for Waypoint metadata, OpenAPI diffs and local diagnostics.',
  openLabButton: 'Open the Lab',

  usersTitle: 'User management',
  usersDescription: 'Manage user accounts, roles and invite codes.',
  openUsersButton: 'Manage users',

  interfaceTitle: 'Interface',
  langLabel: 'Language',
  langFr: 'Français',
  langEn: 'English',
  timezoneLabel: 'Time zone',
  showRecords: 'Show records',
  normalizeModeLabels: 'Normalize mode labels',
  excludeBTB: 'Exclude BTB from career ranking',
  refreshClearsCaches: 'Clear caches on refresh',

  discordTitle: 'Discord notifications',
  discordEnabled: 'Enable notifications',
  discordNotifySync: 'Notify on sync',
  discordNotifyBackfill: 'Notify on backfill',
  discordNotifyNewMedia: 'Notify on new media',
  discordNoWebhook:
    'Discord notifications are enabled but no webhook URL is configured.',

  mediaTitle: 'Media',
  mediaWatcherEnabled: 'Automatic media watcher',
  mediaToleranceLabel: 'Association tolerance (min)',
  mediaNoBaseDir: 'Media watcher is enabled but no source folder is defined.',

  spnkrTitle: 'Periodic synchronisation',
  spnkrAutoSync: 'Automatic synchronisation',
  spnkrAutoSyncInterval: 'Interval (hours)',
  spnkrAutoSyncIntervalUnit: 'h',
  spnkrAutoSyncIntervalMinutes: 'Interval (minutes)',
  spnkrAutoSyncIntervalMinutesUnit: 'min',
  spnkrRefreshWithBackfill: 'Run a backfill after each synchronisation',

  watcherTitle: 'Presence detection',
  watcherPresenceEnabled: 'Automatic detection via Xbox RTA',
  watcherPresenceDescription:
    'Automatically detects when you launch or quit Halo Infinite to trigger a sync. Requires a valid XSTS token.',

  backfillTitle: 'Data to include in backfill',
  backfillMedals: 'Medals',
  backfillSkill: 'Ranking (CSR/MMR)',
  backfillAliases: 'Gamertag aliases',
  backfillPersonalScores: 'Personal scores',
  backfillPerfScores: 'Performance scores',
  backfillLUSR: 'LUSR',
  backfillEvents: 'Events',
  backfillWeapons: 'Weapons',
}

const TEXT: Record<SettingsLocale, SettingsText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeSettingsLocale(locale?: string | null): SettingsLocale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getSettingsText(locale?: string | null): SettingsText {
  return TEXT[normalizeSettingsLocale(locale)]
}
