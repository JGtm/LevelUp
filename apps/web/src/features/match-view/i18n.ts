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
  outcomeWin: string
  outcomeLoss: string
  outcomeDraw: string
  outcomeDnf: string
  fromDate: string
  toDate: string
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
    outcomeWin: 'Victoires',
    outcomeLoss: 'Défaites',
    outcomeDraw: 'Égalités',
    outcomeDnf: 'Non terminés',
    fromDate: 'Depuis',
    toDate: "Jusqu'au",
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
    outcomeWin: 'Wins',
    outcomeLoss: 'Losses',
    outcomeDraw: 'Draws',
    outcomeDnf: 'DNF',
    fromDate: 'From',
    toDate: 'To',
  },
}

/**
 * buildContextLabel — produit un label localisé depuis un MatchFilterSpec.
 *
 * Phase 2b : utilisé quand la cascade tombe sur l'API avec spec URL (pas de
 * filtersLabel pré-localisé dans le matchNavContext). Format compact :
 *   "Classée Arena · Victoires · Depuis 01/04/2026"
 */
import type { MatchFilterSpec } from '@/lib/match-nav/navContext'

export function buildContextLabel(
  spec: MatchFilterSpec | null | undefined,
  locale: MatchViewLocale,
): string {
  if (!spec) return ''
  const t = MATCH_VIEW_TEXT[locale]
  const parts: string[] = []
  if (spec.playlist_name) parts.push(spec.playlist_name)
  if (spec.mode_category) parts.push(spec.mode_category)
  if (spec.outcome) {
    const map: Record<string, string> = {
      win: t.outcomeWin,
      loss: t.outcomeLoss,
      draw: t.outcomeDraw,
      dnf: t.outcomeDnf,
    }
    const lbl = map[spec.outcome]
    if (lbl) parts.push(lbl)
  }
  if (spec.date_from || spec.date_to) {
    const intlLocale = locale === 'en' ? 'en-GB' : 'fr-FR'
    const fmt = (iso: string) => {
      const d = new Date(iso)
      if (isNaN(d.getTime())) return iso
      return new Intl.DateTimeFormat(intlLocale, { day: '2-digit', month: '2-digit', year: 'numeric' }).format(d)
    }
    if (spec.date_from && spec.date_to) {
      parts.push(`${fmt(spec.date_from)} → ${fmt(spec.date_to)}`)
    } else if (spec.date_from) {
      parts.push(`${t.fromDate} ${fmt(spec.date_from)}`)
    } else if (spec.date_to) {
      parts.push(`${t.toDate} ${fmt(spec.date_to)}`)
    }
  }
  if (spec.session_id) parts.push(`#${spec.session_id}`)
  return parts.join(' · ')
}
