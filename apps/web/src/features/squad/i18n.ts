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
    allTeammates: string
  }
  nav: {
    synergies: string
    contributions: string
    v2: string
  }
  selection: {
    placeholder: (count: number) => string
    prompt: string
  }
  filter: {
    experience: string
    playlist: string
    allExperiences: string
    allPlaylists: string
    analyse: string
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
    winRateVsHistoryTitle: string
    winRateVsHistorySession: string
    winRateVsHistoryHistory: string
    winRateVsHistoryBulletTitle: string
    winRateVsHistoryBulletParity: string
    winRateVsHistoryBulletZero: string
    mapPerfVsHistoryTitle: string
    mapPerfVsHistorySession: string
    mapPerfVsHistoryHistory: string
    outcomeSequenceTitle: string
  }
  history: {
    title: string
    description: string
    date: string
    map: string
    playlist: string
    mode: string
    outcome: string
    kda: string
    accuracy: string
    perf: string
    teamMmr: string
    session: string
    outcomeLabel: { win: string; loss: string; draw: string; dnf: string }
    prev: string
    next: string
    pageOf: (cur: number, total: number) => string
    totalRows: (n: number) => string
  }
  timeline: {
    title: string
    perf: string
    winRate: string
    teamMmr: string
    perfAxis: string
    mmrAxis: string
  }
  heatmap: {
    title: string
    pieceTier1: string
    pieceTier2: string
    pieceTier3: string
    pieceTier4: string
    pieceTier5: string
    noScore: string
  }
  impact: {
    title: string
    description: string
    colPlayer: string
    colScore: string
    colBadge: string
    badgeChampion: string
    badgeChampionShort: string
    badgeWeakLink: string
    badgeWeakLinkShort: string
    badgeStowaway: string
    badgeStowawayShort: string
    badgeNames: Record<string, string>
  }
  perMinute: {
    title: string
    description: string
    frags: string
    deaths: string
    assists: string
    suffix: string
  }
  synergyRadar: {
    title: string
    description: string
    axes: { combat: string; survival: string; support: string; score: string; objective: string; impact: string }
  }
  intensity: {
    title: string
    description: string
    toggleLabel: string
    allLabel: string
    zLabel: string
  }
  performanceCharts: {
    title: string
    description: string
    killsDeathsTitle: string
    killsLabel: string
    deathsLabel: string
    assistsTitle: string
    kdaTitle: string
    accuracyTitle: string
    avgLifeTitle: string
    performanceTitle: string
    maxSpreeTitle: string
    hsPerfectTitle: string
    hsLabel: string
    perfectLabel: string
  }
  weaponKills: {
    title: string
    description: string
  }
  firstEvents: {
    title: string
    description: string
    fragLabel: string
    deathLabel: string
    matchesSuffix: string
  }
  units: {
    perGame: string
  }
  medals: {
    title: string
    dominantCategory: string
    topMedals: string
    statsDistinct: string
    statsAvg: string
    statsPeak: string
    expandLabel: string
    collapseLabel: string
    noMedals: string
    categoryLabels: {
      multikill: string
      spree: string
      skill: string
      style: string
      mode: string
      proficiency: string
      other: string
    }
  }
  errors: {
    loadError: (message: string) => string
  }
}

