/** Textes de la feature Paramètres en anglais (contrat : `contract.ts`). */
import type { SettingsText } from './contract'

export const EN_TEXT = {
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

  groupsCardTitle: 'Friends and groups',
  groupsCardDescription:
    "Manage your profile's friends list (it marks your squad games) and your groups: members of the same group share mutual access to their profiles and stats.",
  groupsCardOpen: 'Manage my friends and groups',

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
  refreshClearsCaches: 'Clear caches on refresh',
  showWaypointColumn: 'Halo Waypoint column on match lists',
  showWaypointColumnHint:
    "Adds a direct link to the match's Halo Waypoint page in match tables. Automatically hidden if the current title doesn't expose it.",

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
  watcherAuthCodeCopied: 'Code copied',
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
  watcherDurationUnderMin: 'less than 1 min',
  watcherDurationHours: '{n} hr',
  watcherDurationDayOne: '{n} day',
  watcherDurationDayMany: '{n} days',
  demoModeNotice: 'Demo mode: settings are frozen. Only language and accessibility can be changed (this session only).',

  backfillTitle: 'Backfill',
  backfillMedals: 'Medals',
  backfillSkill: 'Ranking (CSR/MMR)',
  backfillAliases: 'Gamertag aliases',
  backfillPersonalScores: 'Personal scores',
  backfillPerfScores: 'Performance scores',
  backfillLUSR: 'LUSR',
  backfillCSR: 'Per-match CSR (API re-fetch)',
  backfillEvents: 'Events',
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
  syncToastBusy: 'Another sync of this player is already running. Try again in a moment.',
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

  // Analyse tab — 2D replay
  replayRetentionTitle: '2D replay',
  replayRetentionLabel: 'Replay retention window',
  replayRetentionUnit: 'months',
  replayRetentionHint:
    '2D replays of matches older than this window are purged, and new matches outside the ' +
    'window are not built. 0 = unlimited (keep everything). ' +
    'Downloaded films are never deleted.',
  replayLocationLabel: 'Build location',
  replayLocationAuto: 'Automatic',
  replayLocationLocal: 'This server',
  replayLocationWorker: 'Remote worker',
  replayLocationOff: 'No building',
  replayLocationHint:
    'Decoding a film costs about a minute of CPU. "This server" builds in place (development ' +
    'only); "Remote worker" queues the work, another machine picks it up and sends the replay ' +
    'back; "No building" uses only the replays already there. Automatic picks the remote ' +
    'worker in production.',

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
  previewSquadPlayersLabel: 'Squad players',
  previewOutcome: { win: 'win', loss: 'loss', draw: 'draw', dnf: 'DNF' },
  previewDivergent: { pos: 'positive', neutral: 'neutral', neg: 'negative' },

  teamColorsTitle: 'In-game colours',
  teamColorsDescription: 'Match the colours you use in Halo Infinite (Settings › Gameplay & Accessibility). Charts will use these colours to distinguish your team from opponents.',
  allyColorLabel: 'Ally colour',
  enemyColorLabel: 'Enemy colour',
  teamColorDefault: 'Palette default',

  replaySoundTitle: 'Replay sounds',
  replaySoundDescription:
    'Weapon sounds are extracted from the game as they are. These two settings replay what the game engine applies on every shot; they apply to the whole instance.',
  replaySoundVariationLabel: 'Variation',
  replaySoundVariationHint:
    'On every shot the game shifts the volume and pitch slightly. 100 % applies the game ranges as they are, 0 % always plays the file identically.',
  replaySoundDistanceLabel: 'Distance',
  replaySoundDistanceHint:
    'A distant shot is quieter and duller. At 0 % the sound goes through playback untouched.',
  replaySoundOff: 'None',
  replaySoundFull: 'Maximum',

  tabBackup: 'Backup',
  backupStatusEnabled: 'Enabled',
  backupStatusNotConfigured: 'Not configured',
  backupStatusResticMissing: 'Restic not found',
  backupNotConfiguredHint:
    'Backups are not configured. To enable them: install the restic binary (on PATH), set BACKUP_ENABLED=true, and configure RESTIC_REPOSITORY (+ RESTIC_PASSWORD) in the server environment (.env.local). See docs/BACKUP_RESTORE.md.',
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
} satisfies SettingsText
