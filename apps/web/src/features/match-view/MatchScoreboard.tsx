/**
 * MatchScoreboard — tableau de score du match (mock 4b corrigé).
 *
 * Spec visuelle : .ai/mocks/__mockup_tables.html section "4b — MatchScoreboard.tsx corrigé".
 *
 * Différences vs ancienne version :
 *  - Colonnes Rang + Score ajoutées en tête (après Joueur).
 *  - Colonne Outil destruct. (top_weapon_label) ajoutée avant Résultat.
 *  - Vie moy. formattée en mm:ss via formatDurationMMSS().
 *  - MVP/LVP recalculés sur multi-best/worst (≥ 2 best ou ≥ 2 worst), plus
 *    seulement sur kills.
 *  - Construit avec TanStack Table v8 (cohérence avec SquadMatchHistoryTable).
 */
import { Fragment, useMemo, useState } from 'react'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { PlayerDetailPanel } from './PlayerDetailPanel'
import type {
  MatchCitationSnippet,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewRank,
} from '@/lib/api/types'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { formatDurationMMSS } from '@/lib/formatters'
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchViewText } from './i18n'
import { parseTeamSideID, resolveTeamName } from './teamNames'
import {
  cellState,
  cellStyle,
  formatRank,
  formatScore,
  getExtremes,
  getMvpLvp,
  type CellState,
  type ColDef,
  type Extremes,
} from './MatchScoreboard.logic'

function buildHighlightCols(fieldMappings?: FieldMappingsResponse): ColDef[] {
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  return [
    { key: 'kills', label: labelOf('kills'), inverted: false },
    { key: 'deaths', label: labelOf('deaths'), inverted: true },
    { key: 'assists', label: labelOf('assists'), inverted: false },
    { key: 'headshot_kills', label: labelOf('headshot_kills'), inverted: false },
    { key: 'max_killing_spree', label: labelOf('max_killing_spree'), inverted: false },
    { key: 'perfect_kills', label: 'Perf', inverted: false },
    { key: 'power_weapon_kills', label: labelOf('power_weapon_kills'), inverted: false },
    { key: 'melee_kills', label: labelOf('melee_kills'), inverted: false },
    { key: 'shots_fired', label: labelOf('shots_fired'), inverted: false },
    { key: 'shots_hit', label: labelOf('shots_hit'), inverted: false },
    { key: 'kda', label: labelOf('kda'), inverted: false, fmt: (v) => v.toFixed(2) },
    { key: 'damage_dealt', label: labelOf('damage_dealt'), inverted: false, fmt: (v) => v.toFixed(0) },
    { key: 'damage_taken', label: labelOf('damage_taken'), inverted: true, fmt: (v) => v.toFixed(0) },
    { key: 'offensive_conversion', label: 'Rendement', inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%` },
    { key: 'defensive_resistance', label: 'Résistance', inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%` },
    { key: 'damage_per_kill', label: 'Dmg/Frags', inverted: true, fmt: (v) => v.toFixed(0) },
  ]
}


interface ScoreboardRowVM extends MatchScoreboardRow {
  /** Composite cell-state map pour highlight, recalculé par team. */
  _cellStates: Record<string, CellState>
  _isMvp: boolean
  _isLvp: boolean
}

interface Props {
  rows: MatchScoreboardRow[]
  /** Pairs killer→victim du match — utilisés par PlayerDetailPanel pour Nemesis/Bully. */
  killerVictim?: MatchKillerVictimPair[] | null
  /** Citations du joueur principal — section "Médailles & citations" du panneau (is_me only). */
  citations?: MatchCitationSnippet[]
  /** Header — perf score + had_bot_teammate, utilisé par la section Local du panneau (is_me). */
  header?: MatchViewHeader
  /** Rank — LUSR/CSR + delta, utilisé par la section Local du panneau (is_me). */
  rank?: MatchViewRank
  /** I18n match-view (FR/EN). */
  t: MatchViewText
}

