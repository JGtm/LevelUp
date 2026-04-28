export type SettingsLocale = 'fr' | 'en'

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
  tabSync: string
  tabAnalyse: string
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

  // Admin — Fournisseur d'authentification
  authProviderTitle: string
  authProviderLabel: string
  authProviderMsal: string
  authProviderSisu: string
  authProviderHint: string

  // Onglet Accessibilité
  tabAccessibility: string
  accessibilityTitle: string
  accessibilityDescription: string
  paletteLabel: string
  paletteDefault: string
  paletteDefaultDesc: string
  paletteOkabeIto: string
  paletteOkabeItoDesc: string
  previewLabel: string
}

const FR_TEXT: SettingsText = {
  pageTitle: 'Paramètres',
  pageSubtitle: "Configuration de l'application",
  savedStatus: '✓ Enregistré',
  errorStatus: '✗ Erreur lors de la sauvegarde',
  loading: 'Chargement des paramètres…',

  tabGeneral: 'Général',
  tabSync: 'Synchronisation',
  tabAnalyse: 'Analyse',
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
  discordNotifyFriends: 'Notifier pour les amis (ajout + sessions reclassées)',
  discordNoWebhook:
    "Les notifications Discord sont activées mais aucun webhook URL n'est configuré.",

  mediaTitle: 'Médias',
  mediaWatcherEnabled: 'Surveillance automatique des médias',
  mediaToleranceLabel: 'Tolérance association (min)',
  mediaNoBaseDir: "La surveillance des médias est activée mais aucun dossier source n'est défini.",
  mediaBaseDirLabel: 'Dossier des captures',
  mediaBaseDirPlaceholder: 'Ex : C:\\Users\\Moi\\Videos\\Captures ou /mnt/captures',
  mediaBaseDirHint: 'Sous-dossiers par gamertag attendus : {chemin}/{gamertag}/',  mediaScanButton: 'Indexer les médias',
  mediaScanRunning: 'Indexation en cours…',
  mediaScanDone: '✓ Indexation lancée',
  mediaScanError: '✗ Échec de l’indexation',

  autoSyncTitle: 'Synchronisation automatique',
  schedulerSectionTitle: 'Synchronisation planifiée',
  watcherSectionTitle: 'Synchronisation par détection de présence Xbox',

  spnkrTitle: 'Synchronisation périodique',
  spnkrAutoSync: 'Synchronisation automatique',
  spnkrAutoSyncInterval: 'Intervalle (heures)',
  spnkrAutoSyncIntervalUnit: 'h',
  spnkrAutoSyncIntervalMinutes: 'Intervalle (minutes)',
  spnkrAutoSyncIntervalMinutesUnit: 'min',

  watcherTitle: 'Détection de présence',
  watcherPresenceEnabled: 'Détection automatique de présence Xbox',
  watcherPresenceDescription:
    'Détecte automatiquement quand vous lancez ou quittez Halo Infinite pour déclencher une synchronisation. Nécessite un jeton XSTS valide.',
  watcherAuthButton: 'Connecter via Xbox',
  watcherAuthReconnect: 'Reconnecter Xbox',
  watcherAuthInstructions: 'Rendez-vous sur {url} et entrez le code ci-dessous :',
  watcherAuthCopyCode: 'Copier le code',
  watcherAuthOpenLink: 'Ouvrir le lien',
  watcherAuthPending: 'En attente de validation…',
  watcherAuthSuccess: 'Connexion réussie ! Token XSTS valide.',
  watcherAuthFailed: 'Échec de la connexion.',
  watcherTokenValid: 'Jeton valide jusqu’au {date} ({gamertag})',
  watcherTokenExpired: 'Jeton expiré',
  watcherTokenMissing: 'Aucun jeton Xbox configuré',
  watcherPlayersLabel: 'Joueurs surveillés',
  watcherPlayersAll: 'Tous les joueurs',
  watcherSubscriptionsUpdated: 'Abonnements mis à jour',
  watcherRtaConnected: 'Connexion Xbox active',
  watcherRtaDisconnected: 'Connexion Xbox inactive',
  watcherSubscribeError: 'Échec de la surveillance',
  watcherStateIdle: 'En attente',
  watcherStateWatching: 'En surveillance',
  watcherStateSyncing: 'Synchronisation',
  watcherStateCooling: 'Cooldown',
  watcherInGame: 'En jeu',

  backfillTitle: 'Recalcul rétroactif',
  backfillMedals: 'Médailles',
  backfillSkill: 'Classement (CSR/MMR)',
  backfillAliases: 'Alias gamertag',
  backfillPersonalScores: 'Scores personnels',
  backfillPerfScores: 'Scores performance',
  backfillLUSR: 'LUSR',
  backfillEvents: 'Événements',
  backfillWeapons: 'Armes',
  backfillPlayerLabel: 'Joueur',
  backfillForceLabel: 'Forcer le recalcul complet pour les options sélectionnées',
  backfillRunButton: 'Lancer le recalcul rétroactif',
  backfillRunningLabel: 'Recalcul rétroactif en cours…',
  backfillNoScopeHint: 'Sélectionnez au moins un type de données.',
  backfillForceConfirmTitle: 'Forcer un rescan complet ?',
  backfillForceConfirmBody:
    'Cette opération rescanne tous les matchs existants pour les options sélectionnées ' +
    '(y compris ceux déjà traités). Peut prendre plusieurs minutes selon l\'historique.',
  backfillForceConfirmOk: 'Forcer',
  backfillForceConfirmCancel: 'Annuler',
  backfillWarningsHeader: 'Avertissements',

  backfillToastStarted: 'Recalcul rétroactif démarré',
  backfillToastStartFailed: 'Impossible de démarrer le recalcul',
  backfillToastSucceeded: 'Recalcul rétroactif terminé',
  backfillToastSucceededWithWarnings: 'Recalcul terminé avec avertissements',
  backfillToastFailed: 'Recalcul rétroactif échoué',
  backfillToastCancelled: 'Recalcul rétroactif annulé',

  syncToastStarted: 'Synchronisation démarrée',
  syncToastStartFailed: 'Impossible de démarrer la synchronisation',
  syncToastSucceeded: 'Synchronisation terminée',
  syncToastSucceededWithWarnings: 'Synchronisation terminée avec avertissements',
  syncToastFailed: 'Synchronisation échouée',
  syncToastCancelled: 'Synchronisation annulée',

  // Onglet Analyse — Sessions
  analyseTitle: 'Paramètres d’analyse',
  sessionGroupingTitle: 'Regroupement de sessions',
  sessionGapLabel: 'Délai entre deux sessions',
  sessionGapUnit: 'min',
  sessionGapHint:
    'Deux matchs séparés de plus de X minutes appartiennent à des sessions différentes. ' +
    '120 min (2 h) est le réglage recommandé pour une soirée de jeu classique.',
  sessionTeamChangeLabel: 'Composition de l’équipe',
  sessionTeamChangeIgnore: 'Ignorer la composition',
  sessionTeamChangeGroup: 'Changement de groupe',
  sessionTeamChangeFriends: 'Amis seulement (défaut)',
  sessionTeamChangeHint: {
    intro: 'Détermine si un changement de composition de votre groupe déclenche une nouvelle session.',
    items: [
      '« Ignorer la composition » : les changements de groupe n\'ont aucun effet sur le découpage.',
      '« Changement de groupe » : toute arrivée ou départ d\'un joueur (y compris des inconnus) démarre une nouvelle session.',
      '« Amis seulement » : seules les arrivées\/départs des joueurs de votre liste « Mon escouade » sont observés. Idéal si vous jouez souvent avec des inconnus, mais que votre noyau d\'amis reste stable toute la soirée (défaut).',
    ],
  },
  sessionSplitRankedLabel: 'Dissocier si passage classé ↔ social',
  sessionSplitRankedHint:
    'Active une nouvelle session dès que vous basculez entre une playlist classée et une playlist sociale. ' +
    'Utile si vous séparez vos stats ranked des sessions casual.',
  sessionRecalcButton: 'Recalculer les sessions',
  sessionRecalcRunning: 'Synchronisation en cours…',
  sessionRecalcPending: 'Recalcul programmé à la fin du job en cours',
  sessionRecalcConfirmTitle: 'Recalculer les sessions ?',
  sessionRecalcConfirmBody:
    'Les sessions existantes seront supprimées et recalculées à partir de vos matchs. ' +
    'L\'opération est rapide mais irréversible.',
  sessionRecalcConfirmOk: 'Recalculer',
  sessionRecalcConfirmCancel: 'Annuler',

  // Onglet Analyse — Badges de performance
  badgesTitle: 'Badges de performance',
  badgeSensitivityLabel: 'Sensibilité des badges',
  badgeSensitivityRelaxed: 'Souple',
  badgeSensitivityStandard: 'Standard',
  badgeSensitivityStrict: 'Strict',
  badgeSensitivityHint: {
    intro:
      "LevelUp analyse l'évolution du score d'équipe pour détecter les moments décisifs. " +
      "Un « écart » est l'avance d'une équipe exprimée en % du score final maximum. " +
      "Exemple : un match terminé 50-30 a un écart de 40 %.",
    items: [
      '« Souple » : écart ≥ 25 % — plus de badges.',
      '« Standard » : écart ≥ 40 % — seuils historiques (recommandé).',
      '« Strict » : écart ≥ 60 % — uniquement les matchs très marquants.',
    ],
    outro: 'Ce réglage nécessite un recalcul des badges (job automatique).',
  },
  badgeExcludeBotsFromBadgesLabel: 'Exclure les matchs avec bots des attributions de badges',
  badgeExcludeBotsFromBadgesHint:
    'Quand Halo Infinite manque de joueurs, il remplace les absents par des bots. ' +
    'Les performances des bots étant variables, un badge Domination ou Humiliation obtenu ' +
    'contre des bots ne reflète pas votre niveau réel.',
  badgeExcludeBotsFromRecordsLabel: 'Exclure les matchs avec bots des records carrière',
  badgeExcludeBotsFromRecordsHint:
    'Les matchs avec bots peuvent produire des stats atypiques ' +
    'qui fausseraient vos records personnels.',

  authProviderTitle: "Fournisseur d'authentification",
  authProviderLabel: 'Provider',
  authProviderMsal: 'MSAL (Azure)',
  authProviderSisu: 'SISU (Xbox natif)',
  authProviderHint:
    'MSAL utilise une app Azure enregistrée. ' +
    'SISU utilise le client Xbox natif (000000004c20a908) sans configuration Azure. ' +
    'Modification prise en compte au redémarrage du serveur.',

  tabAccessibility: 'Accessibilité',
  accessibilityTitle: 'Accessibilité visuelle',
  accessibilityDescription: 'Choisissez une palette de couleurs adaptée à votre vision. La palette Okabe-Ito (2008) est conçue pour être perceptible par les personnes daltonniennes.',
  paletteLabel: 'Palette de couleurs',
  paletteDefault: 'Standard (défaut)',
  paletteDefaultDesc: 'Palette originale de LevelUp.',
  paletteOkabeIto: 'Okabe-Ito (daltonisme)',
  paletteOkabeItoDesc: 'Palette universellement lisible, distinguable en cas de deutéranopie, protanopie et tritanopie.',
  previewLabel: 'Aperçu',
}

