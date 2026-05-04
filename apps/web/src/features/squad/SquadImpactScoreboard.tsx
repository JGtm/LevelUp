/**
 * SquadImpactScoreboard — teammates.07 : tableau scoreboard "Impact des coéquipiers".
 *
 * Spec : .ai/charts_specs/teammates/07_impact_taquinerie.yaml
 *
 * Implémentation TanStack Table :
 *  - Colonnes dynamiques : Joueur (sticky) + 1 par match avec ≥1 badge + 8 colonnes
 *    agrégat (1 par badge_key) + Score + Badge ranking.
 *  - Cellule joueur×match : pictos Fluent Flat empilés 2/ligne (BadgeIcon).
 *  - Cellule agrégat : compte ou "—" si 0. Couleur best/worst selon extrêmes (best
 *    inversé pour les badges "négatifs" cf. impactInverted).
 *  - Header de colonne match : fond coloré selon outcome du joueur principal.
 *  - Tri serveur : players DESC par score → Champion (rank=1) / Passager
 *    clandestin (rank=N, score≥0) / Maillon faible (rank=N, score<0).
 */
import { useMemo } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'

import type {
  SquadImpactMatrix,
  SquadImpactPlayerSummary,
  SquadImpactCell,
} from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { tokenCssVar } from '@/lib/accessibility'
import { BadgeIcon } from '@/components/feedback/BadgeIcon'
import { Tooltip } from '@/components/ui/tooltip'
import { Card, CardContent } from '@/components/ui/card'
import { getSquadText } from './i18n'

/** Badges où un count élevé est PIRE (rouge=worst au lieu de best). */
const BADGE_INVERTED: Record<string, true> = {
  last_casualty: true,
  last_group_kill: true,
  first_group_death: true,
  false_brother: true,
}

function outcomeBg(outcome: number): string {
  switch (outcome) {
    case 2:
      return `color-mix(in srgb, ${tokenCssVar('outcome-win')} 30%, transparent)`
    case 3:
      return `color-mix(in srgb, ${tokenCssVar('outcome-loss')} 30%, transparent)`
    default:
      return `color-mix(in srgb, ${tokenCssVar('outcome-dnf')} 15%, transparent)`
  }
}

interface ImpactRow {
  player: string
  /** match_id → liste de badge_keys */
  perMatch: Record<string, string[]>
  /** badge_key → count agrégé */
  perBadge: Record<string, number>
  score: number
  rank: number
  badge: 'champion' | 'middle' | 'maillon-faible' | 'passager-clandestin'
}

function buildRows(matrix: SquadImpactMatrix): ImpactRow[] {
  const cellByPM = new Map<string, SquadImpactCell>()
  for (const c of matrix.cells) {
    cellByPM.set(`${c.player}|${c.match_id}`, c)
  }
  return matrix.players.map((p, idx) => {
    const perMatch: Record<string, string[]> = {}
    for (const m of matrix.matches) {
      perMatch[m.match_id] = cellByPM.get(`${p.player}|${m.match_id}`)?.badge_keys ?? []
    }
    const perBadge: Record<string, number> = {}
    for (const b of p.counts) perBadge[b.badge_key] = b.count
    const rank = idx + 1
    const isLast = idx === matrix.players.length - 1
    const lastScore = matrix.players[matrix.players.length - 1]?.score ?? 0
    let badge: ImpactRow['badge'] = 'middle'
    if (rank === 1) badge = 'champion'
    else if (isLast && lastScore < 0) badge = 'maillon-faible'
    else if (isLast) badge = 'passager-clandestin'
    return { player: p.player, perMatch, perBadge, score: p.score, rank, badge }
  })
}

/** Pour chaque badge_key, calcule (min, max) parmi les counts non nuls. */
function computeExtremes(players: SquadImpactPlayerSummary[]): Record<string, { min: number; max: number }> {
  const out: Record<string, { min: number; max: number }> = {}
  for (const p of players) {
    for (const c of p.counts) {
      const v = c.count
      if (v <= 0) continue
      const cur = out[c.badge_key]
      if (!cur) out[c.badge_key] = { min: v, max: v }
      else out[c.badge_key] = { min: Math.min(cur.min, v), max: Math.max(cur.max, v) }
    }
  }
  // Conserver uniquement les badges avec ≥2 valeurs distinctes pour avoir un best/worst.
  for (const k of Object.keys(out)) {
    if (out[k].min === out[k].max) delete out[k]
  }
  return out
}

function aggCellClass(badgeKey: string, count: number, ext: { min: number; max: number } | undefined): string {
  if (!ext) return ''
  const inverted = BADGE_INVERTED[badgeKey] === true
  if (count === ext.max) return inverted ? 'text-[var(--ac-perf-tier-5)] font-semibold' : 'text-[var(--ac-perf-tier-1)] font-semibold'
  if (count === ext.min) return inverted ? 'text-[var(--ac-perf-tier-1)] font-semibold' : 'text-[var(--ac-perf-tier-5)] font-semibold'
  return ''
}

/** Découpe une liste de badge_keys en lignes de 2 pour l'empilage en cellule. */
function chunkPairs<T>(arr: T[]): T[][] {
  const out: T[][] = []
  for (let i = 0; i < arr.length; i += 2) out.push(arr.slice(i, i + 2))
  return out
}

interface SquadImpactScoreboardProps {
  matrix: SquadImpactMatrix
}

