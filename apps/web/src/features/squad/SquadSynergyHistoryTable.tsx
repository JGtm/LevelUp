/**
 * SquadSynergyHistoryTable — historique des matchs partagés pour la page Synergies.
 *
 * Colonnes contextuelles (pas de stats personnelles) :
 *   Ouvrir | Waypoint | Date | Carte | Playlist | Mode |
 *   Résultat | Taux hist. | Score | Durée | MMR équipe | MMR adv. | Écart MMR
 *
 * Colonne « Waypoint » (I19) : lien externe vers la page de détail du match sur
 * Halo Waypoint (logo thème clair/sombre, remplace l'ancien lien texte « ↗ wp »),
 * gatée par la capability `waypoint_match_url` (absente pour Halo 5) ET par la
 * préférence locale `localUiPrefs.showWaypointColumn` (Settings → Apparence).
 *
 * Utilise TanStack Table v8. Pagination 20/page côté client.
 * Labels carte/playlist via useFieldMappings (assets titre).
 *
 * Tri CLIENT par clic sur les en-têtes (I16) : toutes les colonnes de données sont
 * triables (helpers partagés `explorerMatchesClientSort.ts`), sauf Ouvrir/Waypoint.
 * Aucun tri actif par défaut (ordre serveur = chronologique ASC, cf. `sortedRows`) ;
 * le tri réinitialise la pagination en page 1 (pattern RelationsTable).
 */
import { useCallback, useMemo, useState, type ReactNode } from 'react'
import {
  type ColumnDef,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type { SquadMatchHistoryRow } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { tokenCssVar } from '@/lib/accessibility'
import { getOutcomeColor, outcomeKey } from '@/lib/outcome-color'
import { formatDate, formatDurationMinSec } from '@/lib/formatters'
import { useCapability } from '@/lib/capabilities/capabilities'
import { getSquadText } from './i18n'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { buildWaypointMatchUrl, waypointLogoSrc } from '@/lib/match-nav/waypointUrl'
import {
  NUMERIC_SORT,
  dateTimeSortingFn,
  localeTextSortingFn,
} from '@/features/explorer/explorerMatchesClientSort'

const PAGE_SIZE = 20

const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
}

// Classe des cellules d'en-tête (inchangée par le tri — I16).
const HISTORY_TH_CLASS =
  'px-3 py-2 text-left whitespace-nowrap text-xs font-medium text-muted-foreground border-r border-border last:border-r-0'

/** aria-sort du <th> pour un état de tri TanStack (I16, miroir ExplorerMatchesTable). */
function sortAriaValue(dir: false | 'asc' | 'desc'): 'ascending' | 'descending' | 'none' {
  if (dir === 'asc') return 'ascending'
  if (dir === 'desc') return 'descending'
  return 'none'
}

interface SquadSynergyHistoryTableProps {
  rows: SquadMatchHistoryRow[]
  playerSlug: string
}

function fmtMmr(v: number | undefined): string {
  if (v === undefined || v === null) return '-'
  return Math.round(v).toLocaleString()
}

