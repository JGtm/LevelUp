/**
 * RelationsTable — tableau unique du hub Communauté > Relations.
 *
 * Réutilise le langage visuel de match-view/MatchEncountersTable : SplitBar pour
 * les rencontres (allié | ennemi) et les frags / morts, badges narratifs et
 * gamertag cliquable (→ Explorer mode joueur). Couleurs via tokens accessibilité.
 */
import {
  type Column,
  type ColumnDef,
  type PaginationState,
  type SortingState,
  flexRender,
  getCoreRowModel,
  getPaginationRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useMemo, useState, type ReactNode } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import { formatPercent } from '@/lib/formatters'
import { ratioColor } from '@/lib/colors/outcomePalette'
import type { RelationInsight } from '@/lib/api/types'

import { RelationBadges } from './RelationBadges'
import type { PalmaresText } from './i18n'

type RelationsLabels = PalmaresText['relations']

// Le tableau liste tout le réseau ; on borne l'affichage par pages pour rester
// lisible (binôme / bête noire / noyau dur restent en tête de page).
const RELATIONS_PAGE_SIZE = 25

function percentColor(v: number | null | undefined): string | undefined {
  if (v == null || !Number.isFinite(v)) return undefined
  return v >= 0.5 ? tokenCssVar('outcome-win') : tokenCssVar('outcome-loss')
}

// numOrUndef — valeur triable : les nuls/NaN deviennent `undefined` pour être
// systématiquement relégués en fin de liste (via `sortUndefined: 'last'`), quel
// que soit le sens du tri (A2).
function numOrUndef(v: number | null | undefined): number | undefined {
  return v != null && Number.isFinite(v) ? v : undefined
}

// lastSeenEpoch — timestamp epoch (ms) de la dernière rencontre, `undefined` si
// absent/illisible (relégué en fin de liste au tri).
function lastSeenEpoch(iso: string | null): number | undefined {
  if (!iso) return undefined
  const ts = new Date(iso).getTime()
  return Number.isFinite(ts) ? ts : undefined
}

function ariaSortValue(sorted: false | 'asc' | 'desc'): 'ascending' | 'descending' | 'none' {
  if (sorted === 'asc') return 'ascending'
  if (sorted === 'desc') return 'descending'
  return 'none'
}

/**
 * SortLabel — en-tête triable : bouton cliquable (cycle desc→asc→none pour les
 * colonnes numériques, asc→desc→none pour l'alpha), indicateur de sens en
 * caractère texte (pas d'icône). Le `<button>` porte le clic ; `aria-sort` est
 * posé sur le `<th>` (RelationsTable). Pour la colonne « Ratio », le Tooltip
 * enveloppe ce bouton (`div > button` = HTML valide, jamais l'inverse).
 */
function SortLabel({
  column,
  children,
}: {
  column: Column<RelationInsight, unknown>
  children: ReactNode
}) {
  const sorted = column.getIsSorted()
  const indicator = sorted === 'asc' ? '↑' : sorted === 'desc' ? '↓' : ''
  return (
    <button
      type="button"
      onClick={column.getToggleSortingHandler()}
      className="inline-flex items-center gap-1 select-none transition-colors hover:text-foreground"
    >
      {children}
      {indicator && (
        <span aria-hidden="true" className="font-mono text-[0.9em] text-foreground">
          {indicator}
        </span>
      )}
    </button>
  )
}

function formatRatio(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return '—'
  return v.toFixed(2)
}

function SplitBar({
  leftCount,
  rightCount,
  leftColor,
  rightColor,
  leftTooltip,
  rightTooltip,
}: {
  leftCount: number
  rightCount: number
  leftColor: string
  rightColor: string
  leftTooltip: string
  rightTooltip: string
}) {
  const total = leftCount + rightCount
  if (total === 0) return <span className="font-mono">—</span>
  const leftPct = Math.round((leftCount / total) * 100)
  return (
    <span className="inline-flex items-center gap-1 font-mono tabular-nums">
      <Tooltip content={leftTooltip}>
        <span style={{ color: leftColor }}>{leftCount}</span>
      </Tooltip>
      <span className="inline-flex h-2 w-12 border border-border overflow-hidden">
        <span style={{ width: `${leftPct}%`, backgroundColor: leftColor }} />
        <span style={{ flex: 1, backgroundColor: rightColor }} />
      </span>
      <Tooltip content={rightTooltip}>
        <span style={{ color: rightColor }}>{rightCount}</span>
      </Tooltip>
    </span>
  )
}

