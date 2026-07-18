/**
 * ExplorerMatchesTable — tableau historique mode "Matchs" de l'Explorer.
 *
 * Pattern visuel STRICTEMENT aligné sur features/squad/SquadSynergyHistoryTable.tsx :
 *  - thead : `bg-muted border-b`, th `px-3 py-2 text-left whitespace-nowrap
 *    text-xs font-medium text-muted-foreground border-r border-border last:border-r-0`
 *  - tbody : row `transition-colors hover:bg-primary/10` (pas d'onClick ni
 *    role="button" : ouverture du match exclusivement via l'icône de la
 *    1re colonne pour éviter les ouvertures accidentelles).
 *  - td : `px-3 py-2 whitespace-nowrap border-r border-border last:border-r-0`
 *  - outcome rendu en texte coloré via getOutcomeColor (pas de Badge, pas de tinting)
 *  - bouton "Ouvrir" en début de ligne — seul déclencheur de navigation
 *  - formatDate + formatDurationMinSec pour les colonnes formatées
 *  - useFieldMappings pour libellés map/playlist
 *  - Playlist + Mode tronqués à 12 chars (truncateName) avec tooltip natif
 *    sur le label complet via attribut HTML `title`.
 *
 * Colonnes : Ouvrir | Date | Carte | Playlist | Mode | Contexte | Résultat |
 *            Dominance | K | D | A | FDA | Score | Durée |
 *            Perf (color) | ΔPerf | Rang | MMR équipe | MMR adv. | ΔMMR
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

import type { ExplorerMatchRow } from '@/lib/api/types'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useAppShellStore } from '@/stores/appShellStore'
import { localizeTierLabel, skillTierSortValue } from '@/lib/skillTiers'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { mmrDeltaScale, kdaDivergentScale } from '@/lib/accessibility/scales'
import { getOutcomeColor, outcomeKey } from '@/lib/outcome-color'
import { formatDate, formatDurationMMSS, displayRatingLabel, intlLocale as toIntlLocale } from '@/lib/formatters'
import { useNavigateToMatch } from '@/lib/match-nav/useNavigateToMatch'
import { filterContextToMatchFilterSpec } from '@/lib/match-nav/fromFilterContext'
import type { ContextDescriptor, MatchFilterSpec } from '@/lib/match-nav/navContext'
import { useSoloFilterStore } from '@/stores/soloFilterStore'
import { useProvidesTeamMmr } from '@/lib/damage/effectiveHp'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import {
  NUMERIC_SORT,
  SORT_ARIA_LABEL_KEYS,
  dateTimeSortingFn,
  localeTextSortingFn,
} from './explorerMatchesClientSort'
import {
  columnHighlightStyle,
  computeColumnDeciles,
  explorerHlExtract,
  isExplorerHighlightKey,
} from './ExplorerMatchesTable.highlight'

const PAGE_SIZE = 20
const HISTORY_DATE_OPTS: Intl.DateTimeFormatOptions = {
  day: '2-digit',
  month: '2-digit',
  year: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
}

/** Bannière d'équipe affichée en 1ère ligne du thead (variant ally/enemy).
 *  Couleur pleine (flat) color-mix sur token team-ally / team-enemy,
 *  configurable par les réglages d'accessibilité du joueur. Utilisé en mode
 *  Joueur pour distinguer les tableaux "matchs où la cible était alliée" vs
 *  "ennemie". */
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
  /** filterSpec explicite à propager dans le matchNavContext (Phase 4). Quand
   *  fourni, prime sur la dérivation depuis le soloFilterStore — permet à
   *  l'Explorer de piloter la nav contextuelle depuis SES filtres locaux (et
   *  non le store global, vide en contexte Explorer). Les tables mode Joueur
   *  ne le passent pas → fallback soloFilterStore inchangé. */
  filterSpecOverride?: MatchFilterSpec
  /** Forcer l'affichage du bloc pagination même si rows.length ≤ PAGE_SIZE.
   *  Utile en mode Joueur (tableaux ally/enemy) pour rendre la pagination
   *  toujours visible indépendamment du volume de données. */
  alwaysShowPagination?: boolean
  /** Taille de page initiale (par défaut PAGE_SIZE=20). Quand inférieure à
   *  PAGE_SIZE, un bouton "Voir tout" est affiché pour passer au mode plein
   *  (PAGE_SIZE). Mode Joueur Explorer utilise 10 pour réduire la densité. */
  defaultPageSize?: number
  /** Visibilité des colonnes (état TanStack). Clé = id de colonne (accessorKey,
   *  ou 'open' pour la 1re colonne), valeur `false` = masquée. Permet à un
   *  consommateur (ex. vue comparaison session en panneaux 50%) de réduire le set
   *  de colonnes SANS dupliquer le tableau ni changer le style. Défaut : toutes
   *  visibles (aucun impact sur la page Explorer). */
  columnVisibility?: Record<string, boolean>
  /** Colonnes SUPPLÉMENTAIRES injectées par le consommateur, insérées juste après
   *  la colonne d'id `extraColumnsAfterId` (ou en fin si absent/non trouvé). Permet
   *  à la vue session d'ajouter une colonne « Δ rang » SANS l'exposer sur la page
   *  Explorer (qui ne passe pas cette prop). Défaut : aucune. */
  extraColumns?: ColumnDef<ExplorerMatchRow>[]
  extraColumnsAfterId?: string
  /** Active le tri CLIENT par clic sur les en-tetes, sur TOUTES les colonnes
   *  (getSortedRowModel TanStack). Le tableau possede son propre etat de tri
   *  (defaut : date descendante, comme l'ordre backend). Chaque colonne trie sur
   *  sa valeur SOUS-JACENTE (numerique / timestamp / alpha / champ brut), jamais
   *  le libelle formate. Reserve au mode Matchs (toutes les lignes du scope sont
   *  chargees d'un coup, cap 10000). Cas extreme : si un scope depasse 10000
   *  matchs, seules les 10000 plus recentes sont chargees donc triees (le compteur
   *  affiche le vrai total) — aucune regression vs l'existant, non corrige.
   *  Absent/false (mode Joueur ally/ennemi, vue session) → en-tetes statiques,
   *  aucun tri (comportement inchange). */
  sortable?: boolean
  /** Contenu injecté à GAUCHE du pied de tableau, à la place du compteur
   *  « N matchs trouvés » (fallback). Quand fourni : le pied devient visible même
   *  sans pagination (permet à l'Explorer d'y ancrer le bouton Export CSV, visible
   *  aussi pour les scopes ≤ PAGE_SIZE). Les autres consommateurs (mode Joueur
   *  ally/ennemi, vue session) ne le passent pas → compteur conservé, pied visible
   *  ssi pagination (rendu inchangé). */
  footerLeadingSlot?: ReactNode
}

