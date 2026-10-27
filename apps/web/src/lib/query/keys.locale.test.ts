/**
 * Tests — invariance des query keys locale-dependantes (defis / season pass).
 *
 * Contexte (item backlog [i18n]) : le backend baque les libelles localises (titres
 * de defis, nom du pass, map/mode) dans le payload selon le header X-LevelUp-Locale
 * a l'instant du fetch. Si la locale n'entre PAS dans la query key, un switch de
 * langue laisse le cache (y compris le fetch background prefetch/poll) baken dans
 * l'ancienne langue. Ce test verrouille l'invariant : deux locales -> deux cles
 * distinctes (donc refetch naturel a la bascule), locale identique -> cle stable.
 */
import { describe, it, expect } from 'vitest'
import { queryKeys } from './keys'

describe('query keys locale-aware (bascule de langue -> refetch naturel)', () => {
  describe('home (defis de l accueil)', () => {
    it('FR vs EN : cles distinctes (le switch de langue change la cle)', () => {
      const fr = queryKeys.home('player-1', 'halo_infinite', 'fr')
      const en = queryKeys.home('player-1', 'halo_infinite', 'en')
      expect(fr).not.toEqual(en)
      expect(en).toContain('en')
      expect(fr).toContain('fr')
    })

    it('meme locale : cle stable (cache reutilise)', () => {
      expect(queryKeys.home('player-1', 'halo_infinite', 'fr')).toEqual(
        queryKeys.home('player-1', 'halo_infinite', 'fr'),
      )
    })

    it('la locale est le dernier segment de la cle', () => {
      expect(queryKeys.home('p', 'halo_5', 'en')).toEqual(['home', 'p', 'halo_5', 'en'])
    })
  })

  describe('seasonPass (defis / libelles du pass)', () => {
    it('FR vs EN : cles distinctes', () => {
      const fr = queryKeys.seasonPass('player-1', 'halo_infinite', 'fr')
      const en = queryKeys.seasonPass('player-1', 'halo_infinite', 'en')
      expect(fr).not.toEqual(en)
    })

    it('forme attendue avec titleSlug puis locale en dernier segment', () => {
      expect(queryKeys.seasonPass('p', 'halo_infinite', 'en')).toEqual([
        'palmares',
        'p',
        'halo_infinite',
        'season-pass',
        'en',
      ])
    })
  })
})

/**
 * Lot B de l'étape 6 du plan i18n (ADR 0040 D-3) : les endpoints dont le serveur résout les
 * libellés dans la langue de la requête (succès, actifs, jalons, profil, coach, modèles de défi,
 * presets d'arcs, défis d'escouade) portent la locale dans la clé. Deux langues = deux clés, donc
 * un fetch en vol survivant à la bascule ne repeuple pas la mauvaise langue.
 */
