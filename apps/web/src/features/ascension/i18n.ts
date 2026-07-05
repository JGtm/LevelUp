/**
 * i18n FR/EN de la feature Ascension (V2 progression).
 *
 * Pattern : dictionnaire typé par locale, accessible via getAscensionText(locale).
 * Aligné avec features/{notifications,settings,help,...}/i18n.ts.
 */
import type { StreakType, ContextType, BehaviorType } from './types'

export type AscensionLocale = 'fr' | 'en'

export interface AscensionText {
  // Page wrapper (layout 2 onglets — refonte 2026-05-26)
  pageTitle: string
  pageSubtitle: string
  tabsAriaLabel: string
  tabProfile: string
  tabCoaching: string
  tabRealisations: string
  tipsTickerAriaLabel: string
  prestigeLayerTitle: string
  prestigeLayerDescription: string
  prestigeDisabledHint: string // tooltip bouton désactivé (backend non implémenté)
  // Onglet Profil & objectifs — labels ex-inline (I4b, 2026-07-05)
  profileSelectPlayer: string
  profileMyObjectives: string
  profilePrestigeNotEnabled: string
  profileAbandonConfirm: string
  profileMyActiveObjectives: string
  profileFreeObjectives: string
  profileNoFreeObjective: string
  profileNewObjective: string
  profilePilotedObjectives: string
  profilePilotDisabled: string
  profilePilotHelp: string
  profilePilotMode: string
  profileNewArc: string
  profileBrowsePresets: string
  profileMyActiveArcs: string
  profileNoArc: string
  squadPrestigeTitle: string
  squadPrestigeMaxTier: string
  squadPrestigeYou: string
  ascensionLayerTitle: string
  ascensionLayerDescription: string

  // Streaks
  streaksSectionTitle: string
  streaksEmpty: string
  streakActive: string
  streakPaused: string
  streakBroken: string
  streakBadgeAriaLabel: string // "{count} jours d'affilée"
  streakBadgeAriaEmpty: string // "Aucune série active"
  streakCurrentLength: string // "{n} jour(s)"
  streakUnitDay: string // unité période daily_* (jour/jours)
  streakUnitWeek: string // unité période weekly_* (semaine/semaines)
  streakBestLength: string // "Record perso : {n}"
  streakStarted: string // "Commencée le {date}"
  streakBrokenAt: string // "Cassée le {date}"
  streakShieldsAvailable: string // "{n} bouclier(s) disponible(s) ce mois"
  streakShieldsUsed: string // "{n} bouclier(s) utilisé(s)"
  streakPPMultiplier: string // "Multiplicateur PP : ×{value}"
  streakNextMilestone: string // "Prochain multiplicateur : ×{mul} (à {n} jours)"
  streakTypeName: Record<StreakType, string>
  streakAtMaxMultiplier: string

  // Records
  recordsSectionTitle: string
  recordsTimelineTitle: string
  recordsPersonalBestsTitle: string
  recordsEmpty: string
  recordsHistoryEmpty: string
  recordsValueLabel: string
  recordsAchievedAt: string
  recordsPreviousValue: string

  // Milestones
  milestonesSectionTitle: string
  milestonesEmpty: string
  milestonesEarnedAt: string
  milestonesLocked: string
  milestonesEarned: string
  milestonesEarnedCount: string // "{n}/{total}"
  milestonesThreshold: string // "Seuil : {n}"

  // Métriques (labels)
  metric: Record<string, string>
  // Périodes
  period: Record<'30d' | '90d' | 'all_time', string>

  // Loading / errors
  loading: string
  errorLoading: string

  // ── Profile ────────────────────────────────────────────────────────────────
  profileSectionTitle: string
  profileStrengths: string
  profileImprovements: string
  profileNotEnoughData: string
  profileMatchesPerDay: string
  profileLeveragesTitle: string
  profileSuggestedChallenges: string
  radarAxis: Record<string, string>
  styleKey: Record<string, string>
  engagementTier: Record<string, string>
  lusrTierLabel: string
  lusrGapToNext: string
  lusrTop20: string
  lusrTargetForTier: string
  lusrComponent: Record<string, string>

  // ── Patterns ───────────────────────────────────────────────────────────────
  patternsSectionTitle: string
  patternsNotEnoughData: string
  contextType: Record<ContextType, string>
  patternWinRate: string
  patternMatches: string
  signalStrength: string
  signalWeakness: string
  signalNeutral: string
  squadVsSoloTitle: string
  squadVsSoloSolo: string
  squadVsSoloSquad: string