/** Classe des cellules d'en-tete (partagee entre en-tetes statiques et triables).
 *  SANS `text-align` : l'alignement est appliqué PAR COLONNE via `alignClass`
 *  (numériques à droite, texte à gauche — DEC-ALIGN). */
const HEADER_TH_CLASS =
  'px-2 py-1 whitespace-nowrap text-3xs font-medium text-muted-foreground border-r border-border last:border-r-0'

/** Colonnes NUMÉRIQUES (par id TanStack) alignées à DROITE (en-tête ET cellules).
 *  Les autres colonnes (texte : date, carte, playlist, mode, contexte, résultat,
 *  dominance, rang, note) restent à gauche. Jamais centré (DEC-ALIGN). */
const RIGHT_ALIGNED_COLUMNS = new Set<string>([
  'kills',
  'deaths',
  'assists',
  'kda',
  'score_label',
  'duration_seconds',
  'perf_score',
  'delta_perf',
  'team_mmr',
  'enemy_mmr',
  'delta_mmr',
])

/** Classe d'alignement horizontal d'une colonne selon son id (droite si
 *  numérique, gauche sinon). */
function alignClass(colId: string): string {
  return RIGHT_ALIGNED_COLUMNS.has(colId) ? 'text-right' : 'text-left'
}

/** Indicateur de tri de la colonne active : ▲ ascendant / ▼ descendant. Rien sur
 *  les colonnes inactives (l'affordance clic vient du <button> + hover). Tokens
 *  semantiques uniquement, aucune couleur. */
