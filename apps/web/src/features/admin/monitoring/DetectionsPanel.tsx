/**
 * DetectionsPanel — table de triage des détections persistées (cycle de vie).
 * Remplace l'ancien panneau « erreurs récurrentes » mémoire (perdu au reboot).
 * Table TanStack (tri par occurrences / dernière vue), filtre de statut
 * client-side, actions par ligne : Reconnaître / Sourdine / Résoudre / Rouvrir.
 *
 * Couleurs : uniquement des tokens sémantiques (tokenCssVar).
 */
import { useMemo, useState } from 'react'
import {
  createColumnHelper,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type SortingState,
} from '@tanstack/react-table'

import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import type { MonitoringDetection } from '@/lib/api/types'
import { useMonitoringDetections } from './queries'
import { useSetDetectionStatus, type DetectionStatus } from './mutations'
import { detectionLevelToken, detectionStatusMeta, filterDetectionsByStatus } from './detectionDisplay'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

const STATUS_FILTERS = ['all', 'open', 'acked', 'muted', 'resolved'] as const
type StatusFilter = (typeof STATUS_FILTERS)[number]

const columnHelper = createColumnHelper<MonitoringDetection>()

export function DetectionsPanel() {
  const { data, isError } = useMonitoringDetections()
  const tA = useAdminT()
  const locale = useAdminLocale()
  const setStatus = useSetDetectionStatus()
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('open')
  const [sorting, setSorting] = useState<SortingState>([{ id: 'last_seen', desc: true }])

  const detections = useMemo(
    () => filterDetectionsByStatus(data?.detections ?? [], statusFilter),
    [data?.detections, statusFilter],
  )

  function act(fingerprint: string, status: DetectionStatus) {
    // W4 (revue 2026-07) : Annuler le prompt (retour null) DOIT tout annuler — aucune
    // mutation. L'ancien `?? undefined` transformait null en undefined puis mutait quand
    // même (changement de statut non voulu). Chaîne vide (OK sans texte) reste une
    // validation → note absente mais mutation effectuée.
    const note = window.prompt(tA('admin.detections.note_prompt'))
    if (note === null) return
    setStatus.mutate({ fingerprint, status, note: note || undefined })
  }

  const columns = useMemo(
    () => [
      columnHelper.accessor('count', {
        header: () => tA('admin.detections.col_count'),
        cell: (info) => (
          <span className="font-mono font-semibold tabular-nums text-foreground">{info.getValue()}</span>
        ),
      }),
      columnHelper.accessor('level', {
        header: () => tA('admin.detections.col_level'),
        cell: (info) => <LevelBadge level={info.getValue()} />,
        enableSorting: false,
      }),
      columnHelper.accessor('module', {
        header: () => tA('admin.detections.col_module'),
        cell: (info) => <span className="font-mono text-xs text-muted-foreground">{info.getValue() ?? ''}</span>,
        enableSorting: false,
      }),
      columnHelper.accessor('message', {
        header: () => tA('admin.detections.col_message'),
        cell: (info) => {
          const row = info.row.original
          return (
            <div className="min-w-0">
              <span className="text-foreground">{info.getValue()}</span>
              {row.sample_detail && (
                <span
                  className="ml-2 break-all font-mono text-xs"
                  style={{ color: tokenCssVar('destructive') }}
                  title={row.sample_detail}
                >
                  {row.sample_detail}
                </span>
              )}
            </div>
          )
        },
        enableSorting: false,
      }),
      columnHelper.accessor('last_seen', {
        header: () => tA('admin.detections.col_last'),
        cell: (info) => (
          <span className="text-xs text-muted-foreground" title={adminAbsoluteTime(info.getValue(), locale)}>
            {adminRelativeTime(info.getValue(), locale)}
          </span>
        ),
      }),
      columnHelper.accessor('status', {
        header: () => tA('admin.detections.col_status'),
        cell: (info) => <DetectionStatusBadge status={info.getValue()} />,
        enableSorting: false,
      }),
      columnHelper.display({
        id: 'actions',
        header: () => tA('admin.detections.col_actions'),
        cell: (info) => {
          const row = info.row.original
          return (
            <div className="flex flex-wrap justify-end gap-1">
              {row.status === 'resolved' ? (
                <Button size="sm" variant="outline" disabled={setStatus.isPending} onClick={() => act(row.fingerprint, 'open')}>
                  {tA('admin.detections.action_reopen')}
                </Button>
              ) : (
                <>
                  {row.status !== 'acked' && (
                    <Button size="sm" variant="outline" disabled={setStatus.isPending} onClick={() => act(row.fingerprint, 'acked')}>
                      {tA('admin.detections.action_ack')}
                    </Button>
                  )}
                  {row.status !== 'muted' && (
                    <Button size="sm" variant="outline" disabled={setStatus.isPending} onClick={() => act(row.fingerprint, 'muted')}>
                      {tA('admin.detections.action_mute')}
                    </Button>
                  )}
                  <Button size="sm" variant="outline" disabled={setStatus.isPending} onClick={() => act(row.fingerprint, 'resolved')}>
                    {tA('admin.detections.action_resolve')}
                  </Button>
                </>
              )}
            </div>
          )
        },
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [tA, locale, setStatus.isPending],
  )

  const table = useReactTable({
    data: detections,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  return (
    <section className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        <div>
          <SectionHeader title={tA('admin.detections.section')} />
          <p className="text-xs text-muted-foreground">{tA('admin.detections.subtitle')}</p>
        </div>
        <label className="ml-auto flex items-center gap-1.5 text-xs text-muted-foreground">
          {tA('admin.detections.filter_status')}
          <select
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
            className="rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
          >
            {STATUS_FILTERS.map((filter) => (
              <option key={filter} value={filter}>
                {tA(filter === 'all' ? 'admin.detections.filter_all' : `admin.detections.status_${filter}`)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {isError ? (
        <p className="text-sm text-destructive">{tA('admin.detections.unavailable')}</p>
      ) : detections.length === 0 ? (
        <EmptyStateNotice title={tA('admin.detections.empty')} description="" />
      ) : (
        <div className="overflow-x-auto rounded-md border">
          <table className="w-full text-sm">
            <thead>
              {table.getHeaderGroups().map((hg) => (
                <tr key={hg.id} className="border-b bg-muted/40 text-left text-xs uppercase tracking-wide text-muted-foreground">
                  {hg.headers.map((header) => (
                    <th
                      key={header.id}
                      className={`px-3 py-2 font-medium ${header.column.getCanSort() ? 'cursor-pointer select-none' : ''}`}
                      onClick={header.column.getToggleSortingHandler()}
                    >
                      {flexRender(header.column.columnDef.header, header.getContext())}
                      {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? ''}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody>
              {table.getRowModel().rows.map((row) => (
                <tr key={row.id} className="border-b last:border-b-0 hover:bg-muted/30">
                  {row.getVisibleCells().map((cell) => (
                    <td key={cell.id} className="px-3 py-2 align-top">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  )
}

/** Badge du niveau (ERROR/WARN) — token dédié, caps flat hard-edge. */
function LevelBadge({ level }: { level: string }) {
  const color = tokenCssVar(detectionLevelToken(level))
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide"
      style={{ color }}
    >
      <span aria-hidden className="inline-block h-2 w-2 flex-none" style={{ backgroundColor: color }} />
      {level}
    </span>
  )
}

/** Badge du statut de cycle de vie (token optionnel — sourdine = neutre). */
function DetectionStatusBadge({ status }: { status: string }) {
  const tA = useAdminT()
  const meta = detectionStatusMeta(status)
  const color = meta.token ? tokenCssVar(meta.token) : undefined
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-sm bg-muted px-2 py-0.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground"
      style={color ? { color } : undefined}
    >
      {color && <span aria-hidden className="inline-block h-2 w-2 flex-none" style={{ backgroundColor: color }} />}
      {tA(meta.labelKey)}
    </span>
  )
}
