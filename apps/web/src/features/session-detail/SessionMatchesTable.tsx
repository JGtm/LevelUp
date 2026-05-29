/**
 * SessionMatchesTable — tableau détaillé des matchs d'une session.
 *
 * Dupliqué du pattern `features/explorer/ExplorerMatchesTable.tsx` (TanStack Table,
 * thead `bg-muted`, cellules colorées via tokens) mais branché sur
 * `SessionDetailMatchRow` et piloté par un preset de colonnes :
 *  - variant="full" (vue session unique, pleine largeur) : toutes les colonnes.
 *  - variant="compact" (mode comparaison, panneaux 50%) : Issue · Mode · K/D/A ·
 *    KDA · Perf · Rating · ΔMMR — set réduit pour une comparaison normalisée.
 *
 * Navigation : clic sur l'icône d'ouverture → page match avec contexte session
 * (prev/next dans la session, descriptor `session`), identique à l'ancienne version.
 */
import { useCallback, useMemo, type ReactNode } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import { EmptyStateNotice } from '@/components/ui/empty-state'
import type { SessionDetailMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdScale, mmrDeltaScale } from '@/lib/accessibility/scales'
import { getOutcomeColor } from '@/lib/outcome-color'
import { formatDurationMMSS } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'

import { formatNumber, formatPercent, formatShortDateTime, useSessionT } from './_shared'

type Variant = 'full' | 'compact'

interface Props {
  matches: SessionDetailMatchRow[]
  playerSlug: string
  variant?: Variant
}

// Presets de colonnes (ids résolus depuis columnsById).
const FULL_COLS = [
  'open', 'time', 'mode', 'map', 'playlist', 'outcome',
  'kda', 'kdaRatio', 'accuracy', 'duration', 'perf', 'rating', 'deltaMmr',
] as const
const COMPACT_COLS = ['outcome', 'mode', 'kda', 'kdaRatio', 'perf', 'rating', 'deltaMmr'] as const

const TRUNCATE_MAX = 14
function truncate(s: string | null | undefined): string {
  if (!s) return '—'
  return s.length <= TRUNCATE_MAX ? s : s.slice(0, TRUNCATE_MAX - 1) + '…'
}

function fmtDeltaMmr(v: number | null | undefined): ReactNode {
  if (v == null) return '—'
  const sign = v >= 0 ? '+' : ''
  return (
    <span className="font-mono tabular-nums" style={{ color: tokenCssVar(mmrDeltaScale(v)) }}>
      {sign}
      {Math.round(v)}
    </span>
  )
}

