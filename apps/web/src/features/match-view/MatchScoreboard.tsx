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
  type SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { NUMERIC_SORT, localeTextSortingFn } from '@/features/explorer/explorerMatchesClientSort'
import { ariaSortOf, sortSuffixOf } from './sortHeader'
import { useParams, useNavigate } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'
import { useCapability } from '@/lib/capabilities/capabilities'
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
import { HeaderInfoTooltip } from '@/lib/table/columnMeta'
import type { MatchViewText } from './i18n'
import { MatchObjectivesSection } from './MatchObjectivesSection'
import { displayTierLabel } from './MatchHeader.utils'
import { localizeTierLabel } from '@/lib/skillTiers'
import { useAppShellStore } from '@/stores/appShellStore'
import {
  labelHasTeamWord,
  parseTeamSideID,
  resolveTeamColorFromID,
  resolveTeamName,
  teamLogoPath,
} from '@/lib/halo/teamNames'
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

/**
 * Clés des mécaniques de kill natives Halo 5 (assassinats + compétences spartiate :
 * coup au sol / charge spartane). Colonnes gatées par la capability
 * `native_kill_mechanics` : présentes sur Halo 5, ABSENTES sur Halo Infinite (le jeu
 * ne les track pas → stockées 0, pas null, donc le masquage data-driven `presentKeys`
 * ne suffit pas — `0 != null`). Source unique des clés pour les DEUX points de
 * définition de colonnes (buildHighlightCols + le tableau hlDef de TeamScoreboard).
 */
const NATIVE_KILL_MECHANIC_KEYS = [
  'assassination_kills',
  'ground_pound_kills',
  'shoulder_bash_kills',
] as const

