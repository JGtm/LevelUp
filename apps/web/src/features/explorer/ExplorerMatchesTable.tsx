/**
 * ExplorerMatchesTable — tableau historique mode "Matchs" de l'Explorer.
 *
 * Repris depuis features/squad/SquadMatchHistoryTable.tsx (même TanStack Table v8,
 * même style visuel, même pattern de pagination 20/page). Adapté à ExplorerMatchRow :
 *  - Colonnes : Date | Carte | Playlist | Mode | Résultat | FDA | Perf (color) | ΔPerf | Rang | ΔRang
 *  - Pas de win_rate_hist
 *  - Lignes colorées par outcome (OUTCOME_ROW_BG)
 *  - Clic ligne → /players/$slug/matches/$id avec contexte filterSpec
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
import { Badge } from '@/components/ui/badge'
import { tokenVar, tokenCssVar } from '@/lib/accessibility'
import { mmrDeltaScale } from '@/lib/accessibility/scales'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'

const PAGE_SIZE = 20

const OUTCOME_ROW_BG: Record<number, string> = {
  1: 'bg-info/20',
  2: 'bg-success/20',
  3: 'bg-destructive/20',
  4: 'bg-muted/20',
}

const OUTCOME_BADGE: Record<number, 'success' | 'destructive' | 'secondary' | 'outline'> = {
  1: 'secondary',
  2: 'success',
  3: 'destructive',
  4: 'outline',
}

interface Props {
  rows: ExplorerMatchRow[]
  playerSlug: string
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
  const labelOfPlaylist = (p?: string) => (p ? (playlistAssets?.[p]?.label ?? p) : '—')

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

  const columns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
    () => [
      {
        accessorKey: 'start_time',
        header: t('explorer.matches.col_date'),
        cell: (ctx) => {
          const r = ctx.row.original
          return (
            <span className="text-muted-foreground whitespace-nowrap">
              {r.start_time_label ?? new Date(r.start_time).toLocaleDateString(intlLocale)}
            </span>
          )
        },
      },
      {
        accessorKey: 'map_ui',
        header: t('explorer.filters.map'),
        cell: (ctx) => labelOfMap(ctx.getValue<string>()),
      },
      {
        accessorKey: 'playlist_label',
        header: t('explorer.filters.playlist'),
        cell: (ctx) => labelOfPlaylist(ctx.getValue<string | undefined>()),
      },
      {
        accessorKey: 'mode_ui',
        header: t('explorer.filters.mode'),
        cell: (ctx) => ctx.getValue<string | undefined>() ?? '—',
      },
      {
        accessorKey: 'outcome_code',
        header: t('explorer.matches.col_outcome'),
        cell: (ctx) => {
          const r = ctx.row.original
          return (
            <Badge variant={OUTCOME_BADGE[r.outcome_code] ?? 'secondary'}>{r.outcome_label}</Badge>
          )
        },
      },
      {
        id: 'fda',
        header: t('explorer.matches.col_fda'),
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.kills == null) return <span className="text-muted-foreground">—</span>
          return (
            <span className="font-mono text-xs text-muted-foreground">
              {r.kills}/{r.deaths ?? '?'}/{r.assists ?? '?'}
            </span>
          )
        },
      },
      {
        accessorKey: 'perf_score',
        header: t('explorer.matches.col_perf'),
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.perf_score == null || !r.perf_tier) {
            return <span className="text-muted-foreground text-xs">—</span>
          }
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
        header: t('explorer.matches.col_delta_perf'),
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return <span className="text-muted-foreground">—</span>
          const color =
            v > 0
              ? tokenVar('perf-tier-1' as SemanticToken)
              : v < 0
                ? tokenVar('perf-tier-5' as SemanticToken)
                : undefined
          return (
            <span className="font-mono text-xs" style={{ color }}>
              {v >= 0 ? '+' : ''}
              {v}
            </span>
          )
        },
      },
      {
        accessorKey: 'skill_tier_label',
        header: t('explorer.matches.col_rank'),
        cell: (ctx) => {
          const v = ctx.getValue<string | null | undefined>()
          return v ?? <span className="text-muted-foreground">—</span>
        },
      },
      {
        accessorKey: 'delta_mmr',
        header: t('explorer.matches.col_delta_rank'),
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return <span className="text-muted-foreground">—</span>
          return (
            <span
              className="font-mono text-sm"
              style={{ color: tokenCssVar(mmrDeltaScale(v)) }}
            >
              {v >= 0 ? '+' : ''}
              {v.toFixed(0)}
            </span>
          )
        },
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
    <div className="space-y-2">
      <div className="overflow-x-auto rounded-md border border-border bg-card">
        <table className="w-full text-sm">
          <thead className="bg-muted border-b">
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => (
                  <th
                    key={h.id}
                    className="px-3 py-2 text-left whitespace-nowrap text-xs font-medium text-muted-foreground"
                  >
                    {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-border">
            {table.getRowModel().rows.map((row) => {
              const bg = OUTCOME_ROW_BG[row.original.outcome_code] ?? ''
              return (
                <tr
                  key={row.id}
                  className={`cursor-pointer transition-colors hover:brightness-125 ${bg}`}
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
              )
            })}
          </tbody>
        </table>
      </div>

      {showPagination && (
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          <span>
            {t('explorer.matches.count_label', { n: rows.length })}
          </span>
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
