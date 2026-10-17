/**
 * filmGapStrings.ts — les textes des sections de l'onglet Emprise dont le film du périmètre n'est pas
 * lu : « Analyse du film en cours » (au moins un match sans résumé d'usage dont l'analyse tourne) et
 * « Film non analysé » (aucun en cours). Fichier à part : `empriseStrings.ts` frôle le seuil de taille.
 * Parité FR / EN garantie par le typage `Record<Locale, …>`. Descriptions factuelles, sans personne.
 */
import type { Locale } from '@/lib/i18n/locale'

export interface FilmGapText {
  /** Libellé de l'emplacement tant que l'analyse du film tourne. */
  pending: string
  /** Placeholder quand le film n'est pas analysé et que rien n'est en cours. */
  missingTitle: string
  missingDescription: string
}

export const FILM_GAP_TEXT: Record<Locale, FilmGapText> = {
  fr: {
    pending: 'Analyse du film en cours',
    missingTitle: 'Film non analysé',
    missingDescription: 'Le film des matchs de cette sélection n’a pas encore été lu : la section se remplit quand le film est analysé.',
  },
  en: {
    pending: 'Film analysis in progress',
    missingTitle: 'Film not analysed',
    missingDescription: 'The film of the matches in this selection has not been read yet: the section fills in once the film is analysed.',
  },
}
