/**
 * ExplorerBriefingModules — modules conditionnels du bandeau de briefing (Lot C).
 *
 * Rendus sous le socle quand l'échantillon est suffisant :
 *   - Dimensions (par carte / mode / playlist) : top/flop avec note (palier 1..5).
 *   - Contexte solo/escouade + Moments forts (dominance).
 *
 * Tendance, Classement et Séries ne sont plus des cartes ici : ils vivent dans le
 * socle (sparkline + tuiles, V3 compaction — cf. Strip + ExplorerBriefingTiles).
 * Chaque module s'omet proprement si son bloc backend est nil (dégradation par
 * omission, jamais de placeholder vide ni de NaN). Tokens sémantiques uniquement.
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { kdaNetColor, winRateColor } from '@/lib/colors/outcomePalette'
import { formatPercentInt } from '@/lib/formatters'
import { formatMessage } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'
import type {
  ExplorerBriefing,
  ExplorerBriefingContextGroup,
  ExplorerBriefingContextSplit,
  ExplorerBriefingDimension,
  ExplorerBriefingDimensionEntry,
  ExplorerBriefingDominance,
} from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { matchViewManifest, type MatchViewManifestKey } from '@/lib/i18n/generated/match_view'
import { BriefingSectionCard } from './BriefingSectionCard'
import { deltaToken, formatSignedPoints, signOf } from './ExplorerBriefing.logic'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string
// TMV : résout un libellé du manifest match_view (réutilisé pour les libellés
// dominance narrative.dominance.* — item 9, P-9 : ne pas recréer de clés).
type TMV = (key: MatchViewManifestKey) => string

const DIM_TITLE_KEY: Record<string, ExplorerManifestKey> = {
  map: 'explorer.briefing.dim_map',
  mode: 'explorer.briefing.dim_mode',
  playlist: 'explorer.briefing.dim_playlist',
}

const PERF_TIER_KEY: Record<number, ExplorerManifestKey> = {
  1: 'explorer.filters.perf_tier_excellent',
  2: 'explorer.filters.perf_tier_bon',
  3: 'explorer.filters.perf_tier_correct',
  4: 'explorer.filters.perf_tier_faible',
  5: 'explorer.filters.perf_tier_mauvais',
}

// Catégories de moments forts : chaque compteur de ExplorerBriefingDominance,
// avec son libellé (manifest match_view narrative.dominance.*) et son token
// sémantique — RÉUTILISE le mapping du tableau Explorer (ExplorerMatchesTable
// DOMINANCE_LABEL_KEYS / DOMINANCE_COLOR_TOKENS, item 9 / P-9). L'ordre suit la
// priorité narrative (domination → contre-remontada).
const DOMINANCE_ITEMS: {
  field: keyof ExplorerBriefingDominance
  labelKey: MatchViewManifestKey
  token: SemanticToken
}[] = [
  { field: 'dominations', labelKey: 'narrative.dominance.domination', token: 'narrative-dominant' },
  { field: 'humiliations', labelKey: 'narrative.dominance.humiliation', token: 'narrative-humiliation' },
  { field: 'remontadas', labelKey: 'narrative.dominance.remontada', token: 'narrative-remontada' },
  { field: 'debandades', labelKey: 'narrative.dominance.debandade', token: 'narrative-debacle' },
  {
    field: 'contre_remontadas',
    labelKey: 'narrative.dominance.contre_remontada',
    token: 'narrative-contre-remontada',
  },
]

export function ExplorerBriefingModules({
  briefing,
  t,
  hideDelta,
}: {
  briefing: ExplorerBriefing
  t: T
  // hideDelta : plein historique (scope == baseline) → deltas « vs habituel »
  // nuls par construction, colonne delta des lignes de dimension masquée (P-1).
  hideDelta: boolean
}) {
  const locale = useAppShellStore((s) => s.locale)
  const tMV: TMV = (key) => formatMessage(matchViewManifest, key, locale)
  const dimensions = briefing.dimensions ?? []
  const contextSplit = briefing.context_split ?? null
  // Moments forts : carte omise si aucune catégorie non nulle (item 13).
  const dominance = briefing.dominance ?? null
  const showDominance = dominance != null && DOMINANCE_ITEMS.some((it) => (dominance[it.field] ?? 0) > 0)
  if (dimensions.length === 0 && contextSplit == null && !showDominance) return null

  return (
    <div className="space-y-2 pt-1">
      {/* Rangée « Par… » : cartes de dimension + carte « Par contexte » (4e cellule)
          dans une seule grille responsive (DEC-3 : 1 → 2 → 4 colonnes). */}
      {(dimensions.length > 0 || contextSplit != null) && (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4">
          {dimensions.map((d) => (
            <DimensionCard key={d.dimension} dim={d} t={t} hideDelta={hideDelta} />
          ))}
          {contextSplit != null && <ContextSplitCard split={contextSplit} t={t} />}
        </div>
      )}
      {showDominance && (
        <DominanceBand dominance={dominance as ExplorerBriefingDominance} t={t} tMV={tMV} />
      )}
    </div>
  )
}

// ─── Module dimensions (C1) ──────────────────────────────────────────────────

