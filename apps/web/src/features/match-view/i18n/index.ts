/**
 * i18n strings — feature match-view (header refonte 2026-05-05, mock C).
 *
 * Strings UI dédiées au header (nav + actions + labels), un fichier par langue (`fr.ts`,
 * `en.ts`) pour un type déclaré une fois (`contract.ts`). Les libellés de
 * stats Halo (modes, maps, playlists) viennent du pipeline backend, dans la langue
 * de la requête (asset_translations / mode_name_tr) — voir buildMatchHeader Go.
 */

import { intlLocale } from '@/lib/formatters/intlLocale'
import type { Locale } from '@/lib/i18n/locale'
import { labelOfKey, type MatchFilterSpec } from '@/lib/match-nav/navContext'
import type { MatchViewText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { MatchViewText } from './contract'

/** Alias de compat du type central `Locale` (lib/i18n/locale) : conservé car consommé dans plus de 5 fichiers du header match-view. */
export type MatchViewLocale = Locale

export const MATCH_VIEW_TEXT: Record<MatchViewLocale, MatchViewText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

/**
 * buildContextLabel — produit un label localisé depuis un MatchFilterSpec.
 *
 * Phase 2b : utilisé quand la cascade tombe sur l'API avec spec URL (pas de
 * filtersLabel pré-localisé dans le matchNavContext). Format compact :
 *   "Classée Arena · Victoires · Depuis 01/04/2026"
 */
export function buildContextLabel(
  spec: MatchFilterSpec | null | undefined,
  locale: MatchViewLocale,
): string {
  if (!spec) return ''
  const t = MATCH_VIEW_TEXT[locale]
  const parts: string[] = []
  if (spec.playlist_names?.length) {
    parts.push(spec.playlist_names.map((k) => labelOfKey(k, spec.playlist_labels)).join(', '))
  }
  if (spec.mode_categories?.length) {
    parts.push(spec.mode_categories.map((k) => labelOfKey(k, spec.mode_labels)).join(', '))
  }
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
    const fmt = (iso: string) => {
      const d = new Date(iso)
      if (isNaN(d.getTime())) return iso
      return new Intl.DateTimeFormat(intlLocale(locale), { day: '2-digit', month: '2-digit', year: 'numeric' }).format(d)
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

// Note Phase 2c (2026-05-07) : `buildDescriptorLabel` extrait dans
// `./descriptorLabel.ts` pour respecter la limite de 500 lignes/fichier
// (CLAUDE.md §5). `buildContextLabel` (au-dessus, fallback filterSpec)
// reste ici car il dépend uniquement de MATCH_VIEW_TEXT et n'est pas
// amené à grossir.
export { buildDescriptorLabel } from '../descriptorLabel'
