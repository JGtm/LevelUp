/**
 * contract.ts — Le TYPE du dictionnaire de la feature Ascension : une seule déclaration, que
 * `fr.ts`, `en.ts` et toute langue ajoutée doivent satisfaire (`satisfies AscensionText`).
 */
import type { StreakType, ContextType, BehaviorType } from '../types'

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
  activityCalendarUnavailable: string
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