  // ── Behaviors ──────────────────────────────────────────────────────────────
  behaviorsSectionTitle: string
  behaviorType: Record<BehaviorType, string>
  behaviorAdvice: Partial<Record<BehaviorType, string>>
  behaviorConfirmed: string
  patternSeverity: Record<'low' | 'medium' | 'high', string>

  // ── Levers ─────────────────────────────────────────────────────────────────
  leversSectionTitle: string
  leverImpact: string
  leverCurrent: string
  leverTarget: string
  leverHorizonMatches: string
  leverAxis: Record<string, string>
}

const METRIC_LABEL_FR: Record<string, string> = {
  performance_score: 'Score de performance',
  kda: 'KDA',
  kpm: 'Tueries / minute',
  accuracy: 'Précision',
  pspm: 'Score perso / minute',
  matches_played: 'Matchs joués',
  wins: 'Victoires',
  kills: 'Éliminations',
  headshots: 'Tirs à la tête',
  assists: 'Assistances',
  accuracy_threshold_days: 'Jours réguliers',
}

const METRIC_LABEL_EN: Record<string, string> = {
  performance_score: 'Performance score',
  kda: 'KDA',
  kpm: 'Kills per minute',
  accuracy: 'Accuracy',
  pspm: 'Personal score per minute',
  matches_played: 'Matches played',
  wins: 'Wins',
  kills: 'Kills',
  headshots: 'Headshots',
  assists: 'Assists',
  accuracy_threshold_days: 'Consistent days',
}