describe('query keys des réponses localisées par le serveur (lot B, étape 6)', () => {
  const P = 'player-1'
  const T = 'halo_infinite'
  const fabriques: Record<string, (locale: string) => readonly unknown[]> = {
    achievements: (l) => queryKeys.achievements(P, T, l),
    assetMaps: (l) => queryKeys.assetMaps(T, 'q', l),
    assetWeapons: (l) => queryKeys.assetWeapons(T, 'q', l),
    assetMedals: (l) => queryKeys.assetMedals(T, 'q', l),
    progressionMilestones: (l) => queryKeys.progressionMilestones(P, T, l),
    progressionProfile: (l) => queryKeys.progressionProfile(P, T, l),
    'playerProfile.profile': (l) => queryKeys.playerProfile.profile(P, T, l, 30),
    coachProposals: (l) => queryKeys.coachProposals(P, T, l),
    'prestige.templates': (l) => queryKeys.prestige.templates(P, T, l),
    'arc.presets': (l) => queryKeys.arc.presets(P, T, l),
    'arc.list': (l) => queryKeys.arc.list(P, T, l),
    'challenge.list': (l) => queryKeys.challenge.list(P, T, l),
    'challenge.history': (l) => queryKeys.challenge.history(P, T, l),
    'squad.challenges': (l) => queryKeys.squad.challenges('sq-1', l),
    notifications: (l) => queryKeys.notifications(P, T, l, {}),
  }

  for (const [nom, fabrique] of Object.entries(fabriques)) {
    it(`${nom} : FR et EN donnent deux clés distinctes, la même langue une clé stable`, () => {
      expect(fabrique('fr')).not.toEqual(fabrique('en'))
      expect(fabrique('fr')).toEqual(fabrique('fr'))
      expect(fabrique('en')).toContain('en')
    })
  }

  it('squad.challengesAll reste un préfixe des clés de toutes les langues', () => {
    const prefixe = queryKeys.squad.challengesAll('sq-1')
    for (const l of ['fr', 'en']) {
      expect(queryKeys.squad.challenges('sq-1', l).slice(0, prefixe.length)).toEqual(prefixe)
    }
  })

  it('arc.listAll et challenge.listAll restent des préfixes des clés de toutes les langues', () => {
    const arcs = queryKeys.arc.listAll(P, T)
    const defis = queryKeys.challenge.listAll(P, T)
    for (const l of ['fr', 'en']) {
      expect(queryKeys.arc.list(P, T, l).slice(0, arcs.length)).toEqual(arcs)
      expect(queryKeys.challenge.list(P, T, l).slice(0, defis.length)).toEqual(defis)
    }
  })
})

/**
 * Lot C de l'étape 6 du plan i18n (ADR 0040 D-3) : les endpoints dont le serveur résout les noms
 * d'armes, de cartes, de lieux et de rangs dans la langue de la requête (carrière, profil de la cible
 * de l'Explorateur, vue du match, synthèse, séries temporelles, comparaison, onglet Tactique) portent
 * la locale dans la clé. Deux langues = deux clés.
 */
describe('query keys des réponses localisées par le serveur (lot C, étape 6)', () => {
  const P = 'player-1'
  const T = 'halo_infinite'
  const fabriques: Record<string, (locale: string) => readonly unknown[]> = {
    career: (l) => queryKeys.career(P, T, l),
    explorerPlayer: (l) => queryKeys.explorerPlayer(P, T, 'gt', 'x', 1, l),
    matchView: (l) => queryKeys.matchView(P, T, 'm', l),
    synthesis: (l) => queryKeys.synthesis(P, T, 'h', l),
    timeseries: (l) => queryKeys.timeseries(P, T, 'h', l),
    comparePlayer: (l) => queryKeys.comparePlayer(P, T, 'gt', l),
    tacticalMaps: (l) => queryKeys.tacticalMaps(P, T, 'h', l),
    tacticalRaster: (l) => queryKeys.tacticalRaster(P, T, 'map-1', 'h', l),
    tacticalCellule: (l) => queryKeys.tacticalCellule(P, T, 'map-1', 'h', l),
  }

  for (const [nom, fabrique] of Object.entries(fabriques)) {
    it(`${nom} : FR et EN donnent deux clés distinctes, la même langue une clé stable`, () => {
      expect(fabrique('fr')).not.toEqual(fabrique('en'))
      expect(fabrique('fr')).toEqual(fabrique('fr'))
      expect(fabrique('en')).toContain('en')
    })
  }

  it('careerAll reste un préfixe des clés de la page Carrière, quelle que soit la langue', () => {
    const prefixe = queryKeys.careerAll(P)
    for (const l of ['fr', 'en']) {
      expect(queryKeys.career(P, T, l).slice(0, prefixe.length)).toEqual(prefixe)
    }
  })
})

/**
 * Item 8.8.d du plan i18n (ADR 0040 D-3, D-15) : les cartes, modes, playlists et saisons des listes, des
 * options de filtre et des séries sont servis dans la langue de la requête (la valeur d'un filtre reste le
 * nom de la langue par défaut). Deux langues = deux clés, et les préfixes larges restent valides.
 */
