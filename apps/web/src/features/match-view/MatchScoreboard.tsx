/**
 * MatchScoreboard — tableau de score du match.
 *
 * Ordre des colonnes : Joueur · Rang · Score · Frags · Morts · Assist. · FDA
 *   · Outil de destr. · Folie meurt. · Tirs à la Tête · Frags parfaits
 *   · Tirs · Tirs au but · Précision · Corps à corps · Armes lourdes
 *   · Dégâts infligés · Dégâts subis · Vie moy.
 *   · Rendement · Résist. · Dégâts/Frags
 */
import { Fragment, useMemo, useState } from 'react'
import { useFieldLabel } from '@/lib/i18n/fieldMappings'
import {
  type ColumnDef,
  flexRender,
  getCoreRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { useParams, useNavigate } from '@tanstack/react-router'
import { PlayerDetailPanel } from './PlayerDetailPanel'
import type {
  MatchCitationSnippet,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewHeader,
  MatchViewRank,
} from '@/lib/api/types'
import { formatDurationMMSS } from '@/lib/formatters'
import { tokenCssVar } from '@/lib/accessibility'
import type { MatchViewText } from './i18n'
import { parseTeamSideID, resolveTeamName } from '@/lib/halo/teamNames'
import {
  cellState,
  cellStyle,
  getExtremes,
  getMvpLvp,
  type CellState,
  type ColDef,
  type Extremes,
} from './MatchScoreboard.logic'

/**
 * Colonne grenade : `grenade_kills` est disponible pour les deux titres mais la
 * colonne n'est PAS encore activée (décision produit, h5-finitions). Ce flag est
 * l'unique point de contrôle (métadonnée + colonne rendue) → bascule atomique,
 * aucun highlight/MVP fantôme tant qu'il vaut `false`.
 */
const SHOW_GRENADE_KILLS_COLUMN: boolean = false

function buildHighlightCols(t: MatchViewText, offensiveLabel: string, defensiveLabel: string): ColDef[] {
  return [
    { key: 'rank', label: 'Rang', inverted: true },
    { key: 'score', label: 'Score', inverted: false, fmt: (v) => new Intl.NumberFormat('fr-FR').format(v) },
    { key: 'kills', label: t.combatKillsLabel, inverted: false },
    { key: 'deaths', label: t.combatDeathsLabel, inverted: true },
    { key: 'assists', label: 'Assist.', inverted: false },
    { key: 'kda', label: t.sbColKda, inverted: false, fmt: (v) => v.toFixed(2) },
    { key: 'max_killing_spree', label: 'Folie meurt.', inverted: false },
    { key: 'headshot_kills', label: 'Tirs à la Tête', inverted: false },
    { key: 'perfect_kills', label: 'Frags parfaits', inverted: false },
    { key: 'shots_fired', label: 'Tirs', inverted: false },
    { key: 'shots_hit', label: t.sbColShotsHit, inverted: false },
    { key: 'accuracy', label: t.sbColAccuracy, inverted: false, fmt: (v) => `${v.toFixed(1)}%` },
    { key: 'melee_kills', label: t.sbColMeleeKills, inverted: false },
    { key: 'power_weapon_kills', label: 'Armes lourdes', inverted: false },
    ...(SHOW_GRENADE_KILLS_COLUMN ? [{ key: 'grenade_kills', label: t.labelGrenade, inverted: false } as ColDef] : []),
    // Mécaniques de kill natives Halo 5 (assassinats + compétences spartiate).
    // Auto-masquées hors H5 : `null` pour Infinite → retirées par le filtre
    // data-driven (cf. presentKeys / visibleColumn) plus bas.
    { key: 'assassination_kills', label: t.labelAssassination, inverted: false },
    { key: 'ground_pound_kills', label: t.labelGroundPound, inverted: false },
    { key: 'shoulder_bash_kills', label: t.labelShoulderBash, inverted: false },
    { key: 'damage_dealt', label: t.sbColDamageDealt, inverted: false, fmt: (v) => v.toFixed(0) },
    { key: 'damage_taken', label: t.sbColDamageTaken, inverted: true, fmt: (v) => v.toFixed(0) },
    { key: 'avg_life_seconds', label: 'Vie moy.', inverted: false, fmt: (v) => formatDurationMMSS(v, '—') },
    { key: 'offensive_conversion', label: offensiveLabel, inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%` },
    { key: 'defensive_resistance', label: defensiveLabel, inverted: false, fmt: (v) => v < 0 ? '∞' : `${((v - 1) * 100).toFixed(0)}%` },
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
  const { playerSlug } = useParams({ strict: false }) as { playerSlug?: string }
  const navigate = useNavigate()
  const offensiveLabel = useFieldLabel('offensive_conversion')
  const defensiveLabel = useFieldLabel('defensive_resistance')

  const highlightCols = useMemo(() => buildHighlightCols(t, offensiveLabel, defensiveLabel), [t, offensiveLabel, defensiveLabel])
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

  // Masquage data-driven des colonnes : une colonne statistique dont AUCUNE
  // ligne du lobby (bots inclus) ne porte de valeur est retirée. Règle
  // title-agnostic, sans test de titre — supprime « Dégâts subis » / « Résist. »
  // en Halo 5 (non fournis par l'API) et les mécaniques natives hors H5 (null).
  // Calculée au niveau LOBBY → les deux équipes affichent les mêmes colonnes.
  const presentKeys = useMemo(() => {
    const present = new Set<string>()
    for (const c of highlightCols) {
      if (rows.some((r) => (r[c.key] as number | null | undefined) != null)) {
        present.add(String(c.key))
      }
    }
    return present
  }, [rows, highlightCols])

  function goToExplorer(gamertag: string, e: React.MouseEvent) {
    if (!playerSlug) return
    e.stopPropagation()
    void navigate({
      to: '/players/$playerSlug/explorer',
      params: { playerSlug },
      search: { mode: 'player', target: gamertag },
    })
  }

  if (rows.length === 0) {
    return (
      <div className="flex min-h-[200px] items-center justify-center text-sm text-muted-foreground">
        {t.scoreboardNoData}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {teams.map((side) => (
        <TeamScoreboard
          key={side || 'unknown'}
          teamSide={side}
          isMyTeam={side !== '' && side === myTeamSide}
          rows={rows.filter((r) => (r.team_side ?? '') === side)}
          highlightCols={highlightCols}
          presentKeys={presentKeys}
          extremesByKey={extremesByKey}
          mvpXuid={mvpXuid}
          lvpXuid={lvpXuid}
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
    </div>
  )
}

interface TeamScoreboardProps {
  /** team_side du backend (ex: "t0", "t1") — utilisé pour résoudre Eagle/Cobra. */
  teamSide: string
  /** True si ce groupe correspond à l'équipe du joueur principal (`is_me`). */
  isMyTeam: boolean
  rows: MatchScoreboardRow[]
  highlightCols: ColDef[]
  /** Clés de colonnes avec au moins une valeur sur le lobby (masquage data-driven). */
  presentKeys: Set<string>
  /** Extremes (min/max) par colonne, calculés au niveau LOBBY (toutes équipes). */
  extremesByKey: Record<string, Extremes>
  /** MVP/LVP du LOBBY (un seul de chaque, partagé par tous les TeamScoreboard). */
  mvpXuid: string | null
  lvpXuid: string | null
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
  presentKeys,
  extremesByKey,
  mvpXuid: mvp,
  lvpXuid: lvp,
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

  const isRanked = !!header?.is_ranked

  const columns = useMemo<ColumnDef<ScoreboardRowVM>[]>(() => {
    const hlMap = new Map(highlightCols.map((c) => [String(c.key), c]))

    function hlDef(key: string): ColumnDef<ScoreboardRowVM> {
      const c = hlMap.get(key)
      return {
        id: key,
        header: c?.label ?? key,
        cell: (ctx) => {
          const val = ctx.row.original[key as keyof MatchScoreboardRow] as number | null
          const formatted = val == null ? '—' : c?.fmt ? c.fmt(val) : String(val)
          return <span className="font-mono">{formatted}</span>
        },
      }
    }

    const rankBadgeCol: ColumnDef<ScoreboardRowVM> = {
      id: 'csr_badge',
      header: isRanked ? t.sbColCsr : t.sbDetailLusr,
      cell: (ctx) => {
        const url = ctx.row.original.skill_rank?.icon_url
        const label = ctx.row.original.skill_rank?.tier_label
        // Pas d'icône mais un palier connu (CSR Halo 5 sans badge résolu) → afficher le
        // libellé de palier plutôt qu'un « — » (cf. signalement #2). « — » réservé au cas
        // sans icône NI palier.
        if (!url) {
          return label ? (
            <span className="font-mono text-2xs">{label}</span>
          ) : (
            <span className="font-mono text-muted-foreground">—</span>
          )
        }
        return (
          <img
            src={url}
            alt={label ?? (isRanked ? t.sbColCsr : t.sbDetailLusr)}
            className="h-7 w-7 object-contain mx-auto"
            loading="lazy"
          />
        )
      },
    }

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
          const displayGamertag = r.gamertag
          return (
            <span className="whitespace-nowrap">
              <span className="mr-1 text-muted-foreground">{isExpanded ? '▾' : '▸'}</span>
              {linkable ? (
                <button
                  type="button"
                  className="font-medium text-foreground hover:text-primary hover:underline transition-colors"
                  onClick={(e) => onPlayerClick(r.gamertag, e)}
                  title={`Voir l'historique avec ${displayGamertag}`}
                >
                  {displayGamertag}
                </button>
              ) : (
                <span className={`font-medium ${r.is_bot ? 'text-muted-foreground italic' : 'text-foreground'}`}>{displayGamertag}</span>
              )}
              {r.is_bot && (
                <span className="ml-1 rounded px-1 py-0 text-2xs font-bold bg-muted text-muted-foreground uppercase tracking-wide">Bot</span>
              )}
              {r._isMvp && (
                <span
                  className="ml-1 rounded px-1 py-0 text-2xs font-bold uppercase tracking-wide"
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
                  className="ml-1 rounded px-1 py-0 text-2xs font-bold uppercase tracking-wide"
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
      rankBadgeCol,
      hlDef('rank'),
      hlDef('score'),
      hlDef('kills'),
      hlDef('deaths'),
      hlDef('assists'),
      hlDef('kda'),
      {
        id: 'top_weapon',
        header: 'Outil de destr.',
        cell: (ctx) => {
          const lbl = ctx.row.original.top_weapon_label
          return <span className="text-muted-foreground">{lbl ?? '—'}</span>
        },
      },
      hlDef('max_killing_spree'),
      hlDef('headshot_kills'),
      hlDef('perfect_kills'),
      hlDef('shots_fired'),
      hlDef('shots_hit'),
      hlDef('accuracy'),
      hlDef('melee_kills'),
      hlDef('power_weapon_kills'),
      hlDef('damage_dealt'),
      hlDef('damage_taken'),
      hlDef('avg_life_seconds'),
      hlDef('offensive_conversion'),
      hlDef('defensive_resistance'),
    ]

    // Retire les colonnes statistiques sans aucune valeur sur le lobby (cf.
    // presentKeys). Les colonnes non statistiques (gamertag, csr_badge,
    // top_weapon) ne figurent pas dans highlightCols → toujours conservées.
    const hlKeys = new Set(highlightCols.map((c) => String(c.key)))
    return cols.filter((c) => !hlKeys.has(c.id ?? '') || presentKeys.has(c.id ?? ''))
  }, [highlightCols, presentKeys, expandedXuid, playerSlug, onPlayerClick, isRanked, t])

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
  const teamHeaderBg = `color-mix(in oklab, ${teamColorVar} 30%, transparent)`
  const teamHeaderBorder = `2px solid color-mix(in oklab, ${teamColorVar} 55%, transparent)`
  const teamHeaderColor = `color-mix(in oklab, ${teamColorVar} 80%, var(--foreground))`

  return (
    <div className="rounded-lg overflow-hidden border-2 border-border">
      <div className="overflow-x-auto">
      <table className="w-full border-collapse text-3xs">
        <thead>
          <tr>
            <th
              colSpan={columns.length}
              className="border border-border px-3 py-2 text-left text-sm font-bold uppercase tracking-wider"
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
            <tr key={hg.id} className="text-muted-foreground">
              {hg.headers.map((h) => {
                const isPlayerCol = h.column.id === 'gamertag'
                const align = isPlayerCol ? 'text-left' : 'text-right'
                return (
                  <th
                    key={h.id}
                    className={`border border-border border-b-2 px-2 pb-1 pt-1 ${align}`}
                  >
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
                  className={`cursor-pointer hover:bg-accent transition-colors ${rowBg}`}
                  onClick={() => onToggleExpand(r.xuid)}
                >
                  {row.getVisibleCells().map((cell) => {
                    const isPlayerCol = cell.column.id === 'gamertag'
                    const align = isPlayerCol ? 'text-left' : 'text-right'
                    const highlight = r._cellStates[cell.column.id]
                    const tone: React.CSSProperties = isPlayerCol
                      ? gamertagCellStyle
                      : highlight
                        ? cellStyle(highlight)
                        : {}
                    return (
                      <td
                        key={cell.id}
                        className={`border border-border px-2 py-1 ${align} whitespace-nowrap`}
                        style={tone}
                      >
                        {flexRender(cell.column.columnDef.cell, cell.getContext())}
                      </td>
                    )
                  })}
                </tr>
                {isExpanded && (
                  <tr>
                    <td colSpan={columns.length} className="border border-border p-0">
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