const FR: AscensionText = {
  pageTitle: 'Ascension',
  pageSubtitle: 'Ton profil de jeu, tes objectifs et tes accomplissements.',
  tabsAriaLabel: 'Sections Ascension',
  tabProfile: 'Profil & objectifs',
  tabCoaching: 'Entraînement',
  tabRealisations: 'Réalisations',
  tipsTickerAriaLabel: 'Astuces de jeu pour progresser',
  prestigeLayerTitle: 'Prestige — Objectifs et arcs',
  prestigeLayerDescription:
    'Système autonome pour te fixer des objectifs personnels et suivre ta progression. Tu peux l\'utiliser seul, sans coaching.',
  prestigeDisabledHint: 'Phase 5 minimale : non implémenté côté backend',
  profileSelectPlayer: 'Sélectionne un joueur pour voir tes objectifs.',
  profileMyObjectives: 'Mes objectifs',
  profilePrestigeNotEnabled: "Le module Prestige n'est pas activé sur ce serveur.",
  profileAbandonConfirm: 'Abandonner cet objectif ? Cooldown 24h sur la métrique.',
  profileMyActiveObjectives: 'Mes objectifs actifs',
  profileFreeObjectives: 'Objectifs libres',
  profileNoFreeObjective: 'Aucun objectif libre actif.',
  profileNewObjective: '+ Nouvel objectif',
  profilePilotedObjectives: 'Objectifs pilotés',
  profilePilotDisabled: 'Désactivé',
  profilePilotHelp: "Le système t'attribue des objectifs quotidiens, hebdo et mensuels avec des plafonds.",
  profilePilotMode: 'Mode pilote',
  profileNewArc: '+ Nouvel arc',
  profileBrowsePresets: 'Parcourir les presets',
  profileMyActiveArcs: 'Mes arcs en cours',
  profileNoArc: 'Aucun arc en cours. Adopte un arc preset ou crée le tien.',
  squadPrestigeTitle: 'Progression Prestige',
  squadPrestigeMaxTier: 'Niveau max',
  squadPrestigeYou: 'moi',
  ascensionLayerTitle: 'Ascension — Coaching d\'amélioration',
  ascensionLayerDescription:
    'Analyse ton historique pour te proposer des angles d\'amélioration ciblés. S\'appuie sur Prestige (les campagnes deviennent des objectifs) — tu peux ignorer cette section si tu préfères piloter toi-même.',
  streaksSectionTitle: 'Mes séries',
  streaksEmpty:
    "Aucune série en cours. Joue un match aujourd'hui pour en démarrer une !",
  streakActive: 'En cours',
  streakPaused: 'Préservée par un bouclier',
  streakBroken: 'Cassée',
  streakBadgeAriaLabel: 'Série de {count} jours',
  streakBadgeAriaEmpty: 'Aucune série active',
  streakCurrentLength: '{n} jour{plural}',
  streakUnitDay: 'jour{plural}',
  streakUnitWeek: 'semaine{plural}',
  streakBestLength: 'Record perso : {n} jour{plural}',
  streakStarted: 'Commencée le {date}',
  streakBrokenAt: 'Cassée le {date}',
  streakShieldsAvailable:
    '{n} bouclier{plural} disponible{plural} ce mois',
  streakShieldsUsed: '{n} bouclier{plural} utilisé{plural} ce mois',
  streakPPMultiplier: 'Multiplicateur PP : ×{value}',
  streakNextMilestone: 'Prochain multiplicateur : ×{mul} (à {n} {unit})',
  streakAtMaxMultiplier: 'Multiplicateur PP maximum atteint (×1.75)',
  streakTypeName: {
    daily_play: 'Match par jour',
    daily_perf: 'Performance par jour',
    weekly_play: '5 matchs par semaine',
    weekly_kda_threshold: 'KDA hebdomadaire',
  },
  recordsSectionTitle: 'Mes records',
  recordsTimelineTitle: 'Historique des records battus',
  recordsPersonalBestsTitle: 'Records personnels',
  recordsEmpty: 'Pas encore de record. Joue quelques matchs pour faire émerger tes meilleurs scores.',
  recordsHistoryEmpty: 'Aucun record battu pour le moment.',
  recordsValueLabel: 'Valeur',
  recordsAchievedAt: 'Atteint le {date}',
  recordsPreviousValue: 'Précédent : {value}',
  milestonesSectionTitle: 'Mes milestones',
  milestonesEmpty: 'Aucun milestone configuré pour ce titre.',
  milestonesEarnedAt: 'Débloqué le {date}',
  milestonesLocked: 'À débloquer',
  milestonesEarned: 'Débloqué',
  milestonesEarnedCount: '{n}/{total} débloqué{plural}',
  milestonesThreshold: 'Seuil : {n}',
  metric: METRIC_LABEL_FR,
  period: {
    '30d': '30 jours',
    '90d': '90 jours',
    all_time: 'Carrière',
  },
  loading: 'Chargement…',
  errorLoading: 'Erreur lors du chargement',

  // Profile
  profileSectionTitle: 'Profil de jeu',
  profileStrengths: 'Points forts',
  profileImprovements: 'Axes de progression',
  profileNotEnoughData: 'Joue encore quelques matchs pour débloquer ton profil complet (30 minimum).',
  profileMatchesPerDay: 'match(s)/jour',
  profileLeveragesTitle: 'Leviers prioritaires',
  profileSuggestedChallenges: 'Défis suggérés',
  radarAxis: {
    combat: 'Combat',
    survival: 'Survie',
    support: 'Support',
    score: 'Score',
    objective: 'Objectif',
    impact: 'Impact',
  },
  styleKey: {
    opportunistic_finisher: 'Finisher opportuniste',
    overextended: 'Surexposé',
    hyper_engaged: 'Hyper engagé',
    passive: 'Passif',
  },
  engagementTier: {
    low: 'Engagement faible',
    regular: 'Régulier',
    high: 'Actif',
    intense: 'Intensif',
  },
  lusrTierLabel: 'Rang LUSR',
  lusrGapToNext: '+{n} μ pour',
  lusrTop20: 'Top 20% :',
  lusrTargetForTier: 'Cible pour le rang',
  lusrComponent: {
    kills_vs_expected: 'Kills vs attendu',
    deaths_vs_expected: 'Morts vs attendu',
    win_factor: 'Facteur victoire',
    damage_efficiency: 'Efficacité dégâts',
    accuracy_delta: 'Delta précision',
    medal_exploit: 'Exploit médailles',
    offensive_conversion: 'Conversion offensive',
    defensive_resistance: 'Résistance défensive',
  },

  // Patterns
  patternsSectionTitle: 'Patterns de jeu',
  patternsNotEnoughData: 'Pas assez de matchs pour analyser tes patterns (10 minimum).',
  contextType: {
    by_mode: 'Par mode',
    by_map: 'Par carte',
    by_squad: 'Solo vs Escouade',
  },
  patternWinRate: 'Win rate',
  patternMatches: 'matchs',
  signalStrength: 'Force',
  signalWeakness: 'Faiblesse',
  signalNeutral: 'Neutre',
  squadVsSoloTitle: 'Comparaison Solo / Escouade',
  squadVsSoloSolo: 'Solo',
  squadVsSoloSquad: 'Escouade',

  // Behaviors
  behaviorsSectionTitle: 'Comportements détectés',
  behaviorType: {
    tilt: 'Tilt',
    session_fatigue: 'Fatigue de session',
    engagement_drop: 'Désengagement',
    accuracy_plateau: 'Plateau de précision',
    perf_ceiling: 'Plafond de performance',
  },
  behaviorAdvice: {
    tilt: 'Fais une pause après 3 défaites consécutives.',
    session_fatigue: 'Limite tes sessions à 4-5 matchs pour maintenir ton niveau.',
    engagement_drop: 'Varie les modes pour retrouver de la motivation.',
    accuracy_plateau: 'Concentre-toi sur la précision avant la cadence de tir.',
    perf_ceiling: 'Travaille les axes les plus faibles de ton radar.',
  },
  behaviorConfirmed: 'Confirmé',
  patternSeverity: {
    low: 'Faible',
    medium: 'Moyen',
    high: 'Élevé',
  },

  // Levers
  leversSectionTitle: 'Leviers calibrés',
  leverImpact: 'impact',
  leverCurrent: 'Actuel',
  leverTarget: 'Cible',
  leverHorizonMatches: 'matchs',
  leverAxis: {
    mode_selection: 'Choix de mode',
    map_avoidance: 'Carte à éviter',
    squad_play: 'Jeu en escouade',
    session_management: 'Gestion de session',
    session_length: 'Durée de session',
    engagement: 'Engagement',
    accuracy: 'Précision',
    radar_axis: 'Axe radar',
    csr_ranked: 'CSR classé',
  },
}