describe('query keys des noms de jeu servis dans la langue de la requête (item 8.8.d)', () => {
  const P = 'player-1'
  const T = 'halo_infinite'
  const fabriques: Record<string, (locale: string) => readonly unknown[]> = {
    filtersResolve: (l) => queryKeys.filtersResolve(P, T, 'h', 'all', l),
    filtersPreview: (l) => queryKeys.filtersPreview(P, T, 'h', l),
    matchHistory: (l) => queryKeys.matchHistory(P, T, 'h', 1, [], l),
    explorer: (l) => queryKeys.explorer(P, T, 'h', [], [], '', [], '', l),
    careerHighlightMatches: (l) => queryKeys.careerHighlightMatches(P, T, 'f', l),
    engagementMatch: (l) => queryKeys.engagementMatch(P, T, 'm', l),
    engagementTimeseries: (l) => queryKeys.engagementTimeseries(P, T, 'h', 50, l),
    engagementSquadSession: (l) => queryKeys.engagementSquadSession(P, T, ['m'], ['x'], l),
    palmaresRelations: (l) => queryKeys.palmaresRelations(P, T, l),
  }

  for (const [nom, fabrique] of Object.entries(fabriques)) {
    it(`${nom} : FR et EN donnent deux clés distinctes, la même langue une clé stable`, () => {
      expect(fabrique('fr')).not.toEqual(fabrique('en'))
      expect(fabrique('fr')).toEqual(fabrique('fr'))
      expect(fabrique('en')).toContain('en')
    })
  }

  it('les préfixes larges restent des préfixes des clés de toutes les langues', () => {
    for (const l of ['fr', 'en']) {
      const histo = queryKeys.matchHistoryAll(P)
      expect(queryKeys.matchHistory(P, T, 'h', 1, [], l).slice(0, histo.length)).toEqual(histo)
      const resolve = queryKeys.filtersResolveAll(P)
      expect(queryKeys.filtersResolve(P, T, 'h', 'all', l).slice(0, resolve.length)).toEqual(resolve)
      const carriere = queryKeys.careerAll(P)
      expect(queryKeys.careerHighlightMatches(P, T, 'f', l).slice(0, carriere.length)).toEqual(carriere)
    }
  })
})

/**
 * Item 11.R du plan i18n (revue adversariale du 2026-10-27) : inventaire COMPLET des fabriques de
 * `queryKeys`, classées exactement une fois en « dépend de la langue de la requête » ou « n'en dépend
 * pas » (raison courte). Une fabrique de clé de réponse localisée qui oublie la locale sert, après une
 * bascule de langue, la réponse de l'autre langue (cache TanStack, fetch en vol) : trois fabriques
 * (patterns, médias, rail média) étaient dans ce cas, trois autres (CSR de la carrière, top de
 * l'Escouade, candidats de match d'un média) l'étaient aussi et ont été trouvées par cet inventaire.
 *
 * Trois contrôles : (1) toute fabrique est classée, ni deux fois, ni sous un nom disparu ; (2) toute
 * fabrique classée « dépend » produit une clé qui contient la langue ; (3) toute fabrique qui déclare
 * un paramètre `locale` / `lang` est classée « dépend » (le paramètre ne peut pas être décoratif),
 * et toute fabrique classée « dépend » en déclare un.
 */
const L = '__LOCALE_SENTINELLE__'
const PL = 'player-1'
const TI = 'halo_infinite'

