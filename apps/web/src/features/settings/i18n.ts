import type { Locale } from '@/lib/i18n/locale'

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
  excludeBTB: string
  refreshClearsCaches: string

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
  backfillWeapons: string
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

  // Couleurs d'équipe (outline Halo)
  teamColorsTitle: string
  teamColorsDescription: string
  allyColorLabel: string
  enemyColorLabel: string
  teamColorDefault: string

  // Onglet Backup
  tabBackup: string
  backupTitle: string
  backupStatusEnabled: string
  backupStatusDisabled: string
  backupStatusResticMissing: string
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

const FR_TEXT: SettingsText = {
  pageTitle: 'Paramètres',
  pageSubtitle: "Configuration de l'application",
  savedStatus: '✓ Enregistré',
  errorStatus: '✗ Erreur lors de la sauvegarde',
  loading: 'Chargement des paramètres…',

  tabGeneral: 'Général',
  tabTitles: 'Jeux',
  titlesSectionTitle: 'Jeux suivis',
  titlesSectionDesc:
    'Active ou met en pause la synchronisation par jeu. Mettre en pause conserve les données ; au moins un jeu doit rester actif.',
  titleStatusActive: 'Actif',
  titleStatusPaused: 'En pause',
  titleStatusNotTracked: 'Non suivi',
  titlePurgeButton: 'Purger',
  titlePurgeConfirm: 'Supprimer définitivement les données de ce jeu pour ce joueur ?',
  titleLastActiveHint: 'Au moins un jeu doit rester actif.',
  titlePurgeResidualWarning: 'Jeu désactivé, mais certains fichiers de données n’ont pas pu être supprimés.',
  titleActionError: 'L’opération a échoué. Réessaie.',
  tabSync: 'Synchronisation',
  tabAnalyse: 'Analyse',
  tabAppearance: 'Apparence & Accessibilité',
  tabData: 'Données & Médias',
  tabAccount: 'Compte',
  tabNotifications: 'Notifications',

  manualSyncTitle: 'Synchronisation manuelle',
  manualSyncButton: '↻ Synchroniser tous les joueurs',
  manualSyncRunning: 'Synchronisation en cours…',
  manualSyncDescription: 'Lance une synchronisation delta immédiate pour tous les joueurs configurés.',

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
  discordNotifyNewVersion: 'Notifier les nouvelles versions',
  discordNotifyFriends: 'Notifier les ajouts d\'amis et le reclassement des parties en escouade',
  discordNoWebhook:
    "Les notifications Discord sont activées mais aucun webhook URL n'est configuré.",

  mediaTitle: 'Médias',
  mediaWatcherEnabled: 'Surveillance automatique des médias',
  mediaToleranceLabel: 'Tolérance association (min)',
  mediaNoBaseDir: "La surveillance des médias est activée mais aucun dossier source n'est défini.",
  mediaBaseDirLabel: 'Dossier des captures',
  mediaBaseDirPlaceholder: 'Ex : C:\\Users\\Moi\\Videos\\Captures ou /mnt/captures',
  mediaBaseDirHint: 'Sous-dossiers par gamertag attendus : {chemin}/{gamertag}/',
  mediaDeleteSource: 'Supprimer les originaux après conversion HLS',
  mediaDeleteSourceHint:
    "Recommandé sur le serveur (espace disque limité). À désactiver en local pour conserver les fichiers d'enregistrement d'origine.",
  mediaScanButton: 'Indexer les médias',
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
  watcherAuthReconnect: 'Rafraîchir Xbox',
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
  watcherStateIdle: 'Absent',
  watcherStateWatching: 'En surveillance',
  watcherStateSyncing: 'Synchronisation',
  watcherStateCooling: 'Cooldown',
  watcherInGame: 'En jeu',
  watcherPresenceOnline: 'En ligne',
  watcherPresenceAway: 'Absent',
  watcherPresenceOffline: 'Hors-ligne',
  watcherPresenceUnknown: '—',
  watcherTitleXboxDashboard: "l'accueil Xbox",
  watcherLastSeenRelative: 'Vu il y a {duration} sur {title}',
  watcherLastSeenAbsolute: 'Vu le {date} sur {title}',
  watcherNeverSeen: 'Jamais vu en jeu',

  backfillTitle: 'Recalcul rétroactif',
  backfillMedals: 'Médailles',
  backfillSkill: 'Classement (CSR/MMR)',
  backfillAliases: 'Alias gamertag',
  backfillPersonalScores: 'Scores personnels',
  backfillPerfScores: 'Scores performance',
  backfillLUSR: 'LUSR',
  backfillCSR: 'CSR par match (re-fetch API)',
  backfillEvents: 'Événements',
  backfillWeapons: 'Armes',
  backfillEngagementScores: "Score d'engagement",
  backfillEngagementCoefficients: "Coefficients d'engagement (recalcul rapide)",
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
      '« Amis seulement » : seules les arrivées/départs des joueurs de votre liste « Mon escouade » sont observés. Idéal si vous jouez souvent avec des inconnus, mais que votre noyau d\'amis reste stable toute la soirée (défaut).',
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

  progressionTitle: 'Progression long-terme',
  showProgressionLabel: 'Afficher Objectifs & Prestige',
  progressionHint:
    "Les Objectifs sont des défis personnalisés (frags, FDA, précision…) sur une fenêtre " +
    "temporelle ; chaque objectif complété rapporte des Points de Prestige qui déterminent " +
    "votre palier (Normal, Héroïque, Légendaire, Mythique). Désactiver masque la section " +
    "Prestige sur l'Accueil et l'entrée Objectifs dans la barre de navigation.",
  progressionGlossaryLink: 'En savoir plus dans le glossaire',

  coachProactiveTitle: 'Coach proactif',
  coachProactiveLabel: 'Activer les suggestions du coach',
  coachProactiveHint:
    "Quand activé, le coach propose des objectifs et des arcs Prestige calibrés sur vos " +
    "tendances récentes (LOWESS positive, near-miss records, patterns de combat). Les " +
    "propositions apparaissent dans le centre de notifications avec des boutons Accepter / " +
    "Ignorer. Vous gardez la main : aucune création automatique. Opt-in (désactivé par défaut).",

  rendementTitle: 'Rendement combat',
  rendementExcludeAssistsLabel: 'Calculer le rendement sans les assistances',
  rendementExcludeAssistsHint:
    "Par défaut, le rendement (conversion offensive) compte chaque assistance comme 1/3 " +
    "d'élimination (convention Halo). Activez cette option pour ignorer totalement les " +
    "assistances : le rendement devient 225 × éliminations / dégâts. S'applique partout " +
    "(Accueil, Timeseries, Sessions, Explorer, Escouade, Match view). Désactivé par défaut.",


  accessibilityTitle: 'Accessibilité visuelle',
  accessibilityDescription: 'Choisissez une palette de couleurs adaptée à votre vision. Plusieurs palettes optimisées pour le daltonisme sont disponibles.',
  paletteLabel: 'Palette de couleurs',
  paletteDefault: 'Standard (défaut)',
  paletteDefaultDesc: 'Palette originale de LevelUp.',
  paletteOkabeIto: 'Okabe-Ito (daltonisme)',
  paletteOkabeItoDesc: 'Palette universellement lisible, distinguable en cas de deutéranopie, protanopie et tritanopie.',
  paletteCividis: 'Cividis (séquentiel CVD)',
  paletteCividisDesc: 'Palette perceptuellement uniforme conçue pour la déficience visuelle des couleurs (PLOS ONE 2018). Idéale pour heatmaps et gradients.',
  paletteTolBright: 'Tol Bright (catégoriel)',
  paletteTolBrightDesc: 'Palette catégorielle 7 couleurs optimisée daltonisme par Paul Tol (SRON 2018). Recommandée pour les graphes multi-séries.',
  previewLabel: 'Aperçu',

  teamColorsTitle: 'Couleurs de jeu',
  teamColorsDescription: 'Choisissez les mêmes couleurs que dans Halo Infinite (Paramètres › Gameplay & Accessibilité). Les graphes utiliseront ces couleurs pour distinguer votre équipe des adversaires.',
  allyColorLabel: 'Couleur alliés',
  enemyColorLabel: 'Couleur ennemis',
  teamColorDefault: 'Défaut palette',

  tabBackup: 'Sauvegarde',
  backupTitle: 'Sauvegarde des bases DuckDB',
  backupStatusEnabled: 'Activée',
  backupStatusDisabled: 'Désactivée',
  backupStatusResticMissing: 'Restic introuvable',
  backupLastBackup: 'Dernière sauvegarde',
  backupNever: 'Jamais sauvegardé',
  backupSnapshotId: 'Snapshot',
  backupDatabases: 'Bases sauvegardées',
  backupDuration: 'Durée',
  backupRunButton: 'Sauvegarder maintenant',
  backupRunning: 'Sauvegarde en cours…',
  backupRunDone: 'Sauvegarde terminée',
  backupRunSkipped: 'Aucune modification — cycle ignoré',
  backupRunError: 'Erreur lors de la sauvegarde',
  backupConfigTitle: 'Configuration',
  backupConfigInterval: 'Intervalle',
  backupConfigRetention: 'Rétention',
  backupConfigRetentionValue: '{daily}j / {weekly}s / {monthly}m',
  backupIntegrityLabel: 'Intégrité',
}