export function MatchScoreboard({ rows, killerVictim, citations, header, rank, t }: Props) {
  const [expandedXuid, setExpandedXuid] = useState<string | null>(null)
  const { data: fieldMappings } = useFieldMappings()
  const { playerSlug } = useParams({ strict: false }) as { playerSlug?: string }
  const navigate = useNavigate()

  const highlightCols = useMemo(() => buildHighlightCols(fieldMappings), [fieldMappings])
  const teams = useMemo(
    () => Array.from(new Set(rows.map((r) => r.team_side ?? ''))).sort(),
    [rows],
  )
  // Détecte le team_side du joueur principal pour distinguer "Mon équipe" vs
  // "Équipe adverse" (seul critère fiable — l'ancien check `side === '0'`
  // était faux quand le joueur est sur Cobra/Hades/etc.).
  const myTeamSide = useMemo(
    () => rows.find((r) => r.is_me)?.team_side ?? null,
    [rows],
  )
  // Best/worst + MVP/LVP calculés au niveau LOBBY (toutes équipes confondues),
  // pas par équipe. Bots exclus (un bot avec le plus de kills ne doit pas
  // voler le badge MVP). Le résultat est partagé par tous les TeamScoreboard.
  const humanRows = useMemo(() => rows.filter((r) => !r.is_bot), [rows])
  const extremesByKey = useMemo(
    () => Object.fromEntries(highlightCols.map((c) => [String(c.key), getExtremes(humanRows, c.key)])),
    [humanRows, highlightCols],
  )
  const { mvp: mvpXuid, lvp: lvpXuid } = useMemo(
    () => getMvpLvp(humanRows, highlightCols, extremesByKey),
    [humanRows, highlightCols, extremesByKey],
  )

  function goToExplorer(gamertag: string, e: React.MouseEvent) {
    if (!playerSlug) return
    e.stopPropagation()
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  return (
    <Card>
      <CardContent className="py-4">
        <p className="mb-3 text-sm font-semibold text-foreground">{t.scoreboardTitle}</p>
        {teams.map((side) => (
          <TeamScoreboard
            key={side || 'unknown'}
            teamSide={side}
            isMyTeam={side !== '' && side === myTeamSide}
            rows={rows.filter((r) => (r.team_side ?? '') === side)}
            highlightCols={highlightCols}
            extremesByKey={extremesByKey}
            mvpXuid={mvpXuid}
            lvpXuid={lvpXuid}
            fieldMappings={fieldMappings}
            expandedXuid={expandedXuid}
            onToggleExpand={(xuid) => setExpandedXuid((cur) => (cur === xuid ? null : xuid))}
            onPlayerClick={(gt, e) => goToExplorer(gt, e)}
            playerSlug={playerSlug}
            killerVictim={killerVictim}
            citations={citations}
            header={header}
            rank={rank}
            t={t}
          />
        ))}
      </CardContent>
    </Card>
  )
}

interface TeamScoreboardProps {
  /** team_side du backend (ex: "t0", "t1") — utilisé pour résoudre Eagle/Cobra. */
  teamSide: string
  /** True si ce groupe correspond à l'équipe du joueur principal (`is_me`). */
  isMyTeam: boolean
  rows: MatchScoreboardRow[]
  highlightCols: ColDef[]
  /** Extremes (min/max) par colonne, calculés au niveau LOBBY (toutes équipes). */
  extremesByKey: Record<string, Extremes>
  /** MVP/LVP du LOBBY (un seul de chaque, partagé par tous les TeamScoreboard). */
  mvpXuid: string | null
  lvpXuid: string | null
  fieldMappings?: FieldMappingsResponse
  expandedXuid: string | null
  onToggleExpand: (xuid: string) => void
  onPlayerClick: (gamertag: string, e: React.MouseEvent) => void
  playerSlug?: string
  killerVictim?: MatchKillerVictimPair[] | null
  citations?: MatchCitationSnippet[]
  header?: MatchViewHeader
  rank?: MatchViewRank
  t: MatchViewText
}

function TeamScoreboard({
  teamSide,
  isMyTeam,
  rows,
  highlightCols,
  extremesByKey,
  mvpXuid: mvp,
  lvpXuid: lvp,
  fieldMappings,
  expandedXuid,
  onToggleExpand,
  onPlayerClick,
  playerSlug,
  killerVictim,
  citations,
  header,
  rank,
  t,
}: TeamScoreboardProps) {

  const data: ScoreboardRowVM[] = useMemo(
    () =>
      rows.map((r) => {
        const states: Record<string, CellState> = {}
        // Pas de highlight best/worst sur les lignes bot (cellule neutre).
        if (!r.is_bot) {
          for (const c of highlightCols) {
            states[String(c.key)] = cellState(r[c.key] as number | null, extremesByKey[String(c.key)] as Extremes, c.inverted)
          }
        }
        return {
          ...r,
          _cellStates: states,
          _isMvp: !r.is_bot && r.xuid === mvp,
          _isLvp: !r.is_bot && r.xuid === lvp,
        }
      }),
    [rows, highlightCols, extremesByKey, mvp, lvp],
  )

  const columns = useMemo<ColumnDef<ScoreboardRowVM>[]>(() => {
    const cols: ColumnDef<ScoreboardRowVM>[] = [
      {
        id: 'gamertag',
        header: 'Joueur',
        cell: (ctx) => {
          const r = ctx.row.original
          const isExpanded = expandedXuid === r.xuid
          // Pas de lien vers Explorer pour les bots : ils n'existent pas hors
          // de ce match (leur xuid 'bid(N.0)' n'a aucun historique cross-match).
          const linkable = !r.is_me && !r.is_bot && playerSlug
          return (
            <span className="whitespace-nowrap">
              <span className="mr-1 text-muted-foreground">{isExpanded ? '▾' : '▸'}</span>
              {linkable ? (
                <button
                  type="button"
                  className="font-medium text-foreground hover:text-primary hover:underline transition-colors"
                  onClick={(e) => onPlayerClick(r.gamertag, e)}
                  title={`Voir l'historique avec ${r.gamertag}`}
                >
                  {r.gamertag}
                </button>
              ) : (
                <span className={`font-medium ${r.is_bot ? 'text-muted-foreground italic' : 'text-foreground'}`}>{r.gamertag}</span>
              )}
              {r.is_bot && (
                <span className="ml-1 rounded px-1 py-0 text-[10px] font-bold bg-muted text-muted-foreground uppercase tracking-wide">Bot</span>
              )}
              {r._isMvp && (
                <span
                  className="ml-1 rounded px-1 py-0 text-[10px] font-bold uppercase tracking-wide"
                  style={{
                    backgroundColor: 'color-mix(in oklab, var(--ac-outcome-win) 80%, transparent)',
                    color: 'var(--foreground)',
                  }}
                >
                  MVP
                </span>
              )}
              {r._isLvp && (
                <span
                  className="ml-1 rounded px-1 py-0 text-[10px] font-bold uppercase tracking-wide"
                  style={{
                    backgroundColor: 'color-mix(in oklab, var(--ac-outcome-loss) 80%, transparent)',
                    color: 'var(--foreground)',
                  }}
                >
                  LVP
                </span>
              )}
            </span>
          )
        },
      },
      {
        id: 'rank',
        header: 'Rang',
        cell: (ctx) => <span className="font-mono">{formatRank(ctx.row.original.rank)}</span>,
      },
      {
        id: 'score',
        header: 'Score',
        cell: (ctx) => <span className="font-mono">{formatScore(ctx.row.original.score)}</span>,
      },
      ...highlightCols.map<ColumnDef<ScoreboardRowVM>>((c) => ({
        id: String(c.key),
        header: c.label,
        cell: (ctx) => {
          const val = ctx.row.original[c.key] as number | null
          const formatted = val == null ? '—' : c.fmt ? c.fmt(val) : String(val)
          return <span className="font-mono">{formatted}</span>
        },
      })),
      {
        id: 'avg_life',
        header: 'Vie moy.',
        cell: (ctx) => {
          const r = ctx.row.original
          const seconds = r.avg_life_seconds ?? null
          return <span className="font-mono">{formatDurationMMSS(seconds, '—')}</span>
        },
      },
      {
        id: 'top_weapon',
        header: 'Outil destruct.',
        cell: (ctx) => {
          const lbl = ctx.row.original.top_weapon_label
          return <span className="text-muted-foreground">{lbl ?? '—'}</span>
        },
      },
      {
        id: 'outcome',
        header: fieldMappings?.fields['outcome']?.label ?? 'Résultat',
        cell: (ctx) => <span className="text-muted-foreground whitespace-nowrap">{ctx.row.original.outcome_label}</span>,
      },
    ]
    return cols
  }, [highlightCols, expandedXuid, playerSlug, fieldMappings, onPlayerClick])

  const table = useReactTable<ScoreboardRowVM>({
    data,
    columns,
    getCoreRowModel: getCoreRowModel(),
  })

  // Résolution du nom officiel d'équipe (Eagle / Cobra / ...) depuis team_side.
  // Fallback : "Équipe N" si team_id connu mais hors map, "Équipe inconnue"
  // si team_side malformé. Aligné sur src/config.py::TEAM_MAP (Python main).
  const officialName = resolveTeamName(teamSide)
  const teamID = parseTeamSideID(teamSide)
  const teamLabel = officialName
    ? t.teamLabelFmt(officialName)
    : teamID != null
      ? t.teamNumberedFmt(teamID)
      : t.teamUnknown
  // Couleur de fond du header : token sémantique team-ally / team-enemy,
  // overridable par les réglages d'accessibilité du joueur (cf. AccessibilityTab
  // → OutlineColorPicker, thought_log 2026-05-07). Gradient horizontal pour
  // garder l'effet du Python (`linear-gradient(90deg, color 38%, bg 88%)`).
  const teamColorVar = isMyTeam ? tokenCssVar('team-ally') : tokenCssVar('team-enemy')
  const teamHeaderBg = `linear-gradient(90deg, color-mix(in oklab, ${teamColorVar} 35%, transparent), transparent 88%)`
  const teamHeaderBorder = `2px solid color-mix(in oklab, ${teamColorVar} 55%, transparent)`
  const teamHeaderColor = `color-mix(in oklab, ${teamColorVar} 80%, var(--foreground))`

  // Nombre total de colonnes pour le colspan du header d'équipe.
  // = 1 (Joueur) + 2 (Rang/Score) + N (highlightCols) + 3 (Vie moy./Outil/Résultat).
  const totalCols = 1 + 2 + highlightCols.length + 3

  return (
    <div className="mb-4">
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr>
              <th
                colSpan={totalCols}
                className="px-3 py-2 text-left text-sm font-bold uppercase tracking-wider"
                style={{
                  background: teamHeaderBg,
                  color: teamHeaderColor,
                  borderBottom: teamHeaderBorder,
                }}
                aria-label={isMyTeam ? t.teamMine : t.teamEnemy}
              >
                {teamLabel}
              </th>
            </tr>
            {table.getHeaderGroups().map((hg) => (
              <tr key={hg.id} className="border-b border-border text-muted-foreground">
                {hg.headers.map((h, idx) => {
                  const isPlayerCol = h.column.id === 'gamertag'
                  const align = isPlayerCol ? 'text-left' : 'text-right'
                  const px = idx === 0 ? 'pr-2' : 'px-2'
                  return (
                    <th key={h.id} className={`pb-1 ${align} ${px} whitespace-nowrap`}>
                      {flexRender(h.column.columnDef.header, h.getContext())}
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody>
            {table.getRowModel().rows.map((row) => {
              const r = row.original
              const isExpanded = expandedXuid === r.xuid
              // Fond de ligne discret pour me / MVP / LVP (subtle, ne dispute
              // pas l'attention avec la border outcome de la cellule gamertag).
              const rowBg = r.is_me ? 'bg-info/10' : ''
              // Border outcome-win/outcome-loss sur la cellule gamertag du
              // MVP/LVP du LOBBY. Tinted background léger en plus de la border
              // pour faire ressortir la cellule sans hurler visuellement.
              const gamertagCellStyle: React.CSSProperties = r._isMvp
                ? {
                    boxShadow: 'inset 3px 0 0 0 var(--ac-outcome-win)',
                    backgroundColor: 'color-mix(in oklab, var(--ac-outcome-win) 12%, transparent)',
                  }
                : r._isLvp
                  ? {
                      boxShadow: 'inset 3px 0 0 0 var(--ac-outcome-loss)',
                      backgroundColor: 'color-mix(in oklab, var(--ac-outcome-loss) 12%, transparent)',
                    }
                  : {}
              return (
                <Fragment key={r.xuid}>
                  <tr
                    className={`cursor-pointer border-b border-border hover:bg-accent transition-colors ${rowBg}`}
                    onClick={() => onToggleExpand(r.xuid)}
                  >
                    {row.getVisibleCells().map((cell, idx) => {
                      const isPlayerCol = cell.column.id === 'gamertag'
                      const align = isPlayerCol ? 'text-left' : 'text-right'
                      const px = idx === 0 ? 'pr-2' : 'px-2'
                      const highlight = r._cellStates[cell.column.id]
                      const tone: React.CSSProperties = isPlayerCol
                        ? gamertagCellStyle
                        : highlight
                          ? cellStyle(highlight)
                          : {}
                      return (
                        <td key={cell.id} className={`py-1 ${align} ${px}`} style={tone}>
                          {flexRender(cell.column.columnDef.cell, cell.getContext())}
                        </td>
                      )
                    })}
                  </tr>
                  {isExpanded && (
                    <tr>
                      <td colSpan={columns.length} className="p-0">
                        <PlayerDetailPanel
                          row={r}
                          killerVictim={killerVictim}
                          citations={r.is_me ? citations : undefined}
                          header={header}
                          rank={rank}
                          playerSlug={playerSlug}
                          t={t}
                        />
                      </td>
                    </tr>
                  )}
                </Fragment>
              )
            })}
          </tbody>
        </table>
      </div>
    </div>
  )
}
