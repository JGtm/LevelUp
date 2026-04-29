/**
 * SynthesisPage --- Vue synthese / bilan periodique (Slice 7).
 * Types ref: SynthesisPageResponse, SynthesisKPIs, ComparisonMetricItem, HeatmapCell, TopWeekItem
 */
import { useState, useCallback, useMemo } from 'react'
import { useParams } from '@tanstack/react-router'
import type { EChartsCoreOption } from 'echarts/core'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useFieldMappings, useFieldLabel } from '@/lib/i18n/fieldMappings'
import { StatCard } from '@/components/cards/StatCard'
import { useSynthesisPage } from './queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SynthesisHighlightsSection } from './SynthesisHighlightsSection'
import { SynthesisRelationsPreview } from './SynthesisRelationsPreview'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { Heatmap2DChart, type ChartPointHeatmap } from '@/components/charts/Heatmap2DChart'
import { resolveToken } from '@/lib/accessibility'
import {
  CHART_BG,
  TEXT_COLOR,
  ZERO_LINE,
  axisBase,
  legendBase,
  tooltipBase,
} from '@/components/charts/_utils'
import type {
  ComparisonMetricItem,
  SynthesisBreakdowns,
  SynthesisKPIs,
  SynthesisOverview,
  SynthesisScope,
  SynthesisQueryRequest,
  TopWeekItem,
} from '@/lib/api/types'

const PERIOD_OPTIONS = [
  { value: 'all', label: 'Tout' },
  { value: '2y', label: '2 ans' },
  { value: '1y', label: '1 an' },
  { value: '1m', label: '1 mois' },
  { value: '1w', label: '1 semaine' },
] as const
type Period = typeof PERIOD_OPTIONS[number]['value']

const DOW_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']

/**
 * P7.1 (revue 2026-04-29) — formatage côté front d'une valeur de
 * ComparisonMetricItem (DTO sans solo_text/squad_text). Le format dépend de
 * la clé canonique (cf. fields.toml). Les valeurs brutes sont des `number`.
 */
function formatComparisonMetric(metricKey: string | undefined, value: number | undefined): string {
  if (value == null) return ''
  switch (metricKey) {
    case 'win_rate':
    case 'accuracy':
      return `${(value * 100).toFixed(1)}%`
    case 'performance_score':
      return value.toFixed(0)
    default:
      return value.toFixed(2)
  }
}

// ─── Graphique bipolaire Solo ← / → Escouade ─────────────────────────────────