const dependDeLaLangue: Record<string, () => readonly unknown[]> = {
  filtersResolve: () => queryKeys.filtersResolve(PL, TI, 'h', 'all', L),
  filtersPreview: () => queryKeys.filtersPreview(PL, TI, 'h', L),
  career: () => queryKeys.career(PL, TI, L),
  careerHighlightMatches: () => queryKeys.careerHighlightMatches(PL, TI, 'f', L),
  careerCSRs: () => queryKeys.careerCSRs(PL, TI, L),
  achievements: () => queryKeys.achievements(PL, TI, L),
  matchHistory: () => queryKeys.matchHistory(PL, TI, 'h', 1, [], L),
  explorer: () => queryKeys.explorer(PL, TI, 'h', [], [], '', [], '', L),
  explorerPlayer: () => queryKeys.explorerPlayer(PL, TI, 'gt', 'x', 1, L),
  matchView: () => queryKeys.matchView(PL, TI, 'm', L),
  engagementMatch: () => queryKeys.engagementMatch(PL, TI, 'm', L),
  engagementTimeseries: () => queryKeys.engagementTimeseries(PL, TI, 'h', 50, L),
  engagementSquadSession: () => queryKeys.engagementSquadSession(PL, TI, [], [], L),
  home: () => queryKeys.home(PL, TI, L),
  homeMatchesPage: () => queryKeys.homeMatchesPage(PL, TI, L, 'c'),
  homeFavorites: () => queryKeys.homeFavorites(PL, TI, L),
  seasonPass: () => queryKeys.seasonPass(PL, TI, L),
  palmaresRelations: () => queryKeys.palmaresRelations(PL, TI, L),
  teammates: () => queryKeys.teammates(PL, TI, 'h', [], L),
  teammatesTop: () => queryKeys.teammatesTop(PL, TI, L),
  tacticalMaps: () => queryKeys.tacticalMaps(PL, TI, 'h', L),
  tacticalRaster: () => queryKeys.tacticalRaster(PL, TI, 'map-1', 'h', L),
  tacticalCellule: () => queryKeys.tacticalCellule(PL, TI, 'map-1', 'h', L),
  synthesis: () => queryKeys.synthesis(PL, TI, 'h', L),
  media: () => queryKeys.media(PL, TI, 'h', L),
  mediaRail: () => queryKeys.mediaRail(PL, TI, 5, L),
  mediaMatchCandidates: () => queryKeys.mediaMatchCandidates(PL, TI, null, 5, L),
  citations: () => queryKeys.citations(PL, TI, 'h', L),
  medals: () => queryKeys.medals(PL, TI, L),
  commendationTotals: () => queryKeys.commendationTotals(PL, TI, L),
  timeseries: () => queryKeys.timeseries(PL, TI, 'h', L),
  trends: () => queryKeys.trends(PL, TI, 'solo', 'ranked_slayer', L, '', false),
  sessionDetail: () => queryKeys.sessionDetail(PL, TI, 'h', 's', 'c', false, L),
  comparePlayer: () => queryKeys.comparePlayer(PL, TI, 'gt', L),
  leaderboardCatalog: () => queryKeys.leaderboardCatalog(PL, TI, L),
  notifications: () => queryKeys.notifications(PL, TI, L, {}),
  assetMaps: () => queryKeys.assetMaps(TI, 'q', L),
  assetWeapons: () => queryKeys.assetWeapons(TI, 'q', L),
  assetMedals: () => queryKeys.assetMedals(TI, 'q', L),
  progressionMilestones: () => queryKeys.progressionMilestones(PL, TI, L),
  progressionProfile: () => queryKeys.progressionProfile(PL, TI, L),
  progressionPatterns: () => queryKeys.progressionPatterns(PL, TI, L),
  coachProposals: () => queryKeys.coachProposals(PL, TI, L),
  releaseNotes: () => queryKeys.releaseNotes(L),
  'prestige.templates': () => queryKeys.prestige.templates(PL, TI, L),
  'arc.list': () => queryKeys.arc.list(PL, TI, L),
  'arc.presets': () => queryKeys.arc.presets(PL, TI, L),
  'challenge.list': () => queryKeys.challenge.list(PL, TI, L),
  'challenge.history': () => queryKeys.challenge.history(PL, TI, L),
  'squad.challenges': () => queryKeys.squad.challenges('sq-1', L),
  'playerProfile.profile': () => queryKeys.playerProfile.profile(PL, TI, L, 30),
}

