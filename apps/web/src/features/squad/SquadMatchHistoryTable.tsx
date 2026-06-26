/**
 * SquadMatchHistoryTable — tableau historique des matchs partagés (teammates.11).
 *
 * Spec : .ai/charts_specs/teammates/11_friends_history_table.yaml
 *
 * Construit avec TanStack Table v8 :
 *  - Toutes les lignes retournées par le backend pour le scope filtré actuel
 *    (cascade, période, sessions escouade, sélection coéquipiers).
 *  - Tri par défaut : start_time DESC (déjà appliqué côté serveur).
 *  - Pagination client 20/page (boutons prev/next + indicateur page X/Y).
 *  - Clic sur une ligne → /players/$playerSlug/matches/$matchId.
 *
 * NB : le titre original « 250 derniers » du spec est volontairement écarté —
 * on liste tout ce que le backend retourne pour les filtres actifs.
 */
import { useMemo, useState } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type { SquadMatchHistoryRow } from '@/lib/api/types'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { getOutcomeColor, outcomeKey } from '@/lib/outcome-color'
import { formatDate } from '@/lib/formatters'
import { getSquadText } from './i18n'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import { useSquadFilterStore } from '@/stores/squadFilterStore'
import { useProvidesTeamMmr } from '@/lib/damage/effectiveHp'

const PAGE_SIZE = 20
const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
}

interface SquadMatchHistoryTableProps {
  rows: SquadMatchHistoryRow[]
  playerSlug: string
}

function fmtNumber(v: number | undefined | null, decimals = 1): string {
  if (v === undefined || v === null || !Number.isFinite(v)) return '-'
  return v.toFixed(decimals)
}

export function SquadMatchHistoryTable({ rows, playerSlug }: SquadMatchHistoryTableProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const intlLocale = t.intlLocale
  const { data: mappings } = useFieldMappings()
  const navigateToMatch = useNavigateToMatch(playerSlug)
  const filterContext = useSquadFilterStore((s) => s.filterContext)
  // Title-agnostic : la colonne MMR n'a de sens que si le titre courant fournit
  // un MMR d'équipe par match (gating via capability, jamais via slug). Faux pour
  // Halo 5 → on retire la colonne (pas juste la valeur).
  const providesTeamMmr = useProvidesTeamMmr()
  // Backend envoie DESC (newest first) — on inverse pour oldest-first (chronologique).
  const sortedRows = useMemo(() => [...rows].reverse(), [rows])
  const allMatchIds = useMemo(() => sortedRows.map((r) => r.match_id), [sortedRows])
  const goToSquadMatch = (matchId: string) => {
    const filterSpec = filterContextToMatchFilterSpec(filterContext)
    navigateToMatch(matchId, {
      source: 'session',
      matchIds: allMatchIds,
      filterSpec: filterSpec ?? undefined,
    })
  }

  const mapAssets = mappings?.assets?.['map']
  const playlistAssets = mappings?.assets?.['playlist']
  const labelOfMap = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const labelOfPlaylist = (p?: string) => (p ? (playlistAssets?.[p]?.label ?? p) : '-')

  const labels = t.history

  const columns = useMemo<ColumnDef<SquadMatchHistoryRow>[]>(
    () => [
      {
        accessorKey: 'start_time',
        header: labels.date,
        cell: (ctx) => formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS),
      },
      {
        accessorKey: 'map_ui',
        header: labels.map,
        cell: (ctx) => labelOfMap(ctx.getValue<string>()),
      },
      {
        accessorKey: 'playlist_name',
        header: labels.playlist,
        cell: (ctx) => labelOfPlaylist(ctx.getValue<string | undefined>()),
      },
      {
        accessorKey: 'mode_ui',
        header: labels.mode,
        cell: (ctx) => ctx.getValue<string | undefined>() || ctx.row.original.pair_name || '-',
      },
      {
        accessorKey: 'outcome',
        header: labels.outcome,
        cell: (ctx) => {
          const o = ctx.getValue<number>()
          const key = outcomeKey(o)
          return (
            <span style={{ color: getOutcomeColor(o), fontWeight: 600 }}>{labels.outcomeLabel[key]}</span>
          )
        },
      },
      {
        id: 'kda',
        header: labels.kda,
        cell: (ctx) => {
          const r = ctx.row.original
          return `${r.kills}/${r.deaths}/${r.assists}`
        },
      },
      {
        accessorKey: 'accuracy',
        header: labels.accuracy,
        cell: (ctx) => {
          // accuracy est déjà en pourcentage 0..100 (match_participants) — pas de ×100.
          const v = ctx.getValue<number | undefined>()
          return v === undefined || v === null ? '-' : `${v.toFixed(1)}%`
        },
      },
      {
        accessorKey: 'performance_score',
        header: labels.perf,
        cell: (ctx) => fmtNumber(ctx.getValue<number | undefined>(), 1),
      },
      // Colonne MMR équipe : insérée uniquement si le titre courant fournit le MMR.
      ...(providesTeamMmr
        ? [
            {
              accessorKey: 'team_mmr_avg',
              header: labels.teamMmr,
              cell: (ctx) => fmtNumber(ctx.getValue<number>(), 0),
            } as ColumnDef<SquadMatchHistoryRow>,
          ]
        : []),
      {
        accessorKey: 'session_label',
        header: labels.session,
        cell: (ctx) => ctx.getValue<string | null | undefined>() ?? '-',
      },
    ],
    // mapAssets / playlistAssets / labels sont stables sur la durée du render —
    // memoize sur leur identité pour reconstruire si la locale ou les mappings changent.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [labels, mapAssets, playlistAssets, intlLocale, providesTeamMmr],
  )

  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: PAGE_SIZE })

  const table = useReactTable<SquadMatchHistoryRow>({
    data: sortedRows,
    columns,
    state: { pagination },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  // Empty-state visible plutôt qu'un tableau qui disparaît silencieusement
  // (backlog UI 2026-06-05 : couverture empty-state).
  if (rows.length === 0) {
    return <EmptyStateNotice title={t.empty.noDataTitle} description={t.empty.noDataDescription} />
  }

  const pageIndex = table.getState().pagination.pageIndex
  const pageCount = table.getPageCount()
  const showPagination = rows.length > PAGE_SIZE

  return (
    <div className="space-y-2" data-testid="squad-match-history-table">
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
                onClick={() => goToSquadMatch(row.original.match_id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    goToSquadMatch(row.original.match_id)
                  }
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
