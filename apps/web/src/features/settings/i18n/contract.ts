/**
 * contract.ts — Le TYPE du dictionnaire de la feature Paramètres : une seule déclaration, que
 * `fr.ts`, `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies SettingsText`).
 */

export interface HintBullets {
  intro: string
  items: string[]
  outro?: string
}

export interface SettingsText {
  // Page
  pageTitle: string
  pageSubtitle: string
  savedStatus: string
  errorStatus: string
  loading: string

  // Onglets
  tabGeneral: string
  // Onglet « Jeux » (sélection par titre)
  tabTitles: string
  titlesSectionTitle: string
  titlesSectionDesc: string
  titleStatusActive: string
  titleStatusPaused: string
  titleStatusNotTracked: string
  titlePurgeButton: string
  titlePurgeConfirm: string
  titleLastActiveHint: string
  titlePurgeResidualWarning: string
  titleActionError: string
  tabSync: string
  tabAnalyse: string
  tabAppearance: string
  tabData: string
  tabAccount: string
  tabNotifications: string

  // Carte « Groupes et partage » (onglet Compte) — lien vers /groups
  groupsCardTitle: string
  groupsCardDescription: string
  groupsCardOpen: string

  // Sync manuelle
  manualSyncTitle: string
  manualSyncButton: string
  manualSyncRunning: string
  manualSyncDescription: string

  // Instance
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
  refreshClearsCaches: string
  showWaypointColumn: string
  showWaypointColumnHint: string

  // Discord
  discordTitle: string
  discordEnabled: string
  discordNotifySync: string
  discordNotifyBackfill: string
  discordNotifyNewVersion: string
  discordNotifyFriends: string
  discordNoWebhook: string

  // Médias
  mediaTitle: string
  mediaWatcherEnabled: string
  mediaToleranceLabel: string
  mediaNoBaseDir: string
  mediaBaseDirLabel: string
  mediaBaseDirPlaceholder: string
  mediaBaseDirHint: string
  mediaDeleteSource: string
  mediaDeleteSourceHint: string
  mediaScanButton: string
  mediaScanRunning: string
  mediaScanDone: string
  mediaScanError: string

  // Synchronisation automatique (scheduler + watcher fusionnés)
  autoSyncTitle: string
  schedulerSectionTitle: string
  watcherSectionTitle: string

  // Synchronisation SPNKr
  spnkrTitle: string
  spnkrAutoSync: string
  spnkrAutoSyncInterval: string
  spnkrAutoSyncIntervalUnit: string
  spnkrAutoSyncIntervalMinutes: string
  spnkrAutoSyncIntervalMinutesUnit: string

  // Détection de présence
  watcherTitle: string
  watcherPresenceEnabled: string
  watcherPresenceDescription: string
  watcherAuthButton: string
  watcherAuthReconnect: string
  watcherAuthInstructions: string
  watcherAuthCopyCode: string
  /** Libellé du MÊME bouton pendant la fenêtre de confirmation (2 s) qui suit la copie. */
  watcherAuthCodeCopied: string
  watcherAuthOpenLink: string
  watcherAuthPending: string
  watcherAuthSuccess: string
  watcherAuthFailed: string
  watcherTokenValid: string
  watcherTokenExpired: string
  watcherTokenMissing: string
  watcherPlayersLabel: string
  watcherPlayersAll: string
  watcherSubscriptionsUpdated: string
  watcherRtaConnected: string
  watcherRtaDisconnected: string
  watcherSubscribeError: string
  watcherStateIdle: string
  watcherStateWatching: string
  watcherStateSyncing: string
  watcherStateCooling: string
  watcherInGame: string
  /** État Xbox Online (compte connecté à Xbox) */
  watcherPresenceOnline: string
  /** État Xbox Away (idle long sur Xbox) */
  watcherPresenceAway: string
  /** État Xbox Offline (compte déconnecté) */
  watcherPresenceOffline: string
  /** Présence inconnue (pas encore d'event reçu) */
  watcherPresenceUnknown: string
  /** Renommage du titleName "Online" (Dashboard Xbox) côté UI */
  watcherTitleXboxDashboard: string
  /** "Vu il y a {duration} sur {title}" — format relatif + jeu */
  watcherLastSeenRelative: string
  /** "Vu le {date} sur {title}" — format absolu si trop ancien */
  watcherLastSeenAbsolute: string
  /** "Jamais vu en jeu" — pas de last_seen connu */
  watcherNeverSeen: string
  /** Durée du « Vu il y a {duration} » : moins d'une minute, puis heures et jours ({n} = le compte). */
  watcherDurationUnderMin: string
  watcherDurationHours: string
  watcherDurationDayOne: string
  watcherDurationDayMany: string
  /** Bandeau du mode démo : réglages figés, seules la langue et l'accessibilité se modifient. */
  demoModeNotice: string

