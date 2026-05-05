/**
 * i18n strings — feature match-view (header refonte 2026-05-05, mock C).
 *
 * Strings UI dédiées au header (nav + actions + labels). Les libellés de
 * stats Halo (modes, maps, playlists) viennent du pipeline backend FR
 * (asset_translations / mode_name_tr) — voir buildMatchHeader Go.
 */

export type MatchViewLocale = 'fr' | 'en'

export interface MatchViewText {
  prevMatch: string
  nextMatch: string
  matchCounter: (n: number, total: number) => string
  copyMatchId: string
  copied: string
  copyShort: string
  copyTooltip: string
  markIrrelevant: string
  reactivate: string
  excludeShort: string
  excludeTooltip: string
  reactivateTooltip: string
  performance: string
  rank: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  noRank: string
  exitContext: string
}

export const MATCH_VIEW_TEXT: Record<MatchViewLocale, MatchViewText> = {
  fr: {
    prevMatch: 'Match précédent',
    nextMatch: 'Match suivant',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: "Copier l'ID du match",
    copied: 'Copié',
    copyShort: 'Copier ID',
    copyTooltip: "Copier l'identifiant unique de ce match dans le presse-papier",
    markIrrelevant: 'Marquer comme non pertinent',
    reactivate: 'Réactiver',
    excludeShort: 'Exclure',
    excludeTooltip: 'Exclure ce match des statistiques et analyses',
    reactivateTooltip: 'Réintégrer ce match dans les statistiques',
    performance: 'Performance',
    rank: 'Rang',
    addFavorite: 'Ajouter aux favoris',
    removeFavorite: 'Retirer des favoris',
    mapUnknown: 'Map inconnue',
    noRank: 'Pas de rang',
    exitContext: 'Sortir du contexte',
  },
  en: {
    prevMatch: 'Previous match',
    nextMatch: 'Next match',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: 'Copy match ID',
    copied: 'Copied',
    copyShort: 'Copy ID',
    copyTooltip: "Copy this match's unique identifier to clipboard",
    markIrrelevant: 'Mark as irrelevant',
    reactivate: 'Reactivate',
    excludeShort: 'Exclude',
    excludeTooltip: 'Exclude this match from stats and analyses',
    reactivateTooltip: 'Re-include this match in stats and analyses',
    performance: 'Performance',
    rank: 'Rank',
    addFavorite: 'Add to favorites',
    removeFavorite: 'Remove from favorites',
    mapUnknown: 'Unknown map',
    noRank: 'No rank',
    exitContext: 'Exit context',
  },
}
