/**
 * Colonnes de l'historique des matchs de l'escouade qui ne portent pas sur le joueur seul :
 * soirée, dominance, frags / morts / assistances CUMULÉS de l'escouade et FDA d'escouade
 * (calculés côté Go, `attachSquadTotals` ; formule agrégat de l'ADR 0006, jamais le quotient).
 * Extraites de SquadSynergyHistoryTable pour tenir le seuil de taille de fichier.
 */
import type { ReactNode } from 'react'
import type { ColumnDef } from '@tanstack/react-table'

import { tokenCssVar } from '@/lib/accessibility'
import { asDominance } from '@/components/charts/outcomeSequence'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { dominanceLabels, DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'
import type { SquadMatchHistoryRow } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { NUMERIC_SORT, localeTextSortingFn } from '@/features/explorer/explorerMatchesClientSort'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { SQUAD_HISTORY_COLUMNS } from './squadHistoryColumns.i18n'

/** FDA signé à une décimale (« +10,0 », « −6,0 ») dans la langue du client. */
export function formatSquadFda(v: number, intlLocale: string): string {
  const abs = Math.abs(v).toLocaleString(intlLocale, { minimumFractionDigits: 1, maximumFractionDigits: 1 })
  if (v > 0) return `+${abs}`
  if (v < 0) return `\u2212${abs}`
  return abs
}

function countCell(v: number | null | undefined) {
  return (
    <span className="font-mono tabular-nums text-muted-foreground">{v == null ? '-' : v}</span>
  )
}

/** Soirée du match (`session_label`), colonne placée après la date. */
export function eveningColumn(locale: Locale): ColumnDef<SquadMatchHistoryRow> {
  const c = SQUAD_HISTORY_COLUMNS[locale]
  return {
    id: 'evening',
    accessorFn: (r) => r.session_label ?? undefined,
    header: c.evening,
    meta: { headerTooltip: c.eveningTooltip },
    sortingFn: localeTextSortingFn,
    sortDescFirst: false,
    cell: (ctx) => (
      <span className="text-muted-foreground">{ctx.getValue<string | undefined>() || '-'}</span>
    ),
  }
}

/** Dominance puis cumuls de l'escouade et son FDA, colonnes placées après le résultat. */
export function squadTotalsColumns(locale: Locale, intlLocale: string): ColumnDef<SquadMatchHistoryRow>[] {
  const c = SQUAD_HISTORY_COLUMNS[locale]
  const head = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, locale)
  const labels = dominanceLabels(locale)
  const counter = (
    id: 'squad_kills' | 'squad_deaths' | 'squad_assists',
    header: string,
    tip: string,
  ): ColumnDef<SquadMatchHistoryRow> => ({
    id,
    accessorFn: (r) => r[id] ?? undefined,
    header,
    meta: { headerTooltip: tip },
    ...NUMERIC_SORT,
    cell: (ctx) => countCell(ctx.getValue<number | undefined>()),
  })
  return [
    {
      id: 'dominance_flag',
      // Sans badge (0 ou absent) : rangé en bas du tri, comme l'Explorateur.
      accessorFn: (r) => r.dominance_flag || undefined,
      header: head('explorer.matches.col_dominance'),
      meta: { headerTooltip: c.dominanceTooltip },
      ...NUMERIC_SORT,
      cell: (ctx) => {
        const flag = asDominance(ctx.getValue<number | undefined>())
        if (!flag) return <span className="text-muted-foreground">-</span>
        return (
          <NarrativeBadge
            label={labels[flag]}
            colorVar={`--ac-${DOMINANCE_COLOR_TOKENS[flag]}`}
            size="sm"
          />
        )
      },
    },
    counter('squad_kills', head('explorer.matches.col_kills_long'), c.killsTooltip),
    counter('squad_deaths', head('explorer.matches.col_deaths_long'), c.deathsTooltip),
    counter('squad_assists', head('explorer.matches.col_assists_long'), c.assistsTooltip),
    {
      id: 'squad_fda',
      accessorFn: (r) => r.squad_fda ?? undefined,
      header: head('explorer.matches.col_fda'),
      meta: { headerTooltip: c.fdaTooltip },
      ...NUMERIC_SORT,
      cell: (ctx) => {
        const v = ctx.getValue<number | undefined>()
        if (v == null) return <span className="text-muted-foreground">-</span>
        return <span className="font-mono tabular-nums">{formatSquadFda(v, intlLocale)}</span>
      },
    },
  ]
}

/** MMR arrondi, séparateurs de la langue du navigateur ; tiret quand le titre n'en fournit pas. */
export function fmtMmr(v: number | undefined): string {
  if (v === undefined || v === null) return '-'
  return Math.round(v).toLocaleString()
}

/** Écart de MMR signé, coloré par le sens (gain / perte). */
export function fmtDeltaMMR(v: number | undefined): ReactNode {
  if (v === undefined || v === null) return '-'
  const sign = v >= 0 ? '+' : ''
  const color = v > 0 ? tokenCssVar('outcome-win') : v < 0 ? tokenCssVar('outcome-loss') : undefined
  return (
    <span className="font-mono tabular-nums" style={color ? { color } : undefined}>
      {sign}{Math.round(v)}
    </span>
  )
}
