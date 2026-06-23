/**
 * i18n — feature Commendations (totaux à vie natifs Halo 5, AXE B).
 * Pattern feature-local (FR/EN), comme match-view.
 */
export type CommendationsLocale = 'fr' | 'en'

export interface CommendationsText {
  pageTitle: string
  subtitle: string
  empty: string
  emptyHint: string
  total: (n: number) => string
  commendationsCount: (n: number) => string
}

export const COMMENDATIONS_TEXT: Record<CommendationsLocale, CommendationsText> = {
  fr: {
    pageTitle: 'Commendations',
    subtitle: 'Totaux à vie par catégorie',
    empty: 'Aucune commendation',
    emptyHint:
      'Les totaux apparaîtront après une synchronisation des matchs récents.',
    total: (n) => `${n.toLocaleString('fr-FR')}`,
    commendationsCount: (n) => `${n} commendation${n > 1 ? 's' : ''}`,
  },
  en: {
    pageTitle: 'Commendations',
    subtitle: 'Lifetime totals by category',
    empty: 'No commendations',
    emptyHint: 'Totals will appear after syncing recent matches.',
    total: (n) => `${n.toLocaleString('en-US')}`,
    commendationsCount: (n) => `${n} commendation${n > 1 ? 's' : ''}`,
  },
}
