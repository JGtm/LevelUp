/**
 * ExplorerMatchesTable — tableau historique du mode "Matchs" de l'Explorer.
 *
 * COPIE STRICTE du pattern visuel de features/squad/SquadMatchHistoryTable.tsx :
 *  - Mêmes classes Tailwind (rounded-md border, bg-muted thead, hover:bg-primary/10)
 *  - Outcome rendu en texte coloré via getOutcomeColor (pas de Badge, pas de tinting de ligne)
 *  - TanStack Table v8, pagination client 20/page, formatDate, useFieldMappings
 *
 * Différence vs squad : 4 colonnes additionnelles insérées APRÈS FDA :
 *  - Perf (color-graded par perf_tier)
 *  - ΔPerf (signe coloré)
 *  - Rang (skill_tier_label)
 *  - ΔRang (delta_mmr coloré via mmrDeltaScale)
 */
import { useMemo, useState } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type { ExplorerMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenVar, tokenCssVar } from '@/lib/accessibility'
import { mmrDeltaScale } from '@/lib/accessibility/scales'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { getOutcomeColor } from '@/lib/outcome-color'
import { formatDate } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

const PAGE_SIZE = 20
const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
}

interface Props {
  rows: ExplorerMatchRow[]
  playerSlug: string
}

function outcomeKey(outcome: number): 'win' | 'loss' | 'draw' | 'dnf' {
  switch (outcome) {
    case 2:
      return 'win'
    case 3:
      return 'loss'
    case 1:
      return 'draw'
    default:
      return 'dnf'
  }
}

function fmtNumber(v: number | undefined | null, decimals = 1): string {
  if (v === undefined || v === null || !Number.isFinite(v)) return '-'
  return v.toFixed(decimals)
}