export function SquadImpactScoreboard({ matrix }: SquadImpactScoreboardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const i18n = t.impact

  const rows = useMemo(() => buildRows(matrix), [matrix])
  const extremes = useMemo(() => computeExtremes(matrix.players), [matrix.players])
  const scoreExt = useMemo(() => {
    const scores = matrix.players.map((p) => p.score)
    if (scores.length < 2) return undefined
    const min = Math.min(...scores)
    const max = Math.max(...scores)
    return min === max ? undefined : { min, max }
  }, [matrix.players])

  const columns = useMemo<ColumnDef<ImpactRow>[]>(() => {
    const cols: ColumnDef<ImpactRow>[] = [
      {
        id: 'player',
        header: i18n.colPlayer,
        accessorKey: 'player',
        cell: (ctx) => (
          <span className="whitespace-nowrap font-medium">{ctx.row.original.player}</span>
        ),
      },
      ...matrix.matches.map<ColumnDef<ImpactRow>>((m, idx) => ({
        id: `match-${m.match_id}`,
        header: () => (
          <div
            className="text-center text-xs font-semibold"
            style={{ background: outcomeBg(m.outcome), padding: '4px 6px', borderRadius: 3 }}
            title={m.match_id}
          >
            {idx + 1}
          </div>
        ),
        cell: (ctx) => {
          const keys = ctx.row.original.perMatch[m.match_id] ?? []
          if (keys.length === 0) return null
          return (
            <div className="flex flex-col items-center gap-0.5" data-testid="squad-impact-cell">
              {chunkPairs(keys).map((pair, i) => (
                <div key={i} className="flex gap-0.5">
                  {pair.map((k) => (
                    <Tooltip
                      key={k}
                      content={
                        <>
                          <span className="font-semibold">{i18n.badgeNames[k] ?? k}</span>
                          {i18n.badgeDescriptions[k] && (
                            <p className="text-muted-foreground mt-0.5">{i18n.badgeDescriptions[k]}</p>
                          )}
                        </>
                      }
                    >
                      <BadgeIcon badgeKey={k} size={18} />
                    </Tooltip>
                  ))}
                </div>
              ))}
            </div>
          )
        },
      })),
      ...matrix.badge_ord.map<ColumnDef<ImpactRow>>((badgeKey) => ({
        id: `agg-${badgeKey}`,
        header: () => (
          <Tooltip
            content={
              <>
                <span className="font-semibold">{i18n.badgeNames[badgeKey] ?? badgeKey}</span>
                {i18n.badgeDescriptions[badgeKey] && (
                  <p className="text-muted-foreground mt-0.5">{i18n.badgeDescriptions[badgeKey]}</p>
                )}
              </>
            }
          >
            <BadgeIcon badgeKey={badgeKey} size={18} />
          </Tooltip>
        ),
        cell: (ctx) => {
          const v = ctx.row.original.perBadge[badgeKey] ?? 0
          if (v === 0) return <span className="text-muted-foreground">—</span>
          return <span className={aggCellClass(badgeKey, v, extremes[badgeKey])}>{v}</span>
        },
      })),
      {
        id: 'score',
        header: i18n.colScore,
        cell: (ctx) => {
          const s = ctx.row.original.score
          const formatted = s > 0 ? `+${s}` : `${s}`
          let cls = ''
          if (scoreExt) {
            if (s === scoreExt.max) cls = 'text-[var(--ac-perf-tier-1)] font-semibold'
            else if (s === scoreExt.min) cls = 'text-[var(--ac-perf-tier-5)] font-semibold'
          }
          return <span className={cls}>{formatted}</span>
        },
      },
      {
        id: 'badge',
        header: i18n.colBadge,
        cell: (ctx) => {
          switch (ctx.row.original.badge) {
            case 'champion':
              return (
                <span title={i18n.badgeChampion} className="inline-flex items-center gap-1 whitespace-nowrap">
                  <BadgeIcon badgeKey="champion" size={16} /> {i18n.badgeChampionShort}
                </span>
              )
            case 'maillon-faible':
              return (
                <span title={i18n.badgeWeakLink} className="inline-flex items-center gap-1 whitespace-nowrap">
                  <BadgeIcon badgeKey="maillon_faible" size={16} /> {i18n.badgeWeakLinkShort}
                </span>
              )
            case 'passager-clandestin':
              return (
                <span title={i18n.badgeStowaway} className="inline-flex items-center gap-1 whitespace-nowrap">
                  <BadgeIcon badgeKey="passager_clandestin" size={16} /> {i18n.badgeStowawayShort}
                </span>
              )
            default:
              return null
          }
        },
      },
    ]
    return cols
  }, [matrix.matches, matrix.badge_ord, extremes, scoreExt, i18n])

  const table = useReactTable<ImpactRow>({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  if (matrix.matches.length === 0 || matrix.players.length === 0) return null

  return (
    <Card data-testid="squad-impact-section">
      <CardContent className="pt-4 space-y-3">
        <h3 className="text-base font-semibold">{i18n.title}</h3>
        <div
          className="overflow-x-auto rounded-md border border-border"
          data-testid="squad-impact-scoreboard"
        >
          <table className="w-full border-collapse text-sm">
            <thead className="bg-muted">
              {table.getHeaderGroups().map((hg) => (
                <tr key={hg.id} className="border-b border-border">
                  {hg.headers.map((h, idx) => (
                    <th
                      key={h.id}
                      className={`px-2 py-1 text-center align-bottom font-medium ${idx > 0 ? 'border-l border-border/60' : ''}`}
                    >
                      {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                    </th>
                  ))}
                </tr>
              ))}
            </thead>
            <tbody className="divide-y divide-border">
              {table.getRowModel().rows.map((r) => (
                <tr key={r.id}>
                  {r.getVisibleCells().map((cell, idx) => (
                    <td
                      key={cell.id}
                      className={`px-2 py-1 align-middle ${idx > 0 ? 'border-l border-border/60' : ''}`}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </CardContent>
    </Card>
  )
}