const FR_TEXT: SquadText = {
  intlLocale: 'fr-FR',
  title: {
    teammates: 'Coéquipiers',
    allTeammates: 'Tous les coéquipiers',
  },
  nav: {
    synergies: 'Synergies',
    contributions: 'Contributions',
    v2: 'Vue Squad V2',
  },
  selection: {
    placeholder: (count) => `Rechercher parmi ${count} coéquipiers…`,
    prompt: 'Sélectionne jusqu\'à 3 coéquipiers pour analyser vos synergies.',
  },
  filter: {
    experience: 'Expérience',
    playlist: 'Playlist',
    allExperiences: 'Toutes les expériences',
    allPlaylists: 'Toutes les playlists',
    analyse: 'Analyser',
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
    winRateVsHistoryTitle: 'Taux de victoire vs historique par carte',
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'Historique',
    winRateVsHistoryBulletTitle: 'Winrate session vs historique — bullet chart',
    winRateVsHistoryBulletParity: 'Parité 50 %',
    winRateVsHistoryBulletZero: '0 % (toutes défaites)',
    mapPerfVsHistoryTitle: 'Performance par carte — Session vs Historique',
    mapPerfVsHistorySession: 'Session actuelle',
    mapPerfVsHistoryHistory: 'Historique',
    outcomeSequenceTitle: 'Séquence des matchs',
  },
  history: {
    title: 'Historique des matchs avec coéquipiers',
    description: 'Tous les matchs partagés sur les filtres actifs.',
    date: 'Date',
    map: 'Carte',
    playlist: 'Playlist',
    mode: 'Mode',
    outcome: 'Résultat',
    score: 'Score',
    kda: 'K/D/A',
    accuracy: 'Précision',
    perf: 'Perf.',
    teamMmr: 'MMR équipe',
    enemyMmr: 'MMR adv.',
    deltaMMR: 'Écart MMR',
    session: 'Session',
    outcomeLabel: { win: 'Victoire', loss: 'Défaite', draw: 'Égalité', dnf: 'DNF' },
    prev: '← Précédent',
    next: 'Suivant →',
    pageOf: (cur, total) => `Page ${cur} / ${total}`,
    totalRows: (n) => `${n} match${n > 1 ? 's' : ''}`,
  },
  timeline: {
    title: 'Performance d\'escouade par session',
    perf: 'Perf. escouade',
    winRate: 'Taux de victoire',
    teamMmr: 'MMR équipe',
    perfAxis: 'Perf / Win %',
    mmrAxis: 'MMR',
  },
  heatmap: {
    title: 'Performance par joueur × carte',
    pieceTier1: 'Excellente',
    pieceTier2: 'Bonne',
    pieceTier3: 'Moyenne',
    pieceTier4: 'Sous-moyenne',
    pieceTier5: 'Faible',
    noScore: 'Pas de score',
  },
  impact: {
    title: 'Impact des coéquipiers',
    description: 'Tableau matriciel des badges obtenus, par joueur × match. Colonnes agrégat triées par compte décroissant.',
    colPlayer: 'Joueur',
    colScore: 'Score',
    colBadge: 'Rang',
    badgeChampion: 'Champion (rang #1)',
    badgeChampionShort: 'Champion',
    badgeWeakLink: 'Maillon faible (rang dernier, score négatif)',
    badgeWeakLinkShort: 'Maillon faible',
    badgeStowaway: 'Passager clandestin (rang dernier, score positif ou nul)',
    badgeStowawayShort: 'Passager clandestin',
    badgeNames: {
      first_blood: 'Premier sang',
      clutch_finisher: 'Clutch',
      last_casualty: 'Boulet (dernière mort)',
      last_group_kill: 'Touriste (premier kill tardif)',
      first_group_death: 'Première victime',
      silent_hero: 'Héros silencieux',
      false_brother: 'Faux-frère',
      top_killer: 'Bourreau (top kills)',
    },
  },
  perMinute: {
    title: 'Stats par minute — Frags / Morts / Assists',
    description: 'Cadence par joueur sur le scope filtré. Les morts s\'affichent sous l\'axe (couleur joueur atténuée).',
    frags: 'Frags/min',
    deaths: 'Morts/min',
    assists: 'Assists/min',
    suffix: ' /min',
  },
  synergyRadar: {
    title: 'Radar synergie — 6 axes par joueur',
    description: 'Profil de participation calculé sur les matchs où tous les coéquipiers sélectionnés étaient présents. Lignes seules (pas d\'aire), 4 profils superposés max.',
    axes: {
      combat: 'Combat',
      survival: 'Survie',
      support: 'Support',
      score: 'Score',
      objective: 'Objectif',
      impact: 'Impact',
    },
  },
  intensity: {
    title: 'Intensité — kills par phase de match',
    description: 'Densité de kills par tranche de 10 % de la durée du match. Bascule entre toute l\'équipe et un joueur précis.',
    toggleLabel: 'Filtrer par joueur :',
    allLabel: 'Toute l\'équipe',
    zLabel: 'Cadence',
  },
  performanceCharts: {
    title: 'Performance escouade par match',
    description: 'Time-series alignée sur les matchs où tous les coéquipiers étaient présents. 1 ligne par joueur, couleurs cohérentes avec la pill et le multiselect.',
    killsDeathsTitle: 'Frags / Morts',
    killsLabel: 'Frags',
    deathsLabel: 'Morts',
    assistsTitle: 'Assistances',
    kdaTitle: 'FDA',
    accuracyTitle: 'Précision',
    avgLifeTitle: 'Durée de vie moyenne',
    performanceTitle: 'Performance',
    maxSpreeTitle: 'Folie meurtrière max',
    hsPerfectTitle: 'Tirs à la tête & Frags parfaits',
    hsLabel: 'Tirs à la tête',
    perfectLabel: 'Frags parfaits',
  },
  weaponKills: {
    title: 'Kills par arme — comparatif',
    description: 'Kills cumulés par arme sur les matchs partagés. Tri ASC : armes peu utilisées en haut, principales en bas.',
  },
  firstEvents: {
    title: 'Premier frag / première mort — chronologie',
    description: 'Histogramme butterfly : bins de 15 s. Frags positifs en haut, morts négatives en bas (couleur joueur atténuée).',
    fragLabel: 'Premier frag',
    deathLabel: 'Première mort',
    matchesSuffix: 'matchs',
  },
  units: {
    perGame: '/partie',
  },
  medals: {
    title: 'Médailles — Résumé de l\'escouade',
    dominantCategory: 'Catégorie dominante',
    topMedals: 'Top médailles',
    statsDistinct: 'Types distincts',
    statsAvg: 'Moy./match',
    statsPeak: 'Pic',
    expandLabel: 'Voir toutes les médailles',
    collapseLabel: 'Réduire',
    noMedals: 'Aucune médaille disponible pour cette sélection.',
    categoryLabels: {
      multikill: 'Multi-kills',
      spree: 'Séries',
      skill: 'Compétence',
      style: 'Style',
      mode: 'Mode',
      proficiency: 'Maîtrise',
      other: 'Autres',
    },
  },
  errors: {
    loadError: (message) => `Erreur : ${message}`,
  },
}

