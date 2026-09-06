/**
 * MatchSummaryCharts — charts de l'onglet Résumé (côte à côte).
 *
 * C8 : MatchKdaExpectedChart  — match_view.03 : K/D/A réel vs attendu vs hist. moy.
 * C9 : MatchSpreeChart        — match_view.04 : Spree + Headshots + Perfect Kills
 *
 * Mise en forme : 3 traitements visuels distincts
 *   Réel     — plein, opacité haute
 *   Attendu  — opacité 0.6 + bordure + hachures diagonales (décal rect)
 *   Hist.    — opacité 0.35 + cercles (décal circle)
 */
import { useCallback } from 'react'
import { Link } from '@tanstack/react-router'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { buildRadarOption } from '@/components/charts/RadarChart'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import {
  getEChartsThemeColors,
  getAxisBase,
  getTooltipBase,
  getLegendBase,
  CHART_BG,
} from '@/components/charts/_utils'
import type { MatchSummaryKpis, MatchExpectedStats, MatchViewRadarSeries } from '@/lib/api/types'
import type { MatchViewText } from './i18n'

// ---------------------------------------------------------------------------
// Types décal ECharts
// ---------------------------------------------------------------------------

type DecalConfig = {
  symbol: string
  symbolSize: number
  dashArrayX: number[]
  dashArrayY: number[]
  rotation?: number
  color: string
}

type ItemStyle = {
  color: string
  opacity?: number
  borderWidth?: number
  borderColor?: string
  decal?: DecalConfig
}

type BarDataItem = { value: number | null; itemStyle: ItemStyle }
type BarSeries = { name: string; type: 'bar'; barMaxWidth: number; data: BarDataItem[] }

// Décal hachures diagonales (série "Attendu")
const DECAL_HATCH: DecalConfig = {
  symbol: 'rect',
  symbolSize: 0.8,
  dashArrayX: [1, 0],
  dashArrayY: [4, 4],
  rotation: -Math.PI / 4,
  color: 'rgba(255,255,255,0.35)', // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
}

// Décal cercles (série "Hist. Moy.")
const DECAL_DOTS: DecalConfig = {
  symbol: 'circle',
  symbolSize: 1,
  dashArrayX: [4, 8],
  dashArrayY: [4, 8],
  color: 'rgba(255,255,255,0.45)', // color-allow: 2026-09-06 (revue R1, C5) — voile NEUTRE d ombre/fond d infobulle ECharts, pas une couleur de charte ; dette PREEXISTANTE au lot v2 D, a porter sur un token le jour ou un token de voile existera
}

// option aria pour activer les décals par item
const ARIA_DECAL = { decal: { show: true } }

// ---------------------------------------------------------------------------
// Helpers de construction des données par barre
// ---------------------------------------------------------------------------

function makeActualData(values: (number | null)[], tokens: SemanticToken[]): BarDataItem[] {
  return values.map((v, i) => ({
    value: v,
    itemStyle: { color: resolveToken(tokens[i]), opacity: 0.9 },
  }))
}

function makeExpectedData(values: (number | null)[], tokens: SemanticToken[]): BarDataItem[] {
  return values.map((v, i) => ({
    value: v,
    itemStyle: {
      color: resolveToken(tokens[i]),
      opacity: 0.6,
      borderWidth: 2,
      borderColor: resolveToken(tokens[i]),
      decal: DECAL_HATCH,
    },
  }))
}

function makeHistData(values: (number | null)[], tokens: SemanticToken[]): BarDataItem[] {
  return values.map((v, i) => ({
    value: v,
    itemStyle: {
      color: resolveToken(tokens[i]),
      opacity: 0.35,
      decal: DECAL_DOTS,
    },
  }))
}

const DUMMY_SERIES: ChartSeries<number>[] = [{ key: 'data', datapoints: [1] }]

// ---------------------------------------------------------------------------
// C8 — MatchKdaExpectedChart (match_view.03)
// ---------------------------------------------------------------------------

interface MatchKdaExpectedChartProps {
  kpis: MatchSummaryKpis
  expectedStats: MatchExpectedStats
  t: MatchViewText
}

