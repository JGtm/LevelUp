/**
 * contract.ts — Le TYPE du dictionnaire de la feature match-view : une seule déclaration, que
 * `fr.ts`, `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies MatchViewText`).
 */
import type { AnalysisPendingPart } from '@/lib/analysis-pending/analysisPending.logic'

export interface MatchViewText {
  prevMatch: string
  nextMatch: string
  matchCounter: (n: number, total: number) => string
  copyMatchId: string
  copied: string
  copyShort: string
  copyTooltip: string
  replayTooltip: string
  /**
   * Aide du score quand le mode se joue en MANCHES : le nombre affiché est le compte de
   * manches gagnées / perdues, pas le score en points de l'API — lequel, sur ces modes,
   * peut donner la victoire au camp qui en a le moins.
   */
  scoreRoundsHint: string
  /** Étiquette du score de l'API affiché en second plan, à côté du compte de manches. */
  scorePointsAside: (label: string) => string
  markIrrelevant: string
  reactivate: string
  excludeShort: string
  excludeTooltip: string
  reactivateTooltip: string
  // Confirmation dialog d'exclusion / réactivation
  excludeConfirmTitle: string
  excludeConfirmBody: string
  reactivateConfirmTitle: string
  reactivateConfirmBody: string
  confirmAction: string
  cancelAction: string
  excludeRankedDenied: string
  excludeErrorRanked: string
  excludeErrorGeneric: string
  performance: string
  rank: string
  /** Libellé localisé de la sentinelle de tier « Placement » (phase de placement). */
  rankPlacement: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  pageErrorTitle: string
  pageRetry: string
  /** Zone de messages : une source de la vue est en panne (≠ vide). Prime sur noticePartialLead. */
  noticeUnavailable: string
  /** Mention qui suit : la vue se relance seule tant que la panne dure. */
  noticeUnavailableRetry: string
  /** Source vide : amorce de la liste des sections absentes (translatePartialReason). */
  noticePartialLead: string
  // État dédié 404 match_not_found — match absent du substrat local (pas encore
  // synchronisé, ou identifiant invalide). Remplace l'écran d'erreur générique
  // (retiré le 2026-07-25 avec le fallback LIVE du Match view, cf. BACKLOG).
  notSyncedTitle: string
  notSyncedDescription: string
  /** Bouton de la page « indisponible » : la liste des matchs du joueur. */
  navMatches: string
  /** Boutons de la page « indisponible » : accueil du joueur, page précédente. */
  navHome: string
  navBack: string
  /** 404 match_not_participant : le joueur de la page n'a pas joué ce match. */
  notParticipantTitle: string
  notParticipantDescription: string
  /** 403 player_forbidden : joueur non rattaché au compte connecté. */
  accessDeniedTitle: string
  accessDeniedDescription: string
  noRank: string
  exitContext: string
  outcomeWin: string
  outcomeLoss: string
  outcomeDraw: string
  outcomeDnf: string
  fromDate: string
  toDate: string
  // Charts résumé
  chartKdaTitle: string
  chartSpreeTitle: string
  seriesActual: string
  seriesExpected: string
  seriesHistAvg: string
  labelKills: string
  labelDeaths: string
  labelAssists: string
  labelSpree: string
  labelHeadshots: string
  labelPerfectKills: string
  noHistData: string
  duration: string
  // Radar synergie (joueur actif)
  chartSynergyRadarTitle: string
  radarAxisCombat: string
  radarAxisSurvival: string
  radarAxisSupport: string
  radarAxisScore: string
  radarAxisObjective: string
  radarAxisImpact: string
  radarTooltipImpact: string
  radarTooltipCombat: string
  radarTooltipSurvival: string
  radarTooltipSupport: string
  radarTooltipScore: string
  radarTooltipObjective: string
  radarTooltipGlossaryLink: string
  // Labels de mécaniques natives Halo 5 — colonnes du scoreboard (MatchScoreboard).
  labelGrenade: string
  labelAssassination: string
  labelGroundPound: string
  labelShoulderBash: string
  weaponUnknownPrefix: string
  // Distance par arme, par joueur — POC (LOT G.3, 2026-08-30, plan
  // retours-utilisateur §3bis DEC-8). Vue match uniquement, kills du TUEUR
  // seulement (arme/distance de l'assistant hors périmètre).
  killDistanceTitle: string
  killDistanceColKills: string
  killDistanceColAvg: string
  killDistanceMinLabel: string
  killDistanceMaxLabel: string
  /** Distance moyenne formatée locale-aware, ex. « 12,4 m » (FR) / « 12.4 m » (EN). */
  killDistanceAvgFmt: (m: number) => string
  killDistanceReserve: string
  // Section médias (onglet Général) : toujours affichée sous la capability `media`, état vide inclus.
  sectionMedia: string
  mediaNoCaptures: string
  mediaNoCapturesDesc: string
  // Résumé — médailles & citations
  sectionMedals: string
  sectionCitations: string
  newlyMastered: string
  noMedals: string
  noCitations: string
  // Commendations NATIVES (Halo 5) — affichées à la place des citations dérivées
  sectionNativeCommendations: string
  noNativeCommendations: string
  // Onglet Combat — charts en haut (mock match_view.09 / .10 / .11 / .12)
  combatHighlights: string
  combatKdCumulTitle: string
  combatTugOfWarTitle: string
  /**
   * LA COURBE DE SCORE, décodée du film du match (schéma 12). Elle n'apparaît que pour les
   * matchs dont un artefact de rejeu existe — d'où la note de source, qui dit à
   * l'utilisateur pourquoi cette carte est là sur ce match et pas sur le précédent.
   */
  scoreCurveTitle: string
  scoreCurveSource: string
  scoreCurveLead: string
  /**
   * LES POINTS MARQUÉS DANS LE TEMPS — la lecture que prennent les modes qui marquent en
   * trois à cinq fois sur tout le match (drapeau, colline, bombe), là où la courbe serait
   * un escalier vide. Même source, même réserve de pied que la courbe.
   */
  scoreEventsTitle: string
  /** Points pris à cet instant, dans l'infobulle (ex. « +1 point »). */
  scoreEventsScoredFmt: (points: number) => string
  /** Score cumulé du camp juste après cette marque (ex. « Score : 2 »). */
  scoreEventsTotalFmt: (total: number) => string
  combatCadenceTitle: string
  combatKillsLabel: string
  combatDeathsLabel: string
  combatTeamLabel: string
  combatEnemyLabel: string
  // Histogramme momentum (carte Dominance) — libellés de tooltip.
  combatMomentumDelta: string
  combatMomentumCumul: string
  /** Décompte headshot dans le tooltip de vague (G.1) — n toujours >= 1 à l'appel. */
  combatHeadshotCountFmt: (n: number) => string
  combatNemesisTitle: string
  combatBullyTitle: string
  combatNoNemesis: string
  combatKilledMeFmt: (n: number) => string
  combatIKilledFmt: (n: number) => string
  combatNoData: string
  // Overlay capture CTF (charts combat — câblé couche 2)
  combatCtfCaptureLabel: string
  combatCtfCaptureTooltip: (player: string, time: string) => string
  fragDiffNoData: string
  antagonistNoData: string
  impactBadgesNoData: string
  // Libellés des badges d'impact (Match flow), keyés par BadgeKey backend. Le
  // moteur analysis ne produit qu'un libellé FR (BadgeFR) → sous UI EN les cartes
  // restaient en FR (GH-7). Résolution front bilingue par clé, fallback = libellé
  // serveur pour une clé inconnue.
  impactBadgeNames: Record<string, string>
  // Breadcrumb retour (MatchBreadcrumb)
  back: string
  // Onglets de la page (GH2-B2 ; « Détails » scindé en Chronologie + Joueurs ;
  // troisième onglet ajouté le 2026-09-19 — équipement, armes, occupation du terrain —
  // renommé « Armes et terrain » le 2026-09-22 quand il a repris de Général la
  // répartition des frags et la distance des frags)
  tabGeneral: string
  tabChronology: string
  tabArsenal: string
  tabPlayers: string
  // Titre du chart Antagonistes (GH2-B2)
  antagonistTitle: string
  // Graphe des ASSISTANCES (assistant -> tueur assisté). Un match dont le film ne porte pas
  // l'assistance n'a pas de bloc (rien n'est rendu) : le seul état vide écrit est « aucune »,
  // le film lu sans aucune assistance.
  assistTitle: string
  assistNoData: string
  // Note d'infobulle d'un segment : les éliminations volées de ce couple
  // (part de dégâts de l'assistant supérieure à celle du tueur crédité).
  assistStolenNote: (n: number) => string
  // Note d'infobulle d'un segment : la part moyenne de participation de l'assistant sur
  // ce couple. Vocabulaire « part » (comme le kill feed du rejeu), jamais « dégâts ».
  assistAvgShareNote: (pct: number) => string
  // Les deux rôles du graphe (décision utilisateur 2026-09-17, même vocabulaire que la page
  // Escouade) : la barre est le LARBIN (il a assisté), le segment le PATRON (frag crédité).
  // Titres d'axes et infobulle.
  assistRoleAssistant: string
  assistRoleBeneficiary: string
  assistValueAxis: string
  // Sections des onglets Chronologie et Joueurs (titres type-1 du catalogue
  // d'harmonisation)
  sectionFlow: string
  sectionDuels: string
  sectionEncounters: string
  // Sections des onglets Général et Armes et terrain (2026-09-22) — même gabarit de titre
  // type-1. Général : la bande de KPI reste sans titre (comme l'accueil), « Combat » coiffe
  // les trois graphes et « Récompenses » les médailles + citations. Armes et terrain :
  // « Frags et armes » coiffe la répartition des frags + la distance, « Équipement et
  // terrain » les trois blocs tirés du film.
  sectionCombat: string
  sectionRewards: string
  sectionKillsWeapons: string
  sectionEquipmentTerrain: string
  // Onglet Armes et terrain quand AUCUNE de ses deux sections n'a de quoi s'afficher (un
  // titre sans positions de film, sur un match sans frag) : un onglet ne reste jamais vide.
  arsenalEmptyTitle: string
  arsenalEmptyDescription: string
  // Analyse du film non finie (header.analysis_pending) : message de la zone de messages et
  // emplacements d'attente des sections qui dépendent du film. « En cours » seulement quand un
  // décodage porte sur CE match (header.analysis_state), sinon « en attente » ; jamais de délai
  // promis ni formule d'avenir (les données manquent, ce n'est pas une fonctionnalité qui arrive).
  analysisPendingTitleInProgress: string
  analysisPendingTitleQueued: string
  /** Nom court de chaque cause en attente (domain.AnalysisPart*). */
  analysisPendingParts: Record<AnalysisPendingPart, string>
  analysisPendingAutoUpdate: string
  analysisPendingSlotInProgress: string
  analysisPendingSlotQueued: string
  // Scoreboard team header (Eagle / Cobra avec couleur team-ally/enemy)
  scoreboardTitle: string
  scoreboardNoData: string
  teamLabelFmt: (name: string) => string
  teamUnknown: string
  teamNumberedFmt: (n: number) => string
  teamMine: string
  teamEnemy: string
  // Scoreboard expander (port de match_view_scoreboard_detail.py)
  sbDetailWeapons: string
  sbDetailMedalsAndCitations: string
  sbDetailMedalsOnly: string
  sbDetailExpected: string
  sbDetailExpectedKills: string
  sbDetailExpectedDeaths: string
  sbDetailExpectedAssists: string
  sbDetailLocallyEstimated: string
  sbDetailLocallyEstimatedHint: string
  sbDetailAntagonist: string
  sbDetailNemesis: string
  sbDetailBully: string
  sbDetailLocal: string
  sbDetailLusr: string
  sbDetailCsr: string
  sbDetailBotNoteLabel: string
  sbDetailBotNoteValue: string
  sbDetailPlayerDb: string
  sbDetailSharedOnly: string
  sbDetailExplorePlayerFmt: (player: string) => string
  // Libellés colonnes scoreboard (utilisés par buildHighlightCols)
  sbColKda: string
  sbColMeleeKills: string
  sbColDamageDealt: string
  sbColDamageTaken: string
  sbColShotsHit: string
  sbColAccuracy: string
  sbColCsr: string
  sbColRank: string
  sbColScore: string
  sbColAssists: string
  sbColMaxSpree: string
  sbColHeadshots: string
  sbColPerfectKills: string
  sbColShotsFired: string
  sbColPowerWeapons: string
  sbColAvgLife: string
  sbColPlayer: string
  sbColTopWeapon: string
  // Tooltips d'en-tête de colonne (V72-04, icône ⓘ) — colonnes non évidentes.
  sbColCsrTooltip: string
  sbColLusrTooltip: string
  sbColRankTooltip: string
  sbColKdaTooltip: string
  sbColAccuracyTooltip: string
  sbColMaxSpreeTooltip: string
  sbColPerfectKillsTooltip: string
  sbColPowerWeaponsTooltip: string
  sbColMeleeKillsTooltip: string
  sbColAvgLifeTooltip: string
  sbColTopWeaponTooltip: string
  sbColOffensiveTooltip: string
  sbColDefensiveTooltip: string
  /**
   * LES DEUX EN-TÊTES COURTS du tableau des scores (2026-09-13). Le libellé canonique du
   * registre (« Rendement » / « Résistance », servi par `/field-mappings`) reste la source de
   * vérité du NOM de la mesure : il est écrit en tête de l'infobulle d'en-tête. Ces deux
   * chaînes ne sont qu'un affichage abrégé, propre à cette table étroite — ce ne sont PAS des
   * libellés de `FieldKey` et elles n'en forment pas un dictionnaire.
   */
  sbColOffensiveShort: string
  sbColDefensiveShort: string
  sbViewHistoryFmt: (gamertag: string) => string
  /** Format du score (séparateurs locale-sensitive : "12 345" FR / "12,345" EN). */
  sbFormatScore: (v: number) => string
  // Nav contextuelle — Phase 2c (descriptor → label compact)
  ctxRecent: string
  ctxFavorites: string
  ctxMedia: string
  ctxTopMatches: string
  ctxWithPlayerFmt: (gamertag: string) => string
  ctxSessionFmt: (date: string) => string
  /** Jonction date + heure d'un libellé de contexte (« 07/05/26 à 21:30 » / « 07/05/26 at 21:30 »). */
  dateTimeJoinFmt: (date: string, time: string) => string
  ctxPeriodFromToFmt: (from: string, to: string) => string
  ctxPeriodFromFmt: (from: string) => string
  ctxPeriodToFmt: (to: string) => string
  ctxPlaylistFmt: (name: string) => string
  ctxModeFmt: (category: string) => string
  /** Compteur intégré : "Matchs récents 12/47" / "Recent matches 12/47". */
  matchCounterCtxFmt: (label: string, n: number, total: number) => string
  /** Section « Objectifs » du scoreboard (CTF/Zones/Oddball) — V72-03. `cols` :
   *  libellé + tooltip d'en-tête par clé de colonne objectif. */
  objectives: {
    /**
     * LES DEUX CARTES EMPILEES (2026-10-14, deux vues dans une carte « Objectifs » avant).
     * `viewByPlayer` classe les joueurs grandeur par grandeur, `viewTeamTotals` oppose les
     * deux camps sur chacune : le titre de chaque carte est son seul intitule.
     */
    viewByPlayer: string
    viewTeamTotals: string
    /** Infobulle d'une barre de la grille : joueur, grandeur, valeur DEJA ecrite. */
    gridTipFmt: (player: string, metric: string, value: string) => string
    /** Infobulle d'un cote du face-a-face : equipe, grandeur, total DEJA ecrit. */
    duelTipFmt: (team: string, metric: string, value: string) => string
    cols: Record<string, { label: string; tooltip: string }>
  }
}
