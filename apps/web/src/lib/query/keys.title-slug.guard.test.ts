/**
 * Garde-rail structurel (V72-29) — le titre scope TOUTE clé de requête title-dependante.
 *
 * Contexte : une fuite cross-titre (données Halo 5 servies sur Halo Infinite) venait de
 * clés par-joueur SANS `titleSlug` — seule la purge `queryClient.clear()` au switch les
 * séparait, donc toute course (refetch background, réponse en vol) re-exposait des données
 * d'un autre titre. Le correctif : `titleSlug` en 2e segment de chaque clé title-dependante
 * (cf. lib/query/keys.ts en-tête + .ai/archive/V7/PLAN_TITLE_HEADER_LEAK_2026-07.md).
 *
 * Ce test verrouille l'invariant en DEUX volets :
 *   1. INVOCATION : chaque fabrique title-scopée, appelée avec un slug sentinelle, DOIT
 *      produire une clé qui contient ce slug. Une fabrique qui « oublie » le slug échoue.
 *   2. COMPLÉTUDE : toute fabrique / constante de `queryKeys` (y compris les namespaces
 *      imbriqués) est classée EXACTEMENT une fois — title-scopée OU agnostique documentée.
 *      Une NOUVELLE fabrique non classée fait échouer le test : impossible d'ajouter une
 *      clé par-joueur sans décider (et documenter) si elle porte le titre.
 *
 * Ajouter une clé => l'ajouter ici (invocation title-scopée) OU dans l'allowlist agnostique
 * avec une justification (cross-titre par nature, préfixe large, global/admin).
 */
import { describe, it, expect } from 'vitest'
import { queryKeys } from './keys'

const P = '__PLAYER__'
const T = '__TITLE_SENTINEL__'
const U = '__USER__'

/**
 * Fabriques dont la clé DOIT contenir un slug de titre. Chaque entrée invoque la vraie
 * fabrique avec le slug sentinelle `T` à sa position réelle (2e segment pour les clés
 * par-joueur ; 1er argument pour les clés asset title-scopées). Le test échoue si `T`
 * n'apparaît pas dans la clé résultante.
 */
