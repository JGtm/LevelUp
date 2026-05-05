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
  markIrrelevant: string
  reactivate: string
  performance: string
  rank: string
  addFavorite: string
  removeFavorite: string
  mapUnknown: string
  noRank: string
}

export const MATCH_VIEW_TEXT: Record<MatchViewLocale, MatchViewText> = {
  fr: {
    prevMatch: 'Match précédent',
    nextMatch: 'Match suivant',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: "Copier l'ID du match",
    copied: 'ID copié',
    markIrrelevant: 'Marquer comme non pertinent',
    reactivate: 'Réactiver',
    performance: 'Performance',
    rank: 'Rang',
    addFavorite: 'Ajouter aux favoris',
    removeFavorite: 'Retirer des favoris',
    mapUnknown: 'Map inconnue',
    noRank: 'Pas de rang',
  },
  en: {
    prevMatch: 'Previous match',
    nextMatch: 'Next match',
    matchCounter: (n, total) => `Match ${n}/${total}`,
    copyMatchId: 'Copy match ID',
    copied: 'ID copied',
    markIrrelevant: 'Mark as irrelevant',
    reactivate: 'Reactivate',
    performance: 'Performance',
    rank: 'Rank',
    addFavorite: 'Add to favorites',
    removeFavorite: 'Remove from favorites',
    mapUnknown: 'Unknown map',
    noRank: 'No rank',
  },
}