function SortIndicator({ dir }: { dir: false | 'asc' | 'desc' }) {
  if (!dir) return null
  return (
    <span aria-hidden="true" className="text-2xs leading-none">
      {dir === 'asc' ? '▲' : '▼'}
    </span>
  )
}

/** id d'une colonne TanStack (accessorKey ou id explicite type 'open'). */
function columnIdOf(c: ColumnDef<ExplorerMatchRow>): string | undefined {
  return (c as { id?: string }).id ?? (c as { accessorKey?: string }).accessorKey
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

// Mapping flag DominanceFlag (Go canonical.DominanceFlag) → clé i18n
// match_view manifest. 0/undefined = pas de badge → "-" en cellule.
const DOMINANCE_LABEL_KEYS: Record<number, MatchViewManifestKey> = {
  1: 'narrative.dominance.domination',
  2: 'narrative.dominance.humiliation',
  3: 'narrative.dominance.remontada',
  4: 'narrative.dominance.debandade',
  5: 'narrative.dominance.contre_remontada',
}

// Mapping flag → SemanticToken frontend (cf. semantic-tokens.ts).
const DOMINANCE_COLOR_TOKENS: Record<number, SemanticToken> = {
  1: 'narrative-dominant',
  2: 'narrative-humiliation',
  3: 'narrative-remontada',
  4: 'narrative-debacle',
  5: 'narrative-contre-remontada',
}

const NAME_TRUNCATE_MAX = 12

// Tronque un libellé à NAME_TRUNCATE_MAX caractères en remplaçant le dernier
// par "..." (3 points). Si le libellé est plus court ou vide, retourne tel quel.
function truncateName(s: string | null | undefined): string {
  if (!s) return '-'
  if (s.length <= NAME_TRUNCATE_MAX) return s
  return s.slice(0, NAME_TRUNCATE_MAX - 1) + '...'
}

export function ExplorerMatchesTable({ rows, playerSlug, teamBanner, contextDescriptor, filterSpecOverride, alwaysShowPagination, defaultPageSize, columnVisibility, extraColumns, extraColumnsAfterId, sortable, footerLeadingSlot }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey, values?: Record<string, string | number>) =>
    formatMessage(explorerManifest, key, locale, values)
  const tMV = (key: MatchViewManifestKey, values?: Record<string, string | number>) =>
    formatMessage(matchViewManifest, key, locale, values)
  const intlLocale = toIntlLocale(locale)

  const { data: mappings } = useFieldMappings()
  const mapAssets = mappings?.assets?.['map']
  const playlistAssets = mappings?.assets?.['playlist']
  const labelOfMap = (mapUI: string) => mapAssets?.[mapUI]?.label ?? mapUI
  const labelOfPlaylist = (p?: string | null) => (p ? (playlistAssets?.[p]?.label ?? p) : '-')

  // Title-agnostic : les 3 colonnes MMR n'ont de sens que si le titre courant
  // fournit un MMR par match (gating via capability, jamais via slug). Faux pour
  // Halo 5 → on retire les colonnes MMR équipe / MMR adv. / ΔMMR.
  const providesTeamMmr = useProvidesTeamMmr()
  // KDA NET ((k+a/3)−d) pour les deux titres (valeur API Infinite / FDA Halo 5),
  // potentiellement négatif → échelle divergente autour de 0 (jamais kdScale).

  const navigateToMatch = useNavigateToMatch(playerSlug)
  const filterContext = useSoloFilterStore((s) => s.filterContext)
  const allMatchIds = useMemo(() => rows.map((r) => r.match_id), [rows])
  // Seuils de décile (p10/p90) MVP/LVP par colonne, calculés sur TOUT le scope
  // chargé (`rows`, pas la page visible) → indépendants du tri et de la
  // pagination (DEC-DECILE). Mémoïsés.
  const highlightDeciles = useMemo(() => computeColumnDeciles(rows), [rows])
  // useCallback obligatoire : goToMatch est référencé dans le cell renderer du
  // useMemo `columns` ci-dessous. Sans ça, le closure figé au 1er render garde
  // les `allMatchIds` initiaux (pré-filtre) → nav contextuelle hors scope.
  const goToMatch = useCallback(
    (matchId: string) => {
      // Phase 4 : filterSpecOverride (filtres Explorer locaux) prime sur la
      // dérivation soloFilterStore (vide en contexte Explorer).
      const filterSpec =
        filterSpecOverride ?? filterContextToMatchFilterSpec(filterContext) ?? undefined
      navigateToMatch(matchId, {
        source: 'explorer',
        matchIds: allMatchIds,
        filterSpec,
        contextDescriptor,
      })
    },
    [navigateToMatch, allMatchIds, filterContext, contextDescriptor, filterSpecOverride],
  )

  // Labels outcome (pas de Badge, juste texte coloré comme Squad)
  const outcomeLabels: Record<'win' | 'loss' | 'draw' | 'dnf', string> = {
    win: t('explorer.matches.outcome_win'),
    loss: t('explorer.matches.outcome_loss'),
    draw: t('explorer.matches.outcome_draw'),
    dnf: t('explorer.matches.outcome_dnf'),
  }

  const baseColumns = useMemo<ColumnDef<ExplorerMatchRow>[]>(
    () => [
      {
        id: 'open',
        header: '',
        cell: (ctx) => (
          <button
            type="button"
            className="group flex items-center justify-center text-muted-foreground hover:text-foreground transition-colors"
            onClick={() => goToMatch(ctx.row.original.match_id)}
            aria-label={t('explorer.matches.col_open')}
          >
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16" fill="currentColor" className="h-3.5 w-3.5 opacity-50 group-hover:opacity-100 transition-opacity" aria-hidden="true">
              <path d="M6.22 8.72a.75.75 0 0 0 1.06 1.06l5.22-5.22v1.69a.75.75 0 0 0 1.5 0v-3.5a.75.75 0 0 0-.75-.75h-3.5a.75.75 0 0 0 0 1.5h1.69L6.22 8.72Z" />
              <path d="M3.5 6.75c0-.69.56-1.25 1.25-1.25H7A.75.75 0 0 0 7 4H4.75A2.75 2.75 0 0 0 2 6.75v4.5A2.75 2.75 0 0 0 4.75 14h4.5A2.75 2.75 0 0 0 12 11.25V9a.75.75 0 0 0-1.5 0v2.25c0 .69-.56 1.25-1.25 1.25h-4.5c-.69 0-1.25-.56-1.25-1.25v-4.5Z" />
            </svg>
          </button>
        ),
      },
      {
        accessorKey: 'start_time',
        header: t('explorer.matches.col_date'),
        // Tri chronologique sur le timestamp brut (pas le libelle formate).
        sortingFn: dateTimeSortingFn,
        cell: (ctx) => (
          <span className="text-muted-foreground">
            {formatDate(ctx.getValue<string>(), intlLocale, HISTORY_DATE_OPTS)}
          </span>
        ),
      },
      {
        accessorKey: 'map_ui',
        header: t('explorer.filters.map'),
        // Colonnes texte : tri alpha locale-aware, ascendant au 1er clic.
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => labelOfMap(ctx.getValue<string>()),
      },
      {
        accessorKey: 'playlist_label',
        header: t('explorer.filters.playlist'),
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => {
          const full = labelOfPlaylist(ctx.getValue<string | null | undefined>())
          return (
            <span className="text-muted-foreground" title={full}>
              {truncateName(full)}
            </span>
          )
        },
      },
      {
        accessorKey: 'mode_ui',
        header: t('explorer.filters.mode'),
        sortingFn: localeTextSortingFn,
        sortDescFirst: false,
        cell: (ctx) => {
          const full = ctx.getValue<string | null | undefined>() ?? '-'
          return (
            <span className="text-muted-foreground" title={full}>
              {truncateName(full)}
            </span>
          )
        },
      },
      {
        accessorKey: 'is_with_friends',
        header: t('explorer.matches.col_squad'),
        // Contexte Solo/Escouade : tri booleen naturel (Solo puis Escouade en asc).
        sortingFn: 'basic',
        // Pastille reprise du style match-card.tsx (tuiles match home).
        // Les couleurs hex sont autorisées (color-allow) car identifiants UX
        // génériques de catégorie, pas de palette accessibility.
        // Pill "bot" affichée en plus quand had_bot_teammate=true (Career
        // best_matches uniquement, cf. backend asymétrie WIN/LOSS) — couleur
        // amber tolérée car badge d'état système, pas signification métier.
        cell: (ctx) => {
          const row = ctx.row.original
          const isSquad = ctx.getValue<boolean>()
          return (
            <span className="inline-flex items-center gap-1">
              <span
                className="rounded-full px-2 py-0.5 text-2xs font-bold uppercase tracking-wider leading-none"
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
              {row.had_bot_teammate && (
                <span
                  className="rounded-full px-2 py-0.5 text-2xs font-bold lowercase tracking-wider leading-none"
                  style={{ backgroundColor: 'rgba(245,158,11,0.15)', color: '#f59e0b' }} // color-allow: amber pour badge d'état système
                  title={t('explorer.matches.bot_pill_tooltip')}
                >
                  {t('explorer.matches.bot_pill')}
                </span>
              )}
            </span>
          )
        },
      },
      {
        accessorKey: 'outcome_code',
        header: t('explorer.matches.col_outcome'),
        // Tri sur le code brut (2=V, 3=D, 1=N, 4=DNF), jamais le libelle traduit.
        sortingFn: 'basic',
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
        // Tri sur le flag brut (0..5). Les matchs sans badge (0 ou absent) sont
        // coalesces en `undefined` → toujours ranges en bas (sortUndefined:'last').
        accessorFn: (r) => r.dominance_flag || undefined,
        id: 'dominance_flag',
        header: t('explorer.matches.col_dominance'),
        ...NUMERIC_SORT,
        cell: (ctx) => {
          const flag = ctx.getValue<number | null | undefined>() ?? 0
          const labelKey = DOMINANCE_LABEL_KEYS[flag]
          const colorToken = DOMINANCE_COLOR_TOKENS[flag]
          if (!labelKey || !colorToken) return <span className="text-muted-foreground">-</span>
          const color = tokenCssVar(colorToken)
          return (
            <span
              className="inline-flex items-center rounded-full border px-2 py-0.5 text-2xs font-bold uppercase tracking-wider leading-none whitespace-nowrap"
              style={{
                backgroundColor: `color-mix(in oklab, ${color} 18%, transparent)`,
                borderColor: `color-mix(in oklab, ${color} 55%, transparent)`,
                color,
              }}
            >
              {tMV(labelKey)}
            </span>
          )
        },
      },
      {
        accessorFn: (r) => r.kills ?? undefined,
        id: 'kills',
        header: () => (
          <span title={t('explorer.matches.col_kills_long')}>
            {t('explorer.matches.col_kills')}
          </span>
        ),
        ...NUMERIC_SORT,
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorFn: (r) => r.deaths ?? undefined,
        id: 'deaths',
        header: () => (
          <span title={t('explorer.matches.col_deaths_long')}>
            {t('explorer.matches.col_deaths')}
          </span>
        ),
        ...NUMERIC_SORT,
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorFn: (r) => r.assists ?? undefined,
        id: 'assists',
        header: () => (
          <span title={t('explorer.matches.col_assists_long')}>
            {t('explorer.matches.col_assists')}
          </span>
        ),
        ...NUMERIC_SORT,
        cell: (ctx) => (
          <span className="font-mono tabular-nums">
            {ctx.getValue<number | null | undefined>() ?? '-'}
          </span>
        ),
      },
      {
        accessorFn: (r) => r.kda ?? undefined,
        id: 'kda',
        // KDA net signé (peut être négatif) pour les deux titres → tooltip formule.
        header: () => (
          <span title={t('explorer.matches.col_kda_tooltip')}>
            {t('explorer.matches.col_kda')}
          </span>
        ),
        ...NUMERIC_SORT,
        cell: (ctx) => {
          const v = ctx.getValue<number | null | undefined>()
          if (v == null) return '-'
          // KDA net signé (les deux titres) → échelle divergente autour de 0.
          const colorToken = kdaDivergentScale(v)
          return (
            <span
              className="font-mono tabular-nums font-semibold"
              style={{ color: tokenCssVar(colorToken) }}
            >
              {fmtKDA(v)}
            </span>
          )
        },
      },
      {
        accessorKey: 'score_label',
        header: t('explorer.matches.col_score'),
        // Score « 50-30 » : tri alphanumerique naturel (compare 50 vs 100 en
        // nombres) sur la valeur brute — gere aussi les scores mono-valeur.
        sortingFn: 'alphanumeric',
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono">
            {ctx.getValue<string | undefined>() || '-'}
          </span>
        ),
      },
      {
        accessorFn: (r) => r.duration_seconds ?? undefined,
        id: 'duration_seconds',
        header: t('explorer.matches.col_duration'),
        ...NUMERIC_SORT,
        cell: (ctx) => (
          <span className="text-muted-foreground font-mono tabular-nums">
            {formatDurationMMSS(ctx.getValue<number | null | undefined>() ?? undefined)}
          </span>
        ),
      },
      {
        accessorFn: (r) => r.perf_score ?? undefined,
        id: 'perf_score',
        header: t('explorer.matches.col_perf'),
        ...NUMERIC_SORT,
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
        accessorFn: (r) => r.delta_perf ?? undefined,
        id: 'delta_perf',
        header: t('explorer.matches.col_delta_perf'),
        ...NUMERIC_SORT,
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
        // Famille de note (« CSR » / « LUSR ») : tri alpha sur la valeur brute,
        // nuls (PvE/Custom) coalesces en undefined → en bas.
        accessorFn: (r) => r.rating_type ?? undefined,
        id: 'rating_type',
        header: t('explorer.matches.col_rating'),
        sortingFn: localeTextSortingFn,
        sortUndefined: 'last',
        sortDescFirst: false,
        cell: (ctx) => {
          // Normalisé : la famille LUSR (dont la row d'audit 'LUSR_V2') s'affiche
          // 'LUSR' — la v2 est transparente pour l'utilisateur (cf. displayRatingLabel).
          const v = displayRatingLabel(ctx.getValue<string | null | undefined>())
          return <span className="font-mono">{v ?? '-'}</span>
        },
      },
      {
        // Rang : aucun entier de palier dans ExplorerMatchRow → on trie sur un
        // ordinal derive du libelle (Bronze < … < Onyx < Champion), pas l'alpha.
        // Placement en cours / palier inconnu → undefined → en bas. La cellule lit
        // row.original (l'accessor renvoie un nombre, plus le libelle).
        accessorFn: (r) => skillTierSortValue(r.skill_tier_label),
        id: 'skill_tier_label',
        header: t('explorer.matches.col_rank'),
        ...NUMERIC_SORT,
        cell: (ctx) => {
          const r = ctx.row.original
          if (r.placement_done != null && r.placement_total != null) {
            return <span className="font-mono">{r.placement_done}/{r.placement_total}</span>
          }
          return localizeTierLabel(r.skill_tier_label, locale) ?? '-'
        },
      },
      // Colonnes MMR (équipe / adverse / Δ) : incluses uniquement si le titre
      // courant fournit le MMR par match (Halo 5 → masquées).
      ...(providesTeamMmr
        ? [
            {
              accessorFn: (r) => r.team_mmr ?? undefined,
              id: 'team_mmr',
              header: () => renderTwoLineHeader(t('explorer.matches.col_team_mmr')),
              ...NUMERIC_SORT,
              cell: (ctx) => (
                <span className="text-muted-foreground font-mono tabular-nums">
                  {fmtMmr(ctx.getValue<number | null | undefined>())}
                </span>
              ),
            } as ColumnDef<ExplorerMatchRow>,
            {
              accessorFn: (r) => r.enemy_mmr ?? undefined,
              id: 'enemy_mmr',
              header: () => renderTwoLineHeader(t('explorer.matches.col_enemy_mmr')),
              ...NUMERIC_SORT,
              cell: (ctx) => (
                <span className="text-muted-foreground font-mono tabular-nums">
                  {fmtMmr(ctx.getValue<number | null | undefined>())}
                </span>
              ),
            } as ColumnDef<ExplorerMatchRow>,
            {
              accessorFn: (r) => r.delta_mmr ?? undefined,
              id: 'delta_mmr',
              header: t('explorer.matches.col_delta_mmr'),
              ...NUMERIC_SORT,
              cell: (ctx) => fmtDeltaMMR(ctx.getValue<number | null | undefined>()),
            } as ColumnDef<ExplorerMatchRow>,
          ]
        : []),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [intlLocale, mapAssets, playlistAssets, locale, goToMatch, providesTeamMmr],
  )

  // Tri CLIENT possédé par le tableau (mode Matchs). Défaut : date descendante —
  // reproduit l'ordre backend (les 10000 plus récents) au 1er rendu.
  const [sorting, setSorting] = useState<SortingState>([{ id: 'start_time', desc: true }])

  // Insère les colonnes injectées par le consommateur après `extraColumnsAfterId`
  // (ou en fin), puis active le tri sur TOUTES les colonnes de données quand
  // `sortable` (mode Matchs) ; la colonne d'ouverture reste non triable. Les
  // autres consommateurs (mode Joueur ally/ennemi, vue session) ne passent pas
  // `sortable` → colonnes non triables, en-têtes statiques (comportement inchangé).
  const columns = useMemo<ColumnDef<ExplorerMatchRow>[]>(() => {
    const merged = (() => {
      if (!extraColumns?.length) return baseColumns
      const idx = extraColumnsAfterId
        ? baseColumns.findIndex((c) => columnIdOf(c) === extraColumnsAfterId)
        : -1
      if (idx === -1) return [...baseColumns, ...extraColumns]
      return [...baseColumns.slice(0, idx + 1), ...extraColumns, ...baseColumns.slice(idx + 1)]
    })()
    return merged.map((c) => ({
      ...c,
      enableSorting: sortable === true && columnIdOf(c) !== 'open',
    }))
  }, [baseColumns, extraColumns, extraColumnsAfterId, sortable])

  // Pagination simple : taille de page fixe (defaultPageSize en mode Joueur,
  // sinon PAGE_SIZE=20). La navigation par page suffit pour parcourir tous les
  // matchs — pas d'expander redondant (cf. retour user : 2 contrôles = confusion).
  const [pagination, setPagination] = useState({
    pageIndex: 0,
    pageSize: defaultPageSize ?? PAGE_SIZE,
  })

  const table = useReactTable<ExplorerMatchRow>({
    data: rows,
    columns,
    // columnVisibility piloté par le parent (prop) : pas de toggle interne donc pas
    // de onColumnVisibilityChange. Défaut {} = toutes visibles.
    state: { pagination, columnVisibility: columnVisibility ?? {}, sorting },
    onPaginationChange: setPagination,
    // Tri CLIENT : toutes les lignes du scope sont chargées (cap 10000) et
    // paginées côté client, donc getSortedRowModel trie localement TOUTES les
    // colonnes — instantané. Défaut numérique/date = descendant au 1er clic
    // (colonnes texte : sortDescFirst:false). enableSortingRemoval:false garde un
    // tri toujours actif. Le tri n'est effectif que si des colonnes sont triables
    // (sortable=true) ; sinon getSortedRowModel est un no-op (ordre backend intact).
    onSortingChange: setSorting,
    enableSortingRemoval: false,
    sortDescFirst: true,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
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
  const showPagination = (alwaysShowPagination && rows.length > 0) || rows.length > PAGE_SIZE

  // Bannière d'équipe (variant ally/enemy) : couleur pleine (flat) sur token
  // team-ally/team-enemy, configurable via réglages accessibilité.
  const bannerColCount = table.getVisibleLeafColumns().length
  const bannerStyle = teamBanner
    ? (() => {
        const teamColor = tokenCssVar(teamBanner.variant === 'ally' ? 'team-ally' : 'team-enemy')
        return {
          background: `color-mix(in oklab, ${teamColor} 35%, transparent)`,
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
                {hg.headers.map((h) => {
                  const content = h.isPlaceholder
                    ? null
                    : flexRender(h.column.columnDef.header, h.getContext())
                  // Alignement par colonne (DEC-ALIGN) : numériques à droite,
                  // texte à gauche.
                  const alignRight = RIGHT_ALIGNED_COLUMNS.has(h.column.id)
                  // Colonne non triable (tableau sans `sortable`, ou colonne
                  // d'ouverture) : en-tête statique.
                  if (!h.column.getCanSort()) {
                    return (
                      <th key={h.id} className={`${HEADER_TH_CLASS} ${alignClass(h.column.id)}`}>
                        {content}
                      </th>
                    )
                  }
                  const sortDir = h.column.getIsSorted() // false | 'asc' | 'desc'
                  const ariaSort =
                    sortDir === 'asc' ? 'ascending' : sortDir === 'desc' ? 'descending' : 'none'
                  const labelKey = SORT_ARIA_LABEL_KEYS[h.column.id]
                  return (
                    <th key={h.id} className={`${HEADER_TH_CLASS} ${alignClass(h.column.id)}`} aria-sort={ariaSort}>
                      <button
                        type="button"
                        onClick={h.column.getToggleSortingHandler()}
                        aria-label={labelKey ? t('explorer.matches.sort_by', { col: t(labelKey) }) : undefined}
                        className={
                          // Colonne numérique : bouton pleine largeur poussant
                          // libellé + flèche de tri à DROITE ; texte : à gauche.
                          (alignRight
                            ? 'group flex w-full items-center justify-end gap-1'
                            : 'group inline-flex items-center gap-1') +
                          ' whitespace-nowrap transition-colors hover:text-foreground' +
                          (sortDir ? ' text-foreground' : '')
                        }
                      >
                        {content}
                        <SortIndicator dir={sortDir} />
                      </button>
                    </th>
                  )
                })}
              </tr>
            ))}
          </thead>
          <tbody className="divide-y divide-border">
            {table.getRowModel().rows.map((row) => (
              <tr key={row.id} className="transition-colors hover:bg-primary/10">
                {row.getVisibleCells().map((cell) => {
                  // MVP/LVP : bande de décile (top 10 % / pire 10 %) par colonne
                  // clé surlignée sur tout le scope (style best/worst IMPORTÉ de
                  // MatchScoreboard.logic via le sibling highlight, teinte douce).
                  // Colonnes non clés / hors bande / neutres → {} (no-op).
                  const colId = cell.column.id
                  const hlStyle = isExplorerHighlightKey(colId)
                    ? columnHighlightStyle(colId, explorerHlExtract[colId](row.original), highlightDeciles)
                    : undefined
                  return (
                    <td
                      key={cell.id}
                      className={`px-2 py-1 whitespace-nowrap border-r border-border last:border-r-0 ${alignClass(colId)}`}
                      style={hlStyle}
                    >
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </td>
                  )
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Pied de tableau. GAUCHE : `footerLeadingSlot` injecté par le consommateur
          (Explorer → bouton Export CSV) ou, à défaut, le compteur « N matchs trouvés ».
          DROITE : contrôles de pagination, uniquement si pagination requise. Le pied
          s'affiche dès qu'il y a de la pagination OU un slot à rendre (le CSV reste
          donc visible même sans pagination, scopes ≤ PAGE_SIZE). */}
      {(showPagination || footerLeadingSlot != null) && (
        <div className="flex items-center justify-between gap-2 text-xs text-muted-foreground">
          {footerLeadingSlot ?? <span>{t('explorer.matches.count_label', { n: rows.length })}</span>}
          {showPagination && (
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
          )}
        </div>
      )}

    </div>
  )
}
