/**
 * i18n.ts — strings UI + helper de fallback bilingue pour la feature Achievements.
 *
 * Convention : dictionnaire `Record<Locale, AchievementsText>` centralisé,
 * helper `pickLocalized` pour gérer les fallbacks quand un champ EN ou FR
 * est vide côté API (Xbox renvoie parfois un seul des deux).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { intlLocale } from '@/lib/formatters'

export type AchievementsLocale = ManifestLocale

export interface AchievementsText {
  sectionTitle: string
  summaryUnlocked: string // "Débloqués"
  summaryGamerscore: string
  summaryCompletion: string
  filterAll: string
  filterUnlocked: string
  filterInProgress: string
  filterNotStarted: string
  filterCategoryAll: string
  filterCategoryMultiplayer: string
  filterCategoryCampaign: string
  filterCategoryOther: string
  sortDefault: string
  sortDateAsc: string
  sortDateDesc: string
  empty: string
  emptyHint: string
  loadError: string
  retry: string
  unlockedAt: (date: string) => string
  progress: (current: number, target: number) => string
  scrollHintForward: string // aria-label pour scroll buttons
  scrollHintBack: string
}

export const ACHIEVEMENTS_TEXT: Record<AchievementsLocale, AchievementsText> = {
  fr: {
    sectionTitle: 'Succès Xbox',
    summaryUnlocked: 'Débloqués',
    summaryGamerscore: 'Gamerscore',
    summaryCompletion: 'Complétion',
    filterAll: 'Tous',
    filterUnlocked: 'Débloqués',
    filterInProgress: 'En cours',
    filterNotStarted: 'Non commencé',
    filterCategoryAll: 'Toutes catégories',
    filterCategoryMultiplayer: 'Multijoueur',
    filterCategoryCampaign: 'Campagne',
    filterCategoryOther: 'Autres',
    sortDefault: 'Défaut',
    sortDateAsc: 'Date ↑',
    sortDateDesc: 'Date ↓',
    empty: 'Aucun succès en base.',
    emptyHint: 'Lance le backfill : levelup sync-achievements --gamertag <gt>',
    loadError: 'Erreur lors du chargement des succès.',
    retry: 'Réessayer',
    unlockedAt: (date) => `Débloqué le ${date}`,
    progress: (current, target) => `${current} / ${target}`,
    scrollHintForward: 'Voir les succès suivants',
    scrollHintBack: 'Voir les succès précédents',
  },
  en: {
    sectionTitle: 'Xbox Achievements',
    summaryUnlocked: 'Unlocked',
    summaryGamerscore: 'Gamerscore',
    summaryCompletion: 'Completion',
    filterAll: 'All',
    filterUnlocked: 'Unlocked',
    filterInProgress: 'In progress',
    filterNotStarted: 'Not started',
    filterCategoryAll: 'All categories',
    filterCategoryMultiplayer: 'Multiplayer',
    filterCategoryCampaign: 'Campaign',
    filterCategoryOther: 'Other',
    sortDefault: 'Default',
    sortDateAsc: 'Date ↑',
    sortDateDesc: 'Date ↓',
    empty: 'No achievements in the database.',
    emptyHint: 'Run the backfill: levelup sync-achievements --gamertag <gt>',
    loadError: 'Failed to load achievements.',
    retry: 'Retry',
    unlockedAt: (date) => `Unlocked on ${date}`,
    progress: (current, target) => `${current} / ${target}`,
    scrollHintForward: 'Show next achievements',
    scrollHintBack: 'Show previous achievements',
  },
}

/**
 * pickLocalized — sélectionne la chaîne dans la locale demandée avec fallback.
 *
 * Règles :
 *   - locale="fr" : prend `fr` si non vide, sinon `en`, sinon "" (les deux vides)
 *   - locale="en" : prend `en` si non vide, sinon `fr`, sinon ""
 *
 * Utile parce que l'API Xbox Achievements peut renvoyer un seul des deux
 * pour certains achievements localisés partiellement, ou les deux vides
 * pour des champs optionnels (locked_desc).
 *
 * @param en - chaîne anglaise (peut être vide ou undefined)
 * @param fr - chaîne française (peut être vide ou undefined)
 * @param locale - locale courante de l'app
 */
export function pickLocalized(
  en: string | undefined,
  fr: string | undefined,
  locale: AchievementsLocale,
): string {
  const enStr = en ?? ''
  const frStr = fr ?? ''
  if (locale === 'fr') {
    return frStr || enStr
  }
  return enStr || frStr
}

/**
 * formatUnlockedDate — formate une date ISO en chaîne localisée courte.
 * Retourne null si la date est absente ou invalide.
 */
export function formatUnlockedDate(
  iso: string | undefined,
  locale: AchievementsLocale,
): string | null {
  if (!iso) return null
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return null
  return date.toLocaleDateString(intlLocale(locale), {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}
