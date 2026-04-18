/**
 * TimeseriesPage — page Séries temporelles (5 onglets).
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useTimeseriesPage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import type { PlotlyFigurePayload, TimeseriesKpiCard } from '@/lib/api/types'

type TabId = 'summary' | 'cumul' | 'form' | 'intensity' | 'distributions'

const TABS: { id: TabId; label: string }[] = [
  { id: 'summary', label: 'KPIs' },
  { id: 'cumul', label: 'Cumul' },
  { id: 'form', label: 'Forme' },
  { id: 'intensity', label: 'Intensité' },
  { id: 'distributions', label: 'Distributions' },
]

function ChartCard({ title, figure }: { title: string; figure: PlotlyFigurePayload | null }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{title}</CardTitle>
      </CardHeader>
      <CardContent className="pb-4">
        {figure ? (
          <PlotlyChart figure={figure} />
        ) : (
          <EmptyStateNotice
            title="Graphique indisponible"
            description="Aucune figure exploitable n'a été renvoyée pour cette section."
          />
        )}
      </CardContent>
    </Card>
  )
}

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

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement des séries temporelles…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-red-600">Erreur lors du chargement des séries.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-purple-600 underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) {
    return (
      <div className="flex flex-col">
        <PageHeader
          title="Séries temporelles"
          subtitle="Vue analytique des performances"
        />
        <div className="p-6">
          <EmptyStateCard
            title="Séries temporelles indisponibles"
            description="Le backend n'a renvoyé aucune charge utile pour cette page. Vérifie les filtres, les données locales ou la requête API."
            actionLabel="Réessayer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const { summary_tab, cumul_tab, form_tab, intensity_tab, distributions_tab } = data

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Séries temporelles"
        subtitle={`${data.total_matches.toLocaleString('fr-FR')} parties analysées`}
      />

      {/* Onglets */}
      <div className="flex gap-0 border-b bg-white px-6">
        {TABS.map((tab) => (
          <Button
            key={tab.id}
            variant="ghost"
            size="sm"
            onClick={() => setActiveTab(tab.id)}
            className={`rounded-none border-b-2 px-4 py-3 text-sm ${
              activeTab === tab.id
                ? 'border-purple-600 font-semibold text-purple-700'
                : 'border-transparent text-gray-500 hover:text-gray-800'
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
                      <p className="text-xs text-gray-500">{card.label}</p>
                      <p
                        className="text-xl font-bold"
                        style={{ color: card.color ?? undefined }}
                      >
                        {card.value}
                      </p>
                      {card.delta && (
                        <p className="text-xs text-gray-400">{card.delta}</p>
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
            <ChartCard title="KDA Distribution" figure={summary_tab.kda_dist_chart} />
            <ChartCard title="Taux de victoire" figure={summary_tab.win_rate_chart} />
            <ChartCard title="Score" figure={summary_tab.score_chart} />
          </div>
        )}

        {/* Cumul */}
        {activeTab === 'cumul' && (
          <div className="space-y-6">
            <ChartCard title="Score net cumulé" figure={cumul_tab.cumul_net_chart} />
            <ChartCard title="K/D cumulé" figure={cumul_tab.cumul_kd_chart} />
            <ChartCard title="K/D glissant" figure={cumul_tab.rolling_kd_chart} />
          </div>
        )}

        {/* Forme */}
        {activeTab === 'form' && (
          <div className="space-y-6">
            {/* D1 — Cartes régression K/D */}
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
            <ChartCard title="EWMA K/D" figure={form_tab.ewma_kd_chart} />
            <ChartCard title="Tendance (régression)" figure={form_tab.regression_chart} />
            <ChartCard title="Score net / heure" figure={form_tab.net_score_per_hour_chart} />
          </div>
        )}

        {/* Intensité */}
        {activeTab === 'intensity' && (
          <div className="space-y-6">
            <ChartCard title="Heatmap d'intensité" figure={intensity_tab.intensity_heatmap} />
            <ChartCard title="Score par minute" figure={intensity_tab.score_per_minute_chart} />
          </div>
        )}

        {/* Distributions */}
        {activeTab === 'distributions' && (
          <div className="space-y-6">
            <ChartCard title="Distribution KDA" figure={distributions_tab.kda_distribution} />
            <ChartCard title="Premier événement" figure={distributions_tab.first_kill_dist} />
            {distributions_tab.correlations.map((fig, i) => (
              <Card key={i}>
                <CardContent className="pb-4 pt-4">
                  <PlotlyChart figure={fig} />
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
