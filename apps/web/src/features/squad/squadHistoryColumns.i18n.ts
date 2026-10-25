import type { Locale } from '@/lib/i18n/locale'

/**
 * Libellés FR/EN des colonnes ajoutées à l'historique des matchs de l'escouade (infobulles
 * propres à la dominance et aux cumuls d'escouade). Parité par typage : `Record<Locale, …>`.
 *
 * Les EN-TÊTES de dominance, de frags, de morts et d'assistances ne sont pas écrits ici :
 * ils viennent du manifeste de l'Explorateur (`explorer.matches.col_*`), la même source que le
 * tableau des matchs d'un joueur — un libellé de statistique ne se redéfinit pas par feature
 * (garde-rail `no-field-label-dictionary.test.ts`). La performance d'équipe, propre à l'escouade,
 * a son en-tête ici.
 */
interface SquadHistoryColumnsText {
  dominanceTooltip: string
  killsTooltip: string
  deathsTooltip: string
  assistsTooltip: string
  performanceHeader: string
  performanceTooltip: string
}

export const SQUAD_HISTORY_COLUMNS: Record<Locale, SquadHistoryColumnsText> = {
  fr: {
    dominanceTooltip:
      'Fait marquant du match : domination, humiliation, remontada, débandade ou contre-remontada.',
    killsTooltip: "Frags cumulés de l'escouade sur le match.",
    deathsTooltip: "Morts cumulées de l'escouade sur le match.",
    assistsTooltip: "Assistances cumulées de l'escouade sur le match.",
    performanceHeader: "Performance d'équipe",
    performanceTooltip:
      "Moyenne des scores de performance (0–100) des membres de l'escouade présents sur le match. Chaque score est relatif à l'historique de son joueur : 50 = niveau habituel.",
  },
  en: {
    dominanceTooltip:
      'Match storyline: domination, humiliation, comeback, collapse or counter-comeback.',
    killsTooltip: 'Squad total kills for the match.',
    deathsTooltip: 'Squad total deaths for the match.',
    assistsTooltip: 'Squad total assists for the match.',
    performanceHeader: 'Team performance',
    performanceTooltip:
      "Average performance score (0–100) of the squad members present in the match. Each score is relative to that player's own history: 50 = usual level.",
  },
}
