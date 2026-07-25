/**
 * i18n de l'outillage de revue — libellés des badges (FR + EN).
 *
 * Dictionnaire local plutôt qu'un manifeste TOML : l'outillage est TEMPORAIRE
 * par nature (il vit le temps d'une tournée) et ne doit pas laisser de traces
 * dans les manifestes de pages. La parité FR/EN reste garantie par le typage
 * `Record<Locale, T>`.
 */
import type { Locale } from '@/lib/i18n/locale'

import type { ChartReviewStatus } from './chart-review'

interface ReviewText {
  /** Libellé court affiché dans le badge. */
  label: string
  /** Préfixe accessible (lecteurs d'écran + attribut title). */
  aria: string
}

export const REVIEW_TEXT: Record<Locale, Record<ChartReviewStatus, ReviewText>> = {
  fr: {
    verify: { label: 'À vérifier', aria: 'Graphe à vérifier' },
    new: { label: 'Nouveau', aria: 'Nouveau graphe' },
    removal: { label: 'Suppression ?', aria: 'Graphe candidat à la suppression' },
  },
  en: {
    verify: { label: 'To verify', aria: 'Chart to verify' },
    new: { label: 'New', aria: 'New chart' },
    removal: { label: 'Remove?', aria: 'Chart considered for removal' },
  },
}