const EN_TEXT: SquadText = {
  intlLocale: 'en-US',
  title: {
    teammates: 'Teammates',
    allTeammates: 'All teammates',
  },
  nav: {
    synergies: 'Synergies',
    contributions: 'Contributions',
    v2: 'Squad V2 view',
  },
  selection: {
    placeholder: (count) => `Search among ${count} teammates…`,
    prompt: 'Pick up to 3 teammates to analyze your synergies.',
  },
  filter: {
    experience: 'Experience',
    playlist: 'Playlist',
    allExperiences: 'All experiences',
    allPlaylists: 'All playlists',
    analyse: 'Analyse',
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
    winRateVsHistoryTitle: 'Win rate vs history by map',
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'All time',
    winRateVsHistoryBulletTitle: 'Session winrate vs history — bullet chart',
    winRateVsHistoryBulletParity: '50% parity',
    winRateVsHistoryBulletZero: '0% (all losses)',
    mapPerfVsHistoryTitle: 'Performance per map — Session vs History',
    mapPerfVsHistorySession: 'Current session',
    mapPerfVsHistoryHistory: 'History',
    outcomeSequenceTitle: 'Match sequence',
  },
  history: {
    title: 'Match history with teammates',
    description: 'All shared matches matching the active filters.',
    date: 'Date',
    map: 'Map',
    playlist: 'Playlist',
    mode: 'Mode',
    outcome: 'Result',
    score: 'Score',
    kda: 'K/D/A',
    accuracy: 'Accuracy',
    perf: 'Perf.',
    teamMmr: 'Team MMR',
    enemyMmr: 'Enemy MMR',
    deltaMMR: 'MMR Gap',
    session: 'Session',
    outcomeLabel: { win: 'Win', loss: 'Loss', draw: 'Tie', dnf: 'DNF' },
    prev: '← Previous',
    next: 'Next →',
    pageOf: (cur, total) => `Page ${cur} / ${total}`,
    totalRows: (n) => `${n} match${n > 1 ? 'es' : ''}`,
  },
  timeline: {
    title: 'Squad performance by session',
    perf: 'Squad perf',
    winRate: 'Win rate',
    teamMmr: 'Team MMR',
    perfAxis: 'Perf / Win %',
    mmrAxis: 'MMR',
  },
  heatmap: {
    title: 'Performance per player × map',
    pieceTier1: 'Excellent',
    pieceTier2: 'Good',
    pieceTier3: 'Average',
    pieceTier4: 'Below average',
    pieceTier5: 'Poor',
    noScore: 'No score',
  },
  impact: {
    title: 'Teammates impact',
    description: 'Player × match badge matrix. Aggregate columns sorted by decreasing count.',
    colPlayer: 'Player',
    colScore: 'Score',
    colBadge: 'Rank',
    badgeChampion: 'Champion (rank #1)',
    badgeChampionShort: 'Champion',
    badgeWeakLink: 'Weak link (last rank, negative score)',
    badgeWeakLinkShort: 'Weak link',
    badgeStowaway: 'Stowaway (last rank, non-negative score)',
    badgeStowawayShort: 'Stowaway',
    badgeNames: {
      first_blood: 'First blood',
      clutch_finisher: 'Clutch',
      last_casualty: 'Last casualty',
      last_group_kill: 'Late starter',
      first_group_death: 'First down',
      silent_hero: 'Silent hero',
      false_brother: 'False brother',
      top_killer: 'Top killer',
    },
  },
  perMinute: {
    title: 'Per-minute stats — Frags / Deaths / Assists',
    description: 'Per-player cadence over the filtered scope. Deaths render below the axis (muted player color).',
    frags: 'Frags/min',
    deaths: 'Deaths/min',
    assists: 'Assists/min',
    suffix: ' /min',
  },
  synergyRadar: {
    title: 'Synergy radar — 6 axes per player',
    description: 'Participation profile computed on matches where all selected teammates were present. Lines only (no fill), max 4 overlaid profiles.',
    axes: {
      combat: 'Combat',
      survival: 'Survival',
      support: 'Support',
      score: 'Score',
      objective: 'Objective',
      impact: 'Impact',
    },
  },
  intensity: {
    title: 'Intensity — kills per match phase',
    description: 'Kill density per 10% slice of match duration. Toggle between the whole team and a specific player.',
    toggleLabel: 'Filter by player:',
    allLabel: 'Whole team',
    zLabel: 'Cadence',
  },
  performanceCharts: {
    title: 'Squad performance per match',
    description: 'Time-series aligned on matches where every teammate was present. One line per player, colors mirror the active-player pill and multiselect.',
    killsDeathsTitle: 'Frags / Deaths',
    killsLabel: 'Frags',
    deathsLabel: 'Deaths',
    assistsTitle: 'Assists',
    kdaTitle: 'KDA',
    accuracyTitle: 'Accuracy',
    avgLifeTitle: 'Avg lifespan',
    performanceTitle: 'Performance',
    maxSpreeTitle: 'Max killing spree',
    hsPerfectTitle: 'Headshots & Perfect kills',
    hsLabel: 'Headshots',
    perfectLabel: 'Perfect kills',
  },
  weaponKills: {
    title: 'Weapon kills — comparison',
    description: 'Cumulative kills per weapon over shared matches. Sorted ASC: rare weapons on top, primaries at the bottom.',
  },
  firstEvents: {
    title: 'First frag / first death — chronology',
    description: 'Butterfly histogram with 15 s bins. Frags above the axis, deaths below (muted player color).',
    fragLabel: 'First frag',
    deathLabel: 'First death',
    matchesSuffix: 'matches',
  },
  units: {
    perGame: '/game',
  },
  medals: {
    title: 'Medals — Squad summary',
    dominantCategory: 'Dominant category',
    topMedals: 'Top medals',
    statsDistinct: 'Distinct types',
    statsAvg: 'Avg/match',
    statsPeak: 'Peak',
    expandLabel: 'Show all medals',
    collapseLabel: 'Collapse',
    noMedals: 'No medals available for this selection.',
    categoryLabels: {
      multikill: 'Multi-kills',
      spree: 'Spree',
      skill: 'Skill',
      style: 'Style',
      mode: 'Mode',
      proficiency: 'Proficiency',
      other: 'Other',
    },
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