  // Backfill
  backfillTitle: string
  backfillMedals: string
  backfillSkill: string
  backfillAliases: string
  backfillPersonalScores: string
  backfillPerfScores: string
  backfillLUSR: string
  backfillCSR: string
  backfillEvents: string
  backfillEngagementScores: string
  backfillEngagementCoefficients: string
  backfillPlayerLabel: string
  backfillForceLabel: string
  backfillRunButton: string
  backfillRunningLabel: string
  backfillNoScopeHint: string
  backfillForceConfirmTitle: string
  backfillForceConfirmBody: string
  backfillForceConfirmOk: string
  backfillForceConfirmCancel: string
  backfillWarningsHeader: string

  // Backfill — toasts
  backfillToastStarted: string
  backfillToastStartFailed: string
  backfillToastSucceeded: string
  backfillToastSucceededWithWarnings: string
  backfillToastFailed: string
  backfillToastCancelled: string

  // Sync manuelle — toasts
  syncToastStarted: string
  syncToastStartFailed: string
  syncToastSucceeded: string
  syncToastSucceededWithWarnings: string
  syncToastFailed: string
  syncToastBusy: string
  syncToastCancelled: string

  // Onglet Analyse — Sessions
  analyseTitle: string
  sessionGroupingTitle: string
  sessionGapLabel: string
  sessionGapUnit: string
  sessionGapHint: string
  sessionTeamChangeLabel: string
  sessionTeamChangeIgnore: string
  sessionTeamChangeGroup: string
  sessionTeamChangeFriends: string
  sessionTeamChangeHint: HintBullets
  sessionSplitRankedLabel: string
  sessionSplitRankedHint: string
  sessionRecalcButton: string
  sessionRecalcRunning: string
  sessionRecalcPending: string
  sessionRecalcConfirmTitle: string
  sessionRecalcConfirmBody: string
  sessionRecalcConfirmOk: string
  sessionRecalcConfirmCancel: string

  // Onglet Analyse — Rejeu 2D
  replayRetentionTitle: string
  replayRetentionLabel: string
  replayRetentionUnit: string
  replayRetentionHint: string
  replayLocationLabel: string
  replayLocationAuto: string
  replayLocationLocal: string
  replayLocationWorker: string
  replayLocationOff: string
  replayLocationHint: string

  // Onglet Analyse — Badges de performance
  badgesTitle: string
  badgeSensitivityLabel: string
  badgeSensitivityRelaxed: string
  badgeSensitivityStandard: string
  badgeSensitivityStrict: string
  badgeSensitivityHint: HintBullets
  badgeExcludeBotsFromBadgesLabel: string
  badgeExcludeBotsFromBadgesHint: string
  badgeExcludeBotsFromRecordsLabel: string
  badgeExcludeBotsFromRecordsHint: string

  // Onglet Analyse — Progression Objectifs/Prestige
  progressionTitle: string
  showProgressionLabel: string
  progressionHint: string
  progressionGlossaryLink: string

  // Onglet Analyse — Coach proactif (pont coach → Prestige, ADR 0020)
  coachProactiveTitle: string
  coachProactiveLabel: string
  coachProactiveHint: string

  // Onglet Analyse — Rendement combat (OffensiveConversion)
  rendementTitle: string
  rendementExcludeAssistsLabel: string
  rendementExcludeAssistsHint: string

  // Onglet Accessibilité
  accessibilityTitle: string
  accessibilityDescription: string
  paletteLabel: string
  paletteDefault: string
  paletteDefaultDesc: string
  paletteOkabeIto: string
  paletteOkabeItoDesc: string
  paletteCividis: string
  paletteCividisDesc: string
  paletteTolBright: string
  paletteTolBrightDesc: string
  previewLabel: string
  previewSquadPlayersLabel: string
  /** Légende des pastilles de résultat de l'aperçu de palette. */
  previewOutcome: Record<'win' | 'loss' | 'draw' | 'dnf', string>
  /** Légende des pastilles d'écart (positif / neutre / négatif) de l'aperçu de palette. */
  previewDivergent: Record<'pos' | 'neutral' | 'neg', string>

  // Couleurs d'équipe (outline Halo)
  teamColorsTitle: string
  teamColorsDescription: string
  allyColorLabel: string
  enemyColorLabel: string
  teamColorDefault: string

  // Admin · Système — Sons du rejeu 2D (réglages d'instance)
  replaySoundTitle: string
  replaySoundDescription: string
  replaySoundVariationLabel: string
  replaySoundVariationHint: string
  replaySoundDistanceLabel: string
  replaySoundDistanceHint: string
  replaySoundOff: string
  replaySoundFull: string

  // Onglet Backup
  tabBackup: string
  backupStatusEnabled: string
  backupStatusNotConfigured: string
  backupStatusResticMissing: string
  backupNotConfiguredHint: string
  backupLastBackup: string
  backupNever: string
  backupSnapshotId: string
  backupDatabases: string
  backupDuration: string
  backupRunButton: string
  backupRunning: string
  backupRunDone: string
  backupRunSkipped: string
  backupRunError: string
  backupConfigTitle: string
  backupConfigInterval: string
  backupConfigRetention: string
  backupConfigRetentionValue: string
  backupIntegrityLabel: string
}