const titleScopedInvocations: Record<string, () => readonly unknown[]> = {
  // Filtres (déjà title-scopés avant V72-29 — verrouillés ici).
  filtersResolve: () => queryKeys.filtersResolve(P, T, 'h'),
  filtersPreview: () => queryKeys.filtersPreview(P, T, 'h'),
  // `player` retiré le 2026-07-25 : fabrique sans AUCUN call-site (enrichie du
  // titre « par cohérence » en V72-29, jamais consommée) — code mort supprimé,
  // règle CLAUDE.md n°7. Le volet COMPLÉTUDE ci-dessous (symétrie `stale`) fait
  // échouer ce test si un nom classé disparaît de queryKeys sans être retiré ici.
  // Carrière.
  career: () => queryKeys.career(P, T),
  careerEncounters: () => queryKeys.careerEncounters(P, T),
  careerHighlightMatches: () => queryKeys.careerHighlightMatches(P, T),
  careerTopEncounters: () => queryKeys.careerTopEncounters(P, T),
  careerRivals: () => queryKeys.careerRivals(P, T),
  careerCSRs: () => queryKeys.careerCSRs(P, T),
  achievements: () => queryKeys.achievements(P, T),
  // Matchs / explorer.
  matchHistory: () => queryKeys.matchHistory(P, T, 'h', 1),
  explorer: () => queryKeys.explorer(P, T, 'h'),
  explorerPlayer: () => queryKeys.explorerPlayer(P, T, 'gt', 'x', 1),
  matchView: () => queryKeys.matchView(P, T, 'm'),
  matchNeighbors: () => queryKeys.matchNeighbors(P, T, 'm'),
  // Calques décodés du film : un match_id ne vaut que dans son titre, et les trois
  // artefacts (timeline objectif, positions, rejeu 2D) sont produits par titre.
  matchObjectiveEvents: () => queryKeys.matchObjectiveEvents(P, T, 'm'),
  matchPositions: () => queryKeys.matchPositions(P, T, 'm'),
  matchReplay: () => queryKeys.matchReplay(P, T, 'm'),
  matchReplayBackground: () => queryKeys.matchReplayBackground(P, T, 'm'),
  matchReplayBackgroundImage: () => queryKeys.matchReplayBackgroundImage(P, T, 'm'),
  matchReplayCallouts: () => queryKeys.matchReplayCallouts(P, T, 'm'),
  // Engagement.
  engagementMatch: () => queryKeys.engagementMatch(P, T, 'm'),
  engagementProfile: () => queryKeys.engagementProfile(P, T),
  engagementTimeseries: () => queryKeys.engagementTimeseries(P, T, 'h', 50),
  engagementSquadSession: () => queryKeys.engagementSquadSession(P, T, [], []),
  // Accueil / palmarès / social.
  home: () => queryKeys.home(P, T, 'fr'),
  seasonPass: () => queryKeys.seasonPass(P, T, 'fr'),
  palmaresRelations: () => queryKeys.palmaresRelations(P, T),
  // Escouade / synthèse / sessions / compare.
  teammates: () => queryKeys.teammates(P, T, 'h', []),
  synthesis: () => queryKeys.synthesis(P, T, 'h'),
  sessionDetail: () => queryKeys.sessionDetail(P, T, 'h', 's', 'c', false, 'fr'),
  comparePlayer: () => queryKeys.comparePlayer(P, T, 'gt'),
  // Médias (feuilles ; mediaBase est un préfixe large -> agnostique).
  media: () => queryKeys.media(P, T, 'h'),
  mediaRail: () => queryKeys.mediaRail(P, T, 5),
  mediaAuthors: () => queryKeys.mediaAuthors(P, T),
  mediaAudioConfig: () => queryKeys.mediaAudioConfig(P, T),
  mediaMatchCandidates: () => queryKeys.mediaMatchCandidates(P, T, null, 5),
  // Citations / médailles / commendations / timeseries.
  citations: () => queryKeys.citations(P, T, 'h', 'fr'),
  medals: () => queryKeys.medals(P, T, 'fr'),
  commendationTotals: () => queryKeys.commendationTotals(P, T, 'fr'),
  timeseries: () => queryKeys.timeseries(P, T, 'h'),
  // Classement.
  leaderboard: () => queryKeys.leaderboard(P, T),
  leaderboardCatalog: () => queryKeys.leaderboardCatalog(P, T, 'fr'),
  // Notifications (feuilles ; notificationsAll est un préfixe large -> agnostique).
  notifications: () => queryKeys.notifications(P, T, {}),
  notificationsUnreadCount: () => queryKeys.notificationsUnreadCount(P, T),
  notificationsPreferences: () => queryKeys.notificationsPreferences(P, T),
  // Asset drawer (title-scopé par le 1er argument).
  assetMaps: () => queryKeys.assetMaps(T, 'q'),
  assetWeapons: () => queryKeys.assetWeapons(T, 'q'),
  assetMedals: () => queryKeys.assetMedals(T, 'q'),
  // Progression (Ascension).
  progressionStreaks: () => queryKeys.progressionStreaks(P, T),
  progressionRecords: () => queryKeys.progressionRecords(P, T),
  progressionMilestones: () => queryKeys.progressionMilestones(P, T),
  progressionProfile: () => queryKeys.progressionProfile(P, T),
  progressionPatterns: () => queryKeys.progressionPatterns(P, T),
  progressionActivity: () => queryKeys.progressionActivity(P, T),
  // Coach.
  coachProposals: () => queryKeys.coachProposals(P, T),
  // Namespaces imbriqués title-scopés.
  'prestige.me': () => queryKeys.prestige.me(U, T),
  'prestige.templates': () => queryKeys.prestige.templates(U, T),
  'prestige.pilotMode': () => queryKeys.prestige.pilotMode(U, T),
  'arc.list': () => queryKeys.arc.list(U, T),
  'arc.presets': () => queryKeys.arc.presets(U, T),
  'challenge.list': () => queryKeys.challenge.list(U, T),
  'challenge.history': () => queryKeys.challenge.history(U, T),
  'playerProfile.profile': () => queryKeys.playerProfile.profile(P, T, 30),
  'playerProfile.activeCampaign': () => queryKeys.playerProfile.activeCampaign(P, T),
  'playerProfile.campaign': () => queryKeys.playerProfile.campaign(P, T, 'id'),
  'playerProfile.campaignHistory': () => queryKeys.playerProfile.campaignHistory(P, T),
}

/**
 * Clés INTENTIONNELLEMENT sans titre — chaque nom porte sa justification :
 *   - GLOBAL / AUTH / ADMIN : pas de dimension titre côté produit (setup, session, santé,
 *     console admin — le slug admin est un filtre utilisateur explicite, pas la cible de
 *     la fuite par switch de titre).
 *   - CROSS-TITRE PAR NATURE : la donnée enjambe volontairement les titres (recherche
 *     gamertag Xbox, énumération des titres d'un joueur, squads/relations d'identité).
 *   - PRÉFIXE LARGE : clé `['x', playerSlug]` (ou statique) utilisée pour invalider TOUT
 *     le sous-arbre d'un joueur ; reste un préfixe VALIDE des clés enrichies du titre,
 *     donc sa sémantique « tout le joueur » est préservée (cf. keys.ts en-tête).
 */
