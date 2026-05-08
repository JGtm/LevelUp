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
import { tokenCssVar } from '@/lib/accessibility'
import { mmrDeltaScale, kdScale } from '@/lib/accessibility/scales'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { getOutcomeColor } from '@/lib/outcome-color'
import { formatDate, formatDurationMinSec } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'
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

/** Bannière d'équipe affichée en 1ère ligne du thead (variant ally/enemy).
 *  Reproduit le pattern visuel de MatchScoreboard.tsx (gradient horizontal
 *  color-mix sur token team-ally / team-enemy, configurable par les réglages
 *  d'accessibilité du joueur). Utilisé en mode Joueur pour distinguer les
 *  tableaux "matchs où le cible était allié" vs "ennemi". */
export interface TeamBanner {
  variant: 'ally' | 'enemy'
  label: string
}

interface Props {
  rows: ExplorerMatchRow[]
  playerSlug: string
  teamBanner?: TeamBanner
  /** Descriptor typé propagé dans le matchNavContext (Phase 2c) — détermine
   *  le label compact "Matchs <ctx> X/Y" affiché dans la nav bar de la page
   *  match-view. Si undefined, pas de label spécifique (fallback Q25 global). */
  contextDescriptor?: ContextDescriptor
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

/** Split sur le 1er espace pour rendre l'en-tête sur 2 lignes (utile quand
 *  plusieurs mots dans le label, ex: "MMR équipe" → "MMR" + "équipe"). */
function renderTwoLineHeader(label: string): ReactNode {
  const idx = label.indexOf(' ')
  if (idx === -1) return label
  return (
    <span className="leading-tight">
      {label.slice(0, idx)}
      <br />
      {label.slice(idx + 1)}
    </span>
  )
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

export function ExplorerMatchesTable({ rows, playerSlug, teamBanner, contextDescriptor }: Props) {
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
      source: 'explorer',
      matchIds: allMatchIds,
      filterSpec: filterSpec ?? undefined,
      contextDescriptor,
    })
  }

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
        // Pastille reprise du style match-card.tsx (tuiles match home).
        // Les couleurs hex sont autorisées (color-allow) car identifiants UX
        // génériques de catégorie, pas de palette accessibility.
        cell: (ctx) => {
          const isSquad = ctx.getValue<boolean>()
          return (
            <span
              className="rounded-full px-2 py-0.5 text-[10px] font-bold uppercase tracking-wider leading-none"
              style={
                isSquad
                  ? { backgroundColor: 'rgba(56,189,248,0.15)', color: '#38bdf8' } // color-allow: bleu sky pour pill "Escouade"
                  : { backgroundColor: 'rgba(168,85,247,0.15)', color: '#a855f7' } // color-allow: violet pour pill "Solo"
              }
            >
              {isSquad
                ? t('explorer.matches.squad_party')
                : t('explorer.matches.squad_solo')}
            </span>
          )
        },
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
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return '-'
          return (
            <span
              className="font-mono tabular-nums font-semibold"
              style={{ color: tokenCssVar(kdScale(v)) }}
            >
              {fmtKDA(v)}
            </span>
          )
        },
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
              style={{ color: tokenCssVar(`perf-tier-${r.perf_tier}` as SemanticToken) }}
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
              ? tokenCssVar('perf-tier-1' as SemanticToken)
              : v < 0
                ? tokenCssVar('perf-tier-5' as SemanticToken)
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
        cell: (ctx) => ctx.getValue<string | null | undefined>() ?? '-',
      },
      {
        accessorKey: 'team_mmr',
        header: () => renderTwoLineHeader(t('explorer.matches.col_team_mmr')),
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {fmtMmr(ctx.getValue<number | null | undefined>())}
          </span>
        ),
      },
      {
        accessorKey: 'enemy_mmr',
        header: () => renderTwoLineHeader(t('explorer.matches.col_enemy_mmr')),
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

  // Bannière d'équipe (variant ally/enemy) : pattern aligné sur
  // features/match-view/MatchScoreboard.tsx — gradient horizontal color-mix
  // sur token team-ally/team-enemy, configurable via réglages accessibilité.
  const bannerColCount = table.getAllLeafColumns().length
  const bannerStyle = teamBanner
    ? (() => {
        const teamColor = tokenCssVar(teamBanner.variant === 'ally' ? 'team-ally' : 'team-enemy')
        return {
          background: `linear-gradient(90deg, color-mix(in oklab, ${teamColor} 35%, transparent), transparent 88%)`,
          borderBottom: `2px solid color-mix(in oklab, ${teamColor} 55%, transparent)`,
          color: `color-mix(in oklab, ${teamColor} 80%, var(--foreground))`,
        }
      })()
    : null

  return (
    <div className="space-y-2" data-testid="explorer-matches-table">
      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full text-xs">
          <thead className="bg-muted border-b">
            {teamBanner && (
              <tr>
                <th
                  colSpan={bannerColCount}
                  className="px-3 py-2 text-left text-sm font-bold uppercase tracking-wider"
                  style={bannerStyle ?? undefined}
                  aria-label={teamBanner.label}
                >
                  {teamBanner.label}
                </th>
              </tr>
            )}
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id}>
                {hg.headers.map((h) => (
                  <th
                    key={h.id}
                    className="px-2 py-1.5 text-left whitespace-nowrap text-[11px] font-medium text-muted-foreground border-r border-border last:border-r-0"
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
                    className="px-2 py-1.5 whitespace-nowrap border-r border-border last:border-r-0"
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
