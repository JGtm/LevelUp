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
    dynamique: string
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
    /** Option « composition exacte » (désactivée par défaut) + son explication. */
    exactComposition: string
    exactCompositionTitle: string
  }
  /** Bandeau de données partielles (chargements dégradés remontés par l'API). */
  dataIssues: {
    title: string
    teammateMatches: (detail: string) => string
    heatmapTeammate: (detail: string) => string
    mainTeamParticipants: string
    mapStats: string
    unknown: (code: string) => string
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
    winRateVsHistorySession: string
    winRateVsHistoryHistory: string
    winRateVsHistoryBulletTitle: string
    winRateVsHistoryBulletParity: string
    winRateVsHistoryBulletZero: string
    winRateVsHistoryBulletCounts: (session: number, history?: number) => string
    /** Tooltip d'aide : explique le suffixe « (n) » sur les libellés d'axe Y (I9). */
    winRateVsHistoryBulletMapCountTooltip: string
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
    /** aria-label/tooltip du lien « Ouvrir sur Halo Waypoint » (I19). */
    waypointAriaLabel: string
    /** aria-label du bouton de tri d'un en-tête « Trier par {col} » (I16). */
    sortByAriaLabel: (col: string) => string
    /** Tooltips d'en-tête de colonne (V72-04, icône ⓘ). */
    winRateHistTooltip: string
    winProbTooltip: string
    teamMmrTooltip: string
    enemyMmrTooltip: string
    deltaMmrTooltip: string
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
    /** Tooltips d'en-tête de colonne (V72-04, icône ⓘ). */
    colScoreTooltip: string
    colBadgeTooltip: string
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
    /** Préfixe de la valeur BRUTE affichée au survol, à côté de la valeur normalisée. */
    rawLabel: string
  }
  intensity: {
    title: string
    /** Sous-titre de la carte Intensité (profil médian + enveloppe). */
    subtitle: string
    /** Tooltip d'aide : courbe / zone d'irrégularité / repère / courbe d'équipe. */
    tooltip: string
    /** Libellé tooltip du trait médian. */
    medianLabel: string
    /** Libellé tooltip de la fourchette interquartile. */
    envelopeLabel: string
    /** Libellé du repère 10 % (activité uniforme). */
    refLabel: string
    /** Libellé de la courbe agrégée d'équipe (superposée à partir de 3 joueurs). */
    teamLabel: string
  }
  efficiencySeries: {
    /** Titre de la carte Rendement (écart à la frontière élite OC du titre). */
    rendementCardTitle: string
    /** Titre de la carte Résistance (écart à la frontière élite DR du titre). */
    resistanceCardTitle: string
    /** Aide ⓘ : comment lire les pistes empilées et le zéro = frontière élite. */
    help: string
    /** Nom de l'indicateur offensif au survol (« Rendement »). */
    offensiveMetric: string
    /** Nom de l'indicateur défensif au survol (« Résistance »). */
    defensiveMetric: string
    /** Libellé du repère au survol (« frontière élite »). */
    eliteBoundary: string
    /** Libellé des dégâts bruts offensifs au survol. */
    damageDealt: string
    /** Libellé des dégâts bruts défensifs au survol. */
    damageTaken: string
    /** Unité du ratio offensif au survol (« / frag effectif »). */
    perFrag: string
    /** Unité du ratio défensif au survol (« / mort »). */
    perDeath: string
    noData: string
  }
  performanceCharts: {
    title: string
    description: string
    killsDeathsTitle: string
    killsLabel: string
    deathsLabel: string
    /** Libellé de la série « Bonus » (assistances converties) du chart Frags / Morts. */
    bonusLabel: string
    /** Aide ⓘ de la série Bonus : ce que vaut une assistance dans le FDA (ADR 0006). */
    bonusInfo: string
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
  /** « Balance des dégâts cumulée » (P3 — dégâts nets ÷ PV-pour-tuer, cumulé par joueur). */
  netLives: {
    title: string
    /** Tooltip explicatif de la formule (jeton {{HP}} = barème du titre). */
    tooltip: string
  }
  /** « Écart d'engagement cumulé » (P4 — résidu pace_observed − team_expected × durée, par joueur). */
  engagementGap: {
    title: string
    /** Tooltip explicatif (unité = événements en excès/déficit). */
    tooltip: string
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
  /** KPI objectifs cumulés de l'escouade (CTF/Zones/Oddball) — V72-03. */
  objectives: {
    title: string
    flagCaptures: string
    flagReturns: string
    flagSteals: string
    flagCarrierTime: string
    zoneCaptures: string
    zoneSecures: string
    zoneTime: string
    skullGrabs: string
    skullCarrierTime: string
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
    dynamique: 'Dynamique',
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
    exactComposition: 'Composition stricte',
    exactCompositionTitle:
      'Par défaut, tous les matchs commencés ensemble sont comptés, même si un autre joueur connu vous accompagnait. Cochez pour ne garder que les matchs joués avec exactement cette composition.',
  },
  dataIssues: {
    title: 'Données partielles : certains chiffres sont incomplets.',
    teammateMatches: (detail) => `Matchs de ${detail} non chargés : il manque à la population commune.`,
    heatmapTeammate: (detail) => `Cartes de ${detail} non chargées : sa ligne de la grille est vide.`,
    mainTeamParticipants:
      'Équipes non chargées : la composition stricte n\'a pas pu être appliquée.',
    mapStats: 'Historique par carte non chargé : les références historiques manquent.',
    unknown: (code) => `Chargement incomplet (${code}).`,
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
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'Historique',
    winRateVsHistoryBulletTitle: 'Taux de victoire session vs historique',
    winRateVsHistoryBulletParity: 'Parité 50 %',
    winRateVsHistoryBulletZero: '0 % (toutes défaites)',
    winRateVsHistoryBulletCounts: (session, history) =>
      `Session : ${session} ${session <= 1 ? 'partie' : 'parties'} · Historique : ${
        history === undefined ? '—' : `${history} ${history <= 1 ? 'partie' : 'parties'}`
      }`,
    winRateVsHistoryBulletMapCountTooltip:
      'Le nombre entre parenthèses est le nombre de parties jouées sur cette carte pendant la session.',
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
    waypointAriaLabel: 'Ouvrir sur Halo Waypoint',
    sortByAriaLabel: (col) => `Trier par ${col}`,
    winRateHistTooltip: 'Taux de victoire de cette escouade sur tous ses matchs communs.',
    winProbTooltip: 'Probabilité de victoire estimée avant le match, d\'après les MMR des deux équipes.',
    teamMmrTooltip: 'Niveau de compétence moyen estimé (MMR) de ton équipe.',
    enemyMmrTooltip: 'Niveau de compétence moyen estimé (MMR) de l\'équipe adverse.',
    deltaMmrTooltip: 'Écart de MMR entre ton équipe et l\'équipe adverse.',
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
    colScoreTooltip: 'Score d\'impact cumulé : chaque pastille positive ajoute, chaque négative retire.',
    colBadgeTooltip: 'Rôle dans l\'escouade : Champion (1er), Passager clandestin ou Maillon faible (dernier).',
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
      silent_hero: 'Joueur (hors Bourreau) avec le plus d\'assistances et le moins de morts',
      false_brother: 'Joueur (hors Bourreau) avec le plus de morts et le moins d\'assistances',
      top_killer: 'Joueur avec le plus grand nombre de frags du match',
      top_gun: 'Premier membre de l\'équipe à atteindre 10 frags',
      kamikaze: 'Joueur le plus tué dans les 1,5 s qui suivent ses frags',
    },
  },
  perMinute: {
    title: 'Stats par minute — Frags / Morts / Assistances',
    description: 'Cadence par joueur sur le scope filtré. Les morts s\'affichent sous l\'axe (couleur joueur atténuée).',
    frags: 'Frags/min',
    deaths: 'Morts/min',
    assists: 'Assistances/min',
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
      support: 'Assistances × 50.',
      score: 'Score personnel par minute jouée. P80 ≈ 195/min.',
      objective: 'Participation aux objectifs par opportunité — actions pondérées (captures, prises, retours…) + temps sur l\'objectif, calculées sur les seuls matchs à objectif et calibrées par mode (un joueur au P80 de son mode marque 80). Axe absent si aucun match à objectif.',
      glossaryLink: '→ Glossaire',
    },
    rawLabel: 'brut',
  },
  intensity: {
    title: 'Intensité',
    subtitle: 'Répartition des frags par phase de match',
    tooltip: 'Chaque match est découpé en 10 tranches de durée égale. Le trait plein montre à quel moment les frags du joueur tombent : à gauche le début du match, à droite la fin. La zone colorée autour dit à quel point ça change d\'un match à l\'autre — large, le joueur joue très différemment selon les parties ; étroite, il fait toujours à peu près pareil. Le pointillé horizontal est le niveau d\'un match où les frags seraient répartis également du début à la fin. À partir de 3 joueurs, la courbe pointillée « Équipe » superposée montre le même profil pour l\'escouade entière : au-dessus, le joueur est plus actif que le groupe sur cette tranche.',
    medianLabel: 'Médiane',
    envelopeLabel: 'Enveloppe P25–P75',
    refLabel: '10 %',
    teamLabel: 'Équipe',
  },
  efficiencySeries: {
    rendementCardTitle: 'Rendement — écart à la frontière élite',
    resistanceCardTitle: 'Résistance — écart à la frontière élite',
    help: 'Une piste par joueur. Le trait montre, match par match, l\'écart à la frontière élite du jeu — le niveau atteint par les 20 % de joueurs les plus efficaces. Au-dessus de zéro (vert), tu fais mieux que cette frontière ; en dessous (rouge), moins bien. Rendement = dégâts infligés convertis en frags effectifs (frags + assistances / 3). Résistance = dégâts encaissés avant chaque mort. Toutes les pistes partagent la même échelle : une bosse de même hauteur vaut la même chose chez tous. Survole un point pour les valeurs brutes du match.',
    offensiveMetric: 'Rendement',
    defensiveMetric: 'Résistance',
    eliteBoundary: 'frontière élite',
    damageDealt: 'Dégâts infligés',
    damageTaken: 'Dégâts subis',
    perFrag: '/ frag effectif',
    perDeath: '/ mort',
    noData: 'Aucune donnée d\'efficacité disponible.',
  },
  performanceCharts: {
    title: 'Performance',
    description: 'Time-series alignée sur les matchs où tous les coéquipiers étaient présents. 1 ligne par joueur, couleurs cohérentes avec la pill et le multiselect.',
    killsDeathsTitle: 'Frags / Morts',
    killsLabel: 'Frags',
    deathsLabel: 'Morts',
    bonusLabel: 'Bonus',
    bonusInfo:
      'Bonus = assistances ÷ 3 : dans le FDA, 3 assistances valent 1 frag (FDA = (frags + assistances/3) − morts). La série empile ce bonus au-dessus des frags du match ; elle est masquée par défaut, clique « Bonus » pour l\'afficher.',
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
  netLives: {
    title: 'Balance des dégâts cumulée',
    tooltip: 'Balance des dégâts = (dégâts infligés − dégâts subis) ÷ {{HP}} PV, exprimée en vies. Positif = tu portes l\'équipe ; négatif = tu coûtes plus que tu ne rapportes.',
  },
  engagementGap: {
    title: 'Écart d\'engagement cumulé',
    tooltip: 'Écart d\'engagement = (rythme observé − rythme attendu) × durée du match, cumulé. Exprimé en événements en excès (positif) ou en déficit (négatif) vs l\'attendu.',
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
    title: 'Mécaniques de frag',
    labels: { assassination: 'Assassinats', ground_pound: 'Frappes au sol', shoulder_bash: 'Charges d\'épaule' },
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
  objectives: {
    title: 'Objectifs de l’escouade',
    flagCaptures: 'Captures de drapeau',
    flagReturns: 'Retours de drapeau',
    flagSteals: 'Vols de drapeau',
    flagCarrierTime: 'Temps porteur (drapeau)',
    zoneCaptures: 'Zones capturées',
    zoneSecures: 'Zones sécurisées',
    zoneTime: 'Temps en zone',
    skullGrabs: 'Récupérations du crâne',
    skullCarrierTime: 'Temps porteur (crâne)',
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
    dynamique: 'Dynamics',
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
    exactComposition: 'Strict line-up',
    exactCompositionTitle:
      'By default every match started together is counted, even if another known player was with you. Tick to keep only matches played with exactly this line-up.',
  },
  dataIssues: {
    title: 'Partial data: some numbers are incomplete.',
    teammateMatches: (detail) => `Could not load ${detail}'s matches: they are missing from the shared population.`,
    heatmapTeammate: (detail) => `Could not load ${detail}'s maps: their grid row is empty.`,
    mainTeamParticipants: 'Teams could not be loaded: the strict line-up option was not applied.',
    mapStats: 'Per-map history could not be loaded: historical references are missing.',
    unknown: (code) => `Incomplete load (${code}).`,
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
    winRateVsHistorySession: 'Session',
    winRateVsHistoryHistory: 'All time',
    winRateVsHistoryBulletTitle: 'Session winrate vs history',
    winRateVsHistoryBulletParity: '50% parity',
    winRateVsHistoryBulletZero: '0% (all losses)',
    winRateVsHistoryBulletCounts: (session, history) =>
      `Session: ${session} ${session <= 1 ? 'game' : 'games'} · History: ${
        history === undefined ? '—' : `${history} ${history <= 1 ? 'game' : 'games'}`
      }`,
    winRateVsHistoryBulletMapCountTooltip:
      'The number in parentheses is the number of games played on this map during the session.',
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
    waypointAriaLabel: 'Open on Halo Waypoint',
    sortByAriaLabel: (col) => `Sort by ${col}`,
    winRateHistTooltip: 'Win rate for this squad across all their shared matches.',
    winProbTooltip: 'Win probability estimated before the match, from both teams\' MMR.',
    teamMmrTooltip: 'Average estimated skill level (MMR) of your team.',
    enemyMmrTooltip: 'Average estimated skill level (MMR) of the enemy team.',
    deltaMmrTooltip: 'MMR gap between your team and the enemy team.',
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
    colScoreTooltip: 'Cumulative impact score: each positive pill adds, each negative one subtracts.',
    colBadgeTooltip: 'Role in the squad: Champion (1st), Stowaway or Weak link (last).',
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
      objective: 'Objective participation per opportunity — weighted actions (captures, grabs, returns…) plus time on the objective, computed on objective matches only and calibrated per mode (a P80 player in their mode scores 80). Axis hidden when no objective match.',
      glossaryLink: '→ Glossary',
    },
    rawLabel: 'raw',
  },
  intensity: {
    title: 'Intensity',
    subtitle: 'Frag distribution across match phases',
    tooltip: 'Each match is split into 10 equal slices. The solid line shows when the player\'s kills happen: start of the match on the left, end on the right. The shaded band around it shows how much this changes from match to match — wide means the player plays very differently depending on the game, narrow means they play much the same way every time. The horizontal dashed line is the level of a match where kills would be spread evenly from start to finish. From 3 players on, the overlaid "Team" dashed curve shows the same profile for the whole squad: above it, the player is more active than the group on that slice.',
    medianLabel: 'Median',
    envelopeLabel: 'P25–P75 envelope',
    refLabel: '10%',
    teamLabel: 'Team',
  },
  efficiencySeries: {
    rendementCardTitle: 'Offensive efficiency — gap to elite frontier',
    resistanceCardTitle: 'Defensive resistance — gap to elite frontier',
    help: 'One track per player. The line shows, match by match, the gap to the game\'s elite frontier — the level reached by the top 20% most efficient players. Above zero (green) you beat that frontier; below (red) you fall short. Offensive efficiency = damage dealt converted into effective kills (kills + assists / 3). Defensive resistance = damage absorbed before each death. Every track shares the same scale, so a peak of the same height means the same thing for everyone. Hover a point for the raw match values.',
    offensiveMetric: 'Efficiency',
    defensiveMetric: 'Resistance',
    eliteBoundary: 'elite frontier',
    damageDealt: 'Damage dealt',
    damageTaken: 'Damage taken',
    perFrag: '/ effective kill',
    perDeath: '/ death',
    noData: 'No efficiency data available.',
  },
  performanceCharts: {
    title: 'Performance',
    description: 'Time-series aligned on matches where every teammate was present. One line per player, colors mirror the active-player pill and multiselect.',
    killsDeathsTitle: 'Frags / Deaths',
    killsLabel: 'Frags',
    deathsLabel: 'Deaths',
    bonusLabel: 'Bonus',
    bonusInfo:
      'Bonus = assists ÷ 3: in KDA, 3 assists count as 1 kill (KDA = (kills + assists/3) − deaths). The series stacks that bonus on top of the match kills; it is hidden by default, click "Bonus" to show it.',
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
  netLives: {
    title: 'Cumulative damage balance',
    tooltip: 'Damage balance = (damage dealt − damage taken) ÷ {{HP}} HP, expressed in lives. Positive = you carry the team; negative = you cost more than you bring.',
  },
  engagementGap: {
    title: 'Cumulative engagement gap',
    tooltip: 'Engagement gap = (observed pace − expected pace) × match duration, accumulated. Expressed in events in surplus (positive) or deficit (negative) vs expected.',
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
  objectives: {
    title: 'Squad objectives',
    flagCaptures: 'Flag captures',
    flagReturns: 'Flag returns',
    flagSteals: 'Flag steals',
    flagCarrierTime: 'Flag carrier time',
    zoneCaptures: 'Zones captured',
    zoneSecures: 'Zones secured',
    zoneTime: 'Time in zones',
    skullGrabs: 'Skull grabs',
    skullCarrierTime: 'Skull carrier time',
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
