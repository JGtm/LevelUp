/**
 * i18n.ts — strings UI pour la feature Tactique.
 *
 * Convention : dictionnaire `Record<Locale, TacticalText>` centralisé,
 * accès via `getTacticalText()`.
 */
import type { ManifestLocale } from '@/lib/i18n/format'

export type TacticalLocale = ManifestLocale

export interface TacticalText {
  // Page & section titles
  pageTitle: string
  plansTitle: string
  plansGridFooter: string
  analysisPlansTitle: (mapName: string, question: string) => string
  selectedCellTitle: string
  coordinationTitle: string

  // KPI labels
  matchesRetained: string
  coverage: string
  tradingDeaths: string
  tradeAfterDeath: string
  isolatedDeaths: string
  medianDistanceToTeammate: string

  // Question selector options
  questionWhere: string
  questionDeaths: string
  questionKills: string
  questionWin: string
  questionTime: string
  questionRoutes: string
  questionIsolated: string

  // Question units
  unitDeathsPerMatch: string
  unitKillsPerMatch: string
  unitWinDelta: string
  unitSecondsPerMatch: string
  unitEmpty: string

  // Question descriptions (for titles)
  descWhere: string
  descDeaths: string
  descKills: string
  descWin: string
  descTime: string
  descRoutes: string
  descIsolated: string

  // "Who" segment
  whoMe: string
  whoSquad: string
  whoEnemies: string

  // Spawn segment
  spawnAll: string
  spawnLabel: string

  // Legend labels
  legendFrom: string
  legendTo: string
  legendScale: string

  // Plan card footer
  planFooterQuantile: string
  planFooterDivergent: string
  planFooterRoutes: string

  // Cell card labels
  cellValue: string
  cellNoSelection: string
  cellContributors: string
  cellDateColumn: string
  cellOutcomeColumn: string
  cellOpenColumn: string
  cellMoreMatches: (count: number) => string

  // Coordination card labels
  coordinationTrade: string
  coordinationIso: string
  coordinationDistance: string
  coordinationTradeHelp: string
  coordinationIsoHelp: string
  coordinationDistanceHelp: string
  coordinationDistributionLabel: string
  coordinationThresholdLabel: string

  // States & messages
  loadingMessage: string
  processingMatches: (count: number) => string
  unavailableMatches: (count: number) => string
  emptyState: string
  errorState: string
  retryButton: string

  // Canvas aria labels
  planCanvasLabel: (mapName: string, question: string) => string
  distributionChartLabel: string
}