const agnosticKeys = new Set<string>([
  // Global / setup / auth.
  'bootstrap',
  'players',
  'sessionContext',
  'health',
  'deviceFlow',
  'job',
  'settings',
  'groups',
  'changelog',
  'releaseNotes',
  'feedbackSimilarIssues',
  // Cross-titre par nature.
  'playerTitles', // statut sync de TOUS les titres d'un joueur
  'gamertagSearch', // recherche Xbox globale
  // Préfixes larges (invalidation « tout le joueur »).
  'matchHistoryAll',
  'mediaBase',
  'feedVersion',
  'notificationsAll',
  'coachAll',
  'teammatesAll',
  'filtersResolveAll',
  'adminDataQualityIssuesAll',
  // Admin console (non soumise à la fuite par switch de titre joueur).
  'adminInvariants',
  'adminDbContention',
  'adminTokenHealth',
  'adminMonitoringOverview',
  'adminMonitoringScheduler',
  'adminMonitoringJobs',
  'adminMonitoringConvergence',
  'adminMonitoringPerf',
  'adminMonitoringErrors',
  'adminMonitoringDetections',
  'adminMonitoringFreshness',
  'adminMonitoringResources',
  'adminMonitoringCrons',
  'adminActionJournal',
  'adminWeaponCoverage',
  'adminLusrGaps',
  'adminDataQuality',
  'adminDataQualityIssues',
  'adminLogModules',
  'adminLogTail',
  'adminTitles',
  'adminTitleDetail',
  'adminTitleDiagnostic',
  'adminUsers',
  'adminAppearanceDiagMutation',
  // Namespaces imbriqués agnostiques.
  'prestige.meAll', // préfixe large (userId)
  'arc.one', // identifiant global d'arc
  'challenge.one', // identifiant global de défi
  'squad.mine', // squads d'un utilisateur (identité, cross-titre)
  'squad.challenges', // clé par squadId (identité)
  'squad.orientation', // clé par squadId (identité)
  'playerProfile.campaignAll', // préfixe large (playerSlug)
  'watcher.status',
  'watcher.authPoll',
])

/** Aplati les noms feuilles de `queryKeys` (namespaces imbriqués -> `parent.child`). */
function leafNames(obj: Record<string, unknown>, prefix = ''): string[] {
  const out: string[] = []
  for (const [name, value] of Object.entries(obj)) {
    const full = prefix ? `${prefix}.${name}` : name
    if (typeof value === 'function' || Array.isArray(value)) {
      out.push(full)
    } else if (value && typeof value === 'object') {
      out.push(...leafNames(value as Record<string, unknown>, full))
    }
  }
  return out
}

describe('garde-rail titleSlug dans les query keys (anti-fuite cross-titre V72-29)', () => {
  it('chaque fabrique title-scopée produit une clé CONTENANT le slug de titre', () => {
    const offenders = Object.entries(titleScopedInvocations)
      .filter(([, invoke]) => !invoke().includes(T))
      .map(([name]) => name)
    expect(
      offenders,
      `Clés title-scopées qui n'incluent PAS le slug de titre (fuite cross-titre possible) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it('toute fabrique/constante de queryKeys est classée exactement une fois', () => {
    const all = leafNames(queryKeys as unknown as Record<string, unknown>)
    const titleScoped = new Set(Object.keys(titleScopedInvocations))

    const unclassified = all.filter((n) => !titleScoped.has(n) && !agnosticKeys.has(n))
    expect(
      unclassified,
      `Clé(s) non classée(s) — décide et documente : title-scopée (ajouter une invocation) ` +
        `ou agnostique (ajouter à agnosticKeys avec justification) : ${unclassified.join(', ')}`,
    ).toEqual([])

    const both = all.filter((n) => titleScoped.has(n) && agnosticKeys.has(n))
    expect(both, `Clé(s) classée(s) DEUX fois : ${both.join(', ')}`).toEqual([])

    // Symétrie : pas de nom classé qui n'existe plus dans queryKeys (évite un allowlist mort).
    const allSet = new Set(all)
    const stale = [...titleScoped, ...agnosticKeys].filter((n) => !allSet.has(n))
    expect(stale, `Nom(s) classé(s) absent(s) de queryKeys (nettoyer) : ${stale.join(', ')}`).toEqual([])
  })
})
