/**
 * ExplorerBriefingModules — modules conditionnels du bandeau de briefing (Lot C).
 *
 * Rendus sous le socle quand l'échantillon est suffisant :
 *   - Dimensions (par carte / mode / playlist) : top/flop avec note (palier 1..5).
 *   - Tendance : sparkline du taux de victoire par bucket.
 *   - Classement : progression de paliers PAR TYPE de rating (CSR / LUSR) +
 *     moyenne par match — gaté useCapability('ranked').
 *
 * Chaque module s'omet proprement si son bloc backend est nil (dégradation par
 * omission, jamais de placeholder vide ni de NaN). Tokens sémantiques uniquement.
 */
import {
  TimeseriesLineChart,
  type ChartPoint2D,
} from '@/components/charts/TimeseriesLineChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import { useCapability } from '@/lib/capabilities/capabilities'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'
import { winRateColor } from '@/lib/colors/outcomePalette'
import { formatPercentInt } from '@/lib/formatters'
import type {
  ExplorerBriefing,
  ExplorerBriefingDimension,
  ExplorerBriefingDimensionEntry,
  ExplorerBriefingRanked,
  ExplorerBriefingRankedKind,
  ExplorerBriefingTrend,
} from '@/lib/api/types'
import type { ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import { BriefingSectionCard } from './BriefingSectionCard'
import { formatSignedFixed, formatSignedPoints, signOf } from './ExplorerBriefing.logic'

type T = (key: ExplorerManifestKey, values?: Record<string, string | number>) => string

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

function deltaToken(v: number | null | undefined): SemanticToken {
  const s = signOf(v)
  return s > 0 ? 'outcome-win' : s < 0 ? 'outcome-loss' : 'outcome-draw'
}

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
  const hasRanked = useCapability('ranked')
  const dimensions = briefing.dimensions ?? []
  const showRanked = hasRanked && briefing.ranked != null
  if (dimensions.length === 0 && briefing.trend == null && !showRanked) return null

  return (
    <div className="space-y-2 pt-1">
      {dimensions.length > 0 && (
        <div className="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
          {dimensions.map((d) => (
            <DimensionCard key={d.dimension} dim={d} t={t} hideDelta={hideDelta} />
          ))}
        </div>
      )}
      {briefing.trend != null && <TrendCard trend={briefing.trend} t={t} />}
      {showRanked && <RankedCard ranked={briefing.ranked as ExplorerBriefingRanked} t={t} />}
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

// ─── Module tendance (C2) ─────────────────────────────────────────────────────

function TrendCard({ trend, t }: { trend: ExplorerBriefingTrend; t: T }) {
  const series: ChartSeries<ChartPoint2D>[] = [
    {
      key: 'win_rate',
      colorToken: 'outcome-win',
      datapoints: (trend.points ?? []).map((p) => ({
        x: p.bucket_start,
        y: Math.round(p.win_rate * 100),
      })),
    },
  ]
  return (
    <TimeseriesLineChart
      title={t('explorer.briefing.trend_title')}
      series={series}
      height={120}
      xAxisType="time"
      outcomeMarkers={false}
      seriesNameResolver={() => t('explorer.briefing.win_rate_label')}
    />
  )
}

// ─── Module « Classement » (C3) : une ligne par type de rating (CSR / LUSR) ────

function RankedCard({ ranked, t }: { ranked: ExplorerBriefingRanked; t: T }) {
  const kinds = ranked.kinds ?? []
  if (kinds.length === 0) return null
  return (
    <BriefingSectionCard className="h-full" title={t('explorer.briefing.ranked_title')}>
      <ul className="space-y-1.5">
        {kinds.map((k) => (
          <RankedKindRow key={k.kind} kind={k} t={t} />
        ))}
      </ul>
    </BriefingSectionCard>
  )
}

// rankedProgression compose « palier début → palier fin » (D-C), en résolvant les
// paliers de placement via clés i18n (D-D : jamais parser le libellé FR). Null si
// aucun palier n'est résolvable (segment omis). Paliers égaux → palier seul.
function rankedProgression(k: ExplorerBriefingRankedKind, t: T): string | null {
  const start = k.tier_start_is_placement
    ? t('explorer.briefing.placement')
    : (k.tier_start_label ?? null)
  const end =
    k.tier_end_placement_remaining != null
      ? t('explorer.briefing.placement_remaining', { n: k.tier_end_placement_remaining })
      : (k.tier_end_label ?? null)
  if (start == null && end == null) return null
  if (start != null && end != null) return start === end ? start : `${start} → ${end}`
  return start ?? end
}

function RankedKindRow({ kind, t }: { kind: ExplorerBriefingRankedKind; t: T }) {
  const progression = rankedProgression(kind, t)
  const perMatch =
    kind.delta_per_match != null
      ? t('explorer.briefing.ranked_per_match', {
          delta: formatSignedFixed(kind.delta_per_match, 1),
        })
      : null
  return (
    <li className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-xs">
      <span className="font-semibold uppercase text-foreground">{kind.kind}</span>
      {progression != null && (
        <>
          <span className="text-muted-foreground">·</span>
          <span className="min-w-0 truncate text-foreground" title={progression}>
            {progression}
          </span>
        </>
      )}
      {perMatch != null && (
        <>
          <span className="text-muted-foreground">·</span>
          <span
            className="shrink-0 tabular-nums"
            style={{ color: tokenCssVar(deltaToken(kind.delta_per_match)) }}
          >
            {perMatch}
          </span>
        </>
      )}
    </li>
  )
}
