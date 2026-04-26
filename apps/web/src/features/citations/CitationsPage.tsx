/**
 * CitationsPage — commendations Halo 5 + médailles Halo Infinite.
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useCitationsPage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { tokenCssVar } from '@/lib/accessibility'

export function CitationsPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)

  const { data, isLoading, isError, refetch } = useCitationsPage(
    playerSlug,
    { filters: filterContext },
    filterContextHash,
  )

  if (isLoading) return null

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement des citations.</p>
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
          title="Citations indisponibles"
          description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie le chargement des agrégats de citations."
          actionLabel="Réessayer"
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const { commendations, medals_summary, deltas, distribution_chart } = data

  return (
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
                <p className="text-xs text-muted-foreground">{kpi.label}</p>
                <p className="text-lg font-bold text-foreground">{kpi.value}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Distribution Plotly */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Distribution</CardTitle>
          </CardHeader>
          <CardContent className="pb-4">
            {distribution_chart ? (
              <PlotlyChart figure={distribution_chart} />
            ) : (
              <EmptyStateNotice
                title="Distribution indisponible"
                description="Aucun graphique de distribution n'a été généré pour la sélection actuelle."
              />
            )}
          </CardContent>
        </Card>

        {/* Commendations */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Commendations ({commendations.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="pb-4">
            {commendations.length > 0 ? (
              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
                {commendations.map((c) => (
                  <div
                    key={c.key}
                    className="rounded-lg border p-3 space-y-1"
                    style={{ borderLeftColor: c.color ?? '#a78bfa', borderLeftWidth: 4 }}
                  >
                    <div className="flex items-center justify-between">
                      <p className="text-sm font-semibold text-foreground">{c.label}</p>
                      {c.tier_label && (
                        <Badge variant="secondary" className="text-xs">
                          {c.tier_label}
                        </Badge>
                      )}
                    </div>
                  <p className="text-xs text-muted-foreground">{c.category ?? ''}</p>
                  <div className="flex items-center justify-between text-xs">
                    <span className="text-foreground">Valeur : {c.current_value}</span>
                    {c.mastery_pct != null && (
                      <span className="font-medium text-primary">
                          {c.mastery_pct.toFixed(1)}%
                        </span>
                      )}
                    </div>
                    {c.mastery_pct != null && (
                      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
                        <div
                          className="h-full rounded-full bg-primary"
                          style={{ width: `${Math.min(100, c.mastery_pct)}%` }}
                        />
                      </div>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucune commendation disponible"
                description="Le backend n'a renvoyé aucune commendation pour cette période."
              />
            )}
          </CardContent>
        </Card>

        {/* Médailles — B3 NATIVE_COMPONENTS : grille responsive triée par count */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">
              Médailles ({medals_summary.length})
            </CardTitle>
          </CardHeader>
          <CardContent className="pb-4">
            {medals_summary.length > 0 ? (
              <div className="grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-8">
                {[...medals_summary]
                  .sort((a, b) => b.count_filtered - a.count_filtered)
                  .map((m) => (
                    <div
                      key={m.medal_name_id}
                      className="flex flex-col items-center rounded-lg bg-muted/40 p-2 text-center"
                      title={m.description ?? m.name}
                    >
                      <span className="text-lg font-bold" style={{ color: tokenCssVar('perf-tier-2') }}>{m.count_filtered}</span>
                      <span className="mt-0.5 text-[10px] leading-tight text-muted-foreground line-clamp-2">{m.name}</span>
                      {m.count_total !== m.count_filtered && (
                        <span className="text-[9px] text-muted-foreground">/{m.count_total}</span>
                      )}
                    </div>
                  ))}
              </div>
            ) : (
              <EmptyStateNotice
                title="Aucune médaille disponible"
                description="Aucune médaille n'a été agrégée pour cette période filtrée."
              />
            )}
          </CardContent>
        </Card>
    </div>
  )
}
