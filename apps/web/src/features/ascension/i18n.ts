/**
 * i18n FR/EN de la feature Ascension (V2 progression).
 *
 * Pattern : dictionnaire typé par locale, accessible via getAscensionText(locale).
 * Aligné avec features/{notifications,settings,help,...}/i18n.ts.
 */
import type { StreakType, ContextType, BehaviorType } from './types'
import type { Locale } from '@/lib/i18n/locale'

export interface AscensionText {
  // Page wrapper (layout 4 onglets — refonte 2026-07, DEC-3)
  pageTitle: string
  pageSubtitle: string
  tabsAriaLabel: string
  tabProfile: string
  tabObjectives: string
  tabCoaching: string
  tabRealisations: string
  /** Onglet « Tactique » (5e rang, 2026-09-06) — masqué pour un titre sans rejeu. */
  tabTactical: string
  tipsTickerAriaLabel: string
  profilLayerTitle: string
  profilLayerDescription: string
  prestigeLayerTitle: string
  prestigeLayerDescription: string
  // Onglet Objectifs (couche Prestige) — labels ex-inline (I4b, 2026-07-05)
  profileSelectPlayer: string
  profileMyObjectives: string
  profilePrestigeNotEnabled: string
  profileAbandonObjective: string // bouton « Abandonner » d'un objectif (B5)
  profileAbandonTitle: string // titre de la confirmation AlertDialog (B5)
  profileAbandonConfirm: string // description de la confirmation AlertDialog
  profileAbandonCancel: string // libellé « Annuler » de la confirmation (B5)
  profileMyActiveObjectives: string
  profileFreeObjectives: string
  profileNoFreeObjective: string
  profileNewObjective: string
  profilePilotedObjectives: string
  profilePilotHelp: string
  profilePilotMode: string
  profilePilotEnable: string // bouton d'activation du mode pilote
  profilePilotDisable: string // bouton de désactivation du mode pilote
  profilePilotPending: string // libellé transitoire (mutation en cours)
  profilePilotEnableCta: string // CTA d'activation dans l'empty state objectifs
  profileNewArc: string
  profileBrowsePresets: string
  profileMyActiveArcs: string
  profileNoArc: string
  squadPrestigeTitle: string
  squadPrestigeMaxTier: string
  squadPrestigeYou: string
  squadPrestigeTowardNext: string // "{current} / {target} PP vers {next}" (intra-niveau)
  squadPrestigeTotal: string // "{pp} PP au total" (libellé secondaire)
  realisationsSelectPlayer: string
  realisationsHighlights: string
  realisationsEmpty: string
  // Historique (Lot C) — mémoire complète datée sous les jalons.
  historyTitle: string
  historyObjectivesTitle: string
  historyObjectivesEmpty: string
  historyArcsTitle: string
  historyArcsEmpty: string
  historyArcCompletedOn: string // "Terminé le {date}" / "Completed on {date}"
  historyArcStarted: string // "Créé le {date}" / "Created on {date}"
  historyCampaignsTitle: string
  historyCampaignsEmpty: string
  historyCampaignProgress: string // libellé du delta d'axe
  historyResultCompleted: string
  historyResultExpired: string
  historyResultAbandoned: string
  historyResultArchived: string // « Retiré » — neutre (défi pilote désactivé)
  // Sorties vers les matchs (Lot C, C5).
  patternSeeMatches: string // aria/tooltip carte pattern cliquable
  recordSeePeriod: string // lien « voir la période » d'un record
  coachingSelectPlayer: string
  ascensionLayerTitle: string
  ascensionLayerDescription: string

  // Calendrier d'activité (DEC-5/D3)
  activityCalendarTitle: string
  activityCalendarAria: string
  activityCalendarEmpty: string
  activityCalendarLegendLess: string
  activityCalendarLegendMore: string

  // Streaks
  streaksSectionTitle: string
  streaksEmpty: string
  streakActive: string
  streakPaused: string
  streakBroken: string
  streakBrokenTooltip: string // pill série interrompue : date + reset multiplicateur (AM-6)
  streakBadgeAriaLabel: string // "{count} jours d'affilée"
  streakBadgeAriaEmpty: string // "Aucune série active"
  streakCurrentLength: string // "{n} jour(s)"
  streakUnitDay: string // unité période daily_* (jour/jours)
  streakUnitWeek: string // unité période weekly_* (semaine/semaines)
  streakBestLength: string // "Record perso : {n} {unit}" (unité jour/semaine selon le type)
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
  signalLowSample: string // badge neutre sous le seuil de matchs (DEC-8)
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
  /** Gabarit de phrase par axe (F3) : le backend ne sert plus de phrase, le
   *  front la compose. `{context}` (leviers by_mode/by_map/by_squad) est
   *  interpolé avec le libellé du contexte visé (mode, carte résolue, solo/
   *  escouade) ; les axes comportementaux sont des phrases fixes sans placeholder. */
  leverPhrase: Record<string, string>
}

