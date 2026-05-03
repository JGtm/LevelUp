/**
 * SquadSynergyHistoryTable — historique des matchs partagés pour la page Synergies.
 *
 * Colonnes contextuelles (pas de stats personnelles) :
 *   Ouvrir | ↗ wp | Date | Carte | Playlist | Mode |
 *   Résultat | Taux hist. | Score | Durée | MMR équipe | MMR adv. | Écart MMR
 *
 * Utilise TanStack Table v8. Pagination 20/page côté client.
 * Labels carte/playlist via useFieldMappings (assets titre).
 */
import { useMemo, useState } from 'react'
import { useNavigate } from '@tanstack/react-router'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type { SquadMatchHistoryRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenCssVar } from '@/lib/accessibility'
import { getOutcomeColor } from '@/lib/outcome-color'
import { formatDate, formatDurationMinSec } from '@/lib/formatters'
import { getSquadText } from './i18n'

const PAGE_SIZE = 20

const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
}

interface SquadSynergyHistoryTableProps {
  rows: SquadMatchHistoryRow[]
  playerSlug: string
}

function fmtMmr(v: number | undefined): string {
  if (v === undefined || v === null) return '-'
  return Math.round(v).toLocaleString()
}

function fmtDeltaMMR(v: number | undefined): JSX.Element | string {
  if (v === undefined || v === null) return '-'
  const sign = v >= 0 ? '+' : ''
  const color = v > 0 ? tokenCssVar('outcome-win') : v < 0 ? tokenCssVar('outcome-loss') : undefined
  return (
    <span className="font-mono tabular-nums" style={color ? { color } : undefined}>
      {sign}{Math.round(v)}
    </span>
  )
}

export function SquadSynergyHistoryTable({ rows, playerSlug }: SquadSynergyHistoryTableProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const intlLocale = t.intlLocale
  const labels = t.history
  const navigate = useNavigate()

  const waypointBase = `https://www.halowaypoint.com/halo-infinite/players/${encodeURIComponent(playerSlug)}/matches`

  const columns = useMemo<ColumnDef<SquadMatchHistoryRow>[]>(
    () => [
      {
        id: 'open',
        header: '',
        cell: (ctx) => (
          <button
            type="button"
            className="text-primary underline text-xs whitespace-nowrap"
            onClick={(e) => {
              e.stopPropagation()
              void navigate({
                to: '/players/$playerSlug/matches/$matchId',
                params: { playerSlug, matchId: ctx.row.original.match_id },
              })
            }}
          >
            Ouvrir
          </button>
        ),
      },
      {
        id: 'waypoint',
        header: '',
        cell: (ctx) => (
          <a
            href={`${waypointBase}/${ctx.row.original.match_id}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary text-xs whitespace-nowrap"
            onClick={(e) => e.stopPropagation()}
          >
            ↗ wp
          </a>
        ),
      },
      {
        accessorKey: 'start_time',
        header: labels.date,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS)}
          </span>
        ),
      },
      {
        accessorKey: 'map_ui',
        header: labels.map,
        cell: (ctx) => ctx.getValue<string>() || '-',
      },
      {
        accessorKey: 'playlist_name',
        header: labels.playlist,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'mode_ui',
        header: labels.mode,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {ctx.getValue<string | undefined>() || ctx.row.original.pair_name || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'outcome',
        header: labels.outcome,
        cell: (ctx) => {
          const o = ctx.getValue<number>()
          const key = o === 2 ? 'win' : o === 3 ? 'loss' : o === 1 ? 'draw' : 'dnf'
          return (
            <span style={{ color: getOutcomeColor(o), fontWeight: 600 }}>
              {labels.outcomeLabel[key]}
            </span>
          )
        },
      },
      {
        accessorKey: 'win_rate_hist',
        header: labels.winRateHist,
        cell: (ctx) => {
          const v = ctx.getValue<number | undefined>()
          if (v == null) return <span className="text-muted-foreground">—</span>
          const pct = Math.round(v * 100)
          const color = pct >= 50 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
          const total = ctx.row.original.win_rate_hist_total
          return (
            <span className="font-mono tabular-nums" style={{ color }}>
              {pct}%
              {total != null && (
                <span className="text-muted-foreground text-xs ml-1">({total})</span>
              )}
            </span>
          )
        },
      },
      {
        accessorKey: 'score_label',
        header: labels.score,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'duration_seconds',
        header: labels.duration,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {formatDurationMinSec(ctx.getValue<number | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'team_mmr_avg',
        header: labels.teamMmr,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {fmtMmr(ctx.getValue<number>())}
          </span>
        ),
      },
      {
        accessorKey: 'enemy_mmr_avg',
        header: labels.enemyMmr,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {fmtMmr(ctx.getValue<number | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'delta_mmr',
        header: labels.deltaMMR,
        cell: (ctx) => fmtDeltaMMR(ctx.getValue<number | undefined>()),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [labels, intlLocale, playerSlug, waypointBase],
  )

  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: PAGE_SIZE })

  const table = useReactTable<SquadMatchHistoryRow>({
    data: rows,
    columns,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  if (rows.length === 0) return null

  const pageIndex = table.getState().pagination.pageIndex
  const pageCount = table.getPageCount()
  const showPagination = rows.length > PAGE_SIZE

  return (
    <div className="space-y-2" data-testid="squad-synergy-history-table">
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full text-sm">
          <thead className="bg-muted border-b">
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => (
                  <th key={h.id} className="px-3 py-2 text-left whitespace-nowrap text-xs font-medium text-muted-foreground border-r border-border last:border-r-0">
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
                onClick={() =>
                  void navigate({
                    to: '/players/$playerSlug/matches/$matchId',
                    params: { playerSlug, matchId: row.original.match_id },
                  })
                }
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    void navigate({
                      to: '/players/$playerSlug/matches/$matchId',
                      params: { playerSlug, matchId: row.original.match_id },
                    })
                  }
                }}
              >
                {row.getVisibleCells().map((cell) => (
                  <td key={cell.id} className="px-3 py-2 whitespace-nowrap border-r border-border last:border-r-0">
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
          <span>{labels.totalRows(rows.length)}</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              className="rounded border border-input px-2 py-1 hover:bg-muted disabled:opacity-50"
              onClick={() => table.previousPage()}
              disabled={!table.getCanPreviousPage()}
            >
              {labels.prev}
            </button>
            <span>{labels.pageOf(pageIndex + 1, Math.max(pageCount, 1))}</span>
            <button
              type="button"
              className="rounded border border-input px-2 py-1 hover:bg-muted disabled:opacity-50"
              onClick={() => table.nextPage()}
              disabled={!table.getCanNextPage()}
            >
              {labels.next}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
