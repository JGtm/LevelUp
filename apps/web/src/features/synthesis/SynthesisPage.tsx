/**
 * SynthesisPage --- Vue synthese / bilan periodique (Slice 7).
 * Types ref: SynthesisPageResponse, SynthesisKPIs, ComparisonMetricItem, HeatmapCell, TopWeekItem
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { useSynthesisPage } from './queries'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PageLoader } from '@/components/ui/spinner'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { SynthesisHighlightsSection } from './SynthesisHighlightsSection'
import { SynthesisRelationsPreview } from './SynthesisRelationsPreview'
import type {
  ComparisonMetricItem,
  HeatmapCell,
  PlotlyFigurePayload,
  SynthesisBreakdowns,
  SynthesisKPIs,
  SynthesisOverview,
  SynthesisScope,
  SynthesisQueryRequest,
  TopWeekItem,
} from '@/lib/api/types'
import { resolveToken } from '@/lib/accessibility'

const PERIOD_OPTIONS = [
  { value: 'all', label: 'Tout' },
  { value: '2y', label: '2 ans' },
  { value: '1y', label: '1 an' },
  { value: '1m', label: '1 mois' },
  { value: '1w', label: '1 semaine' },
] as const
type Period = typeof PERIOD_OPTIONS[number]['value']

const DOW_LABELS = ['Lun', 'Mar', 'Mer', 'Jeu', 'Ven', 'Sam', 'Dim']

// ─── Graphique bipolaire Solo ← / → Escouade ─────────────────────────────────

function buildBipolaireChart(metrics: ComparisonMetricItem[]): PlotlyFigurePayload {
  const reversed = [...metrics].reverse()
  const labels = reversed.map((m) => m.label)
  const soloVals = reversed.map((m) => -Math.abs(m.solo_value))
  const squadVals = reversed.map((m) => Math.abs(m.squad_value))
  const soloTexts = reversed.map((m) => m.solo_text)
  const squadTexts = reversed.map((m) => m.squad_text)
  const height = Math.max(320, 70 * metrics.length)

  return {
    data: [
      {
        type: 'bar', name: 'Solo', orientation: 'h',
        x: soloVals, y: labels, text: soloTexts, textposition: 'outside',
        marker: { color: resolveToken('perf-tier-2') },
        hovertemplate: '<b>Solo</b>: %{text}<extra></extra>',
      },
      {
        type: 'bar', name: 'Escouade', orientation: 'h',
        x: squadVals, y: labels, text: squadTexts, textposition: 'outside',
        marker: { color: resolveToken('divergent-pos') },
        hovertemplate: '<b>Escouade</b>: %{text}<extra></extra>',
      },
    ],
    layout: {
      height, barmode: 'overlay', bargap: 0.3,
      margin: { l: 110, r: 80, t: 20, b: 40 },
      xaxis: {
        tickformat: '.2f', zeroline: true, zerolinewidth: 2,
        zerolinecolor: 'rgba(100,116,139,0.8)', showticklabels: false, fixedrange: true,
      },
      yaxis: { automargin: true, fixedrange: true },
      legend: { orientation: 'h', x: 0.5, xanchor: 'center', y: -0.08 },
      plot_bgcolor: 'rgba(0,0,0,0)', paper_bgcolor: 'rgba(0,0,0,0)', font: { size: 12 },
      shapes: [{
        type: 'line', x0: 0, x1: 0, y0: -0.5, y1: labels.length - 0.5,
        xref: 'x', yref: 'y', line: { color: 'rgba(100,116,139,0.8)', width: 2 },
      }],
    },
  }
}

// ─── Heatmap temporelle ───────────────────────────────────────────────────────

function buildHeatmapChart(cells: HeatmapCell[]): PlotlyFigurePayload {
  // Construire une matrice 7 lignes (jours) × 24 colonnes (heures)
  const matrix: number[][] = Array.from({ length: 7 }, () => new Array(24).fill(0))
  for (const c of cells) {
    if (c.dow >= 0 && c.dow < 7 && c.hour >= 0 && c.hour < 24) {
      matrix[c.dow][c.hour] = c.count
    }
  }
  return {
    data: [{
      type: 'heatmap',
      z: matrix,
      x: Array.from({ length: 24 }, (_, i) => `${i}h`),
      y: DOW_LABELS,
      colorscale: 'Blues',
      showscale: true,
      hovertemplate: '%{y} %{x} : %{z} matchs<extra></extra>',
    }],
    layout: {
      height: 220,
      margin: { l: 50, r: 20, t: 10, b: 40 },
      xaxis: { title: 'Heure', fixedrange: true },
      yaxis: { title: '', fixedrange: true },
      plot_bgcolor: 'rgba(0,0,0,0)', paper_bgcolor: 'rgba(0,0,0,0)', font: { size: 11 },
    },
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

function StatCell({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-lg border p-3">
      <span className="text-xs text-muted-foreground block">{label}</span>
      <span className="text-xl font-bold">{value}</span>
    </div>
  )
}

interface SynthesisOverviewSectionProps { overview: SynthesisOverview }
function SynthesisOverviewSection({ overview }: SynthesisOverviewSectionProps) {
  const kd = overview.total_deaths > 0
    ? (overview.total_kills / overview.total_deaths).toFixed(2)
    : String(overview.total_kills)
  return (
    <Card>
      <CardHeader><CardTitle>Vue d'ensemble</CardTitle></CardHeader>
      <CardContent>
        <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
          <StatCell label="Victoires" value={String(overview.total_wins)} />
          <StatCell label="Défaites" value={String(overview.total_losses)} />
          <StatCell label="Kills" value={String(overview.total_kills)} />
          <StatCell label="K/D moyen" value={kd} />
          <StatCell label="Win Rate" value={`${(overview.win_rate * 100).toFixed(1)}%`} />
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
  return (
    <tr className="border-b last:border-0">
      <td className="px-4 py-3 text-sm font-medium text-foreground">{item.label}</td>
      <td className="px-4 py-3 text-center text-sm text-info">{item.solo_text}</td>
      <td className="px-4 py-3 text-center text-sm text-success">{item.squad_text}</td>
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

  if (isLoading) return <PageLoader />
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

  const comparisonMetrics = data.comparison_metrics ?? []
  const heatmapData = data.heatmap_data ?? []
  const topWeeks = data.top_weeks ?? []

  const bipolaireChart = comparisonMetrics.length > 0
    ? buildBipolaireChart(comparisonMetrics)
    : null

  const heatmapChart = heatmapData.length > 0
    ? buildHeatmapChart(heatmapData)
    : null

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
      {bipolaireChart ? (
        <Card>
          <CardHeader>
            <CardTitle>
              Solo <span className="mx-2 text-info">←</span> vs <span className="mx-2 text-success">→</span> Escouade
            </CardTitle>
          </CardHeader>
          <CardContent>
            <PlotlyChart figure={bipolaireChart} />
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
          {heatmapChart ? (
            <PlotlyChart figure={heatmapChart} />
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
