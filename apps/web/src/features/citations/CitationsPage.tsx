/**
 * CitationsPage — commendations Halo 5 + médailles Halo Infinite.
 */
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useCitationsPage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'

export function CitationsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)

  const { data, isLoading, isError, refetch } = useCitationsPage(
    playerSlug,
    { filters: filterContext },
    filterContextHash,
  )

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement des citations…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-red-600">Erreur lors du chargement des citations.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-purple-600 underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) return null

  const { commendations, medals_summary, deltas, distribution_chart } = data

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Citations"
        subtitle="Commendations et médailles dans la période sélectionnée"
      />

      <div className="space-y-6 p-6">
        {/* Delta filtre vs complet */}
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'Total filtré', value: deltas.filtered_total },
            { label: 'Total complet', value: deltas.unfiltered_total },
            { label: 'Delta', value: deltas.delta_count > 0 ? `+${deltas.delta_count}` : String(deltas.delta_count) },
          ].map((kpi) => (
            <Card key={kpi.label}>
              <CardContent className="py-3 text-center">
                <p className="text-xs text-gray-500">{kpi.label}</p>
                <p className="text-lg font-bold text-gray-900">{kpi.value}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Distribution Plotly */}
        {distribution_chart && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Distribution</CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <PlotlyChart figure={distribution_chart} />
            </CardContent>
          </Card>
        )}

        {/* Commendations */}
        {commendations.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Commendations ({commendations.length})
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {commendations.map((c) => (
                  <div
                    key={c.key}
                    className="rounded-lg border p-3 space-y-1"
                    style={{ borderLeftColor: c.color ?? '#a78bfa', borderLeftWidth: 4 }}
                  >
                    <div className="flex items-center justify-between">
                      <p className="text-sm font-semibold text-gray-800">{c.label}</p>
                      {c.tier_label && (
                        <Badge variant="secondary" className="text-xs">
                          {c.tier_label}
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-gray-500">{c.category ?? ''}</p>
                    <div className="flex items-center justify-between text-xs">
                      <span className="text-gray-700">Valeur : {c.current_value}</span>
                      {c.mastery_pct != null && (
                        <span className="font-medium text-purple-700">
                          {c.mastery_pct.toFixed(1)}%
                        </span>
                      )}
                    </div>
                    {c.mastery_pct != null && (
                      <div className="h-1.5 w-full overflow-hidden rounded-full bg-gray-100">
                        <div
                          className="h-full rounded-full bg-purple-500"
                          style={{ width: `${Math.min(100, c.mastery_pct)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Médailles */}
        {medals_summary.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">
                Médailles ({medals_summary.length})
              </CardTitle>
            </CardHeader>
            <CardContent className="pb-4 overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-gray-500">
                    <th className="py-2 pr-4">Médaille</th>
                    <th className="py-2 pr-4 text-right">Filtré</th>
                    <th className="py-2 text-right">Total</th>
                  </tr>
                </thead>
                <tbody>
                  {medals_summary.map((m) => (
                    <tr key={m.medal_name_id} className="border-b last:border-0">
                      <td className="py-1.5 pr-4">
                        <p className="font-medium text-gray-800">{m.name}</p>
                        {m.description && (
                          <p className="text-xs text-gray-400">{m.description}</p>
                        )}
                      </td>
                      <td className="py-1.5 pr-4 text-right font-semibold text-purple-700">
                        {m.count_filtered}
                      </td>
                      <td className="py-1.5 text-right text-gray-500">{m.count_total}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}

        {commendations.length === 0 && medals_summary.length === 0 && (
          <p className="text-center text-sm text-gray-500">
            Aucune citation pour la période sélectionnée.
          </p>
        )}
      </div>
    </div>
  )
}
