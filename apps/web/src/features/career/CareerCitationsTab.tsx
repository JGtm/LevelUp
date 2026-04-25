/**
 * CareerCitationsTab — onglet Citations du hub Carrière.
 * Sprint 55 : version hub sans dépendance au globalFilterStore.
 * Affiche les commendations et médailles dans leur totalité (scope global, pas filtre période).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { PageLoader } from '@/components/ui/spinner'
import { useCitationsPage } from '@/features/citations/queries'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/globalFilterStore'
import { tokenCssVar } from '@/lib/accessibility'

export function CareerCitationsTab() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }

  // Version hub : pas de filtre global — scope complet du joueur.
  const { data, isLoading, isError, refetch } = useCitationsPage(
    playerSlug,
    { filters: DEFAULT_FILTER_CONTEXT },
    'hub-all',
  )

  if (isLoading) {
    return (
      <PageLoader label="Chargement des citations…" />
    )
  }

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
          description="Aucune réponse exploitable n'a été renvoyée pour cette page."
          actionLabel="Réessayer"
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const { commendations, medals_summary, distribution_chart } = data

  // Sprint 55 B5 — Résumé de maîtrise
  const completedCommendations = commendations.filter(c => (c.mastery_pct ?? 0) >= 100).length
  const avgMastery = commendations.length > 0
    ? commendations.reduce((acc, c) => acc + (c.mastery_pct ?? 0), 0) / commendations.length
    : 0
  const totalMedals = medals_summary.reduce((acc, m) => acc + m.count_total, 0)
  const uniqueMedalTypes = medals_summary.length

  return (
    <div className="space-y-6 p-6">
      {/* Sprint 55 B5 — Résumé de maîtrise */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Résumé de maîtrise</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div className="rounded-lg border p-3">
              <span className="text-xs text-muted-foreground block">Commendations complètes</span>
              <span className="text-xl font-bold text-success">{completedCommendations}</span>
              <span className="text-xs text-muted-foreground"> / {commendations.length}</span>
            </div>
            <div className="rounded-lg border p-3">
              <span className="text-xs text-muted-foreground block">Maîtrise moyenne</span>
              <span className="text-xl font-bold">{avgMastery.toFixed(1)}%</span>
            </div>
            <div className="rounded-lg border p-3">
              <span className="text-xs text-muted-foreground block">Total médailles</span>
              <span className="text-xl font-bold">{totalMedals.toLocaleString()}</span>
            </div>
            <div className="rounded-lg border p-3">
              <span className="text-xs text-muted-foreground block">Types uniques</span>
              <span className="text-xl font-bold">{uniqueMedalTypes}</span>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Distribution Plotly */}
      {distribution_chart && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Distribution des citations</CardTitle>
          </CardHeader>
          <CardContent className="pb-4">
            <PlotlyChart figure={distribution_chart} />
          </CardContent>
        </Card>
      )}

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
              description="Le backend n'a renvoyé aucune commendation."
            />
          )}
        </CardContent>
      </Card>

      {/* Médailles */}
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
                    <span className="mt-0.5 text-[10px] leading-tight text-muted-foreground line-clamp-2">
                      {m.name}
                    </span>
                    {m.count_total !== m.count_filtered && (
                      <span className="text-[9px] text-muted-foreground">/{m.count_total}</span>
                    )}
                  </div>
                ))}
            </div>
          ) : (
            <EmptyStateNotice
              title="Aucune médaille disponible"
              description="Aucune médaille n'a été agrégée."
            />
          )}
        </CardContent>
      </Card>
    </div>
  )
}