export function SessionMatchesTable({ matches, playerSlug, variant = 'full' }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const t = useSessionT()
  const navigateToMatch = useNavigateToMatch(playerSlug)

  const allMatchIds = useMemo(() => matches.map((m) => m.match_id), [matches])
  // Début de session = match le plus ancien (start_time ISO UTC → tri lexical = chrono).
  const sessionStartUtc = useMemo(() => {
    if (matches.length === 0) return ''
    return matches.reduce(
      (min, m) => (m.start_time < min ? m.start_time : min),
      matches[0].start_time,
    )
  }, [matches])
  const sessionId = useMemo(
    () => matches.find((m) => m.session_label)?.session_label ?? undefined,
    [matches],
  )

  const goToMatch = useCallback(
    (matchId: string) => {
      navigateToMatch(matchId, {
        source: 'session',
        matchIds: allMatchIds,
        contextDescriptor: { kind: 'session', startTimeUtc: sessionStartUtc },
        filterSpec: sessionId ? { session_id: sessionId } : undefined,
      })
    },
    [navigateToMatch, allMatchIds, sessionStartUtc, sessionId],
  )

  const outcomeLabel = useCallback(
    (outcome: number | null) => {
      const key =
        outcome === 2 ? 'win' : outcome === 3 ? 'loss' : outcome === 1 ? 'tie' : outcome === 4 ? 'dnf' : null
      if (!key) return '—'
      return fieldMappings?.outcomes?.[key]?.label ?? key
    },
    [fieldMappings],
  )

  const columnsById = useMemo<Record<string, ColumnDef<SessionDetailMatchRow>>>(
    () => ({
      open: {
        id: 'open',
        header: '',
        cell: (ctx) => (
          <button
            type="button"
            className="group flex items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
            onClick={() => goToMatch(ctx.row.original.match_id)}
            aria-label={t('session.detail.col_open')}
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3.5 w-3.5 opacity-50 transition-opacity group-hover:opacity-100" aria-hidden="true">
              <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
              <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
            </svg>
          </button>
        ),
      },
      time: {
        accessorKey: 'start_time',
        header: t('session.detail.col_time'),
        cell: (ctx) => (
          <span className="text-muted-foreground">{formatShortDateTime(ctx.getValue<string>())}</span>
        ),
      },
      mode: {
        accessorKey: 'pair_name',
        header: t('session.detail.col_mode'),
        cell: (ctx) => {
          const v = ctx.getValue<string>()
          return (
            <span title={v} className="font-medium">
              {truncate(v)}
            </span>
          )
        },
      },
      map: {
        accessorKey: 'map_name',
        header: t('session.detail.col_map'),
        cell: (ctx) => {
          const v = ctx.getValue<string | undefined>()
          return (
            <span title={v ?? ''} className="text-muted-foreground">
              {truncate(v)}
            </span>
          )
        },
      },
      playlist: {
        accessorKey: 'playlist_name',
        header: t('session.detail.col_playlist'),
        cell: (ctx) => {
          const v = ctx.getValue<string>()
          return (
            <span title={v} className="text-muted-foreground">
              {truncate(v)}
            </span>
          )
        },
      },
      outcome: {
        accessorKey: 'outcome',
        header: t('session.detail.col_outcome'),
        cell: (ctx) => {
          const o = ctx.getValue<number | null>()
          return (
            <span style={{ color: o != null ? getOutcomeColor(o) : undefined, fontWeight: 600 }}>
              {outcomeLabel(o)}
            </span>
          )
        },
      },
      kda: {
        id: 'kda',
        header: t('session.detail.col_kda'),
        cell: (ctx) => {
          const r = ctx.row.original
          return <span className="font-mono tabular-nums">{`${r.kills}/${r.deaths}/${r.assists}`}</span>
        },
      },
      kdaRatio: {
        id: 'kdaRatio',
        accessorKey: 'kda',
        header: t('session.detail.col_kda_ratio'),
        cell: (ctx) => {
          const v = ctx.getValue<number | null>()
          if (v == null) return '—'
          return (
            <span className="font-mono font-semibold tabular-nums" style={{ color: tokenCssVar(kdScale(v)) }}>
              {formatNumber(v, 2)}
            </span>
          )
        },
      },
      accuracy: {
        accessorKey: 'accuracy',
        header: t('session.detail.col_accuracy'),
        cell: (ctx) => <span className="tabular-nums">{formatPercent(ctx.getValue<number | null>())}</span>,
      },
      duration: {
        accessorKey: 'duration_seconds',
        header: t('session.detail.col_duration'),
        cell: (ctx) => (
          <span className="font-mono tabular-nums text-muted-foreground">
            {formatDurationMMSS(ctx.getValue<number | null | undefined>() ?? undefined)}
          </span>
        ),
      },
      perf: {
        accessorKey: 'performance_score',
        header: t('session.detail.col_perf_score'),
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.performance_score == null) return '—'
          const tier = r.perf_tier
          return (
            <span
              className="font-semibold tabular-nums"
              style={tier ? { color: tokenCssVar(`perf-tier-${tier}` as SemanticToken) } : undefined}
            >
              {formatNumber(r.performance_score, 1)}
            </span>
          )
        },
      },
      rating: {
        id: 'rating',
        header: t('session.detail.col_rating'),
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.skill_rating_value == null) return <span className="text-muted-foreground">—</span>
          const type = (r.skill_rating_type ?? '').toUpperCase()
          return (
            <span className="font-mono tabular-nums">
              {Math.round(r.skill_rating_value)}
              {type ? ` ${type}` : ''}
            </span>
          )
        },
      },
      deltaMmr: {
        accessorKey: 'delta_mmr',
        header: t('session.detail.col_delta_mmr'),
        cell: (ctx) => fmtDeltaMmr(ctx.getValue<number | null | undefined>()),
      },
    }),
    [t, goToMatch, outcomeLabel],
  )

  const columns = useMemo(() => {
    const ids = variant === 'compact' ? COMPACT_COLS : FULL_COLS
    return ids.map((id) => columnsById[id])
  }, [variant, columnsById])

  const table = useReactTable({
    data: matches,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  if (matches.length === 0) {
    return (
      <EmptyStateNotice
        title={t('session.detail.matches_empty_title')}
        description={t('session.detail.matches_empty_description')}
      />
    )
  }

  return (
    <div className="overflow-x-auto rounded-md border border-border" data-testid="session-matches-table">
      <table className="w-full text-xs">
        <thead className="border-b bg-muted">
          {table.getHeaderGroups().map((hg) => (
            <tr key={hg.id}>
              {hg.headers.map((h) => (
                <th
                  key={h.id}
                  className="whitespace-nowrap border-r border-border px-2 py-1.5 text-left text-3xs font-medium uppercase tracking-label text-muted-foreground last:border-r-0"
                >
                  {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                </th>
              ))}
            </tr>
          ))}
        </thead>
        <tbody className="divide-y divide-border">
          {table.getRowModel().rows.map((row) => (
            <tr key={row.id} className="transition-colors hover:bg-primary/10">
              {row.getVisibleCells().map((cell) => (
                <td
                  key={cell.id}
                  className="whitespace-nowrap border-r border-border px-2 py-1.5 last:border-r-0"
                >
                  {flexRender(cell.column.columnDef.cell, cell.getContext())}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