function formatRelative(iso: string | null, rel: RelationsLabels['relative']): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return '—'
  const days = Math.round((Date.now() - date.getTime()) / 86_400_000)
  if (days <= 0) return rel.today
  if (days === 1) return rel.yesterday
  if (days < 7) return rel.daysAgo(days)
  if (days < 30) return rel.weeksAgo(Math.round(days / 7))
  if (days < 365) return rel.monthsAgo(Math.round(days / 30))
  return rel.yearsAgo(Math.round(days / 365))
}

function categoryLabel(category: RelationInsight['category'], labels: RelationsLabels): string {
  if (category === 'ally') return labels.category.ally
  if (category === 'enemy') return labels.category.enemy
  return labels.category.mixed
}

function buildColumns(
  labels: RelationsLabels,
  locale: 'fr' | 'en',
  onPlayerClick: (gamertag: string) => void,
): ColumnDef<RelationInsight>[] {
  return [
    {
      id: 'player',
      // Tri alpha insensible à la casse (asc→desc→none).
      accessorFn: (r) => r.gamertag.toLowerCase(),
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.player}</SortLabel>,
      cell: (ctx) => {
        const r = ctx.row.original
        return (
          <span className="whitespace-nowrap">
            <button
              type="button"
              className="font-semibold text-info hover:underline"
              onClick={() => onPlayerClick(r.gamertag)}
            >
              {r.gamertag}
            </button>
            <RelationBadges badges={r.badges} locale={locale} />
          </span>
        )
      },
    },
    {
      id: 'link',
      header: labels.table.link,
      enableSorting: false,
      cell: (ctx) => (
        <span className="text-[0.85em] text-muted-foreground">
          {categoryLabel(ctx.row.original.category, labels)}
        </span>
      ),
    },
    {
      id: 'encounters',
      accessorFn: (r) => r.total_matches,
      sortDescFirst: true,
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.encounters}</SortLabel>,
      cell: (ctx) => {
        const r = ctx.row.original
        return (
          <SplitBar
            leftCount={r.teammate_matches}
            rightCount={r.enemy_matches}
            leftColor={tokenCssVar('team-ally')}
            rightColor={tokenCssVar('team-enemy')}
            leftTooltip={labels.tooltip.matchesAlly(String(r.teammate_matches))}
            rightTooltip={labels.tooltip.matchesEnemy(String(r.enemy_matches))}
          />
        )
      },
    },
    {
      id: 'wr_ally',
      accessorFn: (r) => numOrUndef(r.teammate_win_rate),
      sortUndefined: 'last',
      sortDescFirst: true,
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.winRateAlly}</SortLabel>,
      cell: (ctx) => {
        const v = ctx.row.original.teammate_win_rate
        const color = percentColor(v)
        return (
          <span className="font-mono font-bold" style={color ? { color } : undefined}>
            {formatPercent(v, 0)}
          </span>
        )
      },
    },
    {
      id: 'wr_enemy',
      accessorFn: (r) => numOrUndef(r.enemy_win_rate),
      sortUndefined: 'last',
      sortDescFirst: true,
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.winRateEnemy}</SortLabel>,
      cell: (ctx) => {
        const v = ctx.row.original.enemy_win_rate
        const color = percentColor(v)
        return (
          <span className="font-mono font-bold" style={color ? { color } : undefined}>
            {formatPercent(v, 0)}
          </span>
        )
      },
    },
    {
      id: 'frags_deaths',
      // Tri sur le net frags − morts (kills_dealt − deaths_suffered).
      accessorFn: (r) => r.kills_dealt - r.deaths_suffered,
      sortDescFirst: true,
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.fragsDeaths}</SortLabel>,
      cell: (ctx) => {
        const r = ctx.row.original
        return (
          <SplitBar
            leftCount={r.kills_dealt}
            rightCount={r.deaths_suffered}
            leftColor={tokenCssVar('outcome-win')}
            rightColor={tokenCssVar('outcome-loss')}
            leftTooltip={labels.tooltip.fragsDealt(String(r.kills_dealt))}
            rightTooltip={labels.tooltip.deathsSuffered(String(r.deaths_suffered))}
          />
        )
      },
    },
    {
      id: 'ratio',
      accessorFn: (r) => numOrUndef(r.duel_ratio),
      sortUndefined: 'last',
      sortDescFirst: true,
      // Le Tooltip (rend un <div>) enveloppe le bouton de tri : div > button est
      // valide, l'inverse ne l'est pas.
      header: (ctx) => (
        <Tooltip content={labels.table.ratioTooltip}>
          <SortLabel column={ctx.column}>
            <span className="cursor-help border-b border-dashed border-current">{labels.table.ratio}</span>
          </SortLabel>
        </Tooltip>
      ),
      cell: (ctx) => {
        const v = ctx.row.original.duel_ratio
        const color = ratioColor(v)
        return (
          <span className="font-mono font-bold" style={color ? { color } : undefined}>
            {formatRatio(v)}
          </span>
        )
      },
    },
    {
      id: 'last_seen',
      accessorFn: (r) => lastSeenEpoch(r.last_seen_at),
      sortUndefined: 'last',
      sortDescFirst: true,
      header: (ctx) => <SortLabel column={ctx.column}>{labels.table.lastSeen}</SortLabel>,
      cell: (ctx) => (
        <span className="text-[0.85em] text-muted-foreground">
          {formatRelative(ctx.row.original.last_seen_at, labels.relative)}
        </span>
      ),
    },
  ]
}

