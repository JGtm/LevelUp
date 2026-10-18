/**
 * filmGapStrings.ts — les textes d'une section tirée du film quand le film n'est pas lu : « Analyse du
 * film en cours » (au moins un match dont l’analyse n’est pas finie) et « Film non analysé » (aucun en cours).
 * Parité FR / EN garantie par le typage `Record<Locale, …>`. Constats factuels : ni personne, ni action
 * possible, ni mot interne (garde `textesSansPersonne.test.ts`).
 */
import type { Locale } from '@/lib/i18n/locale'

export interface FilmGapText {
  /** Libellé de l'emplacement tant que l’analyse du film n’est pas finie. */
  pending: string
  /** Placeholder quand le film n'est pas analysé et que rien n'est en cours. */
  missingTitle: string
  missingDescription: string
}

export const FILM_GAP_TEXT: Record<Locale, FilmGapText> = {
  fr: {
    pending: 'Analyse du film en attente',
    missingTitle: 'Film non analysé',
    missingDescription: 'Le film de ces matchs n’a pas été analysé : la section se remplit pour les matchs dont le film est analysé.',
  },
  en: {
    pending: 'Film analysis pending',
    missingTitle: 'Film not analysed',
    missingDescription: 'The film of these matches has not been analysed: the section fills in for the matches whose film is analysed.',
  },
}