const FR: AscensionText = {
  pageTitle: 'Ascension',
  pageSubtitle: 'Profil de jeu, objectifs et accomplissements.',
  tabsAriaLabel: 'Sections Ascension',
  tabProfile: 'Profil',
  tabObjectives: 'Objectifs',
  tabCoaching: 'Entraînement',
  tabRealisations: 'Réalisations',
  tabTactical: 'Tactique',
  tipsTickerAriaLabel: 'Astuces de jeu pour progresser',
  profilLayerTitle: 'Profil de jeu',
  profilLayerDescription:
    'Identité de jeu, style, niveau et tendances par contexte.',
  prestigeLayerTitle: 'Prestige — Objectifs et arcs',
  prestigeLayerDescription:
    'Système autonome pour fixer des objectifs personnels et suivre la progression. Utilisable seul, sans coaching.',
  profileSelectPlayer: 'Sélectionne un joueur pour voir les objectifs.',
  profileMyObjectives: 'Mes objectifs',
  profilePrestigeNotEnabled: "Le module Prestige n'est pas activé sur ce serveur.",
  profileAbandonObjective: 'Abandonner',
  profileAbandonTitle: 'Abandonner cet objectif ?',
  profileAbandonConfirm: 'Un cooldown de 24h s\'applique sur la métrique après l\'abandon.',
  profileAbandonCancel: 'Annuler',
  profileMyActiveObjectives: 'Mes objectifs actifs',
  profileFreeObjectives: 'Objectifs libres',
  profileNoFreeObjective: 'Aucun objectif libre actif.',
  profileNewObjective: '+ Nouvel objectif',
  profilePilotedObjectives: 'Objectifs pilotés',
  profilePilotHelp: "Le système attribue des objectifs quotidiens, hebdo et mensuels avec des plafonds.",
  profilePilotMode: 'Mode pilote',
  profilePilotEnable: 'Activer',
  profilePilotDisable: 'Désactiver',
  profilePilotPending: '…',
  profilePilotEnableCta: 'Activer le mode pilote',
  profileNewArc: '+ Nouvel arc',
  profileBrowsePresets: 'Parcourir les presets',
  profileMyActiveArcs: 'Mes arcs en cours',
  profileNoArc: 'Aucun arc en cours. Adopte un arc preset ou crée le tien.',
  squadPrestigeTitle: 'Progression Prestige',
  squadPrestigeMaxTier: 'Niveau max',
  squadPrestigeYou: 'moi',
  squadPrestigeTowardNext: '{current} / {target} PP vers {next}',
  squadPrestigeTotal: '{pp} PP au total',
  realisationsSelectPlayer: 'Sélectionne un joueur.',
  realisationsHighlights: 'Moments marquants',
  realisationsEmpty: 'Les cartes moments apparaîtront ici à la validation des premiers objectifs.',
  historyTitle: 'Historique',
  historyObjectivesTitle: 'Objectifs passés',
  historyObjectivesEmpty: 'Aucun objectif terminé pour l’instant.',
  historyArcsTitle: 'Arcs terminés',
  historyArcsEmpty: 'Aucun arc terminé.',
  historyArcCompletedOn: 'Terminé le {date}',
  historyArcStarted: 'Créé le {date}',
  historyCampaignsTitle: 'Campagnes closes',
  historyCampaignsEmpty: 'Aucune campagne close.',
  historyCampaignProgress: 'Progression',
  historyResultCompleted: 'Réussi',
  historyResultExpired: 'Expiré',
  historyResultAbandoned: 'Abandonné',
  historyResultArchived: 'Retiré',
  patternSeeMatches: 'Voir les matchs',
  recordSeePeriod: 'Voir la période',
  coachingSelectPlayer: 'Sélectionne un joueur pour voir l\'entraînement.',
  ascensionLayerTitle: 'Ascension — Coaching d\'amélioration',
  ascensionLayerDescription:
    'Analyse l\'historique pour proposer des angles d\'amélioration ciblés. S\'appuie sur Prestige (les campagnes deviennent des objectifs) — cette section peut être ignorée pour piloter sa progression soi-même.',
  activityCalendarTitle: 'Calendrier d\'activité',
  activityCalendarAria:
    "Calendrier des jours joués sur les 90 derniers jours : une case remplie par jour joué, l'intensité reflète le nombre de matchs.",
  activityCalendarEmpty: 'Aucun match sur les 90 derniers jours.',
  activityCalendarLegendLess: 'Moins',
  activityCalendarLegendMore: 'Plus',
  streaksSectionTitle: 'Mes séries',
  streaksEmpty:
    "Aucune série en cours. Joue un match aujourd'hui pour en démarrer une !",
  streakActive: 'En cours',
  streakPaused: 'Préservée par un bouclier',
  streakBroken: 'Interrompue',
  streakBrokenTooltip: 'Série interrompue le {date}. Multiplicateur PP réinitialisé.',
  streakBadgeAriaLabel: 'Série de {count} jours',
  streakBadgeAriaEmpty: 'Aucune série active',
  streakCurrentLength: '{n} jour{plural}',
  streakUnitDay: 'jour{plural}',
  streakUnitWeek: 'semaine{plural}',
  streakBestLength: 'Record perso : {n} {unit}',
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
  recordsEmpty: 'Pas encore de record. Joue quelques matchs pour faire émerger les meilleurs scores.',
  recordsHistoryEmpty: 'Aucun record battu pour le moment.',
  recordsValueLabel: 'Valeur',
  recordsAchievedAt: 'Atteint le {date}',
  recordsPreviousValue: 'Précédent : {value}',
  milestonesSectionTitle: 'Mes jalons',
  milestonesEmpty: 'Aucun jalon configuré pour ce titre.',
  milestonesEarnedAt: 'Débloqué le {date}',
  milestonesLocked: 'À débloquer',
  milestonesEarned: 'Débloqué',
  milestonesEarnedCount: '{n}/{total} débloqué{plural}',
  milestonesThreshold: 'Seuil : {n}',
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
  profileNotEnoughData: 'Joue encore quelques matchs pour débloquer le profil complet (30 minimum).',
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
    kills_vs_expected: 'Frags vs attendu',
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
  patternsNotEnoughData: 'Pas assez de matchs pour analyser les patterns (10 minimum).',
  contextType: {
    by_mode: 'Par mode',
    by_map: 'Par carte',
    by_squad: 'Solo vs Escouade',
  },
  patternWinRate: 'Taux de victoire',
  patternMatches: 'matchs',
  signalStrength: 'Force',
  signalWeakness: 'Faiblesse',
  signalNeutral: 'Neutre',
  signalLowSample: 'Échantillon faible',
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
  leverPhrase: {
    mode_selection: 'Améliore ton taux de victoire en {context}',
    map_avoidance: 'Améliore ton taux de victoire sur {context}',
    squad_play: 'Améliore ton taux de victoire en {context}',
    session_management: 'Gère tes sessions de tilt',
    session_length: 'Ajuste la durée de tes sessions',
    engagement: 'Maintiens ton engagement',
    accuracy: 'Améliore ta précision',
    radar_axis: 'Dépasse ton plafond de performance',
  },
}