function fmtDeltaMMR(v: number | undefined): ReactNode {
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
  const navigateToMatch = useNavigateToMatch(playerSlug)
  // Title-agnostic : les 3 colonnes MMR (équipe / adverse / écart) n'ont de sens
  // que si le titre courant fournit un MMR par match. Faux pour Halo 5 → on
  // RETIRE silencieusement les colonnes (pas de cellule "-" ni de colonne vide).
  const providesTeamMmr = useCapability('team_mmr')
  // Lien « Ouvrir sur Halo Waypoint » (I19) : gating par capability (absente pour
  // Halo 5) ET par préférence LOCALE (Apparence → « Colonne Halo Waypoint sur les
  // listes de matchs », défaut ON).
  const waypointCapability = useCapability('waypoint_match_url')
  const showWaypointColumnPref = useSettingsDraftStore((s) => s.localUiPrefs.showWaypointColumn)
  const showWaypoint = waypointCapability && showWaypointColumnPref
  const theme = useSettingsDraftStore((s) => s.localUiPrefs.theme)
  const currentTitleSlug = useAppShellStore((s) => s.currentTitleSlug)
  // Backend envoie DESC (newest first) — on inverse pour oldest-first (chronologique).
  const sortedRows = useMemo(() => [...rows].reverse(), [rows])
  const allMatchIds = useMemo(() => sortedRows.map((r) => r.match_id), [sortedRows])
  // useCallback obligatoire : goToSynergyMatch est référencé dans le cell renderer
  // du useMemo `columns` ci-dessous. Sans ça, le closure figé au 1er render garde
  // les `allMatchIds` initiaux (pré-filtre) → nav contextuelle hors scope.
  const goToSynergyMatch = useCallback(
    (matchId: string) =>
      navigateToMatch(matchId, { source: 'session', matchIds: allMatchIds }),
    [navigateToMatch, allMatchIds],
  )

  const columns = useMemo<ColumnDef<SquadMatchHistoryRow>[]>(
    () => [
      {
        id: 'open',
        header: '',
        // Icône d'ouverture : jamais triable (I16, comme ExplorerMatchesTable).
        enableSorting: false,
        cell: (ctx) => (
          <button
            type="button"
            className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
            onClick={(e) => {
              e.stopPropagation()
              goToSynergyMatch(ctx.row.original.match_id)
            }}
            aria-label="Ouvrir"
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3.5 w-3.5 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true">
              <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
              <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
            </svg>
          </button>
        ),
      },
      // Lien « Ouvrir sur Halo Waypoint » (I19) : capability + préférence locale
      // (cf. showWaypoint plus haut). Halo 5 → colonne absente (pas de cellule vide).
      ...(showWaypoint
        ? [
            {
              id: 'waypoint',
              header: '',
              // Lien externe : jamais triable (I16, comme ExplorerMatchesTable).
              enableSorting: false,
              cell: (ctx) => (
                <a
                  href={buildWaypointMatchUrl(playerSlug, ctx.row.original.match_id, currentTitleSlug)}
                  target="_blank"
                  rel="noopener noreferrer"
                  onClick={(e) => e.stopPropagation()}
                  aria-label={labels.waypointAriaLabel}
                  title={labels.waypointAriaLabel}
                  className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
                >
                  <img
                    src={waypointLogoSrc(theme)}
                    alt=""
                    aria-hidden
                    className="h-4 w-4 opacity-60 group-hover:opacity-100 transition-opacity"
                  />
                </a>
              ),
            } as ColumnDef<SquadMatchHistoryRow>,
          ]
        : []),
      {
        accessorKey: 'start_time',
        header: labels.date,
        // Tri chronologique sur le timestamp brut (pas le libellé formaté).
        sortingFn: dateTimeSortingFn,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS)}
          </span>
        ),
      },
      {
        accessorKey: 'map_ui',
        header: labels.map,
        // Colonnes texte : tri alpha locale-aware, ascendant au 1er clic.
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => ctx.getValue<string>() || '-',
      },
      {
        accessorKey: 'playlist_name',
        header: labels.playlist,
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'mode_ui',
        header: labels.mode,
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {ctx.getValue<string | undefined>() || ctx.row.original.pair_name || '-'}
          </span>
        ),
      },
      {
        accessorKey: 'outcome',
        header: labels.outcome,
        // Tri sur le code brut (2=V, 3=D, 1=N, 4=DNF), jamais le libellé traduit.
        sortingFn: 'basic',
        cell: (ctx) => {
          const o = ctx.getValue<number>()
          const key = outcomeKey(o)
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
        ...NUMERIC_SORT,
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
        accessorKey: 'expected_win_prob',
        header: labels.winProb,
        ...NUMERIC_SORT,
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return <span className="text-muted-foreground">—</span>
          const pct = Math.round(v * 100)
          const color = pct >= 50 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
          return (
            <span className="font-mono tabular-nums" style={{ color }}>
              {pct}%
            </span>
          )
        },
      },
      {
        accessorKey: 'score_label',
        header: labels.score,
        // Score « 50-30 » : tri alphanumérique naturel (comme ExplorerMatchesTable).
        sortingFn: 'alphanumeric',
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        id: 'duration',
        // Durée réelle de gameplay (countdown retranché) préférée, fallback brut.
        accessorFn: (r) => r.gameplay_duration_seconds ?? r.duration_seconds,
        header: labels.duration,
        ...NUMERIC_SORT,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {formatDurationMinSec(ctx.getValue<number | undefined>())}
          </span>
        ),
      },
      // Colonnes MMR (équipe / adverse / écart) : incluses uniquement si le titre
      // courant fournit le MMR par match (Halo 5 → masquées).
      ...(providesTeamMmr
        ? [
            {
              accessorKey: 'team_mmr_avg',
              header: labels.teamMmr,
              ...NUMERIC_SORT,
              cell: (ctx) => (
                <span className="text-muted-foreground font-mono tabular-nums">
                  {fmtMmr(ctx.getValue<number>())}
                </span>
              ),
            } as ColumnDef<SquadMatchHistoryRow>,
            {
              accessorKey: 'enemy_mmr_avg',
              header: labels.enemyMmr,
              ...NUMERIC_SORT,
              cell: (ctx) => (
                <span className="text-muted-foreground font-mono tabular-nums">
                  {fmtMmr(ctx.getValue<number | undefined>())}
                </span>
              ),
            } as ColumnDef<SquadMatchHistoryRow>,
            {
              accessorKey: 'delta_mmr',
              header: labels.deltaMMR,
              ...NUMERIC_SORT,
              cell: (ctx) => fmtDeltaMMR(ctx.getValue<number | undefined>()),
            } as ColumnDef<SquadMatchHistoryRow>,
          ]
        : []),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [labels, intlLocale, playerSlug, goToSynergyMatch, providesTeamMmr, showWaypoint, theme],
  )

  // I16 : tri CLIENT par clic sur les en-têtes. Pas d'état de tri initial : l'ordre
  // serveur déjà appliqué ci-dessus (`sortedRows` — chronologique ASC) reste affiché
  // tant qu'aucun en-tête n'est cliqué (pattern RelationsTable A3).
  const [sorting, setSorting] = useState<SortingState>([])
  const [pagination, setPagination] = useState({ pageIndex: 0, pageSize: PAGE_SIZE })

  const table = useReactTable<SquadMatchHistoryRow>({
    data: sortedRows,
    columns,
    state: { sorting, pagination },
    // Reset en page 1 au changement de tri (pattern RelationsTable A4).
    onSortingChange: (updater) => {
      setSorting(updater)
      setPagination((p) => ({ ...p, pageIndex: 0 }))
    },
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getPaginationRowModel: getPaginationRowModel(),
  })

  if (rows.length === 0) {
    // Bloc vide : on conserve le cadre bordé (même style que la table) + message.
    return (
      <div
        className="flex min-h-[100px] items-center justify-center rounded-md border border-border text-sm text-muted-foreground"
        data-testid="squad-synergy-history-table"
      >
        {t.empty.noBlockData}
      </div>
    )
  }

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
                {hg.headers.map((h) => {
                  const content = h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())
                  if (!h.column.getCanSort()) {
                    return (
                      <th key={h.id} className={HISTORY_TH_CLASS}>
                        {content}
                      </th>
                    )
                  }
                  const sortDir = h.column.getIsSorted()
                  return (
                    <th key={h.id} className={HISTORY_TH_CLASS} aria-sort={sortAriaValue(sortDir)}>
                      <button
                        type="button"
                        onClick={h.column.getToggleSortingHandler()}
                        aria-label={labels.sortByAriaLabel(String(content ?? ''))}
                        className={`group inline-flex items-center gap-1 whitespace-nowrap transition-colors hover:text-foreground${sortDir ? ' text-foreground' : ''}`}
                      >
                        {content}
                        {sortDir && (
                          <span aria-hidden="true" className="text-2xs leading-none">
                            {sortDir === 'asc' ? '▲' : '▼'}
                          </span>
                        )}
                      </button>
                    </th>
                  )
                })}
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
                onClick={() => goToSynergyMatch(row.original.match_id)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    goToSynergyMatch(row.original.match_id)
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