export function buildBipolaireOption(metrics: ComparisonMetricItem[]): EChartsCoreOption {
  if (metrics.length === 0) return { backgroundColor: CHART_BG }
  const dps = [...metrics].reverse()
  const labels = dps.map((m) => m.label)
  const soloVals = dps.map((m) => -Math.abs(m.solo_value))
  const squadVals = dps.map((m) => Math.abs(m.squad_value))
  return {
    backgroundColor: CHART_BG,
    grid: { top: 24, bottom: 40, left: 120, right: 80 },
    tooltip: { ...tooltipBase, trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { ...legendBase, data: ['Solo', 'Escouade'] },
    xAxis: { ...axisBase, type: 'value', axisLabel: { show: false }, splitLine: { show: false } },
    yAxis: { ...axisBase, type: 'category', data: labels },
    series: [
      {
        name: 'Solo',
        type: 'bar',
        data: soloVals,
        itemStyle: { color: resolveToken('perf-tier-2') },
        label: {
          show: true, position: 'left',
          color: TEXT_COLOR, fontSize: 10,
          // P7.1 (revue 2026-04-29) : formatage côté front à partir des
          // valeurs brutes (solo_text/squad_text retirés du DTO).
          formatter: (p: { dataIndex: number }) =>
            formatComparisonMetric(dps[p.dataIndex]?.label, dps[p.dataIndex]?.solo_value),
        },
        markLine: {
          symbol: 'none', silent: true,
          lineStyle: { color: ZERO_LINE, width: 2 },
          label: { show: false },
          data: [{ xAxis: 0 }],
        },
      },
      {
        name: 'Escouade',
        type: 'bar',
        data: squadVals,
        itemStyle: { color: resolveToken('divergent-pos') },
        label: {
          show: true, position: 'right',
          color: TEXT_COLOR, fontSize: 10,
          formatter: (p: { dataIndex: number }) =>
            formatComparisonMetric(dps[p.dataIndex]?.label, dps[p.dataIndex]?.squad_value),
        },
      },
    ],
  }
}

// ─── Sous-composants ──────────────────────────────────────────────────────────

// ─── Bloc 0 — Scope bar ───────────────────────────────────────────────────────

interface ScopeBarProps { scope: SynthesisScope }
function ScopeBar({ scope }: ScopeBarProps) {
  const periodLabel = PERIOD_OPTIONS.find((o) => o.value === scope.period)?.label ?? scope.period
  const appliedFilters = scope.filters_applied ?? []
  const ignoredFilters = scope.filters_ignored ?? []
  return (
    <Card>
      <CardContent className="py-4">
        <div className="flex flex-wrap gap-6 items-center text-sm" data-testid="scope-bar">
          <span className="text-muted-foreground">Période : <strong className="text-foreground">{periodLabel}</strong></span>
          <span className="text-muted-foreground">Matchs : <strong className="text-foreground">{scope.match_count}</strong></span>
          {appliedFilters.length > 0 && (
            <span className="text-muted-foreground">Filtres : <strong className="text-foreground">{appliedFilters.join(', ')}</strong></span>
          )}
          {ignoredFilters.length > 0 && (
            <span className="text-xs text-amber-500">⚠ Ignorés : {ignoredFilters.join(', ')}</span>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// ─── Bloc 1 — Vue d'ensemble (D4) ─────────────────────────────────────────────

// P8.13 (revue 2026-04-29) : StatCell consolidé dans `components/cards/StatCard`
// (variant 'default'). Wrapper conservé pour la rétrocompat des call sites.
function StatCell({ label, value }: { label: string; value: string }) {
  return <StatCard label={label} value={value} />
}

interface SynthesisOverviewSectionProps { overview: SynthesisOverview }
function SynthesisOverviewSection({ overview }: SynthesisOverviewSectionProps) {
  const { data: fieldMappings } = useFieldMappings()
  const labelOf = (key: string): string =>
    fieldMappings?.fields[key]?.label ?? key
  // P4.4 (revue 2026-04-29 B3) : utilise overview.total_kdr expose par l'API
  // (P2.5) au lieu de sum/sum cote front (mathematiquement faux car
  // sum(K)/sum(D) != avg(K/D)). Fallback sur l'ancien recompute si total_kdr
  // manquant (vieux DTO).
  const kd = overview.total_kdr != null
    ? overview.total_kdr.toFixed(2)
    : overview.total_deaths > 0
    ? (overview.total_kills / overview.total_deaths).toFixed(2)
    : String(overview.total_kills)
  return (
    <Card>
      <CardHeader><CardTitle>Vue d'ensemble</CardTitle></CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <StatCell label="Victoires" value={String(overview.total_wins)} />
          <StatCell label="Défaites" value={String(overview.total_losses)} />
          <StatCell label={labelOf('kills')} value={String(overview.total_kills)} />
          <StatCell label="K/D moyen" value={kd} />
          <StatCell label={labelOf('win_rate')} value={`${(overview.win_rate * 100).toFixed(1)}%`} />
          {overview.best_kills_match != null && (
            <StatCell label="Meilleur match" value={`${overview.best_kills_match}K`} />
          )}
          {(overview.longest_win_streak ?? 0) > 1 && (
            <StatCell label="Série max" value={`${overview.longest_win_streak}V`} />
          )}
        </div>
      </CardContent>
    </Card>
  )
}

interface MetricRowProps { item: ComparisonMetricItem }
function MetricRow({ item }: MetricRowProps) {
  // P7.1 : item.label est une clé canonique (win_rate, kd_ratio…),
  // résolue en libellé via useFieldLabel (TOML fields.toml).
  const displayLabel = useFieldLabel(item.label)
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm font-medium text-foreground">{displayLabel}</td>
      <td className="px-4 py-3 text-center text-sm text-info">
        {formatComparisonMetric(item.label, item.solo_value)}
      </td>
      <td className="px-4 py-3 text-center text-sm text-success">
        {formatComparisonMetric(item.label, item.squad_value)}
      </td>
    </tr>
  )
}

interface KPISectionProps { title: string; kpis: SynthesisKPIs }
function KPISection({ title, kpis }: KPISectionProps) {
  return (
    <div>
      {title && <h3 className="text-sm font-semibold text-muted-foreground mb-2">{title}</h3>}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="rounded-lg border p-3"><span className="text-xs text-muted-foreground block">Win Rate</span><span className="text-xl font-bold">{(kpis.win_rate * 100).toFixed(1)}%</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-muted-foreground block">K/D</span><span className="text-xl font-bold">{kpis.kd_ratio?.toFixed(2) ?? '-'}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-muted-foreground block">Matchs</span><span className="text-xl font-bold">{kpis.match_count}</span></div>
        <div className="rounded-lg border p-3"><span className="text-xs text-muted-foreground block">Perf. <InfoTooltip content="Le Performance Score est un indice composite calculé par LevelUp à partir des kills, assists, objectifs et dégâts. Plus il est élevé, meilleure est la contribution globale au match." /></span><span className="text-xl font-bold">{kpis.performance_score?.toFixed(0) ?? '-'}</span></div>
      </div>
    </div>
  )
}

interface TopWeekRowProps { item: TopWeekItem; rank: number }
function TopWeekRow({ item, rank }: TopWeekRowProps) {
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-2 text-sm text-muted-foreground">{rank}</td>
      <td className="px-4 py-2 text-sm font-mono font-medium">{item.week_label}</td>
      <td className="px-4 py-2 text-center text-sm font-bold">{item.match_count}</td>
      <td className="px-4 py-2 text-center text-sm">{(item.win_rate * 100).toFixed(0)}%</td>
      <td className="px-4 py-2 text-center text-sm">{item.kd_ratio?.toFixed(2) ?? '-'}</td>
    </tr>
  )
}

// ─── Breakdowns D7 ────────────────────────────────────────────────────────────

interface SynthesisBreakdownsSectionProps { breakdowns: SynthesisBreakdowns }
function SynthesisBreakdownsSection({ breakdowns }: SynthesisBreakdownsSectionProps) {
  const hasMaps = breakdowns.top_maps.length > 0
  const hasModes = breakdowns.top_modes.length > 0
  if (!hasMaps && !hasModes) return null
  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {hasMaps && (
        <Card>
          <CardHeader><CardTitle>Par carte</CardTitle></CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted border-b">
                  <tr>
                    <th className="px-4 py-2 text-left">Carte</th>
                    <th className="px-4 py-2 text-center">Matchs</th>
                    <th className="px-4 py-2 text-center">Win%</th>
                  </tr>
                </thead>
                <tbody>
                  {breakdowns.top_maps.map((m) => (
                    <tr key={m.map_name} className="border-b last:border-0">
                      <td className="px-4 py-2 text-sm font-medium">{m.map_name}</td>
                      <td className="px-4 py-2 text-center text-sm">{m.match_count}</td>
                      <td className="px-4 py-2 text-center text-sm">{(m.win_rate * 100).toFixed(0)}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
      {hasModes && (
        <Card>
          <CardHeader><CardTitle>Par mode</CardTitle></CardHeader>
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted border-b">
                  <tr>
                    <th className="px-4 py-2 text-left">Mode</th>
                    <th className="px-4 py-2 text-center">Matchs</th>
                    <th className="px-4 py-2 text-center">Win%</th>
                  </tr>
                </thead>
                <tbody>
                  {breakdowns.top_modes.map((m) => (
                    <tr key={m.mode_name} className="border-b last:border-0">
                      <td className="px-4 py-2 text-sm font-medium">{m.mode_name}</td>
                      <td className="px-4 py-2 text-center text-sm">{m.match_count}</td>
                      <td className="px-4 py-2 text-center text-sm">{(m.win_rate * 100).toFixed(0)}%</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

// ─── Page principale ──────────────────────────────────────────────────────────

export function SynthesisPage() {
  const { playerSlug } = useParams({ from: '/players/$playerSlug/synthesis' })
  const { filterContext } = useGlobalFilterStore()
  const [period, setPeriod] = useState<Period>('all')
  const request: SynthesisQueryRequest = { filters: filterContext, period }
  const { data, isLoading, isError, error } = useSynthesisPage(playerSlug, period, request)

  const comparisonMetrics = data?.comparison_metrics ?? []
  const heatmapData = data?.heatmap_data ?? []
  const topWeeks = data?.top_weeks ?? []

  const bipolaireSeries = useMemo<ChartSeries<ComparisonMetricItem>[]>(
    () => comparisonMetrics.length > 0
      ? [{ key: 'bipolaire', datapoints: comparisonMetrics }]
      : [],
    [comparisonMetrics],
  )
  const bipolaireHeight = Math.max(320, 70 * comparisonMetrics.length)
  const buildBipolaireOptCb = useCallback(
    (s: ChartSeries<ComparisonMetricItem>[]) => buildBipolaireOption(s[0]?.datapoints ?? []),
    [],
  )

  const heatmapSeries = useMemo<ChartSeries<ChartPointHeatmap>[]>(() => {
    if (heatmapData.length === 0) return []
    const countMap = new Map<string, number>()
    for (const c of heatmapData) {
      if (c.dow >= 0 && c.dow < 7 && c.hour >= 0 && c.hour < 24) {
        countMap.set(`${c.dow}-${c.hour}`, c.count)
      }
    }
    const datapoints: ChartPointHeatmap[] = []
    for (let d = 0; d < 7; d++) {
      for (let h = 0; h < 24; h++) {
        datapoints.push({ x: `${h}h`, y: DOW_LABELS[d], value: countMap.get(`${d}-${h}`) ?? 0 })
      }
    }
    return [{ key: 'activity', datapoints }]
  }, [heatmapData])

  if (isLoading) return null
  if (isError) return <div className="p-8 text-center text-destructive">Erreur : {String(error)}</div>
  if (!data) {
    return (
      <div className="px-6">
        <EmptyStateCard
          title="Synthèse indisponible"
          description="Aucune charge utile n'a été renvoyée pour cette page. Vérifie les agrégats solo/escouade et le contexte de filtres."
        />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex justify-end px-6 pt-6">
        <div className="flex gap-1 rounded-lg border p-1">
          {PERIOD_OPTIONS.map((opt) => (
            <button
              key={opt.value}
              onClick={() => setPeriod(opt.value)}
              className={`rounded px-3 py-1 text-sm transition-colors ${period === opt.value ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted'}`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* Bloc 0 — Scope */}
      {data.scope && <ScopeBar scope={data.scope} />}

      {/* Bloc 1 — Vue d'ensemble D4 */}
      {data.overview && <SynthesisOverviewSection overview={data.overview} />}

      {/* KPIs Solo / Escouade */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader><CardTitle>Solo ({data.solo_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.solo_kpis} /></CardContent>
        </Card>
        <Card>
          <CardHeader><CardTitle>Escouade ({data.squad_kpis.match_count} matchs)</CardTitle></CardHeader>
          <CardContent><KPISection title="" kpis={data.squad_kpis} /></CardContent>
        </Card>
      </div>

      {/* Graphique bipolaire */}
      {bipolaireSeries.length > 0 ? (
        <Card>
          <CardHeader>
            <CardTitle>
              Solo <span className="mx-2 text-info">←</span> vs <span className="mx-2 text-success">→</span> Escouade
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ChartCard
              series={bipolaireSeries}
              buildOption={buildBipolaireOptCb}
              height={bipolaireHeight}
            />
            <p className="mt-2 text-xs text-muted-foreground text-center">
              Solo : {data.solo_kpis.match_count} matchs · Escouade : {data.squad_kpis.match_count} matchs
            </p>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardContent className="py-6 text-center text-muted-foreground text-sm">
            Pas assez de données des deux contextes pour afficher la comparaison.
          </CardContent>
        </Card>
      )}

      {/* Comparaison détaillée */}
      <Card>
        <CardHeader><CardTitle>Comparaison détaillée</CardTitle></CardHeader>
        <CardContent className="p-0">
          {comparisonMetrics.length === 0 ? (
            <p className="p-6 text-center text-muted-foreground">Pas assez de données pour cette période.</p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted border-b">
                  <tr>
                    <th className="px-4 py-3 text-left">Métrique</th>
                    <th className="px-4 py-3 text-center text-info">Solo</th>
                    <th className="px-4 py-3 text-center text-success">Escouade</th>
                  </tr>
                </thead>
                <tbody>
                  {comparisonMetrics.map((item, idx) => <MetricRow key={idx} item={item} />)}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Performances marquantes D5 */}
      {data.highlights_preview && (
        <SynthesisHighlightsSection highlights={data.highlights_preview} playerSlug={playerSlug} />
      )}

      {/* Heatmap temporelle */}
      <Card>
        <CardHeader><CardTitle>Activité par jour et heure</CardTitle></CardHeader>
        <CardContent>
          {heatmapSeries.length > 0 ? (
            <Heatmap2DChart series={heatmapSeries} height={220} />
          ) : (
            <EmptyStateNotice
              title="Activité indisponible"
              description="La heatmap temporelle ne peut pas être affichée sans répartition horaire exploitable."
            />
          )}
        </CardContent>
      </Card>

      {/* Top semaines */}
      <Card>
        <CardHeader><CardTitle>Top semaines</CardTitle></CardHeader>
        <CardContent className={topWeeks.length > 0 ? 'p-0' : undefined}>
          {topWeeks.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="bg-muted border-b">
                  <tr>
                    <th className="px-4 py-2 text-left text-muted-foreground">#</th>
                    <th className="px-4 py-2 text-left">Semaine</th>
                    <th className="px-4 py-2 text-center">Matchs</th>
                    <th className="px-4 py-2 text-center">Win%</th>
                    <th className="px-4 py-2 text-center">K/D</th>
                  </tr>
                </thead>
                <tbody>
                  {topWeeks.map((w, i) => <TopWeekRow key={w.week_label} item={w} rank={i + 1} />)}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyStateNotice
              title="Aucune semaine remarquable"
              description="Le classement hebdomadaire est vide pour la période et les filtres sélectionnés."
            />
          )}
        </CardContent>
      </Card>

      {/* Relations / Rivalités D6 */}
      {data.rivalries_preview &&
        (data.rivalries_preview.top_teammates.length > 0 || data.rivalries_preview.top_enemies.length > 0) && (
          <SynthesisRelationsPreview playerSlug={playerSlug} preview={data.rivalries_preview} />
        )}

      {/* Répartition carte / mode D7 */}
      {data.breakdowns && <SynthesisBreakdownsSection breakdowns={data.breakdowns} />}
    </div>
  )
}
