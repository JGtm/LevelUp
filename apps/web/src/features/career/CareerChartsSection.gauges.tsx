/**
 * CareerChartsSection — jauges career.01 (rang) + career.02 (Héros).
 *
 * Découpé depuis CareerChartsSection.tsx (audit #6 god-file split).
 * Sortie : 2 ChartCard<GaugePoint> avec leur footer enrichi.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import {
  getEChartsThemeColors,
  getTooltipBase,
  CHART_BG,
} from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import type { CareerSummary, HeroProgress } from '@/lib/api/types'

// ── Constantes métier (Héros) ──────────────────────────────────────────────
const HERO_XP_TOTAL = 9_319_350
const HERO_RANK_TOTAL_FALLBACK = 272

interface GaugePoint { value: number; label: string; detail: string }

// ── career.01 — jauge rang ─────────────────────────────────────────────────

function rankGaugeSeries(summary: CareerSummary, numLoc: string): ChartSeries<GaugePoint> {
  // progress_pct est déjà en 0..100 côté API Go
  const pct = Math.min(100, summary.progress_pct)
  return {
    key: 'career.gauge.rank',
    datapoints: [{
      value: pct,
      label: summary.rank_label,
      detail: `${summary.current_xp.toLocaleString(numLoc)} / ${summary.xp_for_next_rank.toLocaleString(numLoc)} XP`,
    }],
  }
}

function buildRankGaugeOption(series: ChartSeries<GaugePoint>[]): EChartsCoreOption {
  const point = series[0]?.datapoints[0]
  if (!point) return {}
  return buildGaugeOption(point, resolveToken('chart-series-1'), getEChartsThemeColors())
}

// ── career.02 — jauge Héros ────────────────────────────────────────────────

function heroGaugeSeries(hero: HeroProgress, numLoc: string): ChartSeries<GaugePoint> {
  // percentage est déjà en 0..100 côté API Go (× 100 fait côté service)
  const pct = Math.min(100, hero.percentage)
  const acquired = hero.xp_total_required - hero.xp_remaining
  return {
    key: 'career.gauge.hero',
    datapoints: [{
      value: pct,
      label: 'Progression vers Héros',
      detail: `${acquired.toLocaleString(numLoc)} / ${HERO_XP_TOTAL.toLocaleString(numLoc)} XP`,
    }],
  }
}

function buildHeroGaugeOption(series: ChartSeries<GaugePoint>[]): EChartsCoreOption {
  const point = series[0]?.datapoints[0]
  if (!point) return {}
  return buildGaugeOption(point, resolveToken('perf-tier-2'), getEChartsThemeColors())
}

// ── Gauge builder partagé ──────────────────────────────────────────────────

function buildGaugeOption(
  point: GaugePoint,
  progressColor: string,
  tc: ReturnType<typeof getEChartsThemeColors>,
): EChartsCoreOption {
  const trackColors: [number, string][] = [
    [0.25, resolveToken('perf-tier-1') + '30'],
    [0.5,  resolveToken('perf-tier-2') + '30'],
    [0.75, resolveToken('perf-tier-3') + '30'],
    [1,    resolveToken('perf-tier-4') + '30'],
  ]
  return {
    backgroundColor: CHART_BG,
    tooltip: {
      formatter: () => `${point.label}<br/>${point.detail}<br/>${Math.round(point.value)}%`,
      ...getTooltipBase(tc),
    },
    title: {
      text: point.label,
      subtext: point.detail,
      left: 'center',
      top: '82%',
      textStyle: { color: tc.text, fontSize: 13 },
      subtextStyle: { color: tc.axisLabel, fontSize: 11 },
    },
    series: [{
      type: 'gauge',
      min: 0, max: 100,
      startAngle: 215, endAngle: -35,
      radius: '88%', center: ['50%', '58%'],
      data: [{ value: point.value, name: '' }],
      axisLine: { lineStyle: { width: 16, color: trackColors } },
      progress: { show: true, width: 16, roundCap: false, itemStyle: { color: progressColor } },
      pointer: { show: true, length: '60%', width: 4, itemStyle: { color: tc.text } },
      axisTick: { show: false },
      splitLine: { show: true, distance: 0, length: 5, lineStyle: { color: tc.splitLine, width: 1 } },
      axisLabel: { show: true, distance: 20, color: tc.axisLabel, fontSize: 9, formatter: (v: number) => Math.round(v).toString() },
      title: { show: false },
      detail: { show: true, offsetCenter: [0, '22%'], color: tc.text, fontSize: 28, fontWeight: 'bold', formatter: (v: number) => `${Math.round(v)}%` },
    }],
  }
}

// ── Footer enrichi sous chaque jauge ──────────────────────────────────────

function RankGaugeFooter({
  summary,
  locale,
  intlLocale,
}: {
  summary: CareerSummary
  locale: ManifestLocale
  intlLocale: string
}) {
  const nextRankName =
    locale === 'fr' ? summary.next_rank_name_fr : summary.next_rank_name_en
  return (
    <div className="flex flex-col items-center gap-2 border-t border-border px-3 py-3">
      <div className="text-center">
        <div className="text-xs text-muted-foreground">
          {careerManifest['career.summary.xp_next_rank'][locale]}
        </div>
        <div className="text-base font-semibold">
          {summary.xp_for_next_rank.toLocaleString(intlLocale)}
        </div>
      </div>
      {(summary.rank_image_url || nextRankName || summary.next_rank_image_url) && (
        <div className="flex items-center gap-3 text-xs">
          {summary.rank_image_url && (
            <img
              src={summary.rank_image_url}
              alt={summary.rank_label}
              className="h-12 w-12 object-contain"
              loading="lazy"
              decoding="async"
              onError={(e) => {
                e.currentTarget.style.display = 'none'
              }}
            />
          )}
          <span className="text-muted-foreground">{summary.rank_label}</span>
          {(nextRankName || summary.next_rank_image_url) && (
            <>
              <span className="text-muted-foreground" aria-hidden="true">→</span>
              {summary.next_rank_image_url && (
                <img
                  src={summary.next_rank_image_url}
                  alt={nextRankName ?? ''}
                  className="h-12 w-12 object-contain"
                  loading="lazy"
                  decoding="async"
                  onError={(e) => {
                    e.currentTarget.style.display = 'none'
                  }}
                />
              )}
              {nextRankName && (
                <span className="text-muted-foreground">{nextRankName}</span>
              )}
            </>
          )}
        </div>
      )}
    </div>
  )
}

function HeroGaugeFooter({
  hero,
  locale,
  intlLocale,
}: {
  hero: HeroProgress
  locale: ManifestLocale
  intlLocale: string
}) {
  const totalRanks = hero.total_ranks ?? HERO_RANK_TOTAL_FALLBACK
  return (
    <div className="grid grid-cols-2 gap-4 border-t border-border px-3 py-3 text-center">
      <div>
        <div className="text-xs text-muted-foreground">
          {careerManifest['career.summary.xp_remaining'][locale]}
        </div>
        <div className="text-base font-semibold">
          {hero.xp_remaining.toLocaleString(intlLocale)}
        </div>
      </div>
      <div>
        <div className="text-xs text-muted-foreground">
          {careerManifest['career.summary.rank_position'][locale]}
        </div>
        <div className="text-base font-semibold">
          {hero.current_rank}/{totalRanks}
        </div>
      </div>
    </div>
  )
}

// ── Composants exportés ────────────────────────────────────────────────────

export interface CareerRankGaugeChartProps {
  summary: CareerSummary | null
  locale: ManifestLocale
  intlLocale: string
}

export function CareerRankGaugeChart({ summary, locale, intlLocale }: CareerRankGaugeChartProps) {
  return (
    <ChartCard<GaugePoint>
      title={careerManifest['career.charts.rank_gauge_title'][locale]}
      series={summary ? [rankGaugeSeries(summary, intlLocale)] : []}
      height={280}
      buildOption={buildRankGaugeOption}
      emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
    >
      {summary && !summary.is_max_rank && (
        <RankGaugeFooter summary={summary} locale={locale} intlLocale={intlLocale} />
      )}
    </ChartCard>
  )
}

export interface CareerHeroGaugeChartProps {
  heroProgress: HeroProgress | null
  locale: ManifestLocale
  intlLocale: string
}

export function CareerHeroGaugeChart({ heroProgress, locale, intlLocale }: CareerHeroGaugeChartProps) {
  return (
    <ChartCard<GaugePoint>
      title={careerManifest['career.charts.hero_gauge_title'][locale]}
      series={heroProgress ? [heroGaugeSeries(heroProgress, intlLocale)] : []}
      height={280}
      buildOption={buildHeroGaugeOption}
      emptyMessage={careerManifest['career.charts.placeholder_unavailable'][locale]}
    >
      {heroProgress && (
        <HeroGaugeFooter hero={heroProgress} locale={locale} intlLocale={intlLocale} />
      )}
    </ChartCard>
  )
}