function DimensionCard({
  dim,
  t,
  hideDelta,
}: {
  dim: ExplorerBriefingDimension
  t: T
  hideDelta: boolean
}) {
  const titleKey = DIM_TITLE_KEY[dim.dimension]
  return (
    <BriefingSectionCard className="h-full" title={titleKey ? t(titleKey) : dim.dimension}>
      <ul className="space-y-1">
        {(dim.entries ?? []).map((e) => (
          <DimensionRow key={e.label} entry={e} t={t} hideDelta={hideDelta} />
        ))}
      </ul>
    </BriefingSectionCard>
  )
}

function DimensionRow({
  entry,
  t,
  hideDelta,
}: {
  entry: ExplorerBriefingDimensionEntry
  t: T
  hideDelta: boolean
}) {
  const wr = entry.win_rate
  const dw = entry.delta_win_rate
  return (
    <li className="flex items-center gap-2 text-xs">
      <span className="min-w-0 flex-1 truncate text-foreground" title={entry.label}>
        {entry.label}
      </span>
      <span className="shrink-0 tabular-nums text-muted-foreground">
        {t('explorer.briefing.dim_matches', { n: entry.matches })}
      </span>
      <span className="w-10 shrink-0 text-right tabular-nums font-semibold" style={{ color: winRateColor(wr) }}>
        {formatPercentInt(wr)}
      </span>
      {/* Colonne delta « vs habituel » masquée en plein historique (P-1). */}
      {!hideDelta && (
        <span
          className="w-16 shrink-0 text-right tabular-nums"
          style={{ color: tokenCssVar(deltaToken(dw)) }}
        >
          {signOf(dw) > 0 ? '▲' : signOf(dw) < 0 ? '▼' : '='} {formatSignedPoints(dw)}
        </span>
      )}
      {entry.note_tier != null ? (
        <span
          className="w-20 shrink-0 rounded border px-1.5 py-0.5 text-center text-3xs font-semibold"
          style={{
            color: tokenCssVar(`perf-tier-${entry.note_tier}` as SemanticToken),
            borderColor: tokenCssVar(`perf-tier-${entry.note_tier}` as SemanticToken),
          }}
        >
          {t(PERF_TIER_KEY[entry.note_tier] ?? 'explorer.filters.perf_tier_correct')}
        </span>
      ) : (
        <span className="w-20 shrink-0 text-center text-3xs text-muted-foreground">—</span>
      )}
    </li>
  )
}

// ─── Module contexte solo/escouade (C4) : une ligne par contexte social ────────
// Rendu uniquement si le bloc backend est présent (les deux sous-groupes ≥ seuil,
// scope multi-contexte — item 6, P-5). Libellés « Solo »/« Escouade » réutilisant
// les clés de filtre existantes ; WR coloré via winRateColor (tokens).

function ContextSplitCard({ split, t }: { split: ExplorerBriefingContextSplit; t: T }) {
  return (
    <BriefingSectionCard className="h-full" title={t('explorer.briefing.context_split_title')}>
      <ul className="space-y-1">
        <ContextSplitRow label={t('explorer.filters.context_solo')} group={split.solo} t={t} />
        <ContextSplitRow label={t('explorer.filters.context_squad')} group={split.squad} t={t} />
      </ul>
    </BriefingSectionCard>
  )
}

function ContextSplitRow({
  label,
  group,
  t,
}: {
  label: string
  group: ExplorerBriefingContextGroup
  t: T
}) {
  return (
    <li className="flex items-center gap-2 text-xs">
      <span className="min-w-0 flex-1 truncate text-foreground" title={label}>
        {label}
      </span>
      <span className="shrink-0 tabular-nums text-muted-foreground">
        {t('explorer.briefing.dim_matches', { n: group.matches })}
      </span>
      <span
        className="w-10 shrink-0 text-right tabular-nums font-semibold"
        style={{ color: winRateColor(group.win_rate) }}
      >
        {formatPercentInt(group.win_rate)}
      </span>
      {/* FDA coloré via kdaNetColor (DP-10) — même convention que la tuile socle FDA. */}
      <span className="w-12 shrink-0 text-right tabular-nums" style={{ color: kdaNetColor(group.kda) }}>
        {group.kda.toFixed(2)}
      </span>
    </li>
  )
}

// ─── Bande « Moments forts » (C6) : compteurs de dominance du scope ─────────────
// Bande NUE (DP-5) : plus de BriefingSectionCard ni d'en-tête de carte — un libellé
// discret muted (highlights_title) suivi de la même rangée de pastilles. Une pastille
// par catégorie NON NULLE (zéros omis, item 13), libellés réutilisant
// narrative.dominance.* (manifest match_view) + tokens narrative-* (P-9).

function DominanceBand({
  dominance,
  t,
  tMV,
}: {
  dominance: ExplorerBriefingDominance
  t: T
  tMV: TMV
}) {
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      <span className="text-2xs uppercase tracking-wide text-muted-foreground">
        {t('explorer.briefing.highlights_title')}
      </span>
      {DOMINANCE_ITEMS.map((it) => {
        const count = dominance[it.field] ?? 0
        if (count <= 0) return null
        const color = tokenCssVar(it.token)
        return (
          <span
            key={it.field}
            className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-2xs font-bold uppercase tracking-wider leading-none whitespace-nowrap"
            style={{
              backgroundColor: `color-mix(in oklab, ${color} 18%, transparent)`,
              borderColor: `color-mix(in oklab, ${color} 55%, transparent)`,
              color,
            }}
          >
            {tMV(it.labelKey)}
            <span className="tabular-nums">×{count}</span>
          </span>
        )
      })}
    </div>
  )
}