const PREFIXE_LARGE = "préfixe large d'invalidation"
const CONSOLE_ADMIN = 'console admin : aucun libellé de jeu résolu par la langue de la requête'
const TEXTE_STOCKE = 'texte stocké dans une seule langue, non résolu par la requête (plan §6, étape 7)'

const nePasDependreDeLaLangue: Record<string, string> = {
  // Constantes et clés globales.
  bootstrap: 'amorçage : porte la langue résolue, ne la reçoit pas',
  players: 'liste des profils suivis (gamertags)',
  sessionContext: 'contexte de session (identifiants)',
  health: "état du serveur",
  settings: "réglages d'instance (valeurs)",
  groups: "groupes d'accès (noms saisis par les utilisateurs)",
  changelog: 'un seul fichier, non localisé',
  feedbackSimilarIssues: 'recherche de tickets par texte saisi',
  feedVersion: 'compteur de version du flux média',
  // Identifiants, jobs, statuts.
  playerTitles: 'statut de synchronisation par titre',
  deviceFlow: "état d'un flux d'authentification",
  job: "état d'une tâche",
  playerFriends: 'gamertags des amis',
  gamertagSearch: 'recherche Xbox par gamertag',
  presence: 'joueurs en ligne (gamertags)',
  titleDataCapabilities: 'capacités du titre (clés techniques)',
  'watcher.status': 'état du suivi en direct',
  'watcher.authPoll': "état d'un flux d'authentification",
  // Préfixes larges : jamais une clé de lecture.
  teammatesAll: PREFIXE_LARGE,
  matchHistoryAll: PREFIXE_LARGE,
  careerAll: PREFIXE_LARGE,
  homeAll: PREFIXE_LARGE,
  mediaBase: PREFIXE_LARGE,
  notificationsAll: PREFIXE_LARGE,
  coachAll: PREFIXE_LARGE,
  filtersResolveAll: PREFIXE_LARGE,
  adminDataQualityIssuesAll: PREFIXE_LARGE,
  'prestige.meAll': PREFIXE_LARGE,
  'squad.challengesAll': PREFIXE_LARGE,
  'playerProfile.campaignAll': PREFIXE_LARGE,
  'arc.listAll': PREFIXE_LARGE,
  'challenge.listAll': PREFIXE_LARGE,
  // Carrière, match, tactique, médias : identifiants, scores, mesures.
  careerEncounters: 'joueurs croisés (gamertags, comptes)',
  careerTopEncounters: 'joueurs croisés (gamertags, comptes)',
  careerRivals: 'némésis et souffre-douleur (gamertags, comptes)',
  matchObjectiveEvents: 'événements décodés du film (identifiants, instants)',
  matchPositions: 'positions décodées du film (coordonnées)',
  matchReplay: "artefact de rejeu (toutes les langues y figurent, la page choisit)",
  matchReplayContext: 'contexte du rejeu (identifiants, équipes)',
  matchReplayBackground: "fond de carte du rejeu (métadonnées d'image)",
  matchReplayBackgroundImage: 'image du fond de carte',
  matchReplayCallouts: 'zones nommées : toutes les langues dans la réponse',
  matchNeighbors: 'identifiants des matchs voisins',
  engagementProfile: "coefficients d'engagement (nombres)",
  compositionSessions: 'sessions de composition (libellés de session, comptes)',
  tacticalMatchIDs: 'identifiants de matchs du périmètre',
  tacticalMapBackground: 'fond de carte (image)',
  tacticalMapBackgroundFrame: 'cadre du fond de carte (nombres)',
  tacticalSequence: "séquence d'une mort (trajectoires, joueurs)",
  mediaAuthors: 'auteurs des médias (gamertags)',
  mediaAudioConfig: 'réglage audio',
  leaderboard: 'classement : identifiants de saison et de playlist, scores',
  notificationsUnreadCount: 'compteur',
  notificationsPreferences: 'préférences (clés techniques)',
  progressionStreaks: 'séries (valeurs, dates)',
  progressionRecords: 'records (valeurs, dates)',
  progressionActivity: 'jours joués',
  'prestige.me': 'prestige du joueur (paliers en clés)',
  'prestige.pilotMode': 'clé de mutation',
  'arc.one': `identifiant d'arc : ${TEXTE_STOCKE}`,
  'challenge.one': `identifiant de défi : ${TEXTE_STOCKE}`,
  'squad.mine': 'escouades enregistrées (noms saisis)',
  'squad.orientation': 'axe focal en clé stable',
  'playerProfile.activeCampaign': `campagne active : ${TEXTE_STOCKE}`,
  'playerProfile.campaign': `campagne : ${TEXTE_STOCKE}`,
  'playerProfile.campaignHistory': `campagnes closes : ${TEXTE_STOCKE}`,
  // Console d'administration.
  adminInvariants: CONSOLE_ADMIN,
  adminDbContention: CONSOLE_ADMIN,
  adminTokenHealth: CONSOLE_ADMIN,
  adminIdentities: CONSOLE_ADMIN,
  adminMonitoringOverview: CONSOLE_ADMIN,
  adminMonitoringScheduler: CONSOLE_ADMIN,
  adminMonitoringJobs: CONSOLE_ADMIN,
  adminMonitoringConvergence: CONSOLE_ADMIN,
  adminMonitoringPerf: CONSOLE_ADMIN,
  adminMonitoringDetections: CONSOLE_ADMIN,
  adminMonitoringFreshness: CONSOLE_ADMIN,
  adminMonitoringResources: CONSOLE_ADMIN,
  adminMonitoringCrons: CONSOLE_ADMIN,
  adminMonitoringBuildQueue: CONSOLE_ADMIN,
  adminActionJournal: CONSOLE_ADMIN,
  adminWeaponCoverage: CONSOLE_ADMIN,
  adminLusrGaps: CONSOLE_ADMIN,
  adminDataQuality: 'langue du détecteur fixée par la constante DATA_QUALITY_LOCALE de la requête',
  adminDataQualityIssues: 'langue du détecteur fixée par la constante DATA_QUALITY_LOCALE de la requête',
  adminLogModules: CONSOLE_ADMIN,
  adminLogTail: CONSOLE_ADMIN,
  adminTitles: CONSOLE_ADMIN,
  adminTitleDetail: CONSOLE_ADMIN,
  adminTitleDiagnostic: CONSOLE_ADMIN,
  adminUsers: CONSOLE_ADMIN,
  adminInvites: CONSOLE_ADMIN,
  adminAppearanceDiagMutation: 'clé de mutation',
}

