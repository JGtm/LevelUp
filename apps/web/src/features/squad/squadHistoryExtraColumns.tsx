/**
 * Colonnes de l'historique des matchs de l'escouade qui ne portent pas sur le joueur seul :
 * dominance, frags / morts / assistances CUMULÉS de l'escouade et performance d'équipe
 * (calculés côté Go, `attachSquadTotals` / `attachSquadPerformance`).
 * Extraites de SquadSynergyHistoryTable pour tenir le seuil de taille de fichier.
 */
import type { ReactNode } from 'react'
import type { ColumnDef } from '@tanstack/react-table'

import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { asDominance } from '@/components/charts/outcomeSequence'
import { NarrativeBadge } from '@/components/feedback/NarrativeBadge'
import { dominanceLabels, DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'
import type { SquadMatchHistoryRow } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { SQUAD_HISTORY_COLUMNS } from './squadHistoryColumns.i18n'

/**
 * Les tris que le tableau injecte (ceux de l'Explorateur, importés par lui seul : pas d'import
 * d'une feature à l'autre depuis ce fichier).
 */
export interface HistorySorts {
  /** Tri numérique, valeurs absentes en bas. */
  numeric: { readonly sortingFn: 'basic'; readonly sortUndefined: 'last' }
}

function countCell(v: number | null | undefined) {
  return (
    <span className="font-mono tabular-nums text-muted-foreground">{v == null ? '-' : v}</span>
  )
}

/** Dominance puis cumuls de l'escouade et sa performance, colonnes placées après le résultat. */
export function squadTotalsColumns(
  locale: Locale,
  sorts: HistorySorts,
): ColumnDef<SquadMatchHistoryRow>[] {
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
    ...sorts.numeric,
    cell: (ctx) => countCell(ctx.getValue<number | undefined>()),
  })
  return [
    {
      id: 'dominance_flag',
      // Sans badge (0 ou absent) : rangé en bas du tri, comme l'Explorateur.
      accessorFn: (r) => r.dominance_flag || undefined,
      header: head('explorer.matches.col_dominance'),
      meta: { headerTooltip: c.dominanceTooltip },
      ...sorts.numeric,
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
      id: 'squad_performance',
      accessorFn: (r) => r.squad_performance ?? undefined,
      header: c.performanceHeader,
      meta: { headerTooltip: c.performanceTooltip },
      ...sorts.numeric,
      cell: (ctx) => {
        const r = ctx.row.original
        if (r.squad_performance == null || !r.squad_perf_tier) {
          return <span className="text-muted-foreground">-</span>
        }
        return (
          <span
            className="font-semibold tabular-nums"
            style={{ color: tokenCssVar(`perf-tier-${r.squad_perf_tier}` as SemanticToken) }}
          >
            {r.squad_performance}
          </span>
        )
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
