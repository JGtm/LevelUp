/**
 * i18n FR/EN de la feature Ascension (V2 progression).
 *
 * Pattern : dictionnaire typé par locale, accessible via getAscensionText(locale).
 * Aligné avec features/{notifications,settings,help,...}/i18n.ts.
 */
import type { StreakType } from './types'

export type AscensionLocale = 'fr' | 'en'

export interface AscensionText {
  // Page wrapper
  pageTitle: string

  // Streaks
  streaksSectionTitle: string
  streaksEmpty: string
  streakActive: string
  streakPaused: string
  streakBroken: string
  streakBadgeAriaLabel: string // "{count} jours d'affilée"
  streakBadgeAriaEmpty: string // "Aucune streak active"
  streakCurrentLength: string // "{n} jour(s)"
  streakBestLength: string // "Record perso : {n}"
  streakStarted: string // "Commencée le {date}"
  streakBrokenAt: string // "Cassée le {date}"
  streakShieldsAvailable: string // "{n} bouclier(s) disponible(s) ce mois"
  streakShieldsUsed: string // "{n} bouclier(s) utilisé(s)"
  streakPPMultiplier: string // "Multiplicateur PP : ×{value}"
  streakNextMilestone: string // "Prochain palier : {n} jours (×{mul})"
  streakTypeName: Record<StreakType, string>
  streakAtMaxMultiplier: string

  // Loading / errors
  loading: string
  errorLoading: string
}

const FR: AscensionText = {
  pageTitle: 'Ascension',
  streaksSectionTitle: 'Mes streaks',
  streaksEmpty:
    "Aucune streak en cours. Joue un match aujourd'hui pour démarrer une série !",
  streakActive: 'En cours',
  streakPaused: 'Préservée par un bouclier',
  streakBroken: 'Cassée',
  streakBadgeAriaLabel: 'Streak de {count} jours',
  streakBadgeAriaEmpty: 'Aucune streak active',
  streakCurrentLength: '{n} jour{plural}',
  streakBestLength: 'Record perso : {n} jour{plural}',
  streakStarted: 'Commencée le {date}',
  streakBrokenAt: 'Cassée le {date}',
  streakShieldsAvailable:
    '{n} bouclier{plural} disponible{plural} ce mois',
  streakShieldsUsed: '{n} bouclier{plural} utilisé{plural} ce mois',
  streakPPMultiplier: 'Multiplicateur PP : ×{value}',
  streakNextMilestone: 'Prochain palier : {n} jours (×{mul})',
  streakAtMaxMultiplier: 'Multiplicateur PP maximum atteint (×1.75)',
  streakTypeName: {
    daily_play: 'Match par jour',
    daily_perf: 'Performance par jour',
    weekly_play: '5 matchs par semaine',
    weekly_kda_threshold: 'KDA hebdomadaire',
  },
  loading: 'Chargement…',
  errorLoading: 'Erreur lors du chargement',
}

const EN: AscensionText = {
  pageTitle: 'Ascension',
  streaksSectionTitle: 'My streaks',
  streaksEmpty: 'No active streak. Play a match today to start a series!',
  streakActive: 'Active',
  streakPaused: 'Preserved by a shield',
  streakBroken: 'Broken',
  streakBadgeAriaLabel: '{count}-day streak',
  streakBadgeAriaEmpty: 'No active streak',
  streakCurrentLength: '{n} day{plural}',
  streakBestLength: 'Personal best: {n} day{plural}',
  streakStarted: 'Started on {date}',
  streakBrokenAt: 'Broken on {date}',
  streakShieldsAvailable:
    '{n} shield{plural} available this month',
  streakShieldsUsed: '{n} shield{plural} used this month',
  streakPPMultiplier: 'PP multiplier: ×{value}',
  streakNextMilestone: 'Next milestone: {n} days (×{mul})',
  streakAtMaxMultiplier: 'Maximum PP multiplier reached (×1.75)',
  streakTypeName: {
    daily_play: 'Daily match',
    daily_perf: 'Daily performance',
    weekly_play: '5 matches per week',
    weekly_kda_threshold: 'Weekly KDA',
  },
  loading: 'Loading…',
  errorLoading: 'Loading error',
}

export function getAscensionText(locale: AscensionLocale): AscensionText {
  return locale === 'en' ? EN : FR
}