/** Noms feuilles de `queryKeys`, avec les paramètres déclarés d'une fabrique (texte avant la flèche). */
function fabriques(obj: Record<string, unknown>, prefixe = ''): Array<{ nom: string; parametres: string }> {
  const out: Array<{ nom: string; parametres: string }> = []
  for (const [nom, valeur] of Object.entries(obj)) {
    const complet = prefixe ? `${prefixe}.${nom}` : nom
    if (typeof valeur === 'function') {
      const source = valeur.toString()
      out.push({ nom: complet, parametres: source.slice(0, source.indexOf('=>')) })
    } else if (Array.isArray(valeur)) {
      out.push({ nom: complet, parametres: '' })
    } else if (valeur && typeof valeur === 'object') {
      out.push(...fabriques(valeur as Record<string, unknown>, complet))
    }
  }
  return out
}

const DECLARE_LA_LANGUE = /\b(locale|lang)\b/

describe('inventaire des fabriques de queryKeys par dépendance à la langue (item 11.R)', () => {
  const toutes = fabriques(queryKeys as unknown as Record<string, unknown>)

  it('toute fabrique est classée exactement une fois, sans nom disparu', () => {
    const noms = new Set(toutes.map((f) => f.nom))
    const nonClassees = toutes
      .map((f) => f.nom)
      .filter((n) => !(n in dependDeLaLangue) && !(n in nePasDependreDeLaLangue))
    expect(
      nonClassees,
      `Fabrique(s) non classée(s) : décider si la réponse dépend de la langue de la requête. Oui : paramètre ` +
        `locale et invocation dans dependDeLaLangue ; non : raison dans nePasDependreDeLaLangue : ${nonClassees.join(', ')}`,
    ).toEqual([])
    const deuxFois = Object.keys(dependDeLaLangue).filter((n) => n in nePasDependreDeLaLangue)
    expect(deuxFois, `Fabrique(s) classée(s) deux fois : ${deuxFois.join(', ')}`).toEqual([])
    const disparus = [...Object.keys(dependDeLaLangue), ...Object.keys(nePasDependreDeLaLangue)].filter(
      (n) => !noms.has(n),
    )
    expect(disparus, `Nom(s) classé(s) absent(s) de queryKeys : ${disparus.join(', ')}`).toEqual([])
  })

  it('toute fabrique classée « dépend de la langue » produit une clé qui la contient', () => {
    const sansLangue = Object.entries(dependDeLaLangue)
      .filter(([, invoquer]) => !invoquer().includes(L))
      .map(([nom]) => nom)
    expect(sansLangue, `Clé(s) sans la langue : ${sansLangue.join(', ')}`).toEqual([])
  })

  it('un paramètre de langue déclaré classe la fabrique « dépend », et inversement', () => {
    const declarent = toutes.filter((f) => DECLARE_LA_LANGUE.test(f.parametres)).map((f) => f.nom)
    const declareSansEtreClassee = declarent.filter((n) => !(n in dependDeLaLangue))
    expect(
      declareSansEtreClassee,
      `Paramètre de langue déclaré mais fabrique classée indépendante : ${declareSansEtreClassee.join(', ')}`,
    ).toEqual([])
    const classeeSansParametre = Object.keys(dependDeLaLangue).filter((n) => !declarent.includes(n))
    expect(
      classeeSansParametre,
      `Classée « dépend » sans paramètre de langue : ${classeeSansParametre.join(', ')}`,
    ).toEqual([])
  })
})