const EN_TEXT: SettingsText = {
  pageTitle: 'Settings',
  pageSubtitle: 'Application configuration',
  savedStatus: '✓ Saved',
  errorStatus: '✗ Save failed',
  loading: 'Loading settings…',

  tabGeneral: 'General',
  tabSync: 'Synchronisation',
  tabAnalyse: 'Analysis',
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
  discordNotifyFriends: 'Notify on friend events (add + reclassified sessions)',
  discordNoWebhook:
    'Discord notifications are enabled but no webhook URL is configured.',

  mediaTitle: 'Media',
  mediaWatcherEnabled: 'Automatic media watcher',
  mediaToleranceLabel: 'Association tolerance (min)',
  mediaNoBaseDir: 'Media watcher is enabled but no source folder is defined.',
  mediaBaseDirLabel: 'Captures folder',
  mediaBaseDirPlaceholder: 'e.g. C:\\Users\\Me\\Videos\\Captures or /mnt/captures',
  mediaBaseDirHint: 'Subfolders by gamertag expected: {path}/{gamertag}/',
  mediaScanButton: 'Index media',
  mediaScanRunning: 'Indexing…',
  mediaScanDone: '✓ Indexation started',
  mediaScanError: '✗ Indexation failed',


  autoSyncTitle: 'Automatic synchronisation',
  schedulerSectionTitle: 'Scheduled synchronisation',
  watcherSectionTitle: 'Xbox presence detection triggered synchronisation',

  spnkrTitle: 'Periodic synchronisation',
  spnkrAutoSync: 'Automatic synchronisation',
  spnkrAutoSyncInterval: 'Interval (hours)',
  spnkrAutoSyncIntervalUnit: 'h',
  spnkrAutoSyncIntervalMinutes: 'Interval (minutes)',
  spnkrAutoSyncIntervalMinutesUnit: 'min',

  watcherTitle: 'Presence detection',
  watcherPresenceEnabled: 'Automatic Xbox presence detection',
  watcherPresenceDescription:
    'Automatically detects when you launch or quit Halo Infinite to trigger a sync. Requires a valid XSTS token.',
  watcherAuthButton: 'Connect via Xbox',
  watcherAuthReconnect: 'Reconnect Xbox',
  watcherAuthInstructions: 'Go to {url} and enter the code below:',
  watcherAuthCopyCode: 'Copy code',
  watcherAuthOpenLink: 'Open link',
  watcherAuthPending: 'Waiting for validation…',
  watcherAuthSuccess: 'Connected! XSTS token valid.',
  watcherAuthFailed: 'Connection failed.',
  watcherTokenValid: 'Valid until {date} ({gamertag})',
  watcherTokenExpired: 'Token expired',
  watcherTokenMissing: 'No Xbox token configured',
  watcherPlayersLabel: 'Watched players',
  watcherPlayersAll: 'All players',
  watcherSubscriptionsUpdated: 'Subscriptions updated',
  watcherRtaConnected: 'Xbox connection active',
  watcherRtaDisconnected: 'Xbox connection inactive',
  watcherSubscribeError: 'Monitoring failed',
  watcherStateIdle: 'Idle',
  watcherStateWatching: 'Watching',
  watcherStateSyncing: 'Syncing',
  watcherStateCooling: 'Cooling',
  watcherInGame: 'In game',

  backfillTitle: 'Backfill',
  backfillMedals: 'Medals',
  backfillSkill: 'Ranking (CSR/MMR)',
  backfillAliases: 'Gamertag aliases',
  backfillPersonalScores: 'Personal scores',
  backfillPerfScores: 'Performance scores',
  backfillLUSR: 'LUSR',
  backfillEvents: 'Events',
  backfillWeapons: 'Weapons',
  backfillPlayerLabel: 'Player',
  backfillForceLabel: 'Force full rescan for selected options',
  backfillRunButton: 'Run backfill',
  backfillRunningLabel: 'Backfill running…',
  backfillNoScopeHint: 'Select at least one data type.',
  backfillForceConfirmTitle: 'Force a full rescan?',
  backfillForceConfirmBody:
    'This will rescan all existing matches for the selected options ' +
    '(including ones already processed). May take several minutes depending on history size.',
  backfillForceConfirmOk: 'Force',
  backfillForceConfirmCancel: 'Cancel',
  backfillWarningsHeader: 'Warnings',

  backfillToastStarted: 'Backfill started',
  backfillToastStartFailed: 'Failed to start backfill',
  backfillToastSucceeded: 'Backfill completed',
  backfillToastSucceededWithWarnings: 'Backfill completed with warnings',
  backfillToastFailed: 'Backfill failed',
  backfillToastCancelled: 'Backfill cancelled',

  syncToastStarted: 'Sync started',
  syncToastStartFailed: 'Failed to start sync',
  syncToastSucceeded: 'Sync completed',
  syncToastSucceededWithWarnings: 'Sync completed with warnings',
  syncToastFailed: 'Sync failed',
  syncToastCancelled: 'Sync cancelled',

  // Analyse tab — Sessions
  analyseTitle: 'Analysis settings',
  sessionGroupingTitle: 'Session grouping',
  sessionGapLabel: 'Gap between sessions',
  sessionGapUnit: 'min',
  sessionGapHint:
    'Two matches separated by more than X minutes belong to different sessions. ' +
    '120 min (2 h) is the recommended setting for a typical gaming evening.',
  sessionTeamChangeLabel: 'Team composition',
  sessionTeamChangeIgnore: 'Ignore composition',
  sessionTeamChangeGroup: 'Group change',
  sessionTeamChangeFriends: 'Friends only (default)',
  sessionTeamChangeHint: {
    intro: 'Determines whether a change in squad composition triggers a new session.',
    items: [
      '"Ignore composition": group changes have no effect on splitting.',
      '"Group change": any player joining or leaving (including randoms) starts a new session.',
      '"Friends only": only arrivals/departures of players in your My squad list are observed. Ideal if you often play with strangers who rotate, but your core friends stay together all evening (default).',
    ],
  },
  sessionSplitRankedLabel: 'Split on ranked ↔ social switch',
  sessionSplitRankedHint:
    'Starts a new session whenever you switch between a ranked and a social playlist. ' +
    'Useful if you separate your ranked stats from casual sessions.',
  sessionRecalcButton: 'Recalculate sessions',
  sessionRecalcRunning: 'Sync in progress…',
  sessionRecalcPending: 'Recalculation scheduled after current job',
  sessionRecalcConfirmTitle: 'Recalculate sessions?',
  sessionRecalcConfirmBody:
    'Existing sessions will be deleted and recalculated from your matches. ' +
    'The operation is fast but irreversible.',
  sessionRecalcConfirmOk: 'Recalculate',
  sessionRecalcConfirmCancel: 'Cancel',

  // Analyse tab — Performance badges
  badgesTitle: 'Performance badges',
  badgeSensitivityLabel: 'Badge sensitivity',
  badgeSensitivityRelaxed: 'Relaxed',
  badgeSensitivityStandard: 'Standard',
  badgeSensitivityStrict: 'Strict',
  badgeSensitivityHint: {
    intro:
      'LevelUp analyses the score curve over the match to detect decisive moments. ' +
      'A "gap" is one team\'s lead expressed as % of the final max score. ' +
      'Example: a match ending 50-30 has a 40 % gap (Standard threshold).',
    items: [
      '"Relaxed": gap ≥ 25 % — more badges.',
      '"Standard": gap ≥ 40 % — historical thresholds (recommended).',
      '"Strict": gap ≥ 60 % — only very decisive matches.',
    ],
    outro: 'This setting requires a badge recalculation (automatic job).',
  },
  badgeExcludeBotsFromBadgesLabel: 'Exclude bot matches from badges attributions',
  badgeExcludeBotsFromBadgesHint:
    'When Halo Infinite runs short of players, it substitutes bots. ' +
    'Bots performances may vary a lot, a Domination or Humiliation badge ' +
    'earned against bots does not reflect your real level.',
  badgeExcludeBotsFromRecordsLabel: 'Exclude bot matches from career records',
  badgeExcludeBotsFromRecordsHint:
    'Bot matches can produce atypical stats ' +
    'that would distort your personal records.',

  authProviderTitle: 'Authentication provider',
  authProviderLabel: 'Provider',
  authProviderMsal: 'MSAL (Azure)',
  authProviderSisu: 'SISU (native Xbox)',
  authProviderHint:
    'MSAL uses a registered Azure app. ' +
    'SISU uses the native Xbox client (000000004c20a908) with no Azure configuration required. ' +
    'Change takes effect on server restart.',

  tabAccessibility: 'Accessibility',
  accessibilityTitle: 'Visual accessibility',
  accessibilityDescription: 'Choose a colour palette suited to your vision. The Okabe-Ito (2008) palette is designed to be distinguishable for people with colour vision deficiencies.',
  paletteLabel: 'Colour palette',
  paletteDefault: 'Standard (default)',
  paletteDefaultDesc: 'Original LevelUp palette.',
  paletteOkabeIto: 'Okabe-Ito (colour-blind safe)',
  paletteOkabeItoDesc: 'Universally readable, distinguishable under deuteranopia, protanopia and tritanopia.',
  previewLabel: 'Preview',
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