export function MatchKdaExpectedChart({ kpis, expectedStats, t }: MatchKdaExpectedChartProps) {
  // has_expected_data retiré du contrat (Phase 3a-B) : dérivation strictement équivalente.
  const hasExpected =
    expectedStats.expected_kills != null ||
    expectedStats.expected_deaths != null ||
    expectedStats.expected_assists != null
  const hasHist = expectedStats.has_hist_avg ?? false

  const buildOption = useCallback(
    (): EChartsCoreOption => {
      const tc = getEChartsThemeColors()
      const cats = [t.labelKills, t.labelDeaths, t.labelAssists]
      // color-allow: hex en commentaires de documentation token→couleur
      // K=#00DC82  D=#FF4B4B  A=#33D6FF — tokens qui correspondent exactement
      const tokens: SemanticToken[] = [
        'narrative-dominant', // color-allow: doc token (vert vif #00DC82)
        'heatmap-divergent-low', // color-allow: doc token (rouge vif #FF4B4B)
        'narrative-contre-remontada', // color-allow: doc token (cyan #33D6FF)
      ]

      const seriesList: BarSeries[] = [
        {
          name: t.seriesActual,
          type: 'bar',
          barMaxWidth: 36,
          data: makeActualData(
            [kpis.kills ?? null, kpis.deaths ?? null, kpis.assists ?? null],
            tokens,
          ),
        },
      ]

      if (hasExpected) {
        seriesList.push({
          name: t.seriesExpected,
          type: 'bar',
          barMaxWidth: 36,
          data: makeExpectedData(
            [
              expectedStats.expected_kills ?? null,
              expectedStats.expected_deaths ?? null,
              expectedStats.expected_assists ?? null,
            ],
            tokens,
          ),
        })
      }

      if (hasHist) {
        seriesList.push({
          name: t.seriesHistAvg,
          type: 'bar',
          barMaxWidth: 36,
          data: makeHistData(
            [
              expectedStats.hist_avg_kills ?? null,
              expectedStats.hist_avg_deaths ?? null,
              expectedStats.hist_avg_assists ?? null,
            ],
            tokens,
          ),
        })
      }

      const legendData = [t.seriesActual]
      if (hasExpected) legendData.push(t.seriesExpected)
      if (hasHist) legendData.push(t.seriesHistAvg)

      return {
        aria: ARIA_DECAL,
        backgroundColor: CHART_BG,
        grid: { left: 36, right: 12, top: 16, bottom: 56, containLabel: false },
        tooltip: { ...getTooltipBase(tc), trigger: 'axis' as const },
        legend: { ...getLegendBase(tc), data: legendData },
        xAxis: { ...getAxisBase(tc), type: 'category' as const, data: cats },
        yAxis: { ...getAxisBase(tc), type: 'value' as const, minInterval: 1, min: 0 },
        series: seriesList as EChartsCoreOption['series'],
      }
    },

    [kpis, expectedStats, t, hasExpected, hasHist],
  )

  return (
    <ChartCard
      title={t.chartKdaTitle}
      series={hasExpected || hasHist ? DUMMY_SERIES : []}
      emptyMessage={t.combatNoData}
      buildOption={buildOption}
      height={260}
    />
  )
}

// ---------------------------------------------------------------------------
// C9 — MatchSpreeChart (match_view.04)
// ---------------------------------------------------------------------------

interface MatchSpreeChartProps {
  kpis: MatchSummaryKpis
  expectedStats: MatchExpectedStats
  t: MatchViewText
}

