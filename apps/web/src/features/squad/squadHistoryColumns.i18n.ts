import type { Locale } from '@/lib/i18n/locale'

/**
 * Libellés FR/EN des colonnes ajoutées à l'historique des matchs de l'escouade
 * (dominance, soirée, cumuls de l'escouade, FDA d'escouade). Parité par typage :
 * `Record<Locale, …>`.
 */
interface SquadHistoryColumnsText {
  dominance: string
  dominanceTooltip: string
  evening: string
  eveningTooltip: string
  kills: string
  killsTooltip: string
  deaths: string
  deathsTooltip: string
  assists: string
  assistsTooltip: string
  fda: string
  fdaTooltip: string
}

export const SQUAD_HISTORY_COLUMNS: Record<Locale, SquadHistoryColumnsText> = {
  fr: {
    dominance: 'Dominance',
    dominanceTooltip:
      'Fait marquant du match : domination, humiliation, remontada, débandade ou contre-remontada.',
    evening: 'Soirée',
    eveningTooltip: 'Soirée de jeu du match : plage horaire et nombre de matchs.',
    kills: 'Frags',
    killsTooltip: "Frags cumulés de l'escouade sur le match.",
    deaths: 'Morts',
    deathsTooltip: "Morts cumulées de l'escouade sur le match.",
    assists: 'Assist.',
    assistsTooltip: "Assistances cumulées de l'escouade sur le match.",
    fda: 'FDA',
    fdaTooltip:
      "FDA de l'escouade sur le match = (frags + assistances/3) − morts, cumuls de l'escouade ; peut être négatif.",
  },
  en: {
    dominance: 'Storyline',
    dominanceTooltip:
      'Match storyline: domination, humiliation, comeback, collapse or counter-comeback.',
    evening: 'Evening',
    eveningTooltip: 'Play evening of the match: time range and number of matches.',
    kills: 'Kills',
    killsTooltip: 'Squad total kills for the match.',
    deaths: 'Deaths',
    deathsTooltip: 'Squad total deaths for the match.',
    assists: 'Assists',
    assistsTooltip: 'Squad total assists for the match.',
    fda: 'KDA',
    fdaTooltip:
      'Squad KDA for the match = (kills + assists/3) − deaths, squad totals; can be negative.',
  },
}