export function RelationsTable({
  rows,
  labels,
  locale,
  onPlayerClick,
  emptyMessage,
}: {
  rows: RelationInsight[]
  labels: RelationsLabels
  locale: 'fr' | 'en'
  onPlayerClick: (gamertag: string) => void
  emptyMessage: string
}) {
  const columns = useMemo(
    () => buildColumns(labels, locale, onPlayerClick),
    [labels, locale, onPlayerClick],
  )
  // Pas d'état de tri initial : l'ordre serveur (matchs communs DESC) est
  // conservé tant qu'aucun en-tête n'est cliqué (A3).
  const [sorting, setSorting] = useState<SortingState>([])
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: RELATIONS_PAGE_SIZE,
  })
  const table = useReactTable<RelationInsight>({
    data: rows,
    columns,
    state: { sorting, pagination },
    // Reset explicite en page 1 au changement de tri (A4) : `autoResetPageIndex`
    // ne se déclenche pas de façon fiable ici, on pilote donc l'état.
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
    return <p className="text-sm text-muted-foreground">{emptyMessage}</p>
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="overflow-x-auto rounded-lg border border-border bg-card">
        <table className="w-full border-collapse text-xs">
          <thead>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id} className="text-muted-foreground">
                {hg.headers.map((h, idx) => (
                  <th
                    key={h.id}
                    aria-sort={h.column.getCanSort() ? ariaSortValue(h.column.getIsSorted()) : undefined}
                    className={`border border-border border-b-2 px-2 pb-1 pt-1 ${idx === 0 ? 'text-left' : 'text-right'}`}
                  >
                    {flexRender(h.column.columnDef.header, h.getContext())}
                  </th>
                ))}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => (
              <tr key={row.id} className="hover:bg-accent/40 transition-colors">
                {row.getVisibleCells().map((cell, idx) => (
                  <td
                    key={cell.id}
                    className={`border border-border px-2 py-1.5 ${idx === 0 ? 'text-left' : 'text-right'}`}
                  >
                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {table.getPageCount() > 1 && (
        <div className="flex items-center justify-end gap-2 px-1 text-xs text-muted-foreground">
          <button
            type="button"
            onClick={() => table.previousPage()}
            disabled={!table.getCanPreviousPage()}
            aria-label={labels.table.previous}
            className="rounded border border-border px-2 py-1 leading-none transition-colors hover:text-foreground disabled:opacity-40"
          >
            &lsaquo;
          </button>
          <span className="tabular-nums">
            {table.getState().pagination.pageIndex + 1} / {table.getPageCount()}
          </span>
          <button
            type="button"
            onClick={() => table.nextPage()}
            disabled={!table.getCanNextPage()}
            aria-label={labels.table.next}
            className="rounded border border-border px-2 py-1 leading-none transition-colors hover:text-foreground disabled:opacity-40"
          >
            &rsaquo;
          </button>
        </div>
      )}
    </div>
  )
}
