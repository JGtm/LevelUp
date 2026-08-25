/**
 * Query keys TanStack Query — centralisés pour éviter les doublons.
 *
 * Convention :
 * - Clés en tableau hiérarchique : ['bootstrap'], ['players'], ['career', slug, titre, ...]
 * - Fonctions pour les clés paramétrées : queryKeys.career(playerSlug, titleSlug)
 *
 * titleSlug dans la clé — INVARIANT STRUCTUREL (V72-29). TOUTE clé par-joueur dont le
 * payload dépend du titre porte `titleSlug` en 2e segment (juste après `playerSlug`).
 * Ne PAS s'appuyer sur le seul `queryClient.clear()` du switch (applyActiveTitle) pour
 * séparer les titres : une course (refetch background, réponse en vol au switch) re-clé
 * sinon des DONNÉES d'un autre titre sous la clé du titre courant (fuite cross-titre
 * corrigée le 2026-07 — cf. .ai/archive/V7/PLAN_TITLE_HEADER_LEAK_2026-07.md §Secondaire).
 * La clé EST la garde, indépendante du timing. Position imposée (2e segment) : les
 * préfixes larges d'invalidation `['x', playerSlug]` (ex. matchHistoryAll, mediaBase,
 * notificationsAll, coachAll) restent des préfixes valides des clés enrichies du titre —
 * leur sémantique « tout le joueur » est préservée (ils balaient désormais tous les
 * titres du joueur, ce qui reste correct pour un refresh post-sync).
 *
 * Garde-rail : `keys.title-slug.guard.test.ts` invoque CHAQUE fabrique par-joueur et
 * échoue si le slug n'apparaît pas dans la clé ; toute nouvelle fabrique non classée
 * (title-scopée vs agnostique documentée) y fait échouer la complétude.
 *
 * locale dans la clé — QUAND ? En plus du titre, si le serveur bake un libellé localisé
 * (header X-LevelUp-Locale) dans le payload (ex. `home`, `medals`, `seasonPass`). Sans
 * `locale` en clé, un fetch background survivant à la bascule de langue reste dans
 * l'ancienne langue. Modèle : `home` (playerSlug + titleSlug + locale).
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

  // Par joueur (titleSlug en 2e segment — invariant structurel, cf. en-tête).
  // Le titre courant scope la clé (même motif que `home` ci-dessous) : la
  // résolution de filtres est spécifique au titre (sessions / options cascade du
  // titre actif). Sans lui, un switch de titre — ou une réponse mise en cache
  // avant que le header titre soit affirmé au boot (fenêtre de course « titre
  // implicite ») — servait des options périmées de l'autre titre, la clé ne
  // changeant pas. Défense en profondeur du chantier D7 (titre dans l'URL) — cf.
  // §7 PLAN_TITLE_SLUG_URL.
  filtersResolve: (playerSlug: string, titleSlug: string, filterHash: string) =>
    ['filters-resolve', playerSlug, titleSlug, filterHash] as const,
  filtersPreview: (playerSlug: string, titleSlug: string, filterHash: string) =>
    ['filters-preview', playerSlug, titleSlug, filterHash] as const,

  // Carrière (Slice 2)
  career: (playerSlug: string, titleSlug: string) => ['career', playerSlug, titleSlug] as const,
  careerEncounters: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'encounters'] as const,
  careerHighlightMatches: (playerSlug: string, titleSlug: string, filtersKey = '') =>
    ['career', playerSlug, titleSlug, 'highlight-matches', filtersKey] as const,
  careerTopEncounters: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'top-encounters'] as const,
  careerRivals: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'rivals'] as const,
  careerCSRs: (playerSlug: string, titleSlug: string, season?: string) =>
    ['career', playerSlug, titleSlug, 'csrs', season ?? ''] as const,

  // Achievements Xbox (bilingues EN/FR, statiques après backfill) — par titre (jeu).
  achievements: (playerSlug: string, titleSlug: string) =>
    ['achievements', playerSlug, titleSlug] as const,

  // Historique des matchs (Slice 3)
  matchHistory: (playerSlug: string, titleSlug: string, filterHash: string, page: number, soloSessions: string[] = []) =>
    ['match-history', playerSlug, titleSlug, filterHash, page, [...soloSessions].sort().join(',')] as const,
  /** Préfixe broad — invalide tout le cache match-history d'un joueur. */
  matchHistoryAll: (playerSlug: string) => ['match-history', playerSlug] as const,

  // Explorer (Slice 4)
  explorer: (
    playerSlug: string,
    titleSlug: string,
    filterHash: string,
    perfTiers: number[] = [],
    skillTiers: string[] = [],
    rankedContext = '',
    outcomeFilter: number[] = [],
    matchFiltersKey = '',
  ) =>
    [
      'explorer', playerSlug, titleSlug, filterHash,
      [...perfTiers].sort().join(','),
      [...skillTiers].sort().join(','),
      rankedContext,
      [...outcomeFilter].sort().join(','),
      matchFiltersKey,
    ] as const,
  explorerPlayer: (playerSlug: string, titleSlug: string, targetGamertag: string, targetXuid: string, page: number) =>
    ['explorer-player', playerSlug, titleSlug, targetGamertag, targetXuid, page] as const,
  // gamertagSearch : recherche globale (Xbox), title-agnostic — pas de titleSlug.
  // `live` distingue le repli Xbox explicite (V72-24) de la recherche locale rapide.
  gamertagSearch: (q: string, live = false) => ['gamertag-search', q, live] as const,
  matchView: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-view', playerSlug, titleSlug, matchId] as const,
  matchObjectiveEvents: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-objective-events', playerSlug, titleSlug, matchId] as const,
  matchPositions: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-positions', playerSlug, titleSlug, matchId] as const,
  matchReplay: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay', playerSlug, titleSlug, matchId] as const,

  // Engagement (Phase 4 plan engagement)
  engagementMatch: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['engagement', 'match', playerSlug, titleSlug, matchId] as const,
  engagementProfile: (playerSlug: string, titleSlug: string) =>
    ['engagement', 'profile', playerSlug, titleSlug] as const,
  engagementTimeseries: (playerSlug: string, titleSlug: string, filterHash: string, limit: number) =>
    ['engagement', 'timeseries', playerSlug, titleSlug, filterHash, limit] as const,
  engagementSquadSession: (playerSlug: string, titleSlug: string, matchIds: string[], teammates: string[]) =>
    ['engagement', 'squad-session', playerSlug, titleSlug, matchIds.join(','), teammates.join(',')] as const,

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
  seasonPass: (playerSlug: string, titleSlug: string, locale: string) =>
    ['palmares', playerSlug, titleSlug, 'season-pass', locale] as const,
  // Relations / social (followers) — stockage shared_social PAR TITRE.
  palmaresRelations: (playerSlug: string, titleSlug: string) =>
    ['palmares', playerSlug, titleSlug, 'relations'] as const,

  // Escouade / Teammates (Slice 6)
  // exactComposition : l'option « composition exacte » change la POPULATION servie
  // (matchs commencés ensemble vs composition exclusive) — sans elle dans la clé,
  // le cache resservirait les nombres de l'autre réglage.
  teammates: (playerSlug: string, titleSlug: string, filterHash: string, selectedGts: string[], sessionLabels: string[] = [], locale = '', exactComposition = false) =>
    ['teammates', playerSlug, titleSlug, filterHash, [...selectedGts].sort().join(','), [...sessionLabels].sort().join(','), locale, exactComposition] as const,
  /** Préfixe broad — invalide toutes les queries teammates (ex. après ajout d'ami).
   *  Title-agnostic PAR DESIGN (balaie tous les joueurs/titres). */
  teammatesAll: ['teammates'] as const,

  // Synthèse (Slice 7 — Sprint 55 D8 : scopeHash = period + filtres)
  synthesis: (playerSlug: string, titleSlug: string, scopeHash: string) =>
    ['synthesis', playerSlug, titleSlug, scopeHash] as const,

  // Médias (Slice 8)
  /** Préfixe broad — matche toutes les queries média du joueur (patch like/upload).
   *  Reste un préfixe `['media', playerSlug]` : matche les clés enrichies du titre. */
  mediaBase: (playerSlug: string) => ['media', playerSlug] as const,
  media: (playerSlug: string, titleSlug: string, requestHash: string) =>
    ['media', playerSlug, titleSlug, requestHash] as const,
  mediaRail: (playerSlug: string, titleSlug: string, limit: number, likedOnly = false) =>
    ['media', playerSlug, titleSlug, 'rail', limit, likedOnly] as const,
  mediaAuthors: (playerSlug: string, titleSlug: string) =>
    ['media', playerSlug, titleSlug, 'authors'] as const,
  mediaAudioConfig: (playerSlug: string, titleSlug: string) =>
    ['media', playerSlug, titleSlug, 'audio-config'] as const,
  feedVersion: ['media', 'feed-version'] as const,

  // Citations (Slice 2B) — locale dans la clé : le backend bake les libellés de
  // citations localisés (X-LevelUp-Locale) dans le payload. Sans elle, un fetch
  // background survivant à la bascule de langue reste dans l'ancienne langue —
  // cf. commentaire `home` ci-dessus (même motif).
  citations: (playerSlug: string, titleSlug: string, filterHash: string, locale: string) =>
    ['citations', playerSlug, titleSlug, filterHash, locale] as const,

  // Médailles (sous-page Carrière) — locale dans la clé : le backend bake les
  // libellés/descriptions de médailles localisés (X-LevelUp-Locale). Pas de
  // filterHash : le filtre obtenues/non-obtenues + le tri sont 100% client.
  medals: (playerSlug: string, titleSlug: string, locale: string) =>
    ['medals', playerSlug, titleSlug, locale] as const,

  // Totaux à vie des commendations natives (Halo 5, AXE B) — par titre. Locale
  // dans la clé : le backend bake les libellés/descriptions de commendations
  // localisés (X-LevelUp-Locale) — cf. commentaire `home` ci-dessus.
  commendationTotals: (playerSlug: string, titleSlug: string, locale: string) =>
    ['commendation-totals', playerSlug, titleSlug, locale] as const,

  // Timeseries (Slice 3B) — 'solo' dans la clé pour invalider tout cache pré-fix
  timeseries: (playerSlug: string, titleSlug: string, filterHash: string) =>
    ['timeseries', 'solo', playerSlug, titleSlug, filterHash] as const,

  // Session Detail (session page revamp)
  sessionDetail: (
    playerSlug: string,
    titleSlug: string,
    filterHash: string,
    sessionLabel: string,
    compareSessionLabel: string,
    enableCompare: boolean,
    locale: string,
  ) =>
    ['session-detail', playerSlug, titleSlug, filterHash, sessionLabel, compareSessionLabel, enableCompare, locale] as const,

  // Compare joueur vs joueur (Sprint 54-C)
  comparePlayer: (playerSlug: string, titleSlug: string, targetGamertag: string) =>
    ['compare', playerSlug, titleSlug, targetGamertag] as const,

  // Navigation prev/next entre matchs (V7).
  // Phase 2b : spec optionnel pour différencier les caches global vs filtré.
  matchNeighbors: (
    playerSlug: string,
    titleSlug: string,
    matchId: string,
    spec?: Record<string, unknown> | null,
  ) => ['match-neighbors', playerSlug, titleSlug, matchId, spec ?? null] as const,

  // Classement (CSR mondial + stats communautaires)
  leaderboard: (playerSlug: string, titleSlug: string, category?: string, season?: string, playlist?: string) =>
    ['leaderboard', playerSlug, titleSlug, category ?? '', season ?? '', playlist ?? ''] as const,
  // Locale dans la clé : le catalogue bake les display_name (saisons/playlists) localisés
  // selon X-LevelUp-Locale — cf. commentaire `home` ci-dessus. La clé EST l'invalidation à
  // la bascule de langue (plus d'invalidation ciblée depuis le layout titre).
  leaderboardCatalog: (playerSlug: string, titleSlug: string, locale: string) =>
    ['leaderboard-catalog', playerSlug, titleSlug, locale] as const,

  // Notifications in-app (per-player) — stockage shared_social PAR TITRE (routes
  // notifications title-scopées /t/{slug}/…) → le titre scope les clés feuilles.
  /** Préfixe broad — invalide/matche toutes les queries notifications d'un joueur.
   *  Reste `['notifications', playerSlug]` : préfixe valide des clés enrichies du titre. */
  notificationsAll: (playerSlug: string) => ['notifications', playerSlug] as const,
  notifications: (playerSlug: string, titleSlug: string, filter: object) =>
    ['notifications', playerSlug, titleSlug, 'list', filter] as const,
  notificationsUnreadCount: (playerSlug: string, titleSlug: string) =>
    ['notifications', playerSlug, titleSlug, 'unread-count'] as const,
  notificationsPreferences: (playerSlug: string, titleSlug: string) =>
    ['notifications', playerSlug, titleSlug, 'preferences'] as const,

  // Asset Drawer (Phase 2)
  assetMaps: (titleSlug: string, q: string) => ['assets', titleSlug, 'maps', q] as const,
  assetWeapons: (titleSlug: string, q: string) => ['assets', titleSlug, 'weapons', q] as const,
  assetMedals: (titleSlug: string, q: string) => ['assets', titleSlug, 'medals', q] as const,

  // Progression V2 (Ascension) — cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1.
  // Dérivée des matchs du titre → titleSlug scope la clé.
  progressionStreaks: (playerSlug: string, titleSlug: string) =>
    ['progression', playerSlug, titleSlug, 'streaks'] as const,
  progressionRecords: (playerSlug: string, titleSlug: string, historyLimit?: number) =>
    ['progression', playerSlug, titleSlug, 'records', historyLimit ?? 50] as const,
  progressionMilestones: (playerSlug: string, titleSlug: string) =>
    ['progression', playerSlug, titleSlug, 'milestones'] as const,
  // Pattern Engine (phases 0.2-3)
  progressionProfile: (playerSlug: string, titleSlug: string, windowDays = 30) =>
    ['progression', playerSlug, titleSlug, 'profile', windowDays] as const,
  progressionPatterns: (playerSlug: string, titleSlug: string, n = 50) =>
    ['progression', playerSlug, titleSlug, 'patterns', n] as const,
  // Calendrier d'activité (Réalisations) — jours joués sur la fenêtre (DEC-5/D3).
  progressionActivity: (playerSlug: string, titleSlug: string, days = 90) =>
    ['progression', playerSlug, titleSlug, 'activity', days] as const,

  // Coach Advisor proposals (ADR 0020 Phase 10) — dérivé des matchs du titre.
  coachProposals: (playerSlug: string, titleSlug: string, status?: string) =>
    ['coach', playerSlug, titleSlug, 'proposals', status ?? 'all'] as const,
  /** Préfixe broad — invalide toutes les queries coach d'un joueur.
   *  Reste `['coach', playerSlug]` : préfixe valide des clés enrichies du titre. */
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
  adminMonitoringDetections: ['admin', 'monitoring', 'detections'] as const,
  adminMonitoringFreshness: ['admin', 'monitoring', 'freshness'] as const,
  adminMonitoringResources: ['admin', 'monitoring', 'resources'] as const,
  adminMonitoringCrons: ['admin', 'monitoring', 'crons'] as const,
  adminActionJournal: ['admin', 'actions', 'journal'] as const,
  adminWeaponCoverage: (slug: string) => ['admin', 'monitoring', 'weapon-coverage', slug] as const,
  adminLusrGaps: (slug: string) => ['admin', 'monitoring', 'lusr-gaps', slug] as const,
  adminDataQuality: ['admin', 'data-quality', 'counts'] as const,
  // limit/offset dans la clé : la pagination serveur des xuids orphelins met
  // chaque page en cache distincte (les autres kinds gardent limit=50/offset=0).
  adminDataQualityIssues: (kind: string, limit = 50, offset = 0) =>
    ['admin', 'data-quality', 'issues', kind, limit, offset] as const,
  adminLogModules: ['admin', 'logs', 'modules'] as const,
  adminLogTail: (module: string, level: string, contains: string, limit: number) =>
    ['admin', 'logs', 'tail', module, level, contains, limit] as const,
  // Admin — Titres (PMT-14 volet A : gestion multi-titres)
  adminTitles: ['admin', 'titles'] as const,
  adminTitleDetail: (slug: string) => ['admin', 'titles', slug] as const,
  adminTitleDiagnostic: (slug: string) => ['admin', 'titles', slug, 'diagnostic'] as const,
  // Admin — Gestion des utilisateurs (ex-adminKeys, L5)
  adminUsers: ['admin', 'users'] as const,
  // Admin — Diagnostic apparence Spartan ID (volet 2). MUTATION à la demande
  // (aucune query auto/refetch au focus) : clé stable pour l'identité/devtools.
  adminAppearanceDiagMutation: ['admin', 'diag', 'appearance'] as const,

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
    profile: (playerSlug: string, titleSlug: string, windowDays: number) =>
      ['playerProfile', playerSlug, titleSlug, windowDays] as const,
    activeCampaign: (playerSlug: string, titleSlug: string) =>
      ['playerProfile', 'campaign', 'active', playerSlug, titleSlug] as const,
    campaign: (playerSlug: string, titleSlug: string, id: string) =>
      ['playerProfile', 'campaign', playerSlug, titleSlug, id] as const,
    /** Campagnes closes (historique Réalisations). Sous le préfixe `campaignAll`
     *  → invalidée par les mutations de campagne (close/abandon refont l'historique). */
    campaignHistory: (playerSlug: string, titleSlug: string) =>
      ['playerProfile', 'campaign', playerSlug, titleSlug, 'history'] as const,
    /** Préfixe broad — invalide tous les `campaign(playerSlug, *)`.
     *  Reste `['playerProfile', 'campaign', playerSlug]` : préfixe des clés du titre. */
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
    titleSlug: string,
    // `filePath` peut être null tant que la sélection média n'est pas faite ;
    // le hook associé est alors désactivé (`enabled: !!filePath`) et ne fetch
    // jamais avec cette valeur. La forme de clé pour un filePath non-null reste
    // byte-identique (rien inséré/retiré) — pas d'invalidation de cache.
    filePath: string | null,
    windowMinutes: number,
  ) => ['media', 'match-candidates', playerSlug, titleSlug, filePath, windowMinutes] as const,
  /** Préfixe broad — invalide tous les `filtersResolve(playerSlug, *, *)`.
   *  RESTE broad PAR JOUEUR (n'inclut PAS le titre) : son unique usage
   *  (invalidation post-sync, $playerSlug.tsx) doit rafraîchir la résolution du
   *  joueur, et le préfixe `['filters-resolve', playerSlug]` matche toujours la clé
   *  enrichie du titre. Décision orchestrateur — cf. §7 PLAN_TITLE_SLUG_URL. */
  filtersResolveAll: (playerSlug: string) => ['filters-resolve', playerSlug] as const,
  /** Préfixe broad — invalide tous les `adminDataQualityIssues(*)`. */
  adminDataQualityIssuesAll: ['admin', 'data-quality', 'issues'] as const,
} as const
