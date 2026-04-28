/**
 * Query keys TanStack Query — centralisés pour éviter les doublons.
 *
 * Convention :
 * - Clés en tableau hiérarchique : ['bootstrap'], ['players'], ['player', slug, ...]
 * - Fonctions pour les clés paramétrées : queryKeys.player(slug)
 */

export const queryKeys = {
  bootstrap: ['bootstrap'] as const,
  players: ['players'] as const,
  sessionContext: ['session-context'] as const,
  health: ['health'] as const,

  // Setup & auth (Slice 1)
  // setupStatus supprimé (sprint 29) : GET /setup/status n'existe ni en FastAPI ni en Go
  deviceFlow: (attemptId: string) => ['device-flow', attemptId] as const,
  job: (jobId: string) => ['job', jobId] as const,
  settings: ['settings'] as const,
  labContracts: ['lab', 'contracts'] as const,
  labDiagnostics: ['lab', 'diagnostics'] as const,
  labResources: (requestHash: string) => ['lab', 'resources', requestHash] as const,

  // Par joueur
  player: (playerSlug: string) => ['player', playerSlug] as const,
  filtersResolve: (playerSlug: string, filterHash: string) =>
    ['filters-resolve', playerSlug, filterHash] as const,

  // Carrière (Slice 2)
  career: (playerSlug: string) => ['career', playerSlug] as const,
  careerTopMatches: (playerSlug: string) => ['career', playerSlug, 'top-matches'] as const,
  careerEncounters: (playerSlug: string) => ['career', playerSlug, 'encounters'] as const,

  // Historique des parties (Slice 3)
  matchHistory: (playerSlug: string, filterHash: string, page: number, soloSessions: string[] = []) =>
    ['match-history', playerSlug, filterHash, page, [...soloSessions].sort().join(',')] as const,

  // Explorer (Slice 4)
  explorer: (playerSlug: string, filterHash: string) =>
    ['explorer', playerSlug, filterHash] as const,
  gamertagSearch: (q: string) => ['gamertag-search', q] as const,
  matchView: (playerSlug: string, matchId: string) =>
    ['match-view', playerSlug, matchId] as const,

  // Accueil / Home (Slice 5)
  home: (playerSlug: string) => ['home', playerSlug] as const,
  battlepass: (playerSlug: string) => ['home', playerSlug, 'battlepass'] as const,

  // Palmares
  seasonPass: (playerSlug: string) => ['palmares', playerSlug, 'season-pass'] as const,
  palmaresRelations: (playerSlug: string) => ['palmares', playerSlug, 'relations'] as const,

  // Escouade / Teammates (Slice 6)
  teammates: (playerSlug: string, filterHash: string, selectedGts: string[], sessionLabels: string[] = []) =>
    ['teammates', playerSlug, filterHash, [...selectedGts].sort().join(','), [...sessionLabels].sort().join(',')] as const,

  // Synthèse (Slice 7 — Sprint 55 D8 : scopeHash = period + filtres)
  synthesis: (playerSlug: string, scopeHash: string) =>
    ['synthesis', playerSlug, scopeHash] as const,

  // Médias (Slice 8)
  mediaBase: (playerSlug: string) => ['media', playerSlug] as const,
  media: (playerSlug: string, requestHash: string) => ['media', playerSlug, requestHash] as const,
  mediaRail: (playerSlug: string, limit: number, likedOnly = false) => ['media', playerSlug, 'rail', limit, likedOnly] as const,
  mediaAuthors: (playerSlug: string) => ['media', playerSlug, 'authors'] as const,
  feedVersion: ['media', 'feed-version'] as const,

  // Citations (Slice 2B)
  citations: (playerSlug: string, filterHash: string) =>
    ['citations', playerSlug, filterHash] as const,

  // Timeseries (Slice 3B)
  timeseries: (playerSlug: string, filterHash: string) =>
    ['timeseries', playerSlug, filterHash] as const,

  // Session Compare (Slice 3C)
  sessionCompare: (playerSlug: string, filterHash: string, sessionA: string, sessionB: string) =>
    ['session-compare', playerSlug, filterHash, sessionA, sessionB] as const,

  // Session Detail (session page revamp)
  sessionDetail: (
    playerSlug: string,
    filterHash: string,
    sessionLabel: string,
    compareSessionLabel: string,
    enableCompare: boolean,
  ) => ['session-detail', playerSlug, filterHash, sessionLabel, compareSessionLabel, enableCompare] as const,

  // Compare joueur vs joueur (Sprint 54-C)
  comparePlayer: (playerSlug: string, targetGamertag: string) =>
    ['compare', playerSlug, targetGamertag] as const,

  // Navigation prev/next entre matchs (V7)
  matchNeighbors: (playerSlug: string, matchId: string) =>
    ['match-neighbors', playerSlug, matchId] as const,

  // Classement CSR (Sprint 54-E)
  leaderboard: (playerSlug: string, season?: string, playlist?: string) =>
    ['leaderboard', playerSlug, season ?? '', playlist ?? ''] as const,

  // Notifications in-app (per-player)
  notifications: (playerSlug: string, filter: object) =>
    ['notifications', playerSlug, 'list', filter] as const,
  notificationsUnreadCount: (playerSlug: string) =>
    ['notifications', playerSlug, 'unread-count'] as const,
  notificationsPreferences: (playerSlug: string) =>
    ['notifications', playerSlug, 'preferences'] as const,
} as const
