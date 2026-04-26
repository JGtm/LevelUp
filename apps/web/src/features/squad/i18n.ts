/**
 * i18n.ts — Dictionnaire FR/EN des strings UI de la feature Escouade.
 *
 * Frontière stricte avec les mappings TOML multi-titres :
 *  - Les libellés métier (FieldKey, assets, outcomes) restent dans les TOML
 *    et passent par useFieldLabel / useAssetLabel / useOutcomeLabel.
 *  - Ce fichier ne contient que des strings UI non-titre-bound : titres de
 *    cartes, navigation, empty states, boutons, descriptions, unités.
 *
 * Pattern aligné avec features/compare/i18n.ts et features/home/*.i18n.ts.
 */

export type SquadLocale = 'fr' | 'en'

export interface SquadText {
  intlLocale: string
  title: {
    teammates: string
    statsWith: string
    allTeammates: string
  }
  nav: {
    synergies: string
    contributions: string
  }
  selection: {
    placeholder: (count: number) => string
    prompt: string
  }
  session: {
    label: string
    squad: string
    prev: string
    next: string
    all: string
    reset: string
  }
  table: {
    gamertag: string
    matches: string
    wins: string
    winPct: string
    kd: string
    lastSeen: string
    actions: string
    openCompare: string
    withTeammate: (gamertag: string) => string
  }
  empty: {
    noSelectionTitle: string
    noSelectionDescription: string
    invalidSelectionTitle: string
    invalidSelectionDescription: string
    noChartTitle: string
    noChartDescription: string
    noDataTitle: string
    noDataDescription: string
  }
  synergies: {
    description: string
  }
  contributions: {
    description: string
  }
  charts: {
    hsPkTitle: string
    timelineTitle: string
    timelinePerfName: string
    timelineWinRateName: string
    timelinePerfAxis: string
    timelineWinRateAxis: string
    heatmapTitle: string
    heatmapWinAxis: string
    heatmapMatchesLabel: string
  }
  units: {
    perGame: string
  }
  errors: {
    loadError: (message: string) => string
  }
}

const FR_TEXT: SquadText = {
  intlLocale: 'fr-FR',
  title: {
    teammates: 'Coéquipiers',
    statsWith: 'Synergies avec les coéquipiers sélectionnés',
    allTeammates: 'Tous les coéquipiers',
  },
  nav: {
    synergies: 'Synergies',
    contributions: 'Contributions',
  },
  selection: {
    placeholder: (count) => `Rechercher parmi ${count} coéquipiers…`,
    prompt: 'Sélectionne jusqu\'à 3 coéquipiers pour analyser vos synergies.',
  },
  session: {
    label: 'Session',
    squad: 'Escouade',
    prev: 'Session précédente',
    next: 'Session suivante',
    all: '(toutes)',
    reset: '✕ Réinitialiser',
  },
  table: {
    gamertag: 'Gamertag',
    matches: 'Matchs',
    wins: 'Victoires',
    winPct: 'Win%',
    kd: 'K/D',
    lastSeen: 'Dernière rencontre',
    actions: 'Actions',
    openCompare: 'Face-à-face',
    withTeammate: (gamertag) => `Avec ${gamertag}`,
  },
  empty: {
    noSelectionTitle: 'Analyse de synergies',
    noSelectionDescription: 'Choisis 1 à 3 coéquipiers pour analyser les synergies de ton escouade.',
    invalidSelectionTitle: 'Aucune donnée commune',
    invalidSelectionDescription:
      'Les coéquipiers sélectionnés n\'ont pas joué de match avec toi sur la période filtrée.',
    noChartTitle: 'Graphique indisponible',
    noChartDescription: 'Le graphique n\'a pas pu être construit avec les données actuelles.',
    noDataTitle: 'Données d\'escouade indisponibles',
    noDataDescription:
      'Aucune réponse exploitable n\'a été renvoyée pour cette page. Vérifie les filtres ou la disponibilité des matchs partagés.',
  },
  synergies: {
    description: 'Comparaison de tes stats avec chaque coéquipier sur les matchs joués ensemble.',
  },
  contributions: {
    description: 'Profil de contribution normalisé pour chaque coéquipier sélectionné.',
  },
  charts: {
    hsPkTitle: 'Headshot & Perfect kills par partie',
    timelineTitle: 'Évolution des performances en escouade',
    timelinePerfName: 'Perf. moyenne',
    timelineWinRateName: 'Taux de victoire',
    timelinePerfAxis: 'Score perf.',
    timelineWinRateAxis: 'Taux de victoire',
    heatmapTitle: 'Taux de victoire par carte (escouade)',
    heatmapWinAxis: 'Win rate (%)',
    heatmapMatchesLabel: 'Matchs',
  },
  units: {
    perGame: '/partie',
  },
  errors: {
    loadError: (message) => `Erreur : ${message}`,
  },
}

const EN_TEXT: SquadText = {
  intlLocale: 'en-US',
  title: {
    teammates: 'Teammates',
    statsWith: 'Synergies with selected teammates',
    allTeammates: 'All teammates',
  },
  nav: {
    synergies: 'Synergies',
    contributions: 'Contributions',
  },
  selection: {
    placeholder: (count) => `Search among ${count} teammates…`,
    prompt: 'Pick up to 3 teammates to analyze your synergies.',
  },
  session: {
    label: 'Session',
    squad: 'Squad',
    prev: 'Previous session',
    next: 'Next session',
    all: '(all)',
    reset: '✕ Reset',
  },
  table: {
    gamertag: 'Gamertag',
    matches: 'Matches',
    wins: 'Wins',
    winPct: 'Win%',
    kd: 'K/D',
    lastSeen: 'Last seen',
    actions: 'Actions',
    openCompare: 'Head-to-head',
    withTeammate: (gamertag) => `With ${gamertag}`,
  },
  empty: {
    noSelectionTitle: 'Synergy analysis',
    noSelectionDescription: 'Pick 1 to 3 teammates to analyze the synergies of your squad.',
    invalidSelectionTitle: 'No shared data',
    invalidSelectionDescription:
      'The selected teammates have no shared matches with you in the filtered period.',
    noChartTitle: 'Chart unavailable',
    noChartDescription: 'The chart could not be built with the current data.',
    noDataTitle: 'Squad data unavailable',
    noDataDescription:
      'No usable response was returned for this page. Check filters or shared matches availability.',
  },
  synergies: {
    description: 'Comparison of your stats with each teammate on shared matches.',
  },
  contributions: {
    description: 'Normalized contribution profile for each selected teammate.',
  },
  charts: {
    hsPkTitle: 'Headshot & Perfect kills per game',
    timelineTitle: 'Squad performance over time',
    timelinePerfName: 'Avg. performance',
    timelineWinRateName: 'Win rate',
    timelinePerfAxis: 'Perf. score',
    timelineWinRateAxis: 'Win rate',
    heatmapTitle: 'Win rate by map (squad)',
    heatmapWinAxis: 'Win rate (%)',
    heatmapMatchesLabel: 'Matches',
  },
  units: {
    perGame: '/game',
  },
  errors: {
    loadError: (message) => `Error: ${message}`,
  },
}

const DICTS: Record<SquadLocale, SquadText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

/** Retourne le dictionnaire pour la locale demandée (fallback fr). */
export function getSquadText(locale: SquadLocale | string | undefined): SquadText {
  if (locale === 'en') return DICTS.en
  return DICTS.fr
}

// Exports nommés pour permettre des tests de parité FR/EN sans réimporter le helper.
export { FR_TEXT, EN_TEXT }