const EN: AscensionText = {
  pageTitle: 'Ascension',
  pageSubtitle: 'Play profile, objectives and achievements.',
  tabsAriaLabel: 'Ascension sections',
  tabProfile: 'Profile',
  tabObjectives: 'Objectives',
  tabCoaching: 'Training',
  tabRealisations: 'Achievements',
  tabTactical: 'Tactics',
  tipsTickerAriaLabel: 'Gameplay tips to improve',
  profilLayerTitle: 'Play profile',
  profilLayerDescription:
    'Game identity, style, tier and context tendencies.',
  prestigeLayerTitle: 'Prestige — Objectives and arcs',
  prestigeLayerDescription:
    'Autonomous system to set personal objectives and track progression. Usable on its own, no coaching required.',
  profileSelectPlayer: 'Select a player to view objectives.',
  profileMyObjectives: 'My objectives',
  profilePrestigeNotEnabled: 'The Prestige module is not enabled on this server.',
  profileAbandonObjective: 'Abandon',
  profileAbandonTitle: 'Abandon this objective?',
  profileAbandonConfirm: 'A 24h cooldown applies to the metric after abandoning.',
  profileAbandonCancel: 'Cancel',
  profileMyActiveObjectives: 'My active objectives',
  profileFreeObjectives: 'Free objectives',
  profileNoFreeObjective: 'No free objective active.',
  profileNewObjective: '+ New objective',
  profilePilotedObjectives: 'Piloted objectives',
  profilePilotHelp: 'The system assigns daily/weekly/monthly objectives with caps.',
  profilePilotMode: 'Pilot mode',
  profilePilotEnable: 'Enable',
  profilePilotDisable: 'Disable',
  profilePilotPending: '…',
  profilePilotEnableCta: 'Enable pilot mode',
  profileNewArc: '+ New arc',
  profileBrowsePresets: 'Browse presets',
  profileMyActiveArcs: 'My active arcs',
  profileNoArc: 'No arc in progress. Adopt a preset arc or create your own.',
  squadPrestigeTitle: 'Prestige progression',
  squadPrestigeMaxTier: 'Max tier',
  squadPrestigeYou: 'you',
  squadPrestigeTowardNext: '{current} / {target} PP to {next}',
  squadPrestigeTotal: '{pp} PP total',
  realisationsSelectPlayer: 'Select a player.',
  realisationsHighlights: 'Highlights',
  realisationsEmpty: 'Moment cards will appear here once the first objectives are completed.',
  historyTitle: 'History',
  historyObjectivesTitle: 'Past objectives',
  historyObjectivesEmpty: 'No completed objectives yet.',
  historyArcsTitle: 'Completed arcs',
  historyArcsEmpty: 'No completed arcs.',
  historyArcCompletedOn: 'Completed on {date}',
  historyArcStarted: 'Created on {date}',
  historyCampaignsTitle: 'Closed campaigns',
  historyCampaignsEmpty: 'No closed campaigns.',
  historyCampaignProgress: 'Progress',
  historyResultCompleted: 'Completed',
  historyResultExpired: 'Expired',
  historyResultAbandoned: 'Abandoned',
  historyResultArchived: 'Removed',
  patternSeeMatches: 'See matches',
  recordSeePeriod: 'See the period',
  coachingSelectPlayer: 'Select a player to view coaching.',
  ascensionLayerTitle: 'Ascension — Improvement coaching',
  ascensionLayerDescription:
    'Analyses match history to surface targeted improvement angles. Builds on Prestige (campaigns become objectives) — this section can be ignored to drive progress independently.',
  activityCalendarTitle: 'Activity calendar',
  activityCalendarAria:
    'Calendar of days played over the last 90 days: one filled cell per day played, intensity reflects the number of matches.',
  activityCalendarEmpty: 'No match over the last 90 days.',
  activityCalendarLegendLess: 'Less',
  activityCalendarLegendMore: 'More',
  streaksSectionTitle: 'My streaks',
  streaksEmpty: 'No active streak. Play a match today to start a series!',
  streakActive: 'Active',
  streakPaused: 'Preserved by a shield',
  streakBroken: 'Broken',
  streakBrokenTooltip: 'Streak broken on {date}. PP multiplier reset.',
  streakBadgeAriaLabel: '{count}-day streak',
  streakBadgeAriaEmpty: 'No active streak',
  streakCurrentLength: '{n} day{plural}',
  streakUnitDay: 'day{plural}',
  streakUnitWeek: 'week{plural}',
  streakBestLength: 'Personal best: {n} {unit}',
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
  recordsEmpty: 'No record yet. Play a few matches to start tracking the best scores.',
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
  profileNotEnoughData: 'Play a few more matches to unlock the full profile (30 minimum).',
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
  patternsNotEnoughData: 'Not enough matches to analyze patterns (10 minimum).',
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
  signalLowSample: 'Low sample',
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
  leverPhrase: {
    mode_selection: 'Improve your win rate in {context}',
    map_avoidance: 'Improve your win rate on {context}',
    squad_play: 'Improve your win rate in {context}',
    session_management: 'Manage your tilt sessions',
    session_length: 'Adjust your session length',
    engagement: 'Sustain your engagement',
    accuracy: 'Improve your accuracy',
    radar_axis: 'Break your performance ceiling',
  },
}

export function getAscensionText(locale: Locale): AscensionText {
  return locale === 'en' ? EN : FR
}
