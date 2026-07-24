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
 *
 * EXCEPTION tri client par en-têtes (I16) : volontairement NON triable.
 * L'ordre des lignes EST le classement lui-même (rank dérivé de la position,
 * cf. buildRows) — un tri client casserait la sémantique Champion / Maillon
 * faible / Passager clandestin, qui dépend de la position 1re/dernière.
 */
import { useMemo, type CSSProperties } from 'react'
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
import { HeaderInfoTooltip } from '@/lib/table/columnMeta'
import { getSquadText } from './i18n'

/** Badges où un count élevé est PIRE (rouge=worst au lieu de best). */
const BADGE_INVERTED: Record<string, true> = {
  last_casualty: true,
  last_group_kill: true,
  first_group_death: true,
  false_brother: true,
  kamikaze: true,
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
  for (const c of matrix.cells ?? []) {
    cellByPM.set(`${c.player}|${c.match_id}`, c)
  }
  const players = matrix.players ?? []
  const matches = matrix.matches ?? []
  return players.map((p, idx) => {
    const perMatch: Record<string, string[]> = {}
    for (const m of matches) {
      perMatch[m.match_id] = cellByPM.get(`${p.player}|${m.match_id}`)?.badge_keys ?? []
    }
    const perBadge: Record<string, number> = {}
    for (const b of p.counts ?? []) perBadge[b.badge_key] = b.count
    const rank = idx + 1
    const isLast = idx === players.length - 1
    const lastScore = players[players.length - 1]?.score ?? 0
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
    for (const c of p.counts ?? []) {
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

type CellState = 'best' | 'worst' | 'neutral'

/**
 * Surbrillance best/worst — MÊME rendu que MVP/LVP du scoreboard de match
 * (cf. match-view/MatchScoreboard.logic.ts::cellStyle) : fond teinté 28% +
 * texte pleine couleur du token outcome-win (best) / outcome-loss (worst).
 */
function extremeStyle(state: CellState): CSSProperties {
  if (state === 'neutral') return {}
  const tokenVar = state === 'best' ? 'var(--ac-outcome-win)' : 'var(--ac-outcome-loss)'
  return {
    backgroundColor: `color-mix(in oklab, ${tokenVar} 28%, transparent)`,
    color: tokenVar,
    fontWeight: state === 'best' ? 600 : 500,
  }
}

function aggCellState(badgeKey: string, count: number, ext: { min: number; max: number } | undefined): CellState {
  if (!ext) return 'neutral'
  const inverted = BADGE_INVERTED[badgeKey] === true
  if (count === ext.max) return inverted ? 'worst' : 'best'
  if (count === ext.min) return inverted ? 'best' : 'worst'
  return 'neutral'
}

/**
 * Répartit les badges sur 2 RANGÉES maximum, qui s'élargissent en COLONNES quand
 * il y en a beaucoup (demande user : ajouter des colonnes plutôt que des rangées).
 */
function splitTwoRows<T>(arr: T[]): T[][] {
  if (arr.length === 0) return []
  const half = Math.ceil(arr.length / 2)
  return [arr.slice(0, half), arr.slice(half)].filter((r) => r.length > 0)
}

interface SquadImpactScoreboardProps {
  matrix: SquadImpactMatrix
}

export function SquadImpactScoreboard({ matrix }: SquadImpactScoreboardProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const i18n = t.impact

  // Le contrat OpenAPI expose les tableaux en `T[] | null` (le Go peut renvoyer
  // null). On dérive des vues non-null une fois, réutilisées partout.
  const matrixPlayers = useMemo(() => matrix.players ?? [], [matrix.players])
  const matrixMatches = useMemo(() => matrix.matches ?? [], [matrix.matches])
  const matrixBadgeOrd = useMemo(() => matrix.badge_ord ?? [], [matrix.badge_ord])

  const rows = useMemo(() => buildRows(matrix), [matrix])
  const extremes = useMemo(() => computeExtremes(matrixPlayers), [matrixPlayers])
  const scoreExt = useMemo(() => {
    const scores = matrixPlayers.map((p) => p.score)
    if (scores.length < 2) return undefined
    const min = Math.min(...scores)
    const max = Math.max(...scores)
    return min === max ? undefined : { min, max }
  }, [matrixPlayers])

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
      ...matrixMatches.map<ColumnDef<ImpactRow>>((m, idx) => ({
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
              {splitTwoRows(keys).map((row, i) => (
                <div key={i} className="flex gap-0.5">
                  {row.map((k) => (
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
      ...matrixBadgeOrd.map<ColumnDef<ImpactRow>>((badgeKey) => ({
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
          const st: CellState = v === 0 ? 'neutral' : aggCellState(badgeKey, v, extremes[badgeKey])
          return (
            <div className="-mx-2 -my-1 px-2 py-1 text-center" style={extremeStyle(st)}>
              {v === 0 ? <span className="text-muted-foreground">—</span> : v}
            </div>
          )
        },
      })),
      {
        id: 'score',
        header: i18n.colScore,
        meta: { headerTooltip: i18n.colScoreTooltip },
        cell: (ctx) => {
          const s = ctx.row.original.score
          const formatted = s > 0 ? `+${s}` : `${s}`
          let st: CellState = 'neutral'
          if (scoreExt) {
            if (s === scoreExt.max) st = 'best'
            else if (s === scoreExt.min) st = 'worst'
          }
          return (
            <div className="-mx-2 -my-1 px-2 py-1 text-center" style={extremeStyle(st)}>{formatted}</div>
          )
        },
      },
      {
        id: 'badge',
        header: i18n.colBadge,
        meta: { headerTooltip: i18n.colBadgeTooltip },
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
  }, [matrixMatches, matrixBadgeOrd, extremes, scoreExt, i18n])

  const table = useReactTable<ImpactRow>({
    data: rows,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  if (matrixMatches.length === 0 || matrixPlayers.length === 0) {
    // Bloc vide : on conserve le cadre bordé (même style que la table) avec un
    // message, au lieu de faire disparaître la section.
    return (
      <div
        className="flex min-h-[120px] items-center justify-center rounded-md border border-border text-sm text-muted-foreground"
        data-testid="squad-impact-scoreboard"
      >
        {t.empty.noBlockData}
      </div>
    )
  }

  // Bloc « sorti » : table seule, sans Card ni titre interne. Le titre « Impact
  // des coéquipiers » est porté par un titre de section dans SquadSynergiesPage.
  return (
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
                  <span className="inline-flex items-center gap-1">
                    {h.isPlaceholder ? null : flexRender(h.column.columnDef.header, h.getContext())}
                    <HeaderInfoTooltip text={h.column.columnDef.meta?.headerTooltip} />
                  </span>
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
                  className={`px-2 py-1 align-middle ${idx > 0 ? 'border-l border-border/60 text-center' : ''}`}
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