function buildHighlightCols(
  t: MatchViewText,
  offensiveLabel: string,
  defensiveLabel: string,
  showNativeKillMechanics: boolean,
): ColDef[] {
  return [
    { key: 'rank', label: t.sbColRank, inverted: true, tooltip: t.sbColRankTooltip },
    { key: 'score', label: t.sbColScore, inverted: false, fmt: (v) => t.sbFormatScore(v) },
    { key: 'kills', label: t.combatKillsLabel, inverted: false },
    { key: 'deaths', label: t.combatDeathsLabel, inverted: true },
    { key: 'assists', label: t.sbColAssists, inverted: false },
    { key: 'kda', label: t.sbColKda, inverted: false, fmt: (v) => v.toFixed(2), tooltip: t.sbColKdaTooltip },
    { key: 'max_killing_spree', label: t.sbColMaxSpree, inverted: false, tooltip: t.sbColMaxSpreeTooltip },
    { key: 'headshot_kills', label: t.sbColHeadshots, inverted: false },
    { key: 'perfect_kills', label: t.sbColPerfectKills, inverted: false, tooltip: t.sbColPerfectKillsTooltip },
    { key: 'shots_fired', label: t.sbColShotsFired, inverted: false },
    { key: 'shots_hit', label: t.sbColShotsHit, inverted: false },
    { key: 'accuracy', label: t.sbColAccuracy, inverted: false, fmt: (v) => `${v.toFixed(1)}%`, tooltip: t.sbColAccuracyTooltip },
    { key: 'melee_kills', label: t.sbColMeleeKills, inverted: false, tooltip: t.sbColMeleeKillsTooltip },
    { key: 'power_weapon_kills', label: t.sbColPowerWeapons, inverted: false, tooltip: t.sbColPowerWeaponsTooltip },
    ...(SHOW_GRENADE_KILLS_COLUMN ? [{ key: 'grenade_kills', label: t.labelGrenade, inverted: false } as ColDef] : []),
    // Mécaniques de kill natives Halo 5 (assassinats + compétences spartiate).
    // Gatées par la capability `native_kill_mechanics` : Halo Infinite ne les track
    // pas (stockées 0, pas null → le filtre data-driven `presentKeys` ne les retire
    // pas) → exclues des colonnes quand la capability est absente. Halo 5 : conservées.
    ...(showNativeKillMechanics
      ? ([
          { key: 'assassination_kills', label: t.labelAssassination, inverted: false },
          { key: 'ground_pound_kills', label: t.labelGroundPound, inverted: false },
          { key: 'shoulder_bash_kills', label: t.labelShoulderBash, inverted: false },
        ] as ColDef[])
      : []),
    { key: 'damage_dealt', label: t.sbColDamageDealt, inverted: false, fmt: (v) => v.toFixed(0) },
    { key: 'damage_taken', label: t.sbColDamageTaken, inverted: true, fmt: (v) => v.toFixed(0) },
    { key: 'avg_life_seconds', label: t.sbColAvgLife, inverted: false, fmt: (v) => formatDurationMMSS(v, '—'), tooltip: t.sbColAvgLifeTooltip },
    { key: 'offensive_conversion', label: offensiveLabel, inverted: false, fmt: (v) => `${(v * 100).toFixed(0)}%`, tooltip: t.sbColOffensiveTooltip },
    { key: 'defensive_resistance', label: defensiveLabel, inverted: false, fmt: (v) => v < 0 ? '∞' : `${((v - 1) * 100).toFixed(0)}%`, tooltip: t.sbColDefensiveTooltip },
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
  const titleSlug = useTitleSlug()
  const offensiveLabel = useFieldLabel('offensive_conversion')
  const defensiveLabel = useFieldLabel('defensive_resistance')
  // Mécaniques de kill natives (assassinat / coup au sol / charge spartane) : gatées
  // par capability (présentes Halo 5, absentes Halo Infinite — non trackées par le jeu).
  // Thread jusqu'aux DEUX points de définition des colonnes (buildHighlightCols +
  // le tableau hlDef de TeamScoreboard) — cf. NATIVE_KILL_MECHANIC_KEYS.
  const showNativeKillMechanics = useCapability('native_kill_mechanics')

  const highlightCols = useMemo(
    () => buildHighlightCols(t, offensiveLabel, defensiveLabel, showNativeKillMechanics),
    [t, offensiveLabel, defensiveLabel, showNativeKillMechanics],
  )
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

  // Largeur commune de la colonne « Joueur », alignée sur le plus long gamertag
  // du LOBBY (toutes équipes/tables confondues), pour que chaque TeamScoreboard
  // dimensionne cette colonne à l'identique — sans quoi chaque table auto-sizerait
  // sa colonne joueur selon SES seules lignes (largeurs incohérentes empilées).
  // Calcul déterministe basé sur la longueur de chaîne (pas de mesure DOM fragile) :
  // longueur max en `ch` + réserve fixe pour la flèche d'expansion (▸/▾ + marge) et
  // le padding de cellule. `minWidth` garantit le plancher partagé ; l'ellipsis
  // gère les rares cas où un badge (Bot/MVP/LVP) dépasserait la réserve.
  const playerColWidth = useMemo(() => {
    const maxLen = rows.reduce((m, r) => Math.max(m, (r.gamertag ?? '').length), 0)
    return `${maxLen + 6}ch`
  }, [rows])

  function goToExplorer(gamertag: string, e: React.MouseEvent) {
    if (!playerSlug) return
    e.stopPropagation()
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/explorer',
      params: { titleSlug, playerSlug },
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
          showNativeKillMechanics={showNativeKillMechanics}
          presentKeys={presentKeys}
          playerColWidth={playerColWidth}
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
      {/* Section « Objectifs » (V72-03) : gated capability objective_stats +
          data-driven par mode. Rien affiché hors mode à objectif / titre non supporté. */}
      <MatchObjectivesSection rows={rows} teams={teams} myTeamSide={myTeamSide} t={t} />
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
  /** Capability `native_kill_mechanics` : gate les colonnes assassinat/coup au sol/charge spartane (H5 only). */
  showNativeKillMechanics: boolean
  /** Clés de colonnes avec au moins une valeur sur le lobby (masquage data-driven). */
  presentKeys: Set<string>
  /** Largeur commune de la colonne « Joueur » (ch) — partagée par toutes les tables. */
  playerColWidth: string
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
  showNativeKillMechanics,
  presentKeys,
  playerColWidth,
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
  const locale = useAppShellStore((s) => s.locale)
  // Titre courant : paramètre du chemin d'asset logo `/titles/{slug}/teams/{id}.png`
  // (résolution d'asset, pas une branche de comportement → title-agnostic).
  const slug = useAppShellStore((s) => s.currentTitleSlug)

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
        meta: c?.tooltip ? { headerTooltip: c.tooltip } : undefined,
        // I16 : colonne numérique triable — accessorFn expose la valeur brute
        // (nuls coalescés en `undefined` → toujours en bas, cf. NUMERIC_SORT).
        // Le rendu de cellule reste inchangé (lit row.original directement).
        accessorFn: (r) => (r[key as keyof MatchScoreboardRow] as number | null | undefined) ?? undefined,
        ...NUMERIC_SORT,
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
      meta: { headerTooltip: isRanked ? t.sbColCsrTooltip : t.sbColLusrTooltip },
      // Badge image + libellé de palier : pas de valeur scalaire triable simplement.
      enableSorting: false,
      cell: (ctx) => {
        const url = ctx.row.original.skill_rank?.icon_url
        // Palier baké au sync (FR Infinite / EN H5) → nom localisé, puis
        // « Placement » (sentinelle back) → « En placement », cohérent avec le
        // header (localizeTierLabel + displayTierLabel).
        const label = displayTierLabel(
          localizeTierLabel(ctx.row.original.skill_rank?.tier_label, locale),
          t.rankPlacement,
        )
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
        header: t.sbColPlayer,
        // I16 : tri alpha locale-aware sur le gamertag brut (pas le badge Bot/MVP/LVP).
        accessorFn: (r) => r.gamertag,
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
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
                  title={t.sbViewHistoryFmt(displayGamertag)}
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
        header: t.sbColTopWeapon,
        meta: { headerTooltip: t.sbColTopWeaponTooltip },
        accessorFn: (r) => r.top_weapon_label ?? undefined,
        sortingFn: localeTextSortingFn,
        sortUndefined: 'last',
        sortDescFirst: false,
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
      // Mécaniques natives H5 : gatées par la capability `native_kill_mechanics`. Halo
      // Infinite ne les track pas (stockées 0, pas null → `presentKeys` ne les retire
      // pas) → il FAUT gater ICI aussi : le filtre final conserve toute clé absente de
      // `hlKeys`, donc sans ce gate les colonnes réapparaîtraient. Halo 5 : conservées.
      ...(showNativeKillMechanics ? NATIVE_KILL_MECHANIC_KEYS.map((k) => hlDef(k)) : []),
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
  }, [highlightCols, showNativeKillMechanics, presentKeys, expandedXuid, playerSlug, onPlayerClick, isRanked, t, locale])

  // I16 : tri CLIENT par clic sur les en-têtes (pattern DetectionsPanel minimal).
  // Aucun tri actif par défaut → l'ordre serveur (celui de `rows`) reste affiché
  // tant qu'aucune colonne n'a été cliquée.
  const [sorting, setSorting] = useState<SortingState>([])

  const table = useReactTable<ScoreboardRowVM>({
    data,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  })

  // Résolution du nom d'équipe. Priorité au libellé fourni par le backend (Halo 5 :
  // « Rouge/Bleu » depuis team_colors, déjà localisé côté serveur via team_name) ; à
  // défaut, résolution front des noms officiels Halo Infinite (Eagle / Cobra / ...).
  // Fallback : "Équipe N" si team_id connu mais hors map, "Équipe inconnue" si
  // team_side malformé.
  const backendTeamName = rows.find((r) => r.team_name)?.team_name ?? null
  const officialName = backendTeamName ?? resolveTeamName(teamSide)
  const teamID = parseTeamSideID(teamSide)
  // Un libellé backend déjà complet (Halo 5 : « Équipe Cobra » depuis team_colors
  // localisé) ne doit PAS être re-préfixé par teamLabelFmt, sinon on double le mot
  // (« Équipe Équipe Cobra »). Les noms officiels résolus côté front (Eagle/Cobra)
  // sont nus et attendent le préfixe. Décision sur la DONNÉE, pas sur le slug.
  const teamLabel = officialName
    ? labelHasTeamWord(officialName)
      ? officialName
      : t.teamLabelFmt(officialName)
    : teamID != null
      ? t.teamNumberedFmt(teamID)
      : t.teamUnknown
  // Couleur d'IDENTITÉ de l'équipe, data-driven (jamais slug==) : couleur fournie par
  // le backend (row.team_color, Halo 5 depuis team_colors) en priorité, sinon la map de
  // couleurs officielles par team_id (Halo Infinite : Eagle bleu, Cobra rouge, ...),
  // sinon repli sur le token sémantique ally/enemy existant (overridable par les réglages
  // d'accessibilité). Chaque équipe obtient ainsi sa couleur distincte (> 2 équipes =
  // > 2 couleurs), là où l'ancien schéma ally/enemy n'en offrait que deux.
  const backendTeamColor = rows.find((r) => r.team_color)?.team_color ?? null
  const identityColor = backendTeamColor ?? resolveTeamColorFromID(teamID)
  const teamColorVar = identityColor ?? (isMyTeam ? tokenCssVar('team-ally') : tokenCssVar('team-enemy'))
  // Accent lisible : fond subtil + soulignement + bordure gauche marquée. Le TEXTE reste
  // en `var(--foreground)` (jamais teinté par une couleur d'identité potentiellement vive
  // comme le jaune Valor) pour garantir le contraste.
  const teamHeaderBg = `color-mix(in oklab, ${teamColorVar} 22%, transparent)`
  const teamHeaderBorder = `2px solid color-mix(in oklab, ${teamColorVar} 55%, transparent)`
  const teamHeaderLeftBorder = `4px solid ${teamColorVar}`
  // Logo d'équipe : `/titles/{slug}/teams/{id}.png`. null si slug/team_id absent → pas de
  // logo ; onError masque proprement les team_id sans asset.
  const teamLogoSrc = teamLogoPath(slug, teamID)

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
                borderBottom: teamHeaderBorder,
                borderLeft: teamHeaderLeftBorder,
              }}
              aria-label={isMyTeam ? t.teamMine : t.teamEnemy}
            >
              <span className="inline-flex items-center gap-2 align-middle">
                {teamLogoSrc && (
                  <img
                    src={teamLogoSrc}
                    alt={teamLabel}
                    className="h-6 w-6 shrink-0 object-contain"
                    onError={(e) => {
                      e.currentTarget.style.display = 'none'
                    }}
                  />
                )}
                <span>{teamLabel}</span>
              </span>
            </th>
          </tr>
          {table.getHeaderGroups().map((hg) => (
            <tr key={hg.id} className="text-muted-foreground">
              {hg.headers.map((h) => {
                const isPlayerCol = h.column.id === 'gamertag'
                const align = isPlayerCol ? 'text-left' : 'text-right'
                const canSort = h.column.getCanSort()
                const sortDir = h.column.getIsSorted()
                return (
                  <th
                    key={h.id}
                    className={`border border-border border-b-2 px-2 pb-1 pt-1 ${align} ${canSort ? 'cursor-pointer select-none' : ''}`}
                    style={isPlayerCol ? { width: playerColWidth, minWidth: playerColWidth } : undefined}
                    onClick={canSort ? h.column.getToggleSortingHandler() : undefined}
                    aria-sort={canSort ? ariaSortOf(sortDir) : undefined}
                  >
                    <span className={`inline-flex items-center gap-1 ${isPlayerCol ? '' : 'justify-end'}`}>
                      {flexRender(h.column.columnDef.header, h.getContext())}
                      {canSort ? sortSuffixOf(sortDir) : ''}
                      <HeaderInfoTooltip text={h.column.columnDef.meta?.headerTooltip} />
                    </span>
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
                      ? { ...gamertagCellStyle, width: playerColWidth, minWidth: playerColWidth }
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
