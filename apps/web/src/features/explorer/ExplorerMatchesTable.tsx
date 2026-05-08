/**
 * ExplorerMatchesTable — tableau historique mode "Matchs" de l'Explorer.
 *
 * Pattern visuel STRICTEMENT aligné sur features/squad/SquadSynergyHistoryTable.tsx :
 *  - thead : `bg-muted border-b`, th `px-3 py-2 text-left whitespace-nowrap
 *    text-xs font-medium text-muted-foreground border-r border-border last:border-r-0`
 *  - tbody : row `cursor-pointer transition-colors hover:bg-primary/10`
 *  - td : `px-3 py-2 whitespace-nowrap border-r border-border last:border-r-0`
 *  - outcome rendu en texte coloré via getOutcomeColor (pas de Badge, pas de tinting)
 *  - boutons "Ouvrir" + "↗ wp" en début de ligne (stop propagation)
 *  - formatDate + formatDurationMinSec pour les colonnes formatées
 *  - useFieldMappings pour libellés map/playlist
 *
 * Colonnes : Ouvrir | wp | Date | Carte | Playlist | Mode | Résultat |
 *            Frags | Morts | Assists | FDA | Score | Durée |
 *            Perf (color) | ΔPerf | Rang | ΔRang |
 *            MMR équipe | MMR adv. | ΔMMR
 */
import { useMemo, useState, type ReactNode } from 'react'
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
import { formatDate, formatDurationMinSec } from '@/lib/formatters'
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

function fmtMmr(v: number | null | undefined): string {
  if (v === undefined || v === null) return '-'
  return Math.round(v).toLocaleString()
}

function fmtDeltaMMR(v: number | null | undefined): ReactNode {
  if (v === undefined || v === null) return '-'
  const sign = v >= 0 ? '+' : ''
  return (
    <span
      className="font-mono tabular-nums"
      style={{ color: tokenCssVar(mmrDeltaScale(v)) }}
    >
      {sign}
      {Math.round(v)}
    </span>
  )
}

function fmtKDA(v: number | null | undefined): string {
  if (v === undefined || v === null) return '-'
  return v.toFixed(2)
}

/** Traduit le préfixe EN du skill tier label vers FR ("Diamond IV" → "Diamant IV").
 *  La DB stocke `tier_label` en EN nativement (pas de tier_label_fr column),
 *  donc on remappe côté UI pour respecter la locale.
 */
const SKILL_TIER_FR: Record<string, string> = {
  Bronze: 'Bronze',
  Silver: 'Argent',
  Gold: 'Or',
  Platinum: 'Platine',
  Diamond: 'Diamant',
  Onyx: 'Onyx',
}
function localizeSkillTierLabel(label: string | null | undefined, locale: string): string {
  if (!label) return '-'
  if (locale !== 'fr') return label
  // Le label est typiquement "<Tier> <Sub>" (ex: "Diamond IV"). On split sur le 1er espace.
  const idx = label.indexOf(' ')
  const head = idx === -1 ? label : label.slice(0, idx)
  const tail = idx === -1 ? '' : label.slice(idx)
  return (SKILL_TIER_FR[head] ?? head) + tail
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

  const waypointBase = `https://www.halowaypoint.com/halo-infinite/players/${encodeURIComponent(playerSlug)}/matches`

  // Labels outcome (pas de Badge, juste texte coloré comme Squad)
  const outcomeLabels: Record<'win' | 'loss' | 'draw' | 'dnf', string> = {
    win: t('explorer.matches.outcome_win'),
    loss: t('explorer.matches.outcome_loss'),
    draw: t('explorer.matches.outcome_draw'),
    dnf: t('explorer.matches.outcome_dnf'),
  }

  const columns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
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
              goToMatch(ctx.row.original.match_id)
            }}
          >
            {t('explorer.matches.col_open')}
          </button>
        ),
      },
      {
        id: 'waypoint',
        header: '',
        cell: (ctx) => (
          <a
            href={ctx.row.original.match_url || `${waypointBase}/${ctx.row.original.match_id}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-primary text-xs whitespace-nowrap"
            onClick={(e) => e.stopPropagation()}
          >
            {t('explorer.matches.col_waypoint')}
          </a>
        ),
      },
      {
        accessorKey: 'start_time',
        header: t('explorer.matches.col_date'),
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS)}
          </span>
        ),
      },
      {
        accessorKey: 'map_ui',
        header: t('explorer.filters.map'),
        cell: (ctx) => labelOfMap(ctx.getValue<string>()),
      },
      {
        accessorKey: 'playlist_label',
        header: t('explorer.filters.playlist'),
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {labelOfPlaylist(ctx.getValue<string | null | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'mode_ui',
        header: t('explorer.filters.mode'),
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {ctx.getValue<string | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorKey: 'is_with_friends',
        header: t('explorer.matches.col_squad'),
        cell: (ctx) => (
          <span className="text-muted-foreground text-xs">
            {ctx.getValue<boolean>()
              ? t('explorer.matches.squad_party')
              : t('explorer.matches.squad_solo')}
          </span>
        ),
      },
      {
        accessorKey: 'outcome_code',
        header: t('explorer.matches.col_outcome'),
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
        accessorKey: 'kills',
        header: t('explorer.matches.col_kills'),
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorKey: 'deaths',
        header: t('explorer.matches.col_deaths'),
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorKey: 'assists',
        header: t('explorer.matches.col_assists'),
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorKey: 'kda',
        header: t('explorer.matches.col_kda'),
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {fmtKDA(ctx.getValue<number | null | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'score_label',
        header: t('explorer.matches.col_score'),
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'duration_seconds',
        header: t('explorer.matches.col_duration'),
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {formatDurationMinSec(ctx.getValue<number | null | undefined>() ?? undefined)}
          </span>
        ),
      },
      {
        accessorKey: 'perf_score',
        header: t('explorer.matches.col_perf'),
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
        header: t('explorer.matches.col_delta_perf'),
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
            <span className="font-mono tabular-nums" style={{ color }}>
              {v >= 0 ? '+' : ''}
              {v}
            </span>
          )
        },
      },
      {
        accessorKey: 'skill_tier_label',
        header: t('explorer.matches.col_rank'),
        cell: (ctx) => localizeSkillTierLabel(ctx.getValue<string | null | undefined>(), locale),
      },
      {
        accessorKey: 'team_mmr',
        header: t('explorer.matches.col_team_mmr'),
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {fmtMmr(ctx.getValue<number | null | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'enemy_mmr',
        header: t('explorer.matches.col_enemy_mmr'),
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {fmtMmr(ctx.getValue<number | null | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'delta_mmr',
        header: t('explorer.matches.col_delta_mmr'),
        cell: (ctx) => fmtDeltaMMR(ctx.getValue<number | null | undefined>()),
      },
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [intlLocale, mapAssets, playlistAssets, locale, playerSlug, waypointBase],
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
                  <th
                    key={h.id}
                    className="px-3 py-2 text-left whitespace-nowrap text-xs font-medium text-muted-foreground border-r border-border last:border-r-0"
                  >
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
                  <td
                    key={cell.id}
                    className="px-3 py-2 whitespace-nowrap border-r border-border last:border-r-0"
                  >
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
