/**
 * text.ts — LES TEXTES DE LA MAINMISE, en un seul endroit.
 *
 * La Mainmise (part de la durée jouée passée en tête du score, plus le nombre de retournements)
 * s'affiche sur quatre pages. Son nom, sa phrase d'aide et le message du « pas mesuré » sont les
 * MÊMES partout : un dictionnaire par page les ferait diverger. Chaque langue de
 * `config/locales.toml` y est typée (`Record<Locale, LeadShareText>`) : une langue ajoutée sans
 * ses textes ne compile pas.
 *
 * Le nom anglais est « Lead share » : « dominance » désigne déjà un badge narratif, et « lead
 * time » se lirait comme une durée alors que la grandeur est une part (en pourcentage).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import type { Locale } from '@/lib/i18n/locale'

export interface LeadShareText {
  /** Le nom de la mesure. */
  name: string
  /** UNE phrase : ce que c'est, et pourquoi certains modes affichent un tiret. */
  help: string
  /** Pourquoi un match n'a pas de chiffre (infobulle d'un tiret ou d'une colonne en pointillé). */
  notMeasured: string
  /** `{n}` retournements, au singulier. */
  changesOne: string
  /** `{n}` retournements, au pluriel. */
  changesOther: string
  /** Titre de la carte match par match (Sessions, Escouade). */
  perMatchTitle: string
  /** Titre de la carte par mois (Séries temporelles). */
  perMonthTitle: string
  /** Titre de la carte par carte (Séries temporelles). */
  perMapTitle: string
  /** État vide d'une carte dont aucun match n'est mesuré. */
  emptyNoMeasure: string
  /** Moyenne des matchs mesurés, au singulier : `{pct}` et `{n}`. */
  averageOne: string
  /** Moyenne des matchs mesurés, au pluriel : `{pct}` et `{n}`. */
  averageOther: string
  /** Légende de la colonne en pointillé d'un match non mesuré. */
  notMeasuredLegend: string
  /** Repère de la ligne à 50 %. */
  referenceLabel: string
  /** Infobulle d'un mois ou d'une carte : `{n}` match mesuré (singulier). */
  measuredOne: string
  /** Idem au pluriel. */
  measuredOther: string
  /** Infobulle d'un mois : `{measured}` matchs mesurés sur `{total}`. */
  measuredOfTotal: string
}

export const LEAD_SHARE_TEXT: Record<Locale, LeadShareText> = {
  fr: {
    name: 'Mainmise',
    help:
      "Part de la durée du match passée en tête du score par l'équipe du joueur, avec le nombre de " +
      "retournements. Un mode où le score avance trop peu souvent (drapeau, colline, bombe) n'a pas " +
      "de chiffre : un tiret le remplace.",
    notMeasured: 'Pas de chiffre pour ce match : le score de ce mode avance trop peu souvent.',
    changesOne: '{n} retournement',
    changesOther: '{n} retournements',
    perMatchTitle: 'Mainmise, match par match',
    perMonthTitle: 'Mainmise par mois',
    perMapTitle: 'Mainmise par carte',
    emptyNoMeasure: "Aucun chiffre sur cette période : le score de ces matchs avance trop peu souvent, ou leur film n'a pas été analysé.",
    averageOne: 'Moyenne {pct} sur {n} match',
    averageOther: 'Moyenne {pct} sur {n} matchs',
    notMeasuredLegend: 'Sans chiffre',
    referenceLabel: '50 %',
    measuredOne: '{n} match',
    measuredOther: '{n} matchs',
    measuredOfTotal: '{measured} matchs sur {total} avec un chiffre',
  },
  en: {
    name: 'Lead share',
    help:
      "Share of the match time the player's team spends alone in the lead of the score, with the " +
      'number of lead changes. A mode where the score moves too rarely (flag, hill, bomb) has no ' +
      'figure: a dash replaces it.',
    notMeasured: 'No figure for this match: the score of this mode moves too rarely.',
    changesOne: '{n} lead change',
    changesOther: '{n} lead changes',
    perMatchTitle: 'Lead share, match by match',
    perMonthTitle: 'Lead share by month',
    perMapTitle: 'Lead share by map',
    emptyNoMeasure: 'No figure for this range: the score of these matches moves too rarely, or their film has not been analysed.',
    averageOne: 'Average {pct} over {n} match',
    averageOther: 'Average {pct} over {n} matches',
    notMeasuredLegend: 'No figure',
    referenceLabel: '50%',
    measuredOne: '{n} match',
    measuredOther: '{n} matches',
    measuredOfTotal: '{measured} of {total} matches with a figure',
  },
}

/** Le texte de la langue du shell. */
export function useLeadShareText(): LeadShareText {
  const locale = useAppShellStore((s) => s.locale)
  return LEAD_SHARE_TEXT[locale]
}