export function MatchSpreeChart({ kpis, expectedStats, t }: MatchSpreeChartProps) {
  const hasHist = expectedStats.has_hist_avg ?? false

  const buildOption = useCallback(
    (): EChartsCoreOption => {
      const tc = getEChartsThemeColors()
      const cats = [t.labelSpree, t.labelHeadshots, t.labelPerfectKills]
      // color-allow: hex en commentaires de documentation token→couleur
      // Spree=#8B5CF6  Headshots=#33D6FF  Perfect=#00DC82
      const tokens: SemanticToken[] = [
        'outcome-dnf', // color-allow: doc token (violet #8B5CF6)
        'narrative-contre-remontada', // color-allow: doc token (cyan #33D6FF)
        'narrative-dominant', // color-allow: doc token (vert vif #00DC82)
      ]

      const seriesList: BarSeries[] = [
        {
          name: t.seriesActual,
          type: 'bar',
          barMaxWidth: 36,
          data: makeActualData(
            [
              kpis.max_killing_spree ?? null,
              kpis.headshot_kills ?? null,
              kpis.perfect_kills ?? null,
            ],
            tokens,
          ),
        },
      ]

      if (hasHist) {
        seriesList.push({
          name: t.seriesHistAvg,
          type: 'bar',
          barMaxWidth: 36,
          data: makeHistData(
            [
              expectedStats.hist_avg_spree ?? null,
              expectedStats.hist_avg_headshot_kills ?? null,
              expectedStats.hist_avg_perfect_kills ?? null,
            ],
            tokens,
          ),
        })
      }

      return {
        aria: ARIA_DECAL,
        backgroundColor: CHART_BG,
        grid: { left: 36, right: 12, top: 16, bottom: 56, containLabel: false },
        tooltip: { ...getTooltipBase(tc), trigger: 'axis' as const },
        legend: {
          ...getLegendBase(tc),
          data: hasHist ? [t.seriesActual, t.seriesHistAvg] : [t.seriesActual],
        },
        xAxis: { ...getAxisBase(tc), type: 'category' as const, data: cats },
        yAxis: { ...getAxisBase(tc), type: 'value' as const, minInterval: 1, min: 0 },
        series: seriesList as EChartsCoreOption['series'],
      }
    },

    [kpis, expectedStats, t, hasHist],
  )

  const hasAnyData =
    kpis.max_killing_spree != null ||
    kpis.headshot_kills != null ||
    kpis.perfect_kills != null

  return (
    <ChartCard
      title={t.chartSpreeTitle}
      series={hasAnyData ? DUMMY_SERIES : []}
      emptyMessage={t.combatNoData}
      buildOption={buildOption}
      height={220}
    />
  )
}

// ---------------------------------------------------------------------------
// C10 — MatchSummaryRadarChart (radar synergie joueur actif)
// ---------------------------------------------------------------------------

interface MatchSummaryRadarChartProps {
  radar: MatchViewRadarSeries[] | undefined
  meXUID: string | null
  t: MatchViewText
}

export function MatchSummaryRadarChart({ radar, meXUID, t }: MatchSummaryRadarChartProps) {
  const mySeries = radar?.find((s) => s.xuid === meXUID)

  const axisLabels: Record<string, string> = {
    combat: t.radarAxisCombat,
    survival: t.radarAxisSurvival,
    support: t.radarAxisSupport,
    score: t.radarAxisScore,
    objective: t.radarAxisObjective,
    impact: t.radarAxisImpact,
  }

  const radarPayload = mySeries
    ? [
        {
          key: `match.radar.${mySeries.xuid}`,
          axes: mySeries.axes.map((ax) => ({ axis: ax.Axis, value: ax.Value, raw: ax.Raw })),
          meta: { gamertag: mySeries.gamertag },
        },
      ]
    : []

  const buildOption = useCallback(
    (): EChartsCoreOption => {
      const opt = buildRadarOption(radarPayload, { axisLabels }) as Record<string, unknown>
      // Serie unique (joueur actif) : aucune legende a afficher.
      opt.legend = { show: false }
      return opt as EChartsCoreOption
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [radar, meXUID, t],
  )

  const titleNode = (
    <span className="flex items-center gap-1.5">
      {t.chartSynergyRadarTitle}
      <InfoTooltip
        content={
          <div className="space-y-1">
            <p><span className="font-medium">{t.radarAxisImpact}</span> — {t.radarTooltipImpact}</p>
            <p><span className="font-medium">{t.radarAxisCombat}</span> — {t.radarTooltipCombat}</p>
            <p><span className="font-medium">{t.radarAxisSurvival}</span> — {t.radarTooltipSurvival}</p>
            <p><span className="font-medium">{t.radarAxisSupport}</span> — {t.radarTooltipSupport}</p>
            <p><span className="font-medium">{t.radarAxisScore}</span> — {t.radarTooltipScore}</p>
            <p><span className="font-medium">{t.radarAxisObjective}</span> — {t.radarTooltipObjective}</p>
            <Link to="/help" search={{ tab: 'glossary' }} className="block mt-2 text-primary hover:underline">
              {t.radarTooltipGlossaryLink}
            </Link>
          </div>
        }
      />
    </span>
  )

  return (
    <ChartCard
      title={titleNode}
      series={radarPayload as unknown as { key: string; datapoints: unknown[] }[]}
      emptyMessage={t.combatNoData}
      buildOption={buildOption}
      height={220}
    />
  )
}
