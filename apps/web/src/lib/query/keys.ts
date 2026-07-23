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

  // Sélection par titre (réglages Jeux) — statut sync par titre du joueur courant.
  playerTitles: (playerSlug: string) => ['player-titles', playerSlug] as const,

  // Setup & auth (Slice 1)
  // setupStatus supprimé (sprint 29) : GET /setup/status n'existe ni en FastAPI ni en Go
  deviceFlow: (attemptId: string) => ['device-flow', attemptId] as const,
  job: (jobId: string) => ['job', jobId] as const,
  settings: ['settings'] as const,
  // Groupes/familles (accès mutuel) — gestion end-user
  groups: ['groups'] as const,
  labDiagnostics: ['lab', 'diagnostics'] as const,

  // Par joueur
  player: (playerSlug: string) => ['player', playerSlug] as const,
  filtersResolve: (playerSlug: string, filterHash: string) =>
    ['filters-resolve', playerSlug, filterHash] as const,
  filtersPreview: (playerSlug: string, filterHash: string) =>
    ['filters-preview', playerSlug, filterHash] as const,

  // Carrière (Slice 2)
  career: (playerSlug: string) => ['career', playerSlug] as const,
  careerTopMatches: (playerSlug: string) => ['career', playerSlug, 'top-matches'] as const,
  careerEncounters: (playerSlug: string) => ['career', playerSlug, 'encounters'] as const,
  careerHighlightMatches: (playerSlug: string, filtersKey = '') =>
    ['career', playerSlug, 'highlight-matches', filtersKey] as const,
  careerTopEncounters: (playerSlug: string) => ['career', playerSlug, 'top-encounters'] as const,
  careerRivals: (playerSlug: string) => ['career', playerSlug, 'rivals'] as const,
  careerCSRs: (playerSlug: string, season?: string) =>
    ['career', playerSlug, 'csrs', season ?? ''] as const,

  // Achievements Xbox (bilingues EN/FR, statiques après backfill)
  achievements: (playerSlug: string) => ['achievements', playerSlug] as const,

  // Historique des matchs (Slice 3)
  matchHistory: (playerSlug: string, filterHash: string, page: number, soloSessions: string[] = []) =>
    ['match-history', playerSlug, filterHash, page, [...soloSessions].sort().join(',')] as const,
  /** Préfixe broad — invalide tout le cache match-history d'un joueur. */
  matchHistoryAll: (playerSlug: string) => ['match-history', playerSlug] as const,

  // Explorer (Slice 4)
  explorer: (
    playerSlug: string,
    filterHash: string,
    perfTiers: number[] = [],
    skillTiers: string[] = [],
    rankedContext = '',
    outcomeFilter: number[] = [],
    matchFiltersKey = '',
  ) =>
    [
      'explorer', playerSlug, filterHash,
      [...perfTiers].sort().join(','),
      [...skillTiers].sort().join(','),
      rankedContext,
      [...outcomeFilter].sort().join(','),
      matchFiltersKey,
    ] as const,
  explorerPlayer: (playerSlug: string, targetGamertag: string, targetXuid: string, page: number) =>
    ['explorer-player', playerSlug, targetGamertag, targetXuid, page] as const,
  gamertagSearch: (q: string) => ['gamertag-search', q] as const,
  matchView: (playerSlug: string, matchId: string) =>
    ['match-view', playerSlug, matchId] as const,

  // Engagement (Phase 4 plan engagement)
  engagementMatch: (playerSlug: string, matchId: string) =>
    ['engagement', 'match', playerSlug, matchId] as const,
  engagementProfile: (playerSlug: string) =>
    ['engagement', 'profile', playerSlug] as const,
  engagementTimeseries: (playerSlug: string, filterHash: string, limit: number) =>
    ['engagement', 'timeseries', playerSlug, filterHash, limit] as const,
  engagementSquadSession: (playerSlug: string, matchIds: string[], teammates: string[]) =>
    ['engagement', 'squad-session', playerSlug, matchIds.join(','), teammates.join(',')] as const,

  // Accueil / Home (Slice 5)
  // Le titre courant fait partie de la clé : la réponse /pages/home est
  // spécifique au titre (Spartan ID, playlists récentes, rangs). Sans lui, un
  // switch de titre servait les données périmées du titre précédent (la clé ne
  // changeant pas, TanStack Query réutilisait le cache pendant le staleTime).
  // La LOCALE fait aussi partie de la clé : le backend baque les libellés
  // localisés (titres de défis, noms de map/mode) dans le payload selon le header
  // X-LevelUp-Locale à l'instant du fetch. Sans la locale dans la clé, un switch
  // de langue laissait le cache (y compris le fetch background prefetch/poll)
  // baké dans l'ancienne langue — invalidation naturelle à la bascule.
  home: (playerSlug: string, titleSlug: string, locale: string) =>
    ['home', playerSlug, titleSlug, locale] as const,

  // Palmares
  // Locale dans la clé : mêmes libellés backend-bakés (nom du pass, titres de
  // défis de saison) selon X-LevelUp-Locale — cf. commentaire `home` ci-dessus.
  seasonPass: (playerSlug: string, locale: string) =>
    ['palmares', playerSlug, 'season-pass', locale] as const,
  palmaresRelations: (playerSlug: string) => ['palmares', playerSlug, 'relations'] as const,

  // Escouade / Teammates (Slice 6)
  teammates: (playerSlug: string, filterHash: string, selectedGts: string[], sessionLabels: string[] = [], locale = '') =>
    ['teammates', playerSlug, filterHash, [...selectedGts].sort().join(','), [...sessionLabels].sort().join(','), locale] as const,
  /** Préfixe broad — invalide toutes les queries teammates (ex. après ajout d'ami). */
  teammatesAll: ['teammates'] as const,

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

  // Totaux à vie des commendations natives (Halo 5, AXE B)
  commendationTotals: (playerSlug: string) =>
    ['commendation-totals', playerSlug] as const,

  // Timeseries (Slice 3B) — 'solo' dans la clé pour invalider tout cache pré-fix
  timeseries: (playerSlug: string, filterHash: string) =>
    ['timeseries', 'solo', playerSlug, filterHash] as const,

  // Session Detail (session page revamp)
  sessionDetail: (
    playerSlug: string,
    filterHash: string,
    sessionLabel: string,
    compareSessionLabel: string,
    enableCompare: boolean,
    locale: string,
  ) =>
    ['session-detail', playerSlug, filterHash, sessionLabel, compareSessionLabel, enableCompare, locale] as const,

  // Compare joueur vs joueur (Sprint 54-C)
  comparePlayer: (playerSlug: string, targetGamertag: string) =>
    ['compare', playerSlug, targetGamertag] as const,

  // Navigation prev/next entre matchs (V7).
  // Phase 2b : spec optionnel pour différencier les caches global vs filtré.
  matchNeighbors: (
    playerSlug: string,
    matchId: string,
    spec?: Record<string, unknown> | null,
  ) => ['match-neighbors', playerSlug, matchId, spec ?? null] as const,

  // Classement (CSR mondial + stats communautaires)
  leaderboard: (playerSlug: string, category?: string, season?: string, playlist?: string) =>
    ['leaderboard', playerSlug, category ?? '', season ?? '', playlist ?? ''] as const,
  leaderboardCatalog: (playerSlug: string) => ['leaderboard-catalog', playerSlug] as const,

  // Notifications in-app (per-player)
  /** Préfixe broad — invalide/matche toutes les queries notifications d'un joueur. */
  notificationsAll: (playerSlug: string) => ['notifications', playerSlug] as const,
  notifications: (playerSlug: string, filter: object) =>
    ['notifications', playerSlug, 'list', filter] as const,
  notificationsUnreadCount: (playerSlug: string) =>
    ['notifications', playerSlug, 'unread-count'] as const,
  notificationsPreferences: (playerSlug: string) =>
    ['notifications', playerSlug, 'preferences'] as const,

  // Asset Drawer (Phase 2)
  assetMaps: (titleSlug: string, q: string) => ['assets', titleSlug, 'maps', q] as const,
  assetWeapons: (titleSlug: string, q: string) => ['assets', titleSlug, 'weapons', q] as const,
  assetMedals: (titleSlug: string, q: string) => ['assets', titleSlug, 'medals', q] as const,

  // Progression V2 (Ascension) — cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1
  progressionStreaks: (playerSlug: string) =>
    ['progression', playerSlug, 'streaks'] as const,
  progressionRecords: (playerSlug: string, historyLimit?: number) =>
    ['progression', playerSlug, 'records', historyLimit ?? 50] as const,
  progressionMilestones: (playerSlug: string) =>
    ['progression', playerSlug, 'milestones'] as const,
  // Pattern Engine (phases 0.2-3)
  progressionProfile: (playerSlug: string, windowDays = 30) =>
    ['progression', playerSlug, 'profile', windowDays] as const,
  progressionPatterns: (playerSlug: string, n = 50) =>
    ['progression', playerSlug, 'patterns', n] as const,
  // Calendrier d'activité (Réalisations) — jours joués sur la fenêtre (DEC-5/D3).
  progressionActivity: (playerSlug: string, days = 90) =>
    ['progression', playerSlug, 'activity', days] as const,

  // Coach Advisor proposals (ADR 0020 Phase 10)
  coachProposals: (playerSlug: string, status?: string) =>
    ['coach', playerSlug, 'proposals', status ?? 'all'] as const,
  /** Préfixe broad — invalide toutes les queries coach d'un joueur. */
  coachAll: (playerSlug: string) => ['coach', playerSlug] as const,

  // Admin — Intégrité des données (invariants sync, plan SYNC_INVARIANTS_GATE)
  adminInvariants: ['admin', 'invariants'] as const,
  // Admin — Contention DB (B-swap shared) + santé des tokens auth
  adminDbContention: ['admin', 'db-contention'] as const,
  adminTokenHealth: ['admin', 'token-health'] as const,
  // Admin — Dashboard monitoring (overview agrégé, scheduler + historique,
  // jobs récents du JobStore, convergence, qualité données)
  adminMonitoringOverview: ['admin', 'monitoring', 'overview'] as const,
  adminMonitoringScheduler: ['admin', 'monitoring', 'scheduler'] as const,
  adminMonitoringJobs: ['admin', 'monitoring', 'jobs'] as const,
  adminMonitoringConvergence: ['admin', 'monitoring', 'convergence'] as const,
  adminMonitoringPerf: ['admin', 'monitoring', 'perf'] as const,
  adminMonitoringErrors: ['admin', 'monitoring', 'errors'] as const,
  adminMonitoringDetections: ['admin', 'monitoring', 'detections'] as const,
  adminMonitoringFreshness: ['admin', 'monitoring', 'freshness'] as const,
  adminMonitoringResources: ['admin', 'monitoring', 'resources'] as const,
  adminMonitoringCrons: ['admin', 'monitoring', 'crons'] as const,
  adminWeaponCoverage: (slug: string) => ['admin', 'monitoring', 'weapon-coverage', slug] as const,
  adminLusrGaps: (slug: string) => ['admin', 'monitoring', 'lusr-gaps', slug] as const,
  adminDataQuality: ['admin', 'data-quality', 'counts'] as const,
  adminDataQualityIssues: (kind: string) => ['admin', 'data-quality', 'issues', kind] as const,
  adminLogModules: ['admin', 'logs', 'modules'] as const,
  adminLogTail: (module: string, level: string, contains: string, limit: number) =>
    ['admin', 'logs', 'tail', module, level, contains, limit] as const,
  // Admin — Titres (PMT-14 volet A : gestion multi-titres)
  adminTitles: ['admin', 'titles'] as const,
  adminTitleDetail: (slug: string) => ['admin', 'titles', slug] as const,
  adminTitleDiagnostic: (slug: string) => ['admin', 'titles', slug, 'diagnostic'] as const,
  // Admin — Gestion des utilisateurs (ex-adminKeys, L5)
  adminUsers: ['admin', 'users'] as const,

  // Prestige / Ascension — registres feature centralisés (L5, CLAUDE.md n°13).
  // Tableaux de clés IDENTIQUES aux ex-registres (prestigeKeys/arcKeys/…) → aucun
  // changement de comportement de cache, juste un point d'accès unique.
  prestige: {
    me: (userId: string, titleSlug?: string) =>
      ['prestige', 'me', userId, titleSlug] as const,
    /** Préfixe broad — invalide `me(userId, *)` pour tous les titres. */
    meAll: (userId: string) => ['prestige', 'me', userId] as const,
    templates: (userId: string, titleSlug: string) =>
      ['prestige', 'templates', userId, titleSlug] as const,
    /** Clé de mutation du mode pilote (enable/disable auto-attribution, B3). */
    pilotMode: (userId: string, titleSlug: string) =>
      ['prestige', 'pilot-mode', userId, titleSlug] as const,
  },
  arc: {
    list: (userId: string, titleSlug: string) =>
      ['prestige', 'arcs', userId, titleSlug] as const,
    one: (id: string) => ['prestige', 'arc', id] as const,
    presets: (userId: string, titleSlug: string) =>
      ['prestige', 'arc-presets', userId, titleSlug] as const,
  },
  challenge: {
    list: (userId: string, titleSlug: string) =>
      ['prestige', 'challenges', userId, titleSlug] as const,
    /** Défis terminaux (historique Réalisations) — statuts distincts de `list`
     *  (actifs), donc clé distincte. */
    history: (userId: string, titleSlug: string) =>
      ['prestige', 'challenges', 'history', userId, titleSlug] as const,
    one: (id: string) => ['prestige', 'challenge', id] as const,
  },
  squad: {
    mine: (userId: string) => ['prestige', 'squads', userId] as const,
    challenges: (squadId: string) => ['prestige', 'squad-challenges', squadId] as const,
    orientation: (squadId: string, requestedBy: string) =>
      ['prestige', 'squad-orientation', squadId, requestedBy] as const,
  },
  playerProfile: {
    profile: (playerSlug: string, windowDays: number) =>
      ['playerProfile', playerSlug, windowDays] as const,
    activeCampaign: (playerSlug: string) =>
      ['playerProfile', 'campaign', 'active', playerSlug] as const,
    campaign: (playerSlug: string, id: string) =>
      ['playerProfile', 'campaign', playerSlug, id] as const,
    /** Campagnes closes (historique Réalisations). Sous le préfixe `campaignAll`
     *  → invalidée par les mutations de campagne (close/abandon refont l'historique). */
    campaignHistory: (playerSlug: string) =>
      ['playerProfile', 'campaign', playerSlug, 'history'] as const,
    /** Préfixe broad — invalide tous les `campaign(playerSlug, *)`. */
    campaignAll: (playerSlug: string) =>
      ['playerProfile', 'campaign', playerSlug] as const,
  },
  watcher: {
    status: ['watcher', 'status'] as const,
    authPoll: (attemptId: string) => ['watcher', 'auth', attemptId] as const,
  },

  // Clés feature diverses ex-inline (L5, CLAUDE.md n°13) — centralisées ici.
  changelog: ['changelog'] as const,
  releaseNotes: (lang: string) => ['release-notes', lang] as const,
  feedbackSimilarIssues: (query: string) =>
    ['feedback-drawer', 'similar-issues', query] as const,
  mediaMatchCandidates: (
    playerSlug: string,
    // `filePath` peut être null tant que la sélection média n'est pas faite ;
    // le hook associé est alors désactivé (`enabled: !!filePath`) et ne fetch
    // jamais avec cette valeur. La forme de clé pour un filePath non-null reste
    // byte-identique (rien inséré/retiré) — pas d'invalidation de cache.
    filePath: string | null,
    windowMinutes: number,
  ) => ['media', 'match-candidates', playerSlug, filePath, windowMinutes] as const,
  /** Préfixe broad — invalide tous les `filtersResolve(playerSlug, *)`. */
  filtersResolveAll: (playerSlug: string) => ['filters-resolve', playerSlug] as const,
  /** Préfixe broad — invalide tous les `adminDataQualityIssues(*)`. */
  adminDataQualityIssuesAll: ['admin', 'data-quality', 'issues'] as const,
} as const