const EN: AscensionText = {
  pageTitle: 'Ascension',
  pageSubtitle: 'Your play profile, your objectives and your achievements.',
  tabsAriaLabel: 'Ascension sections',
  tabProfile: 'Profile & objectives',
  tabCoaching: 'Training',
  tabRealisations: 'Achievements',
  tipsTickerAriaLabel: 'Gameplay tips to improve',
  prestigeLayerTitle: 'Prestige — Objectives and arcs',
  prestigeLayerDescription:
    'Autonomous system to set personal objectives and track progression. Usable on its own, no coaching required.',
  prestigeDisabledHint: 'Minimal Phase 5: not yet implemented on the backend',
  profileSelectPlayer: 'Select a player to view objectives.',
  profileMyObjectives: 'My objectives',
  profilePrestigeNotEnabled: 'The Prestige module is not enabled on this server.',
  profileAbandonConfirm: 'Abandon this objective? 24h cooldown on the metric.',
  profileMyActiveObjectives: 'My active objectives',
  profileFreeObjectives: 'Free objectives',
  profileNoFreeObjective: 'No free objective active.',
  profileNewObjective: '+ New objective',
  profilePilotedObjectives: 'Piloted objectives',
  profilePilotDisabled: 'Disabled',
  profilePilotHelp: 'The system assigns you daily/weekly/monthly objectives with caps.',
  profilePilotMode: 'Pilot mode',
  profileNewArc: '+ New arc',
  profileBrowsePresets: 'Browse presets',
  profileMyActiveArcs: 'My active arcs',
  profileNoArc: 'No arc in progress. Adopt a preset arc or create your own.',
  squadPrestigeTitle: 'Prestige progression',
  squadPrestigeMaxTier: 'Max tier',
  squadPrestigeYou: 'you',
  ascensionLayerTitle: 'Ascension — Improvement coaching',
  ascensionLayerDescription:
    'Analyses your history to surface targeted improvement angles. Builds on Prestige (campaigns become objectives) — feel free to ignore this section if you prefer to drive your own progress.',
  streaksSectionTitle: 'My streaks',
  streaksEmpty: 'No active streak. Play a match today to start a series!',
  streakActive: 'Active',
  streakPaused: 'Preserved by a shield',
  streakBroken: 'Broken',
  streakBadgeAriaLabel: '{count}-day streak',
  streakBadgeAriaEmpty: 'No active streak',
  streakCurrentLength: '{n} day{plural}',
  streakUnitDay: 'day{plural}',
  streakUnitWeek: 'week{plural}',
  streakBestLength: 'Personal best: {n} day{plural}',
  streakStarted: 'Started on {date}',
  streakBrokenAt: 'Broken on {date}',
  streakShieldsAvailable:
    '{n} shield{plural} available this month',
  streakShieldsUsed: '{n} shield{plural} used this month',
  streakPPMultiplier: 'PP multiplier: ×{value}',
  streakNextMilestone: 'Next multiplier: ×{mul} (at {n} {unit})',
  streakAtMaxMultiplier: 'Maximum PP multiplier reached (×1.75)',
  streakTypeName: {
    daily_play: 'Daily match',
    daily_perf: 'Daily performance',
    weekly_play: '5 matches per week',
    weekly_kda_threshold: 'Weekly KDA',
  },
  recordsSectionTitle: 'My records',
  recordsTimelineTitle: 'Records broken timeline',
  recordsPersonalBestsTitle: 'Personal bests',
  recordsEmpty: 'No record yet. Play a few matches to start tracking your best scores.',
  recordsHistoryEmpty: 'No record broken yet.',
  recordsValueLabel: 'Value',
  recordsAchievedAt: 'Achieved on {date}',
  recordsPreviousValue: 'Previous: {value}',
  milestonesSectionTitle: 'My milestones',
  milestonesEmpty: 'No milestones configured for this title.',
  milestonesEarnedAt: 'Earned on {date}',
  milestonesLocked: 'Locked',
  milestonesEarned: 'Earned',
  milestonesEarnedCount: '{n}/{total} earned',
  milestonesThreshold: 'Threshold: {n}',
  metric: METRIC_LABEL_EN,
  period: {
    '30d': '30 days',
    '90d': '90 days',
    all_time: 'All-time',
  },
  loading: 'Loading…',
  errorLoading: 'Loading error',

  // Profile
  profileSectionTitle: 'Game Profile',
  profileStrengths: 'Strengths',
  profileImprovements: 'Areas to improve',
  profileNotEnoughData: 'Play a few more matches to unlock your full profile (30 minimum).',
  profileMatchesPerDay: 'match(es)/day',
  profileLeveragesTitle: 'Priority levers',
  profileSuggestedChallenges: 'Suggested challenges',
  radarAxis: {
    combat: 'Combat',
    survival: 'Survival',
    support: 'Support',
    score: 'Score',
    objective: 'Objective',
    impact: 'Impact',
  },
  styleKey: {
    opportunistic_finisher: 'Opportunistic finisher',
    overextended: 'Overextended',
    hyper_engaged: 'Hyper engaged',
    passive: 'Passive',
  },
  engagementTier: {
    low: 'Low engagement',
    regular: 'Regular',
    high: 'Active',
    intense: 'Intensive',
  },
  lusrTierLabel: 'LUSR Rank',
  lusrGapToNext: '+{n} μ for',
  lusrTop20: 'Top 20%:',
  lusrTargetForTier: 'Target for rank',
  lusrComponent: {
    kills_vs_expected: 'Kills vs expected',
    deaths_vs_expected: 'Deaths vs expected',
    win_factor: 'Win factor',
    damage_efficiency: 'Damage efficiency',
    accuracy_delta: 'Accuracy delta',
    medal_exploit: 'Medal exploit',
    offensive_conversion: 'Offensive conversion',
    defensive_resistance: 'Defensive resistance',
  },

  // Patterns
  patternsSectionTitle: 'Play patterns',
  patternsNotEnoughData: 'Not enough matches to analyze your patterns (10 minimum).',
  contextType: {
    by_mode: 'By mode',
    by_map: 'By map',
    by_squad: 'Solo vs Squad',
  },
  patternWinRate: 'Win rate',
  patternMatches: 'matches',
  signalStrength: 'Strength',
  signalWeakness: 'Weakness',
  signalNeutral: 'Neutral',
  squadVsSoloTitle: 'Solo vs Squad comparison',
  squadVsSoloSolo: 'Solo',
  squadVsSoloSquad: 'Squad',

  // Behaviors
  behaviorsSectionTitle: 'Detected behaviors',
  behaviorType: {
    tilt: 'Tilt',
    session_fatigue: 'Session fatigue',
    engagement_drop: 'Disengagement',
    accuracy_plateau: 'Accuracy plateau',
    perf_ceiling: 'Performance ceiling',
  },
  behaviorAdvice: {
    tilt: 'Take a break after 3 consecutive losses.',
    session_fatigue: 'Limit sessions to 4-5 matches to maintain your level.',
    engagement_drop: 'Switch modes to regain motivation.',
    accuracy_plateau: 'Focus on accuracy before fire rate.',
    perf_ceiling: 'Work on the weakest axes of your radar.',
  },
  behaviorConfirmed: 'Confirmed',
  patternSeverity: {
    low: 'Low',
    medium: 'Medium',
    high: 'High',
  },

  // Levers
  leversSectionTitle: 'Calibrated levers',
  leverImpact: 'impact',
  leverCurrent: 'Current',
  leverTarget: 'Target',
  leverHorizonMatches: 'matches',
  leverAxis: {
    mode_selection: 'Mode selection',
    map_avoidance: 'Map to avoid',
    squad_play: 'Squad play',
    session_management: 'Session management',
    session_length: 'Session length',
    engagement: 'Engagement',
    accuracy: 'Accuracy',
    radar_axis: 'Radar axis',
    csr_ranked: 'Ranked CSR',
  },
}

export function getAscensionText(locale: AscensionLocale): AscensionText {
  return locale === 'en' ? EN : FR
}
