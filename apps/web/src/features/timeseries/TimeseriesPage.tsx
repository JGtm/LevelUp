/**
 * TimeseriesPage — page Séries temporelles (5 onglets).
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageLoader } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { useTimeseriesPage, useCombatYieldHistory } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import { CombatYieldTimeseries } from '@/components/ui/combat-yield-timeseries'
import { TimeseriesLineChart } from '@/components/ui/timeseries-line-chart'
import { TimeseriesHistogram } from '@/components/ui/timeseries-histogram'
import { TimeseriesHeatmap } from '@/components/ui/timeseries-heatmap'
import { TimeseriesScatter } from '@/components/ui/timeseries-scatter'
import { TimeseriesKdaBars } from '@/components/ui/timeseries-kda-bars'
import type { TimeseriesKpiCard } from '@/lib/api/types'

type TabId = 'summary' | 'cumul' | 'form' | 'intensity' | 'distributions' | 'combat'

const TABS: { id: TabId; label: string }[] = [
  { id: 'summary', label: 'KPIs' },
  { id: 'cumul', label: 'Cumul' },
  { id: 'form', label: 'Forme' },
  { id: 'intensity', label: 'Intensité' },
  { id: 'distributions', label: 'Distributions' },
  { id: 'combat', label: 'Combat' },
]

export function TimeseriesPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)
  const [activeTab, setActiveTab] = useState<TabId>('summary')

  const { data, isLoading, isError, refetch } = useTimeseriesPage(
    playerSlug,
    { filters: filterContext },
    filterContextHash,
  )

  const { data: combatData, isLoading: combatLoading } = useCombatYieldHistory(
    playerSlug,
    filterContextHash,
    filterContext,
  )

  if (isLoading) {
    return (
      <PageLoader label="Chargement des séries temporelles…" />
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement des séries.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="p-6">
        <EmptyStateCard
          title="Séries temporelles indisponibles"
          description="Le backend n'a renvoyé aucune charge utile pour cette page. Vérifie les filtres, les données locales ou la requête API."
          actionLabel="Réessayer"
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const { summary_tab, cumul_tab, form_tab, intensity_tab, distributions_tab } = data

  return (
    <div className="flex flex-col">
      {/* Onglets */}
      <div className="flex gap-0 border-b bg-background px-6">
        {TABS.map((tab) => (
          <Button
            key={tab.id}
            variant="ghost"
            size="sm"
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-none border-b-2 px-4 py-3 text-sm ${
              activeTab === tab.id
                ? 'border-primary font-semibold text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
          >
            {tab.label}
          </Button>
        ))}
      </div>

      <div className="p-6 space-y-6">
        {/* KPIs */}
        {activeTab === 'summary' && (
          <div className="space-y-6">
            {summary_tab.kpi_cards.length > 0 ? (
              <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
                {summary_tab.kpi_cards.map((card: TimeseriesKpiCard) => (
                  <Card key={card.key}>
                    <CardContent className="py-3 text-center">
                      <p className="text-xs text-muted-foreground">{card.label}</p>
                      <p
                        className="text-xl font-bold"
                        style={{ color: card.color ?? undefined }}
                      >
                        {card.value}
                      </p>
                      {card.delta && (
                        <p className="text-xs text-muted-foreground">{card.delta}</p>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="KPIs indisponibles"
                description="Aucune carte KPI n'a été calculée pour cette période."
              />
            )}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Timeline K/D par match</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesKdaBars rows={data.match_rows ?? []} />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Cumul */}
        {activeTab === 'cumul' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">K/D cumulé</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={[{ name: 'K/D cumulé', points: cumul_tab.cumulative_kd ?? [], color: '#0072B2' }]}
                  yAxisLabel="K/D"
                  referenceY={1}
                  referenceLabel="K/D = 1"
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Score net cumulé</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={[{ name: 'Kills – Morts cumulés', points: cumul_tab.cumulative_net ?? [], color: '#00DC82', fill: 'tozeroy' }]}
                  yAxisLabel="Net"
                  referenceY={0}
                />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">K/D glissant (20 matchs)</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={[{ name: 'K/D glissant', points: cumul_tab.rolling_kd ?? [], color: '#FFB703' }]}
                  yAxisLabel="K/D"
                  referenceY={1}
                  referenceLabel="K/D = 1"
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Forme */}
        {activeTab === 'form' && (
          <div className="space-y-6">
            {form_tab.regression_stats.has_enough_for_trend ? (
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                <DeltaCard
                  label="Pente K/D"
                  value={form_tab.regression_stats.kd_slope?.toFixed(4) ?? '—'}
                  unit="/match"
                  delta={form_tab.regression_stats.kd_slope}
                  warning={
                    (form_tab.regression_stats.r_squared ?? 0) < 0.3
                  }
                  warningText="R² < 0.3 — tendance non significative"
                />
                <DeltaCard
                  label="Pente Win Rate"
                  value={form_tab.regression_stats.winrate_slope != null
                    ? `${(form_tab.regression_stats.winrate_slope * 100).toFixed(2)}%`
                    : '—'}
                  unit="/match"
                  delta={form_tab.regression_stats.winrate_slope}
                />
                <DeltaCard
                  label="R²"
                  value={form_tab.regression_stats.r_squared?.toFixed(3) ?? '—'}
                  warning={(form_tab.regression_stats.r_squared ?? 0) < 0.3}
                  warningText="Tendance peu fiable"
                />
              </div>
            ) : (
              <EmptyStateNotice
                title="Tendance indisponible"
                description="Il faut davantage de matchs pour calculer une régression interprétable sur cette période."
              />
            )}
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">EWMA K/D (α = 0.20)</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={[{ name: 'EWMA K/D', points: form_tab.ewma_kd_points ?? [], color: '#33D6FF' }]}
                  yAxisLabel="K/D lissé"
                  referenceY={1}
                  referenceLabel="K/D = 1"
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Intensité */}
        {activeTab === 'intensity' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Heatmap d'intensité (jour × heure)</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesHeatmap data={intensity_tab.heatmap_data ?? []} colorBy="count" />
              </CardContent>
            </Card>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Score par minute</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesLineChart
                  series={[{ name: 'Score/min', points: intensity_tab.score_per_min_data ?? [], color: '#FFB703', fill: 'tozeroy' }]}
                  yAxisLabel="pts/min"
                />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Distributions */}
        {activeTab === 'distributions' && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Distribution K/D</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <TimeseriesHistogram
                    buckets={distributions_tab.kda_buckets ?? []}
                    color="#33D6FF"
                    xAxisLabel="K/D"
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Distribution Kills</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <TimeseriesHistogram
                    buckets={distributions_tab.kills_buckets ?? []}
                    color="#00DC82"
                    xAxisLabel="Kills / match"
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Distribution Précision</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <TimeseriesHistogram
                    buckets={distributions_tab.accuracy_buckets ?? []}
                    color="#FFB703"
                    xAxisLabel="Précision (%)"
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Distribution Score/min</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <TimeseriesHistogram
                    buckets={distributions_tab.score_per_min_buckets ?? []}
                    color="#8B5CF6"
                    xAxisLabel="Score / min"
                  />
                </CardContent>
              </Card>
              <Card>
                <CardHeader>
                  <CardTitle className="text-sm">Distribution Win Rate glissant</CardTitle>
                </CardHeader>
                <CardContent className="pb-4">
                  <TimeseriesHistogram
                    buckets={distributions_tab.rolling_wr_buckets ?? []}
                    color="#FF4B4B"
                    xAxisLabel="Win Rate (%)"
                  />
                </CardContent>
              </Card>
            </div>
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Corrélations</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                <TimeseriesScatter points={distributions_tab.correlation_points ?? []} />
              </CardContent>
            </Card>
          </div>
        )}

        {/* Combat — deux courbes OC + DR (S56) */}
        {activeTab === 'combat' && (
          <div className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm">Rendement combat par match</CardTitle>
              </CardHeader>
              <CardContent className="pb-4">
                {combatLoading ? (
                  <div className="flex justify-center py-8"><span className="text-muted-foreground text-sm">Chargement…</span></div>
                ) : (
                  <CombatYieldTimeseries rows={combatData?.table.items ?? []} />
                )}
              </CardContent>
            </Card>
          </div>
        )}
      </div>
    </div>
  )
}
