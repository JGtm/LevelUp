/** Textes de la feature Paramètres en français (contrat : `contract.ts`). */
import type { SettingsText } from './contract'

export const FR_TEXT = {
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

  groupsCardTitle: 'Amis et groupes',
  groupsCardDescription:
    "Gère la liste d'amis de ton profil (elle marque tes parties en escouade) et tes groupes : les membres d'un même groupe ont un accès mutuel à leurs profils et statistiques.",
  groupsCardOpen: 'Gérer mes amis et groupes',

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
  refreshClearsCaches: "Vider les caches à l'actualisation",
  showWaypointColumn: 'Colonne Halo Waypoint sur les listes de matchs',
  showWaypointColumnHint:
    "Ajoute un lien direct vers la page du match sur Halo Waypoint dans les tableaux de matchs. Masquée automatiquement si le titre courant ne l'expose pas.",

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
  watcherAuthCodeCopied: 'Code copié',
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
  watcherDurationUnderMin: "moins d'1 min",
  watcherDurationHours: '{n} h',
  watcherDurationDayOne: '{n} j',
  watcherDurationDayMany: '{n} j',
  demoModeNotice: 'Mode démo : les paramètres sont figés. Seules la langue et l’accessibilité sont modifiables (le temps de la session).',

  backfillTitle: 'Recalcul rétroactif',
  backfillMedals: 'Médailles',
  backfillSkill: 'Classement (CSR/MMR)',
  backfillAliases: 'Alias gamertag',
  backfillPersonalScores: 'Scores personnels',
  backfillPerfScores: 'Scores performance',
  backfillLUSR: 'LUSR',
  backfillCSR: 'CSR par match (re-fetch API)',
  backfillEvents: 'Événements',
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
  syncToastBusy: 'Une autre synchronisation de ce joueur est déjà en cours. Réessaie dans un instant.',
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

  // Onglet Analyse — Rejeu 2D
  replayRetentionTitle: 'Rejeu 2D',
  replayRetentionLabel: 'Fenêtre de conservation des rejeux',
  replayRetentionUnit: 'mois',
  replayRetentionHint:
    'Les rejeux 2D des matchs plus anciens que cette fenêtre sont purgés, et les nouveaux ' +
    'matchs hors fenêtre ne sont pas construits. 0 = illimité (tout conserver). ' +
    'Les films téléchargés ne sont jamais supprimés.',
  replayLocationLabel: 'Lieu de construction',
  replayLocationAuto: 'Automatique',
  replayLocationLocal: 'Ce serveur',
  replayLocationWorker: 'Ouvrier distant',
  replayLocationOff: 'Aucune construction',
  replayLocationHint:
    'Décoder un film coûte environ une minute de processeur. « Ce serveur » construit sur ' +
    'place (développement uniquement) ; « Ouvrier distant » met le travail en file, une autre ' +
    'machine le prend et renvoie le rejeu ; « Aucune construction » se contente des rejeux ' +
    'déjà présents. Automatique choisit l\'ouvrier distant en production.',

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
  previewSquadPlayersLabel: 'Joueurs d\'escouade',
  previewOutcome: { win: 'victoire', loss: 'défaite', draw: 'égalité', dnf: 'abandon' },
  previewDivergent: { pos: 'positif', neutral: 'neutre', neg: 'négatif' },

  teamColorsTitle: 'Couleurs de jeu',
  teamColorsDescription: 'Choisissez les mêmes couleurs que dans Halo Infinite (Paramètres › Gameplay & Accessibilité). Les graphes utiliseront ces couleurs pour distinguer votre équipe des adversaires.',
  allyColorLabel: 'Couleur alliés',
  enemyColorLabel: 'Couleur ennemis',
  teamColorDefault: 'Défaut palette',

  replaySoundTitle: 'Sons du rejeu',
  replaySoundDescription:
    "Les sons d'armes sont extraits du jeu tels quels. Ces deux réglages rejouent ce que le moteur du jeu applique à chaque coup ; ils valent pour toute l'instance.",
  replaySoundVariationLabel: 'Variation',
  replaySoundVariationHint:
    "À chaque tir, le jeu déplace légèrement le volume et la hauteur du son. 100 % applique les écarts du jeu tels quels, 0 % joue toujours le fichier à l'identique.",
  replaySoundDistanceLabel: 'Distance',
  replaySoundDistanceHint:
    'Un tir lointain est plus faible et plus sourd. À 0 %, le son traverse la lecture sans aucun traitement.',
  replaySoundOff: 'Aucune',
  replaySoundFull: 'Maximum',

  tabBackup: 'Sauvegarde',
  backupStatusEnabled: 'Activée',
  backupStatusNotConfigured: 'Non configurées',
  backupStatusResticMissing: 'Restic introuvable',
  backupNotConfiguredHint:
    'Sauvegardes non configurées. Pour les activer : installer le binaire restic (sur le PATH), définir BACKUP_ENABLED=true et configurer RESTIC_REPOSITORY (+ RESTIC_PASSWORD) dans l\'environnement du serveur (.env.local). Voir docs/BACKUP_RESTORE.md.',
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
} satisfies SettingsText
