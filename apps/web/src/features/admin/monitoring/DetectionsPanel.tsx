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

import { AlertDialog } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'
import { HeaderLabelTooltip } from '@/lib/table/columnMeta'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { MonitoringDetection } from '@/lib/api/types'
import { useMonitoringDetections } from './queries'
import { useSetDetectionStatus, type DetectionStatus } from './mutations'
import { detectionLevelToken, detectionStatusMeta, filterDetectionsByStatus } from './detectionDisplay'
import { adminAbsoluteTime, adminRelativeTime } from '../format'
import { useAdminT, useAdminLocale } from '../useAdminText'
import { SectionHeader } from '../components/SectionHeader'

const STATUS_FILTERS = ['all', 'open', 'acked', 'muted', 'resolved'] as const
type StatusFilter = (typeof STATUS_FILTERS)[number]

/** Clé i18n du libellé d'action pour chaque statut cible. */
const ACTION_LABEL_KEY: Record<DetectionStatus, AdminManifestKey> = {
  open: 'admin.detections.action_reopen',
  acked: 'admin.detections.action_ack',
  muted: 'admin.detections.action_mute',
  resolved: 'admin.detections.action_resolve',
}

/** Clé i18n de la micro-copie expliquant l'effet de chaque action. */
const ACTION_HELP_KEY: Record<DetectionStatus, AdminManifestKey> = {
  open: 'admin.detections.help_reopen',
  acked: 'admin.detections.help_ack',
  muted: 'admin.detections.help_mute',
  resolved: 'admin.detections.help_resolve',
}

const columnHelper = createColumnHelper<MonitoringDetection>()

export function DetectionsPanel() {
  const { data, isError } = useMonitoringDetections()
  const tA = useAdminT()
  const locale = useAdminLocale()
  const setStatus = useSetDetectionStatus()
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('open')
  const [sorting, setSorting] = useState<SortingState>([{ id: 'last_seen', desc: true }])
  const [pending, setPending] = useState<{ fingerprint: string; status: DetectionStatus } | null>(null)
  const [note, setNote] = useState('')

  const detections = useMemo(
    () => filterDetectionsByStatus(data?.detections ?? [], statusFilter),
    [data?.detections, statusFilter],
  )

  function act(fingerprint: string, status: DetectionStatus) {
    // La note de suivi passe par un dialog in-app (plus de prompt() natif, banni de
    // l'app) : ouvrir le dialog ; la mutation ne part qu'à la validation. Annuler
    // (Escape/backdrop/bouton) n'émet AUCUNE mutation. Note vide = validation sans
    // commentaire (mutation effectuée, note absente).
    setNote('')
    setPending({ fingerprint, status })
  }

  function confirmAction() {
    if (!pending) return
    const trimmed = note.trim()
    setStatus.mutate({ fingerprint: pending.fingerprint, status: pending.status, note: trimmed || undefined })
    setPending(null)
  }

  const columns = useMemo(
    () => [
      columnHelper.accessor('count', {
        header: () => tA('admin.detections.col_count'),
        meta: { headerTooltip: tA('admin.detections.col_count_tooltip') },
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
        meta: { headerTooltip: tA('admin.detections.col_status_tooltip') },
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
                <Button
                  size="sm"
                  variant="outline"
                  className="whitespace-nowrap"
                  title={tA(ACTION_HELP_KEY.open)}
                  disabled={setStatus.isPending}
                  onClick={() => act(row.fingerprint, 'open')}
                >
                  {tA('admin.detections.action_reopen')}
                </Button>
              ) : (
                <>
                  {row.status !== 'acked' && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="whitespace-nowrap"
                      title={tA(ACTION_HELP_KEY.acked)}
                      disabled={setStatus.isPending}
                      onClick={() => act(row.fingerprint, 'acked')}
                    >
                      {tA('admin.detections.action_ack')}
                    </Button>
                  )}
                  {row.status !== 'muted' && (
                    <Button
                      size="sm"
                      variant="outline"
                      className="whitespace-nowrap"
                      title={tA(ACTION_HELP_KEY.muted)}
                      disabled={setStatus.isPending}
                      onClick={() => act(row.fingerprint, 'muted')}
                    >
                      {tA('admin.detections.action_mute')}
                    </Button>
                  )}
                  <Button
                    size="sm"
                    variant="outline"
                    className="whitespace-nowrap"
                    title={tA(ACTION_HELP_KEY.resolved)}
                    disabled={setStatus.isPending}
                    onClick={() => act(row.fingerprint, 'resolved')}
                  >
                    {tA('admin.detections.action_resolve')}
                  </Button>
                </>
              )}
            </div>
          )
        },
      }),
    ],
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
        <div className="space-y-1">
          <SectionHeader title={tA('admin.detections.section')} />
          <p className="text-xs text-muted-foreground">{tA('admin.detections.subtitle')}</p>
          <details className="text-xs text-muted-foreground">
            <summary className="cursor-pointer select-none">{tA('admin.detections.actions_help_summary')}</summary>
            <ul className="mt-1 max-w-2xl list-disc space-y-0.5 pl-5">
              <li>{tA('admin.detections.help_ack')}</li>
              <li>{tA('admin.detections.help_mute')}</li>
              <li>{tA('admin.detections.help_resolve')}</li>
              <li>{tA('admin.detections.help_reopen')}</li>
            </ul>
          </details>
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
                      {/* Aide portée par le libellé ; le tri reste sur le <th>. */}
                      <HeaderLabelTooltip
                        text={header.column.columnDef.meta?.headerTooltip}
                        focusable={!header.column.getCanSort()}
                      >
                        <span className="inline-flex items-center gap-1">
                          {flexRender(header.column.columnDef.header, header.getContext())}
                          {{ asc: ' ↑', desc: ' ↓' }[header.column.getIsSorted() as string] ?? ''}
                        </span>
                      </HeaderLabelTooltip>
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

      <AlertDialog
        open={pending !== null}
        onOpenChange={(o) => {
          if (!o) setPending(null)
        }}
        title={pending ? tA(ACTION_LABEL_KEY[pending.status]) : ''}
        description={tA('admin.detections.note_dialog_desc')}
        confirmLabel={tA('admin.detections.note_confirm')}
        cancelLabel={tA('admin.detections.note_cancel')}
        busy={setStatus.isPending}
        autoFocusConfirm={false}
        onConfirm={confirmAction}
      >
        <textarea
          autoFocus
          rows={3}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={tA('admin.detections.note_placeholder')}
          className="w-full rounded-md border border-input bg-background px-2 py-1 text-sm text-foreground"
        />
      </AlertDialog>
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
