/**
 * contract.ts — Le TYPE du dictionnaire de la feature Escouade : une seule déclaration, que
 * `fr.ts`, `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies SquadText`).
 */

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
    emprise: string
  }
  /**
   * Titres de section transverses (lot 3 « sections », 2026-09-22) — un titre coiffe
   * au moins deux blocs, jamais un bloc seul.
   */
  sections: {
    historique: string
    fragsArmes: string
    /** Titre de la section de Synergies qui coiffe « Appui » et « Rôles de portée ». */
    appuiPortee: string
    /** Aide (i) de cette section : une phrase par carte, la mesure et son périmètre. */
    appuiPorteeHelp: string[]
  }
  selection: {
    placeholder: string
    prompt: string
  }
  filter: {
    experience: string
    playlist: string
    allExperiences: string
    allPlaylists: string
    analyse: string
    /** Option « composition exacte » (activée par défaut) + son explication. */
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
  /** Info-bulle qui explique l'ecart "composition exacte" publie sur la L2
   *  (ADR 0033, D1) : matchs ecartes de la session, coequipier responsable
   *  nomme. Reutilise le patron d'aide d'en-tete existant (V73-L2 2.4c). */
  compositionGap: {
    heading: (excludedCount: number) => string
    excludedLine: (dateLabel: string, mapLabel: string, culpritsLabel: string, culpritCount: number) => string
    culpritUnknown: string
    /** Jonction des coéquipiers responsables d'un match écarté (« A et B »). */
    culpritJoiner: string
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
    /** Onglet Emprise (ex-Usages) sans rien à montrer (aucun film décodé sur la sélection). */
    noDecodedFilmTitle: string
    noDecodedFilmDescription: string
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
    /**
     * En-tête de la colonne Halo Waypoint (retour utilisateur 2026-09-09), abrégé en
     * « HW » : la colonne ne fait qu'une icône de large. Nom propre, donc identique
     * FR/EN — mais declaré dans les deux locales, la parité est un invariant de typage.
     */
    waypointHeader: string
    /** aria-label/tooltip du lien interne vers le rejeu 2D du match. */
    replayAriaLabel: string
    /** En-tête de la colonne « Rejeu » (retour utilisateur 2026-09-09). */
    replayHeader: string
    /** aria-label du bouton de tri d'un en-tête « Trier par {col} » (I16). */
    sortByAriaLabel: (col: string) => string
    /** Tooltips d'en-tête de colonne (V72-04, icône ⓘ). */
    /**
     * Aide de l'en-tete de la colonne Score : sur les modes qui se jouent en manches,
     * ce sont les MANCHES qui sont affichees, pas les points de l'API (ADR 0032).
     */
    scoreTooltip: string
    winRateHistTooltip: string
    winProbTooltip: string
    teamMmrTooltip: string
    enemyMmrTooltip: string
    deltaMmrTooltip: string
  }
  /** Graphe « Appui » (larbin -> patron), page Synergies. */
  assists: {
    title: string
    description: string
    /** Rôle de la BARRE : celui qui a assisté (le larbin). Titre d'axe et infobulle. */
    roleAssistant: string
    /** Rôle du SEGMENT : celui qui a eu le frag crédité (le patron). Infobulle. */
    roleBeneficiary: string
    /** Titre de l'axe des valeurs : ce que la hauteur des segments compte. */
    valueAxis: string
    /** Mesuré, mais aucune assistance interne à l'escouade. */
    noPairs: string
    /** Infobulle d'un segment : la PART de la paire dans les assistances mesurées. */
    tooltipShare: (share: string) => string
    /** Infobulle d'un segment : les assistances de la paire où le larbin a fait plus de
     *  dégâts que le patron (champ `stolen_count` de l'API, requête Q32d côté Go :
     *  `assist_damage_pct > killer_damage_pct`), quand il y en a.
     *  À NE PAS confondre avec le badge « Voleur » de la matrice, dont le critère est
     *  autre (le tueur a fait 10 % des dégâts ou moins) : deux mesures distinctes, d'où
     *  l'absence du mot « volé » ici. */
    tooltipAssistantOutdamaged: (n: number) => string
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
    /** Nom de l'axe X de la heatmap joueur x carte. */
    /** Nom de l'axe Y de la heatmap joueur x carte. */
    yAxis: string
    pieceTier1: string
    pieceTier2: string
    pieceTier3: string
    pieceTier4: string
    pieceTier5: string
    noScore: string
    /** Infobulle d'une case : les deux lignes de valeurs. */
    tooltipPerf: string
    tooltipMatches: string
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
    /** Libellé tooltip du trait médian. */
    medianLabel: string
    /** Libellé tooltip de la fourchette interquartile. */
    envelopeLabel: string
    /** Libellé du repère 10 % (activité uniforme). */
    refLabel: string
    /** Libellé de la courbe de référence ÉQUIPE (alliés du joueur principal, à partir de 3 joueurs). */
    teamLabel: string
    /** Libellé de la courbe de référence LOBBY (tout le match, dès 1 joueur). */
    lobbyLabel: string
  }
  efficiencySeries: {
    /** Titre COURT de la carte Rendement (la définition vit dans l'aide ⓘ). */
    rendementCardTitle: string
    /** Titre COURT de la carte Résistance (la définition vit dans l'aide ⓘ). */
    resistanceCardTitle: string
    /** Aide ⓘ de la carte Rendement : ce que vaut une vie de dégâts infligés. */
    rendementHelp: string
    /** Aide ⓘ de la carte Résistance : ce qui est encaissé avant chaque mort. */
    resistanceHelp: string
    /** Nom de l'indicateur offensif au survol (« Rendement »). */
    offensiveMetric: string
    /** Nom de l'indicateur défensif au survol (« Résistance »). */
    defensiveMetric: string
    /** Libellé du repère 100 % (« 1 vie »), sur le trait et au survol. */
    oneLife: string
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
    /** Aide ⓘ de « Répartition des frags » (texte de la maquette C3EW). */
    fragBreakdownInfo: string
    /** Infobulle d'un segment : joueur, classe, compte et total du joueur. */
    fragBreakdownSegment: (player: string, className: string, kills: number, total: number) => string
    /** Nom accessible de la barre d'un joueur. */
    fragBreakdownBarAria: (player: string, total: number) => string
  }
  /** « Écart cumulé au FDA attendu » (D3/D7 — différentiel FDA réel vs attendu par joueur). */
  fdaGap: {
    title: string
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
    /** Aide ⓘ de « Outils de destruction » (texte de la maquette C3EW). */
    info: string
    /** Ligne des objets explosifs du décor (catégorie de source du film, D8). */
    explosiveObject: string
    /** Ligne de la chute et de l'environnement (catégorie de source du film, D8). */
    environment: string
    /** Infobulle d'une barre : compte de frags et part du total du joueur (en %, déjà arrondie). */
    killsShare: (kills: number, sharePct: number) => string
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
}
