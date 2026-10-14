import type { Locale } from '@/lib/i18n/locale'

/**
 * Libellés FR/EN des colonnes ajoutées à l'historique des matchs de l'escouade (soirée, et les
 * infobulles propres aux cumuls d'escouade). Parité par typage : `Record<Locale, …>`.
 *
 * Les EN-TÊTES de dominance, de frags, de morts, d'assistances et de FDA ne sont pas écrits ici :
 * ils viennent du manifeste de l'Explorateur (`explorer.matches.col_*`), la même source que le
 * tableau des matchs d'un joueur — un libellé de statistique ne se redéfinit pas par feature
 * (garde-rail `no-field-label-dictionary.test.ts`).
 */
interface SquadHistoryColumnsText {
  dominanceTooltip: string
  evening: string
  eveningTooltip: string
  killsTooltip: string
  deathsTooltip: string
  assistsTooltip: string
  fdaTooltip: string
}

export const SQUAD_HISTORY_COLUMNS: Record<Locale, SquadHistoryColumnsText> = {
  fr: {
    dominanceTooltip:
      'Fait marquant du match : domination, humiliation, remontada, débandade ou contre-remontada.',
    evening: 'Soirée',
    eveningTooltip: 'Soirée de jeu du match : plage horaire et nombre de matchs.',
    killsTooltip: "Frags cumulés de l'escouade sur le match.",
    deathsTooltip: "Morts cumulées de l'escouade sur le match.",
    assistsTooltip: "Assistances cumulées de l'escouade sur le match.",
    fdaTooltip:
      "FDA de l'escouade sur le match = (frags + assistances/3) − morts, cumuls de l'escouade ; peut être négatif.",
  },
  en: {
    dominanceTooltip:
      'Match storyline: domination, humiliation, comeback, collapse or counter-comeback.',
    evening: 'Evening',
    eveningTooltip: 'Play evening of the match: time range and number of matches.',
    killsTooltip: 'Squad total kills for the match.',
    deathsTooltip: 'Squad total deaths for the match.',
    assistsTooltip: 'Squad total assists for the match.',
    fdaTooltip:
      'Squad KDA for the match = (kills + assists/3) − deaths, squad totals; can be negative.',
  },
}
