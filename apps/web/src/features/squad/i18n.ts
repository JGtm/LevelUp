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

import type { Locale } from '@/lib/i18n/locale'

export interface SquadText {
  intlLocale: string
  title: {
    teammates: string
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
    /** Message court pour un bloc non-graphe vide (tape, table, scoreboard). */
    noBlockData: string
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
    winRateVsHistoryBulletCounts: (session: number, history?: number) => string
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
    score: string
    winRateHist: string
    winProb: string
    kda: string
    accuracy: string
    perf: string
    duration: string
    teamMmr: string
    enemyMmr: string
    deltaMMR: string
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
    badgeDescriptions: Record<string, string>
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
    tooltip: { impact: string; combat: string; survival: string; support: string; score: string; objective: string; glossaryLink: string }
  }
  intensity: {
    title: string
    description: string
    toggleLabel: string
    allLabel: string
    zLabel: string
  }
  efficiencySeries: {
    title: string
    /** Titre en mode mono-métrique (sans résistance, ex. Halo 5). */
    rendementTitle: string
    description: string
    rendementLabel: string
    resistanceLabel: string
    refLabel: string
    noData: string
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
    rankTitle: string
    mmrLabel: string
    fragBreakdownTitle: string
  }
  /** « Écart cumulé au FDA attendu » (D3/D7 — différentiel FDA réel vs attendu par joueur). */
  fdaGap: {
    title: string
    /** Caption de la rangée de pastilles KPI (écart moyen par match). */
    averageCaption: string
  }
  weaponKills: {
    title: string
    description: string
    /** Ligne agrégée des armes gun au-delà du top-N dans « Outils de destruction ». */
    otherWeapons: string
  }
  /** Comparatif « Précision par rôle » multi-joueurs (Halo 5) : barres groupées horizontales (1 barre/joueur/rôle, longueur = précision %). */
  weaponAccuracy: {
    title: string
    /** Libellé « Tirs » (contexte tooltip). */
    shotsLabel: string
  }
  killMechanics: {
    title: string
    labels: { assassination: string; ground_pound: string; shoulder_bash: string }
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
  },
  selection: {
    placeholder: (count) => `Rechercher parmi ${count} coéquipiers…`,
    prompt: 'Sélectionne jusqu\'à 3 coéquipiers pour analyser vos synergies.',
  },
  filter: {
    experience: 'Expérience',
    playlist: 'Sélection',
    allExperiences: 'Toutes les expériences',
    allPlaylists: 'Toutes les sélections',
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
    winPct: 'Vict.%',
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
    noBlockData: 'Aucune donnée pour cette sélection.',
  },
  synergies: {
    description: 'Comparaison de tes stats avec chaque coéquipier sur les matchs joués ensemble.',
  },
  contributions: {
    description: 'Profil de contribution normalisé pour chaque coéquipier sélectionné.',
  },
  charts: {
    hsPkTitle: 'Tirs à la tête & Frags parfaits par match',
    timelineTitle: 'Évolution des performances en escouade',
    timelinePerfName: 'Perf. moyenne',
    timelineWinRateName: 'Taux de victoire',
    timelinePerfAxis: 'Score perf.',
    timelineWinRateAxis: 'Taux de victoire',
    heatmapTitle: 'Taux de victoire par carte (escouade)',
    heatmapWinAxis: 'Taux de victoire (%)',
    heatmapMatchesLabel: 'Matchs',
    winRateVsHistoryTitle: 'Taux de victoire vs historique par carte',
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'Historique',
    winRateVsHistoryBulletTitle: 'Taux de victoire session vs historique',
    winRateVsHistoryBulletParity: 'Parité 50 %',
    winRateVsHistoryBulletZero: '0 % (toutes défaites)',
    winRateVsHistoryBulletCounts: (session, history) =>
      `Session : ${session} ${session <= 1 ? 'partie' : 'parties'} · Historique : ${
        history === undefined ? '—' : `${history} ${history <= 1 ? 'partie' : 'parties'}`
      }`,
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
    playlist: 'Sélection',
    mode: 'Mode',
    outcome: 'Résultat',
    score: 'Score',
    winRateHist: 'Taux hist.',
    winProb: 'Prob. vic.',
    kda: 'K/D/A',
    accuracy: 'Précision',
    perf: 'Perf.',
    duration: 'Durée',
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
    perfAxis: 'Perf / Vict. %',
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
      clutch_finisher: 'Décisif',
      last_casualty: 'Boulet (dernière mort)',
      last_group_kill: 'Touriste (premier frag tardif)',
      first_group_death: 'Première victime',
      silent_hero: 'Héros silencieux',
      false_brother: 'Faux-frère',
      top_killer: 'Bourreau (top frags)',
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    badgeDescriptions: {
      first_blood: 'Premier frag du match, toutes équipes confondues',
      clutch_finisher: 'Dernier frag réalisé par un joueur de l\'équipe gagnante',
      last_casualty: 'Dernière mort subie par un joueur de l\'équipe perdante',
      last_group_kill: 'Joueur de l\'équipe dont le premier frag arrive le plus tardivement',
      first_group_death: 'Première mort subie par un membre de l\'équipe',
      silent_hero: 'Joueur (hors Bourreau) avec le plus d\'assists et le moins de morts',
      false_brother: 'Joueur (hors Bourreau) avec le plus de morts et le moins d\'assists',
      top_killer: 'Joueur avec le plus grand nombre de frags du match',
      top_gun: 'Premier membre de l\'équipe à atteindre 10 frags',
      kamikaze: 'Joueur le plus tué dans les 1,5 s qui suivent ses frags',
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
    title: 'Radar synergie',
    description: 'Profil de participation calculé sur les matchs où tous les coéquipiers sélectionnés étaient présents. Lignes seules (pas d\'aire), 4 profils superposés max.',
    axes: {
      combat: 'Combat',
      survival: 'Survie',
      support: 'Support',
      score: 'Score',
      objective: 'Objectif',
      impact: 'Impact',
    },
    tooltip: {
      impact: 'Rendement offensif — 225 × (frags + ass/3) / dégâts. P80 = 0,83.',
      combat: 'Frags + tirs à la tête + frags parfaits, pondérés par la précision.',
      survival: 'Résistance défensive — dégâts / (225 × morts). P80 = 1,59.',
      support: 'Assists × 50.',
      score: 'Score personnel par minute jouée. P80 ≈ 195/min.',
      objective: 'Points d\'objectif (PersonalScoreAwards).',
      glossaryLink: '→ Glossaire',
    },
  },
  intensity: {
    title: 'Intensité',
    description: 'Densité de frags par tranche de 10 % de la durée du match. Bascule entre toute l\'équipe et un joueur précis.',
    toggleLabel: 'Filtrer par joueur :',
    allLabel: 'Toute l\'équipe',
    zLabel: 'Cadence',
  },
  efficiencySeries: {
    title: 'Rendement & Résistance',
    rendementTitle: 'Rendement',
    description: 'Dégâts / frag (trait plein) = dégâts infligés / frags. Dégâts / mort (pointillé) = dégâts subis / morts. Repère 225 = 1 vie de Spartan : pour les frags, au plus proche de 225, au plus efficace ; pour les morts, au-dessus de 225 = bonne résistance.',
    rendementLabel: 'Dégâts / frag',
    resistanceLabel: 'Dégâts / mort',
    refLabel: '1 vie ({{HP}})',
    noData: 'Aucune donnée d\'efficacité disponible.',
  },
  performanceCharts: {
    title: 'Performance',
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
    rankTitle: 'Rang & MMR équipe',
    mmrLabel: 'MMR équipe',
    fragBreakdownTitle: 'Répartition des frags',
  },
  fdaGap: {
    title: 'Écart cumulé au FDA attendu',
    averageCaption: 'Écart moyen par match',
  },
  weaponKills: {
    title: 'Outils de destruction',
    description: 'Frags cumulés par arme sur les matchs partagés. Tri ASC : armes peu utilisées en haut, principales en bas.',
    otherWeapons: 'Autres armes',
  },
  weaponAccuracy: {
    title: 'Précision par rôle',
    shotsLabel: 'Tirs',
  },
  killMechanics: {
    title: 'Mécaniques de kill',
    labels: { assassination: 'Assassinats', ground_pound: 'Frappes au sol', shoulder_bash: 'Charges d\'épaule' },
  },
  firstEvents: {
    title: 'Premier frag / première mort',
    description: 'Histogramme butterfly : bins de 15 s. Frags positifs en haut, morts négatives en bas (couleur joueur atténuée).',
    fragLabel: 'Premier frag',
    deathLabel: 'Première mort',
    matchesSuffix: 'matchs',
  },
  units: {
    perGame: '/match',
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
      multikill: 'Multi-frags',
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
    noBlockData: 'No data for this selection.',
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
    winRateVsHistoryBulletTitle: 'Session winrate vs history',
    winRateVsHistoryBulletParity: '50% parity',
    winRateVsHistoryBulletZero: '0% (all losses)',
    winRateVsHistoryBulletCounts: (session, history) =>
      `Session: ${session} ${session <= 1 ? 'game' : 'games'} · History: ${
        history === undefined ? '—' : `${history} ${history <= 1 ? 'game' : 'games'}`
      }`,
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
    winRateHist: 'Hist. win%',
    winProb: 'Win prob.',
    kda: 'K/D/A',
    accuracy: 'Accuracy',
    perf: 'Perf.',
    duration: 'Duration',
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
      top_gun: 'Top Gun',
      kamikaze: 'Kamikaze',
    },
    badgeDescriptions: {
      first_blood: 'First kill of the match, across all teams',
      clutch_finisher: 'Last kill dealt by a player from the winning team',
      last_casualty: 'Last death suffered by a player from the losing team',
      last_group_kill: 'Squad member whose first kill came latest in the match',
      first_group_death: 'First death suffered by a squad member',
      silent_hero: 'Winner (excl. top killer) with most assists and fewest deaths',
      false_brother: 'Loser (excl. top killer) with most deaths and fewest assists',
      top_killer: 'Player with the highest kill count in the match',
      top_gun: 'First squad member to reach 10 kills',
      kamikaze: 'Player killed within 1.5 s after one of their own frags (most frequent in the match)',
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
    title: 'Synergy radar',
    description: 'Participation profile computed on matches where all selected teammates were present. Lines only (no fill), max 4 overlaid profiles.',
    axes: {
      combat: 'Combat',
      survival: 'Survival',
      support: 'Support',
      score: 'Score',
      objective: 'Objective',
      impact: 'Impact',
    },
    tooltip: {
      impact: 'Offensive conversion — 225 × (kills + ass/3) / damage. P80 = 0.83.',
      combat: 'Kills + headshots + perfect kills, weighted by accuracy.',
      survival: 'Defensive resistance — damage / (225 × deaths). P80 = 1.59.',
      support: 'Assists × 50.',
      score: 'Personal score per minute played. P80 ≈ 195/min.',
      objective: 'Objective points (PersonalScoreAwards).',
      glossaryLink: '→ Glossary',
    },
  },
  intensity: {
    title: 'Intensity',
    description: 'Kill density per 10% slice of match duration. Toggle between the whole team and a specific player.',
    toggleLabel: 'Filter by player:',
    allLabel: 'Whole team',
    zLabel: 'Cadence',
  },
  efficiencySeries: {
    title: 'Offensive & Defensive Efficiency',
    rendementTitle: 'Offensive Efficiency',
    description: 'Damage / kill (solid) = damage dealt / kills. Damage / death (dashed) = damage taken / deaths. Reference 225 = one Spartan life: for kills, closer to 225 is more efficient; for deaths, above 225 means good resistance.',
    rendementLabel: 'Damage / kill',
    resistanceLabel: 'Damage / death',
    refLabel: '1 life ({{HP}})',
    noData: 'No efficiency data available.',
  },
  performanceCharts: {
    title: 'Performance',
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
    rankTitle: 'Rank & Team MMR',
    mmrLabel: 'Team MMR',
    fragBreakdownTitle: 'Kill type distribution',
  },
  fdaGap: {
    title: 'Cumulative KDA gap to expected',
    averageCaption: 'Average gap per match',
  },
  weaponKills: {
    title: 'Tools of destruction',
    description: 'Cumulative kills per weapon over shared matches. Sorted ASC: rare weapons on top, primaries at the bottom.',
    otherWeapons: 'Other weapons',
  },
  weaponAccuracy: {
    title: 'Accuracy by role',
    shotsLabel: 'Shots',
  },
  killMechanics: {
    title: 'Kill mechanics',
    labels: { assassination: 'Assassinations', ground_pound: 'Ground Pounds', shoulder_bash: 'Shoulder Bashes' },
  },
  firstEvents: {
    title: 'First frag / first death',
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

const DICTS: Record<Locale, SquadText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

/** Retourne le dictionnaire pour la locale demandée (fallback fr). */
export function getSquadText(locale: Locale | string | undefined): SquadText {
  if (locale === 'en') return DICTS.en
  return DICTS.fr
}

// Exports nommés pour permettre des tests de parité FR/EN sans réimporter le helper.
export { FR_TEXT, EN_TEXT }