const EN_TEXT: SettingsText = {
  pageTitle: 'Settings',
  pageSubtitle: 'Application configuration',
  savedStatus: '✓ Saved',
  errorStatus: '✗ Save failed',
  loading: 'Loading settings…',

  tabGeneral: 'General',
  tabTitles: 'Games',
  titlesSectionTitle: 'Tracked games',
  titlesSectionDesc:
    'Enable or pause syncing per game. Pausing keeps the data; at least one game must stay active.',
  titleStatusActive: 'Active',
  titleStatusPaused: 'Paused',
  titleStatusNotTracked: 'Not tracked',
  titlePurgeButton: 'Purge',
  titlePurgeConfirm: 'Permanently delete this game’s data for this player?',
  titleLastActiveHint: 'At least one game must stay active.',
  titlePurgeResidualWarning: 'Game disabled, but some data files could not be removed.',
  titleActionError: 'The operation failed. Please retry.',
  tabSync: 'Synchronisation',
  tabAnalyse: 'Analysis',
  tabAppearance: 'Appearance & Accessibility',
  tabData: 'Data & Media',
  tabAccount: 'Account',
  tabNotifications: 'Notifications',

  manualSyncTitle: 'Manual synchronisation',
  manualSyncButton: '↻ Synchronise all players',
  manualSyncRunning: 'Synchronisation running…',
  manualSyncDescription: 'Trigger an immediate delta sync for all configured players.',

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
  discordNotifyNewVersion: 'Notify on new version',
  discordNotifyFriends: 'Notify on friend adds and squad re-classification of shared matches',
  discordNoWebhook:
    'Discord notifications are enabled but no webhook URL is configured.',

  mediaTitle: 'Media',
  mediaWatcherEnabled: 'Automatic media watcher',
  mediaToleranceLabel: 'Association tolerance (min)',
  mediaNoBaseDir: 'Media watcher is enabled but no source folder is defined.',
  mediaBaseDirLabel: 'Captures folder',
  mediaBaseDirPlaceholder: 'e.g. C:\\Users\\Me\\Videos\\Captures or /mnt/captures',
  mediaBaseDirHint: 'Subfolders by gamertag expected: {path}/{gamertag}/',
  mediaDeleteSource: 'Delete originals after HLS conversion',
  mediaDeleteSourceHint:
    'Recommended on the server (limited disk space). Disable locally to keep the original recording files.',
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
  watcherAuthReconnect: 'Refresh Xbox',
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
  watcherStateIdle: 'Away',
  watcherStateWatching: 'Watching',
  watcherStateSyncing: 'Syncing',
  watcherStateCooling: 'Cooling',
  watcherInGame: 'In game',
  watcherPresenceOnline: 'Online',
  watcherPresenceAway: 'Away',
  watcherPresenceOffline: 'Offline',
  watcherPresenceUnknown: '—',
  watcherTitleXboxDashboard: 'the Xbox home',
  watcherLastSeenRelative: 'Seen {duration} ago on {title}',
  watcherLastSeenAbsolute: 'Last seen on {date} playing {title}',
  watcherNeverSeen: 'Never seen in game',

  backfillTitle: 'Backfill',
  backfillMedals: 'Medals',
  backfillSkill: 'Ranking (CSR/MMR)',
  backfillAliases: 'Gamertag aliases',
  backfillPersonalScores: 'Personal scores',
  backfillPerfScores: 'Performance scores',
  backfillLUSR: 'LUSR',
  backfillCSR: 'Per-match CSR (API re-fetch)',
  backfillEvents: 'Events',
  backfillWeapons: 'Weapons',
  backfillEngagementScores: 'Engagement score',
  backfillEngagementCoefficients: 'Engagement coefficients (fast recompute)',
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

  progressionTitle: 'Long-term progression',
  showProgressionLabel: 'Show Objectives & Prestige',
  progressionHint:
    'Objectives are custom challenges (kills, KDA, accuracy…) over a time window; ' +
    'each completed objective awards Prestige Points that determine your tier ' +
    '(Normal, Heroic, Legendary, Mythic). Disabling hides the Prestige section on ' +
    'the Home page and the Objectives entry in the navigation bar.',
  progressionGlossaryLink: 'Learn more in the glossary',

  coachProactiveTitle: 'Proactive coach',
  coachProactiveLabel: 'Enable coach suggestions',
  coachProactiveHint:
    'When enabled, the coach proposes objectives and Prestige arcs calibrated on your ' +
    'recent trends (positive LOWESS, near-miss records, combat patterns). Suggestions ' +
    'appear in the notification center with Accept / Dismiss buttons. You stay in ' +
    'control: nothing is created automatically. Opt-in (off by default).',

  rendementTitle: 'Combat yield',
  rendementExcludeAssistsLabel: 'Compute yield without assists',
  rendementExcludeAssistsHint:
    'By default, yield (offensive conversion) counts each assist as 1/3 of a kill (Halo ' +
    'convention). Enable this to ignore assists entirely: yield becomes 225 × kills / ' +
    'damage. Applies everywhere (Home, Timeseries, Sessions, Explorer, Squad, Match view). ' +
    'Off by default.',


  accessibilityTitle: 'Visual accessibility',
  accessibilityDescription: 'Choose a colour palette suited to your vision. Several palettes optimised for colour-blindness are available.',
  paletteLabel: 'Colour palette',
  paletteDefault: 'Standard (default)',
  paletteDefaultDesc: 'Original LevelUp palette.',
  paletteOkabeIto: 'Okabe-Ito (colour-blind safe)',
  paletteOkabeItoDesc: 'Universally readable, distinguishable under deuteranopia, protanopia and tritanopia.',
  paletteCividis: 'Cividis (CVD sequential)',
  paletteCividisDesc: 'Perceptually uniform palette designed for colour vision deficiency (PLOS ONE 2018). Ideal for heatmaps and gradients.',
  paletteTolBright: 'Tol Bright (categorical)',
  paletteTolBrightDesc: 'Categorical 7-colour palette optimised for colour-blindness by Paul Tol (SRON 2018). Recommended for multi-series charts.',
  previewLabel: 'Preview',

  teamColorsTitle: 'In-game colours',
  teamColorsDescription: 'Match the colours you use in Halo Infinite (Settings › Gameplay & Accessibility). Charts will use these colours to distinguish your team from opponents.',
  allyColorLabel: 'Ally colour',
  enemyColorLabel: 'Enemy colour',
  teamColorDefault: 'Palette default',

  tabBackup: 'Backup',
  backupTitle: 'DuckDB backup',
  backupStatusEnabled: 'Enabled',
  backupStatusDisabled: 'Disabled',
  backupStatusResticMissing: 'Restic not found',
  backupLastBackup: 'Last backup',
  backupNever: 'Never backed up',
  backupSnapshotId: 'Snapshot',
  backupDatabases: 'Databases backed up',
  backupDuration: 'Duration',
  backupRunButton: 'Back up now',
  backupRunning: 'Backup in progress…',
  backupRunDone: 'Backup complete',
  backupRunSkipped: 'No changes — cycle skipped',
  backupRunError: 'Backup error',
  backupConfigTitle: 'Configuration',
  backupConfigInterval: 'Interval',
  backupConfigRetention: 'Retention',
  backupConfigRetentionValue: '{daily}d / {weekly}w / {monthly}m',
  backupIntegrityLabel: 'Integrity',
}

const TEXT: Record<Locale, SettingsText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function normalizeSettingsLocale(locale?: string | null): Locale {
  return locale === 'en' ? 'en' : 'fr'
}

export function getSettingsText(locale?: string | null): SettingsText {
  return TEXT[normalizeSettingsLocale(locale)]
}