describe('query keys corrigées par la revue du 2026-10-27 (item 11.R)', () => {
  const fabriquesCorrigees: Record<string, (locale: string) => readonly unknown[]> = {
    progressionPatterns: (l) => queryKeys.progressionPatterns('p', 't', l, 50),
    media: (l) => queryKeys.media('p', 't', 'h', l),
    mediaRail: (l) => queryKeys.mediaRail('p', 't', 5, l, false),
    careerCSRs: (l) => queryKeys.careerCSRs('p', 't', l, 's1'),
    teammatesTop: (l) => queryKeys.teammatesTop('p', 't', l),
    mediaMatchCandidates: (l) => queryKeys.mediaMatchCandidates('p', 't', 'f.mp4', 5, l),
  }
  for (const [nom, fabrique] of Object.entries(fabriquesCorrigees)) {
    it(`${nom} : deux langues, deux clés ; le préfixe médias du joueur reste valide`, () => {
      expect(fabrique('fr')).not.toEqual(fabrique('en'))
      expect(fabrique('en')).toContain('en')
    })
  }

  it('mediaBase reste un préfixe des clés de page et de rail, quelle que soit la langue', () => {
    const prefixe = queryKeys.mediaBase('p')
    for (const l of ['fr', 'en']) {
      expect(queryKeys.media('p', 't', 'h', l).slice(0, prefixe.length)).toEqual(prefixe)
      expect(queryKeys.mediaRail('p', 't', 5, l).slice(0, prefixe.length)).toEqual(prefixe)
    }
  })
})
