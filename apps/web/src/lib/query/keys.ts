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
 * Garde-rail : `keys.locale.test.ts` classe CHAQUE fabrique (dépend de la langue / non, avec la raison)
 * et échoue sur une fabrique non classée ou dont le paramètre de langue ne correspond pas à son classement.
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
  // Amis d'un joueur (liste par profil, pas un réglage d'instance).
  playerFriends: (playerSlug: string) => ['player-friends', playerSlug] as const,

  // Par joueur (titleSlug en 2e segment — invariant structurel, cf. en-tête).
  // Le titre courant scope la clé (même motif que `home` ci-dessous) : la
  // résolution de filtres est spécifique au titre (sessions / options cascade du
  // titre actif). Sans lui, un switch de titre — ou une réponse mise en cache
  // avant que le header titre soit affirmé au boot (fenêtre de course « titre
  // implicite ») — servait des options périmées de l'autre titre, la clé ne
  // changeant pas. Défense en profondeur du chantier D7 (titre dans l'URL) — cf.
  // §7 PLAN_TITLE_SLUG_URL.
  // `matchContext` ('solo' | 'squad' | 'all') : même filterContext, populations
  // différentes — le store escouade résout en 'squad' (lot perf L4a, D4.3,
  // 2026-09-23), le solo sans contexte (= 'all' côté serveur). Sans ce segment,
  // deux stores au même hash partageraient une entrée de cache.
  // Les libellés des options playlist / mode / carte sont servis dans la langue de la requête (la
  // valeur reste le nom de la langue par défaut) : la locale est le dernier segment des deux clés.
  filtersResolve: (playerSlug: string, titleSlug: string, filterHash: string, matchContext: string, locale: string) =>
    ['filters-resolve', playerSlug, titleSlug, filterHash, matchContext, locale] as const,
  filtersPreview: (playerSlug: string, titleSlug: string, filterHash: string, locale: string) =>
    ['filters-preview', playerSlug, titleSlug, filterHash, locale] as const,

  // Carrière (Slice 2)
  // Prefixe large : tout le domaine Carriere d un joueur (invalidation apres une
  // ecriture de sa liste d amis : les rencontres « hors amis » sont filtrees serveur).
  careerAll: (playerSlug: string) => ['career', playerSlug] as const,
  // Page Carrière : les noms de rang (prochain rang, rang sommet) sont servis dans la langue de la
  // requête, la locale est le dernier segment.
  career: (playerSlug: string, titleSlug: string, locale: string) =>
    ['career', playerSlug, titleSlug, locale] as const,
  careerEncounters: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'encounters'] as const,
  // Cartes, modes et playlists des lignes et des listes de filtres servis dans la langue de la requête.
  careerHighlightMatches: (playerSlug: string, titleSlug: string, filtersKey = '', locale = '') =>
    ['career', playerSlug, titleSlug, 'highlight-matches', filtersKey, locale] as const,
  careerTopEncounters: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'top-encounters'] as const,
  careerRivals: (playerSlug: string, titleSlug: string) =>
    ['career', playerSlug, titleSlug, 'rivals'] as const,
  // Noms de playlist et libellés de saison servis dans la langue de la requête.
  careerCSRs: (playerSlug: string, titleSlug: string, locale: string, season?: string) =>
    ['career', playerSlug, titleSlug, 'csrs', locale, season ?? ''] as const,

  // Achievements Xbox (statiques après backfill) — par titre (jeu) ; noms et descriptions servis dans la
  // langue de la requête : la locale est dans la clé.
  achievements: (playerSlug: string, titleSlug: string, locale: string) =>
    ['achievements', playerSlug, titleSlug, locale] as const,

  // Historique des matchs (Slice 3)
  // Cartes, modes et playlists des lignes et des options de filtre servis dans la langue de la requête.
  matchHistory: (
    playerSlug: string,
    titleSlug: string,
    filterHash: string,
    page: number,
    soloSessions: string[] = [],
    locale = '',
  ) =>
    ['match-history', playerSlug, titleSlug, filterHash, page, [...soloSessions].sort().join(','), locale] as const,
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
    locale = '',
  ) =>
    [
      'explorer', playerSlug, titleSlug, filterHash,
      [...perfTiers].sort().join(','),
      [...skillTiers].sort().join(','),
      rankedContext,
      [...outcomeFilter].sort().join(','),
      matchFiltersKey,
      locale,
    ] as const,
  // Profil de la cible : top armes et répartition des frags nommés dans la langue de la requête.
  explorerPlayer: (
    playerSlug: string,
    titleSlug: string,
    targetGamertag: string,
    targetXuid: string,
    page: number,
    locale: string,
  ) => ['explorer-player', playerSlug, titleSlug, targetGamertag, targetXuid, page, locale] as const,
  // gamertagSearch : recherche globale (Xbox), title-agnostic — pas de titleSlug.
  // `live` distingue le repli Xbox explicite (V72-24) de la recherche locale rapide.
  gamertagSearch: (q: string, live = false) => ['gamertag-search', q, live] as const,
  // Vue du match : noms de cartes, modes, armes, médailles... servis dans la langue de la requête.
  matchView: (playerSlug: string, titleSlug: string, matchId: string, locale: string) =>
    ['match-view', playerSlug, titleSlug, matchId, locale] as const,
  matchObjectiveEvents: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-objective-events', playerSlug, titleSlug, matchId] as const,
  matchPositions: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-positions', playerSlug, titleSlug, matchId] as const,
  matchReplay: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay', playerSlug, titleSlug, matchId] as const,
  // Contexte de match de la page de rejeu (en-tête, tableau de score, fil, médias) : une forme
  // à lui, donc une clé à lui — jamais servie depuis le cache de la vue complète (`matchView`).
  matchReplayContext: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay-context', playerSlug, titleSlug, matchId] as const,
  // Fond de carte du rejeu : clé DISTINCTE de l'artefact. Les deux ne s'invalident pas
  // ensemble — l'artefact est propre au match, le fond est propre à la carte et ne change
  // qu'à une re-cuisson.
  matchReplayBackground: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay-background', playerSlug, titleSlug, matchId] as const,
  // L'IMAGE a sa propre cle : elle pese jusqu'a 1,4 Mio et ne se charge qu'apres le calage.
  matchReplayBackgroundImage: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay-background-image', playerSlug, titleSlug, matchId] as const,
  // Zones nommées (callouts) : même logique que le fond — propres à la CARTE, servies par
  // match, figées entre deux régénérations du catalogue versionné.
  matchReplayCallouts: (playerSlug: string, titleSlug: string, matchId: string) =>
    ['match-replay-callouts', playerSlug, titleSlug, matchId] as const,

  // Engagement (Phase 4 plan engagement)
  // Le nom de la carte des matchs (engagementMatch, engagementTimeseries, engagementSquadSession) est servi
  // dans la langue de la requête : la locale est le dernier segment.
  engagementMatch: (playerSlug: string, titleSlug: string, matchId: string, locale = '') =>
    ['engagement', 'match', playerSlug, titleSlug, matchId, locale] as const,
  engagementProfile: (playerSlug: string, titleSlug: string) =>
    ['engagement', 'profile', playerSlug, titleSlug] as const,
  engagementTimeseries: (playerSlug: string, titleSlug: string, filterHash: string, limit: number, locale = '') =>
    ['engagement', 'timeseries', playerSlug, titleSlug, filterHash, limit, locale] as const,
  engagementSquadSession: (
    playerSlug: string,
    titleSlug: string,
    matchIds: string[],
    teammates: string[],
    locale = '',
  ) =>
    ['engagement', 'squad-session', playerSlug, titleSlug, matchIds.join(','), teammates.join(','), locale] as const,

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
  // Prefixe large : toutes les locales et tous les titres de l accueil d un joueur
  // (tuiles escouade filtrees serveur sur la liste d amis).
  homeAll: (playerSlug: string) => ['home', playerSlug] as const,
  home: (playerSlug: string, titleSlug: string, locale: string) =>
    ['home', playerSlug, titleSlug, locale] as const,
  // Pages de tuiles de l'Accueil (GET /pages/home/matches) : sous le préfixe `home` (toute
  // invalidation de l'accueil les recharge), une clé PAR PAGE — le curseur `before` est celui de la
  // page précédente, donc une tête qui change (synchro) change les clés au lieu de laisser des pages
  // périmées se recoller à la nouvelle tête. Locale : les libellés de tuile sont bakés serveur.
  homeMatchesPage: (playerSlug: string, titleSlug: string, locale: string, before: string) =>
    ['home', playerSlug, titleSlug, locale, 'matches', before] as const,
  // Favoris de l'accueil (même route, kind=favorites) : liste rechargée seule après un favori basculé.
  homeFavorites: (playerSlug: string, titleSlug: string, locale: string) =>
    ['home', playerSlug, titleSlug, locale, 'favorites'] as const,

  // Palmares
  // Locale dans la clé : mêmes libellés backend-bakés (nom du pass, titres de
  // défis de saison) selon X-LevelUp-Locale — cf. commentaire `home` ci-dessus.
  seasonPass: (playerSlug: string, titleSlug: string, locale: string) =>
    ['palmares', playerSlug, titleSlug, 'season-pass', locale] as const,
  // Relations / social (followers) — stockage shared_social PAR TITRE.
  // Le nom de la carte de chaque duel est servi dans la langue de la requête.
  palmaresRelations: (playerSlug: string, titleSlug: string, locale = '') =>
    ['palmares', playerSlug, titleSlug, 'relations', locale] as const,

  // Escouade / Teammates (Slice 6)
  // exactComposition : l'option « composition exacte » change la POPULATION servie
  // (matchs commencés ensemble vs composition exclusive) — sans elle dans la clé,
  // le cache resservirait les nombres de l'autre réglage.
  // Pas de segment « sessions pickées » (retiré le 2026-09-23, lot perf L4a, D4.1) :
  // la session pickée vit UNIQUEMENT dans le store escouade
  // (`filterContext.sessions.picked_sessions`), déjà couvert par `filterHash`.
  // Deux segments pour une même information faisaient d'un snap ou d'un clic du
  // rail DEUX clés successives, donc une requête intermédiaire jamais affichée
  // (mesurée : 8,2 s + 26,8 s par clic).
  teammates: (playerSlug: string, titleSlug: string, filterHash: string, selectedGts: string[], locale = '', exactComposition = true) =>
    ['teammates', playerSlug, titleSlug, filterHash, [...selectedGts].sort().join(','), locale, exactComposition] as const,
  // Sessions de la composition SANS la page (GET /pages/teammates/sessions, lot perf L4b,
  // 2026-09-23). Sous le préfixe `teammates` : toute invalidation de l'Escouade la couvre
  // (un ami ajouté entre dans l'extraPool de la composition exacte). Ni filtres ni locale :
  // la réponse se calcule sur l'historique COMPLET de la composition et ne porte aucun
  // libellé traduit.
  compositionSessions: (playerSlug: string, titleSlug: string, selectedGts: string[], exactComposition: boolean) =>
    ['teammates', playerSlug, titleSlug, 'composition-sessions', [...selectedGts].sort().join(','), exactComposition] as const,
  // Top des coéquipiers des matchs avec amis (GET /pages/squad sans coéquipier) : la source de
  // la liste de l'Escouade, lue par les sélecteurs de composition hors Escouade. Sous le préfixe
  // 'teammates' : un ami ajouté l'invalide avec le reste.
  // La page Escouade sert ses noms d'armes et de cartes dans la langue de la requête.
  teammatesTop: (playerSlug: string, titleSlug: string, locale: string) =>
    ['teammates', playerSlug, titleSlug, 'top', locale] as const,
  /** Préfixe broad — invalide toutes les queries teammates (ex. après ajout d'ami).
   *  Title-agnostic PAR DESIGN (balaie tous les joueurs/titres). */
  teammatesAll: ['teammates'] as const,

  // Onglet Tactique (Ascension) — la grille des cartes jouées.
  // `filterHash` : le filtre courant entre dans la clé parce qu'il change la POPULATION
  // servie (quelles cartes, et combien de matchs sur chacune). Sans lui, changer de
  // période resservirait la grille de la période précédente.
  // Les noms de cartes sont servis dans la langue de la requête : la locale est le dernier segment
  // de la grille, du raster (noms des grappes) et du détail de cellule (zone, arme).
  tacticalMaps: (playerSlug: string, titleSlug: string, filterHash: string, locale: string) =>
    ['tactical-maps', playerSlug, titleSlug, filterHash, locale] as const,
  // Le PÉRIMÈTRE de l'onglet : les match_id que la barre L2 fait résoudre par
  // /filters/match-ids. Clé DISTINCTE de la grille (2026-09-06) parce que la
  // résolution est partagée par toutes les lectures de l'onglet — la grille
  // aujourd'hui, la vue par carte demain — et qu'une seule requête doit la servir.
  tacticalMatchIDs: (playerSlug: string, titleSlug: string, filterHash: string) =>
    ['tactical-match-ids', playerSlug, titleSlug, filterHash] as const,
  // Le FOND d'une carte : propre à la CARTE, indépendant du filtre, figé entre deux
  // cuissons — d'où une clé distincte de la grille (même raison que
  // `matchReplayBackgroundImage` vis-à-vis de `matchReplay`), et un staleTime infini.
  //
  // SANS `playerSlug`, et c'est délibéré (revue R1, W4) : l'image d'une carte est une
  // donnée de RÉFÉRENCE du titre, identique pour tout le monde. La clé par joueur retenait
  // N images par joueur consulté dans la session, pour exactement le même contenu.
  // `titleSlug` reste en 1er segment — une carte n'existe que dans son titre — et l'URL de
  // fetch garde le joueur (la route est derrière l'ownership).
  tacticalMapBackground: (titleSlug: string, mapId: string) =>
    ['tactical-map-background', titleSlug, mapId] as const,
  /** Calage monde du fond d'une carte — même donnée de référence que l'image, donc même
   *  régime : versionnée par titre, jamais par joueur. */
  tacticalMapBackgroundFrame: (titleSlug: string, mapId: string) =>
    ['tactical', 'map-background-frame', titleSlug, mapId] as const,

  // Raster de placement pour UNE carte et UNE question.
  tacticalRaster: (playerSlug: string, titleSlug: string, mapId: string, paramHash: string, locale: string) =>
    ['tactical-raster', playerSlug, titleSlug, mapId, paramHash, locale] as const,
  // Détail d'UNE cellule (lien « voir dans le rejeu », lot M1) : mêmes raisons que
  // `tacticalRaster` (une carte n'existe que dans son titre), clé DISTINCTE parce que le
  // périmètre inclut en plus l'adresse de la cellule cliquée.
  tacticalCellule: (playerSlug: string, titleSlug: string, mapId: string, paramHash: string, locale: string) =>
    ['tactical-cellule', playerSlug, titleSlug, mapId, paramHash, locale] as const,
  // Séquence d'UNE mort (vue D) : l'empreinte porte le match, la victime et l'instant.
  tacticalSequence: (playerSlug: string, titleSlug: string, mapId: string, paramHash: string) =>
    ['tactical-sequence', playerSlug, titleSlug, mapId, paramHash] as const,
  // Synthèse (Slice 7 — Sprint 55 D8 : scopeHash = period + filtres)
  // Noms d armes (frags, portée, records) et de cartes servis dans la langue de la requête.
  synthesis: (playerSlug: string, titleSlug: string, scopeHash: string, locale: string) =>
    ['synthesis', playerSlug, titleSlug, scopeHash, locale] as const,

  // Médias (Slice 8)
  /** Préfixe broad — matche toutes les queries média du joueur (patch like/upload).
   *  Reste un préfixe `['media', playerSlug]` : matche les clés enrichies du titre. */
  mediaBase: (playerSlug: string) => ['media', playerSlug] as const,
  // Noms de carte et de mode des médias servis dans la langue de la requête (locale dans la clé).
  media: (playerSlug: string, titleSlug: string, requestHash: string, locale: string) =>
    ['media', playerSlug, titleSlug, requestHash, locale] as const,
  mediaRail: (
    playerSlug: string,
    titleSlug: string,
    limit: number,
    locale: string,
    likedOnly = false,
  ) => ['media', playerSlug, titleSlug, 'rail', limit, locale, likedOnly] as const,
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
  // Noms de cartes et d armes servis dans la langue de la requête : la locale est le dernier segment.
  timeseries: (playerSlug: string, titleSlug: string, filterHash: string, locale: string) =>
    ['timeseries', 'solo', playerSlug, titleSlug, filterHash, locale] as const,

  // Tendances (Solo) — titre en 2e segment ; vue, type de partie, locale, composition
  // (gamertags triés, joints par une virgule ; vide en vue Solo) et option stricte : une autre
  // vue, un autre type, une autre langue de médailles, une autre escouade ou une autre règle
  // de composition est une autre réponse.
  trends: (
    playerSlug: string,
    titleSlug: string,
    view: string,
    gameType: string,
    locale: string,
    composition: string,
    exactComposition: boolean,
  ) =>
    ['trends', playerSlug, titleSlug, view, gameType, locale, composition, exactComposition] as const,

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
  // Noms d armes du profil d armes servis dans la langue de la requête.
  comparePlayer: (playerSlug: string, titleSlug: string, targetGamertag: string, locale: string) =>
    ['compare', playerSlug, titleSlug, targetGamertag, locale] as const,

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
  /** La liste porte des paramètres résolus par le serveur dans la langue de la requête (`title`,
   *  `medal_name`) : la locale entre dans la clé. */
  notifications: (playerSlug: string, titleSlug: string, locale: string, filter: object) =>
    ['notifications', playerSlug, titleSlug, 'list', locale, filter] as const,
  notificationsUnreadCount: (playerSlug: string, titleSlug: string) =>
    ['notifications', playerSlug, titleSlug, 'unread-count'] as const,
  notificationsPreferences: (playerSlug: string, titleSlug: string) =>
    ['notifications', playerSlug, titleSlug, 'preferences'] as const,

  // Capabilities DATA-LEVEL du titre (GET /titles/{slug}/capabilities) — « ce titre
  // produit-il cette donnee ? ». Title-scopee par son PREMIER argument, comme les cles
  // asset* : la reponse EST celle d'un titre, il n'y a pas de dimension joueur.
  // Cf. lib/capabilities/dataCapabilities.ts (regle des deux portes).
  titleDataCapabilities: (titleSlug: string) => ['title-data-capabilities', titleSlug] as const,

  // Asset Drawer (Phase 2)
  // Noms et descriptions servis dans la langue de la requête : la locale est le dernier segment.
  assetMaps: (titleSlug: string, q: string, locale: string) => ['assets', titleSlug, 'maps', q, locale] as const,
  assetWeapons: (titleSlug: string, q: string, locale: string) =>
    ['assets', titleSlug, 'weapons', q, locale] as const,
  assetMedals: (titleSlug: string, q: string, locale: string) =>
    ['assets', titleSlug, 'medals', q, locale] as const,

  // Progression V2 (Ascension) — cf. PLAN_PROGRESSION_TRACKING_ASCENSION.md §8.1.
  // Dérivée des matchs du titre → titleSlug scope la clé.
  progressionStreaks: (playerSlug: string, titleSlug: string) =>
    ['progression', playerSlug, titleSlug, 'streaks'] as const,
  progressionRecords: (playerSlug: string, titleSlug: string, historyLimit?: number) =>
    ['progression', playerSlug, titleSlug, 'records', historyLimit ?? 50] as const,
  // Jalons : titres et conditions servis dans la langue de la requête (locale dans la clé).
  progressionMilestones: (playerSlug: string, titleSlug: string, locale: string) =>
    ['progression', playerSlug, titleSlug, 'milestones', locale] as const,
  // Pattern Engine (phases 0.2-3)
  // Le profil porte des libellés servis dans la langue de la requête (tier, défis suggérés).
  progressionProfile: (playerSlug: string, titleSlug: string, locale: string, windowDays = 30) =>
    ['progression', playerSlug, titleSlug, 'profile', locale, windowDays] as const,
  // Les libellés des contextes (cartes, modes) sont servis dans la langue de la requête.
  progressionPatterns: (playerSlug: string, titleSlug: string, locale: string, n = 50) =>
    ['progression', playerSlug, titleSlug, 'patterns', locale, n] as const,
  // Calendrier d'activité (Réalisations) — jours joués sur la fenêtre (DEC-5/D3).
  progressionActivity: (playerSlug: string, titleSlug: string, days = 90) =>
    ['progression', playerSlug, titleSlug, 'activity', days] as const,

  // Coach Advisor proposals (ADR 0020 Phase 10) — dérivé des matchs du titre.
  // Le motif de chaque proposition est servi dans la langue de la requête (locale dans la clé).
  coachProposals: (playerSlug: string, titleSlug: string, locale: string, status?: string) =>
    ['coach', playerSlug, titleSlug, 'proposals', locale, status ?? 'all'] as const,
  /** Préfixe broad — invalide toutes les queries coach d'un joueur.
   *  Reste `['coach', playerSlug]` : préfixe valide des clés enrichies du titre. */
  coachAll: (playerSlug: string) => ['coach', playerSlug] as const,

  // Admin — Intégrité des données (invariants sync, plan SYNC_INVARIANTS_GATE)
  adminInvariants: ['admin', 'invariants'] as const,
  // Admin — Contention DB (B-swap shared) + santé des tokens auth
  adminDbContention: ['admin', 'db-contention'] as const,
  adminTokenHealth: ['admin', 'token-health'] as const,
  // Admin — Annuaire des joueurs (ADR 0035) : les 4 registres d'identité lus
  // ensemble par xuid (compte, profils, identifiants, suivi live).
  adminIdentities: ['admin', 'identities'] as const,
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
  adminMonitoringBuildQueue: ['admin', 'monitoring', 'build-queue'] as const,
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
  adminInvites: ['admin', 'invites'] as const,
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
    /** Libellés des modèles servis dans la langue de la requête : locale dans la clé. */
    templates: (userId: string, titleSlug: string, locale: string) =>
      ['prestige', 'templates', userId, titleSlug, locale] as const,
    /** Clé de mutation du mode pilote (enable/disable auto-attribution, B3). */
    pilotMode: (userId: string, titleSlug: string) =>
      ['prestige', 'pilot-mode', userId, titleSlug] as const,
  },
  arc: {
    /** Titre et description d'un arc adopté depuis un preset sont servis dans la langue de la
     *  requête (résolus à la lecture depuis le preset) : locale dans la clé. */
    list: (userId: string, titleSlug: string, locale: string) =>
      ['prestige', 'arcs', userId, titleSlug, locale] as const,
    /** Préfixe broad — invalide les arcs du joueur dans toutes les langues. */
    listAll: (userId: string, titleSlug: string) =>
      ['prestige', 'arcs', userId, titleSlug] as const,
    one: (id: string) => ['prestige', 'arc', id] as const,
    /** Titres et descriptions des presets servis dans la langue de la requête. */
    presets: (userId: string, titleSlug: string, locale: string) =>
      ['prestige', 'arc-presets', userId, titleSlug, locale] as const,
  },
  challenge: {
    /** Le libellé d'un défi issu d'un modèle est servi dans la langue de la requête (résolu à la
     *  lecture depuis le modèle) : locale dans la clé. */
    list: (userId: string, titleSlug: string, locale: string) =>
      ['prestige', 'challenges', userId, titleSlug, locale] as const,
    /** Préfixe broad — invalide les défis actifs du joueur dans toutes les langues. */
    listAll: (userId: string, titleSlug: string) =>
      ['prestige', 'challenges', userId, titleSlug] as const,
    /** Défis terminaux (historique Réalisations) — statuts distincts de `list`
     *  (actifs), donc clé distincte. Libellés servis dans la langue de la requête. */
    history: (userId: string, titleSlug: string, locale: string) =>
      ['prestige', 'challenges', 'history', userId, titleSlug, locale] as const,
    one: (id: string) => ['prestige', 'challenge', id] as const,
  },
  squad: {
    mine: (userId: string) => ['prestige', 'squads', userId] as const,
    /** Le libellé de chaque défi est servi dans la langue de la requête : locale dans la clé. */
    challenges: (squadId: string, locale: string) =>
      ['prestige', 'squad-challenges', squadId, locale] as const,
    /** Préfixe broad — invalide les défis de l'escouade dans toutes les langues. */
    challengesAll: (squadId: string) => ['prestige', 'squad-challenges', squadId] as const,
    orientation: (squadId: string, requestedBy: string) =>
      ['prestige', 'squad-orientation', squadId, requestedBy] as const,
  },
  playerProfile: {
    /** Le profil porte des libellés servis dans la langue de la requête : locale dans la clé. */
    profile: (playerSlug: string, titleSlug: string, locale: string, windowDays: number) =>
      ['playerProfile', playerSlug, titleSlug, locale, windowDays] as const,
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

  /** Présence en jeu du shell (joueurs suivis + amis). Title-scopée : la réponse
   *  ne liste que les joueurs du titre courant — sans le slug, un switch de titre
   *  laisserait la manette sur le sélecteur de l'autre titre le temps du refetch. */
  presence: (titleSlug: string) => ['presence', titleSlug] as const,

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
    // Cartes, modes et playlists des suggestions servis dans la langue de la requête.
    locale: string,
  ) => ['media', 'match-candidates', playerSlug, titleSlug, filePath, windowMinutes, locale] as const,
  /** Préfixe broad — invalide tous les `filtersResolve(playerSlug, *, *, *)` (solo ET escouade).
   *  RESTE broad PAR JOUEUR (n'inclut PAS le titre) : son unique usage
   *  (invalidation post-sync, $playerSlug.tsx) doit rafraîchir la résolution du
   *  joueur, et le préfixe `['filters-resolve', playerSlug]` matche toujours la clé
   *  enrichie du titre. Décision orchestrateur — cf. §7 PLAN_TITLE_SLUG_URL. */
  filtersResolveAll: (playerSlug: string) => ['filters-resolve', playerSlug] as const,
  /** Préfixe broad — invalide tous les `adminDataQualityIssues(*)`. */
  adminDataQualityIssuesAll: ['admin', 'data-quality', 'issues'] as const,
} as const