export function ExplorerMatchesTable({ rows, playerSlug }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)
  const intlLocale = locale === 'en' ? 'en-US' : 'fr-FR'

  const { data: mappings } = useFieldMappings()
  const mapAssets = mappings?.assets?.['map']
  const playlistAssets = mappings?.assets?.['playlist']
  const labelOfMap = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const labelOfPlaylist = (p?: string | null) => (p ? (playlistAssets?.[p]?.label ?? p) : '-')

  const navigateToMatch = useNavigateToMatch(playerSlug)
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const allMatchIds = useMemo(() => rows.map((r) => r.match_id), [rows])
  const goToMatch = (matchId: string) => {
    const filterSpec = filterContextToMatchFilterSpec(filterContext)
    navigateToMatch(matchId, {
      source: 'history',
      matchIds: allMatchIds,
      filterSpec: filterSpec ?? undefined,
    })
  }

  // Labels colonnes (i18n)
  const lblDate = t('explorer.matches.col_date')
  const lblMap = t('explorer.filters.map')
  const lblPlaylist = t('explorer.filters.playlist')
  const lblMode = t('explorer.filters.mode')
  const lblOutcome = t('explorer.matches.col_outcome')
  const lblFda = t('explorer.matches.col_fda')
  const lblPerf = t('explorer.matches.col_perf')
  const lblDeltaPerf = t('explorer.matches.col_delta_perf')
  const lblRank = t('explorer.matches.col_rank')
  const lblDeltaRank = t('explorer.matches.col_delta_rank')
  const lblTeamMmr = t('explorer.matches.col_team_mmr')
  const outcomeLabels: Record<'win' | 'loss' | 'draw' | 'dnf', string> = {
    win: t('explorer.matches.outcome_win'),
    loss: t('explorer.matches.outcome_loss'),
    draw: t('explorer.matches.outcome_draw'),
    dnf: t('explorer.matches.outcome_dnf'),
  }

  const columns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
    () => [
      {
        accessorKey: 'start_time',
        header: lblDate,
        cell: (ctx) => formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS),
      },
      {
        accessorKey: 'map_ui',
        header: lblMap,
        cell: (ctx) => labelOfMap(ctx.getValue<string>()),
      },
      {
        accessorKey: 'playlist_label',
        header: lblPlaylist,
        cell: (ctx) => labelOfPlaylist(ctx.getValue<string | null | undefined>()),
      },
      {
        accessorKey: 'mode_ui',
        header: lblMode,
        cell: (ctx) => ctx.getValue<string | undefined>() ?? '-',
      },
      {
        accessorKey: 'outcome_code',
        header: lblOutcome,
        cell: (ctx) => {
          const o = ctx.getValue<number>()
          const key = outcomeKey(o)
          return (
            <span style={{ color: getOutcomeColor(o), fontWeight: 600 }}>
              {outcomeLabels[key]}
            </span>
          )
        },
      },
      {
        id: 'fda',
        header: lblFda,
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.kills == null) return '-'
          return `${r.kills}/${r.deaths ?? 0}/${r.assists ?? 0}`
        },
      },
      {
        accessorKey: 'perf_score',
        header: lblPerf,
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.perf_score == null || !r.perf_tier) return '-'
          return (
            <span
              className="font-semibold tabular-nums"
              style={{ color: tokenVar(`perf-tier-${r.perf_tier}` as SemanticToken) }}
            >
              {r.perf_score}
            </span>
          )
        },
      },
      {
        accessorKey: 'delta_perf',
        header: lblDeltaPerf,
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return '-'
          const color =
            v > 0
              ? tokenVar('perf-tier-1' as SemanticToken)
              : v < 0
                ? tokenVar('perf-tier-5' as SemanticToken)
                : undefined
          return (
            <span style={{ color }}>
              {v >= 0 ? '+' : ''}
              {v}
            </span>
          )
        },
      },
      {
        accessorKey: 'skill_tier_label',
        header: lblRank,
        cell: (ctx) => ctx.getValue<string | null | undefined>() ?? '-',
      },
      {
        accessorKey: 'delta_mmr',
        header: lblDeltaRank,
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return '-'
          return (
            <span style={{ color: tokenCssVar(mmrDeltaScale(v)) }}>
              {v >= 0 ? '+' : ''}
              {v.toFixed(0)}
            </span>
          )
        },
      },
      {
        accessorKey: 'team_mmr',
        header: lblTeamMmr,
        cell: (ctx) => fmtNumber(ctx.getValue<number | null | undefined>(), 0),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [intlLocale, mapAssets, playlistAssets, locale],
  )

  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: PAGE_SIZE })

  const table = useReactTable<ExplorerMatchRow>({
    data: rows,
    columns,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  if (rows.length === 0) {
    return (
      <div className="rounded-md border border-border bg-card px-4 py-8 text-center text-muted-foreground">
        {t('explorer.matches.empty_row')}
      </div>
    )
  }

  const pageIndex = table.getState().pagination.pageIndex
  const pageCount = table.getPageCount()
  const showPagination = rows.length > PAGE_SIZE

  return (
    <div className="space-y-2" data-testid="explorer-matches-table">
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full text-sm">
          <thead className="bg-muted border-b">
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => (
                  <th key={h.id} className="px-3 py-2 text-left whitespace-nowrap">
                    {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-border">
            {table.getRowModel().rows.map((row) => (
              <tr
                key={row.id}
                className="cursor-pointer transition-colors hover:bg-primary/10"
                role="button"
                tabIndex={0}
                onClick={() => goToMatch(row.original.match_id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') goToMatch(row.original.match_id)
                }}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-3 py-2 whitespace-nowrap">
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {showPagination && (
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>{t('explorer.matches.count_label', { n: rows.length })}</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="rounded border border-input px-2 py-1 hover:bg-muted disabled:opacity-50"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              {t('explorer.player.prev_page')}
            </button>
            <span>
              {t('explorer.player.page_info', {
                page: pageIndex + 1,
                total: Math.max(pageCount, 1),
              })}
            </span>
            <button
              type="button"
              className="rounded border border-input px-2 py-1 hover:bg-muted disabled:opacity-50"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              {t('explorer.player.next_page')}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
