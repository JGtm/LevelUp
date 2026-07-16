/**
 * ExplorerBriefingModules — modules conditionnels du bandeau de briefing (Lot C).
 *
 * Rendus sous le socle quand l'échantillon est suffisant :
 *   - Dimensions (par carte / mode / playlist) : top/flop avec note (palier 1..5).
 *   - Tendance : sparkline du taux de victoire par bucket.
 *   - Classé : delta CSR cumulé + attendu vs réel — gaté useCapability('ranked').
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

// ─── Module classé (C3) ───────────────────────────────────────────────────────

function RankedCard({ ranked, t }: { ranked: ExplorerBriefingRanked; t: T }) {
  const hasDelta = (ranked.rating_kind ?? '') !== ''
  const hasPrediction = (ranked.matches_with_prediction ?? 0) > 0
  return (
    <BriefingSectionCard className="h-full" title={t('explorer.briefing.ranked_title')}>
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2">
        {hasDelta && (
          <div>
            <p className="text-3xs uppercase tracking-wide text-muted-foreground">
              {t('explorer.briefing.ranked_delta')}
            </p>
            <p
              className="text-lg font-bold tabular-nums"
              style={{ color: tokenCssVar(deltaToken(ranked.delta_sum)) }}
            >
              {formatSignedFixed(ranked.delta_sum, 0)}
            </p>
          </div>
        )}
        {hasPrediction && (
          <div>
            <p className="text-3xs uppercase tracking-wide text-muted-foreground">
              {t('explorer.briefing.ranked_expected_vs_actual')}
            </p>
            <p className="text-sm tabular-nums text-foreground">
              {t('explorer.briefing.ranked_expected')} {formatPercentInt(ranked.expected_win_rate)}
              {' · '}
              {t('explorer.briefing.ranked_actual')}{' '}
              <span
                className="font-semibold"
                style={{ color: winRateColor(ranked.actual_win_rate) }}
              >
                {formatPercentInt(ranked.actual_win_rate)}
              </span>
            </p>
          </div>
        )}
      </div>
    </BriefingSectionCard>
  )
}