export const TACTICAL_TEXT: Record<TacticalLocale, TacticalText> = {
  fr: {
    pageTitle: 'Tactique',
    plansTitle: 'Cartes jouées',
    plansGridFooter:
      'Les cartes sous 10 matchs sont affichées mais pas ouvrables : sous ce plancher, une zone chaude ne distingue pas une habitude d\'un hasard.',
    analysisPlansTitle: (mapName, question) => `Plan de ${mapName} — ${question}`,
    selectedCellTitle: 'Cellule sélectionnée',
    coordinationTitle: 'Coordination d\'équipe',

    matchesRetained: 'Matchs retenus',
    coverage: 'Couverture',
    tradingDeaths: 'Morts en isolement',
    tradeAfterDeath: 'Échange après ma mort',
    isolatedDeaths: 'Morts en isolement',
    medianDistanceToTeammate: 'Distance médiane à l\'équipier',

    questionWhere: 'La question',
    questionDeaths: 'Où je meurs',
    questionKills: 'Où je tue',
    questionWin: 'Où je gagne',
    questionTime: 'Où je passe mon temps',
    questionRoutes: 'Mes routes de spawn',
    questionIsolated: 'Où je meurs isolé',

    unitDeathsPerMatch: 'morts par match',
    unitKillsPerMatch: 'kills par match',
    unitWinDelta: 'écart V moins D',
    unitSecondsPerMatch: 's par match',
    unitEmpty: '',

    descWhere: 'Où je passe mon temps',
    descDeaths: 'Où je meurs',
    descKills: 'Où je tue',
    descWin: 'Où je gagne',
    descTime: 'Où je passe mon temps',
    descRoutes: 'Mes routes de spawn',
    descIsolated: 'Où je meurs isolé',

    whoMe: 'Moi',
    whoSquad: 'Escouade',
    whoEnemies: 'Adversaires',

    spawnAll: 'Tous',
    spawnLabel: 'Spawn de départ',

    legendFrom: '0',
    legendTo: 'Valeur',
    legendScale: 'Échelle',

    planFooterQuantile:
      'Échelle quantile p50 vers p95, saturée au-delà. Une cellule jamais atteinte reste vide — elle n\'est pas peinte en froid.',
    planFooterDivergent:
      'Échelle divergente : le vert est joué davantage en victoire, le rouge en défaite. Sous 3 matchs distincts la cellule ne s\'affiche pas.',
    planFooterRoutes:
      '15 premières secondes de chaque vie, une ligne par vie à 17 % d\'opacité : la route de consensus apparaît par empilement, pas par densité.',

    cellValue: 'Valeur',
    cellNoSelection: 'Clique une zone chaude du plan.',
    cellContributors: 'matchs distincts ont alimenté cette cellule.',
    cellDateColumn: 'Date',
    cellOutcomeColumn: 'Résultat',
    cellOpenColumn: 'Ouvrir',
    cellMoreMatches: (count) => `${count} autres matchs comptent dans la cellule mais ne te sont pas ouverts : le calque est anonyme, la liste ne l\'est pas.`,

    coordinationTrade: 'Échange après ma mort',
    coordinationIso: 'Morts en isolement',
    coordinationDistance: 'Distance médiane à l\'équipier',
    coordinationTradeHelp: 'Un équipier tue mon tueur dans les 5 s',
    coordinationIsoHelp: 'Plus proche équipier au-delà de 25 m',
    coordinationDistanceHelp: 'Distance à l\'équipier le plus proche, au moment de ma mort',
    coordinationDistributionLabel: 'Distribution de la distance à l\'équipier le plus proche au moment de la mort',
    coordinationThresholdLabel: '25 m',

    loadingMessage: 'Chargement…',
    processingMatches: (count) => `Traitement en cours : ${count} matchs`,
    unavailableMatches: (count) => `Données non disponibles pour ${count} matchs`,
    emptyState: 'Aucune donnée pour cette sélection.',
    errorState: 'Erreur lors du chargement des données.',
    retryButton: 'Réessayer',

    planCanvasLabel: (mapName, question) => `Plan de ${mapName} avec le calque ${question}`,
    distributionChartLabel: 'Distribution des distances à l\'équipier',
  },
  en: {
    pageTitle: 'Tactics',
    plansTitle: 'Maps Played',
    plansGridFooter:
      'Maps under 10 matches are displayed but not openable: below this threshold, a hot zone does not distinguish a habit from chance.',
    analysisPlansTitle: (mapName, question) => `${mapName} Plan — ${question}`,
    selectedCellTitle: 'Selected Cell',
    coordinationTitle: 'Team Coordination',

    matchesRetained: 'Matches Retained',
    coverage: 'Coverage',
    tradingDeaths: 'Deaths While Isolated',
    tradeAfterDeath: 'Trade After My Death',
    isolatedDeaths: 'Deaths While Isolated',
    medianDistanceToTeammate: 'Median Distance to Teammate',

    questionWhere: 'Question',
    questionDeaths: 'Where I Die',
    questionKills: 'Where I Kill',
    questionWin: 'Where I Win',
    questionTime: 'Where I Spend My Time',
    questionRoutes: 'My Spawn Routes',
    questionIsolated: 'Where I Die Isolated',

    unitDeathsPerMatch: 'deaths per match',
    unitKillsPerMatch: 'kills per match',
    unitWinDelta: 'win delta',
    unitSecondsPerMatch: 's per match',
    unitEmpty: '',

    descWhere: 'Where I spend my time',
    descDeaths: 'Where I die',
    descKills: 'Where I kill',
    descWin: 'Where I win',
    descTime: 'Where I spend my time',
    descRoutes: 'My spawn routes',
    descIsolated: 'Where I die isolated',

    whoMe: 'Me',
    whoSquad: 'Squad',
    whoEnemies: 'Enemies',

    spawnAll: 'All',
    spawnLabel: 'Starting Spawn',

    legendFrom: '0',
    legendTo: 'Value',
    legendScale: 'Scale',

    planFooterQuantile:
      'Quantile scale p50 to p95, saturated beyond. A cell never reached remains empty — it is not painted cold.',
    planFooterDivergent:
      'Divergent scale: green is played more in wins, red in losses. Below 3 distinct matches the cell is not displayed.',
    planFooterRoutes:
      'First 15 seconds of each life, one line per life at 17% opacity: the consensus route appears by layering, not by density.',

    cellValue: 'Value',
    cellNoSelection: 'Click a hot zone on the plan.',
    cellContributors: 'distinct matches have fed this cell.',
    cellDateColumn: 'Date',
    cellOutcomeColumn: 'Result',
    cellOpenColumn: 'Open',
    cellMoreMatches: (count) => `${count} other matches count in the cell but are not opened for you: the layer is anonymous, the list is not.`,

    coordinationTrade: 'Trade After My Death',
    coordinationIso: 'Deaths While Isolated',
    coordinationDistance: 'Median Distance to Teammate',
    coordinationTradeHelp: 'A teammate kills my killer within 5 s',
    coordinationIsoHelp: 'Nearest teammate beyond 25 m',
    coordinationDistanceHelp: 'Distance to nearest teammate at the moment of my death',
    coordinationDistributionLabel: 'Distribution of distance to nearest teammate at moment of death',
    coordinationThresholdLabel: '25 m',

    loadingMessage: 'Loading…',
    processingMatches: (count) => `Processing: ${count} matches`,
    unavailableMatches: (count) => `Data unavailable for ${count} matches`,
    emptyState: 'No data for this selection.',
    errorState: 'Error loading data.',
    retryButton: 'Retry',

    planCanvasLabel: (mapName, question) => `${mapName} plan with ${question} layer`,
    distributionChartLabel: 'Distribution of distances to teammates',
  },
}

/**
 * getTacticalText — accède aux strings pour la locale courante.
 */
export function getTacticalText(locale: TacticalLocale): TacticalText {
  return TACTICAL_TEXT[locale]
}
