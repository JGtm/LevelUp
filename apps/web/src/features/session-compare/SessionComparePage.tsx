/**
 * SessionComparePage — comparaison A/B de deux sessions.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { PlotlyChart } from '@/components/ui/plotly-chart'
import { useSessionComparePage } from './queries'
import { useGlobalFilterStore } from '@/stores/globalFilterStore'
import { DeltaCard } from '@/components/ui/delta-card'
import type { SessionCompareEntry, SessionCompareMetricRow } from '@/lib/api/types'

function SessionCard({
  label,
  entry,
  side,
}: {
  label: string
  entry: SessionCompareEntry | null
  side: 'A' | 'B'
}) {
  const color = side === 'A' ? 'text-compare-a' : 'text-compare-b'
  const bg = side === 'A' ? 'bg-compare-a/10 border-compare-a' : 'bg-compare-b/10 border-compare-b'

  return (
    <Card className={`border ${bg}`}>
      <CardHeader className="pb-2">
        <CardTitle className={`text-sm ${color}`}>
          Session {side} {label && `— ${label}`}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 text-xs text-foreground">
        {entry ? (
          <>
            <p>
              <span className="font-medium">Matchs :</span> {entry.total_matches}
            </p>
            <p>
              <span className="font-medium">Victoires :</span> {entry.wins} · Défaites : {entry.losses}
            </p>
            {entry.kda != null && (
              <p>
                <span className="font-medium">KDA :</span> {entry.kda.toFixed(2)}
              </p>
            )}
            {entry.performance_score != null && (
              <p>
                <span className="font-medium">Score perf. :</span>{' '}
                {entry.performance_score.toFixed(1)}
              </p>
            )}
            {entry.dominant_category && (
              <Badge variant="secondary" className="mt-1">
                {entry.dominant_category}
              </Badge>
            )}
          </>
        ) : (
          <p className="text-muted-foreground italic">Non sélectionnée</p>
        )}
      </CardContent>
    </Card>
  )
}

function MetricRow({ row }: { row: SessionCompareMetricRow }) {
  const winnerColor =
    row.winner === 'a'
      ? 'text-compare-a font-semibold'
      : row.winner === 'b'
        ? 'text-compare-b font-semibold'
        : 'text-foreground'

  return (
    <tr className="border-b last:border-0 text-sm">
      <td className="py-2 pr-4 text-muted-foreground">{row.label}</td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'a' ? 'text-compare-a font-semibold' : 'text-foreground'}`}>
        {row.value_a}
      </td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'b' ? 'text-compare-b font-semibold' : 'text-foreground'}`}>
        {row.value_b}
      </td>
      <td className={`py-2 text-right text-xs ${winnerColor}`}>
        {row.delta ?? '—'}
      </td>
    </tr>
  )
}

export function SessionComparePage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const filterContext = useGlobalFilterStore((s) => s.filterContext)
  const filterContextHash = useGlobalFilterStore((s) => s.filterContextHash)

  const [sessionA, setSessionA] = useState('')
  const [sessionB, setSessionB] = useState('')

  const { data, isLoading, isError, refetch } = useSessionComparePage(
    playerSlug,
    {
      filters: filterContext,
      session_a: sessionA || null,
      session_b: sessionB || null,
    },
    filterContextHash,
    sessionA,
    sessionB,
  )

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement de la comparaison…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement.</p>
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
      <div className="flex flex-col">
        <PageHeader
          title="Comparaison de sessions"
          subtitle="Sélectionnez deux sessions pour les comparer"
        />
        <div className="p-6">
          <EmptyStateCard
            title="Comparaison indisponible"
            description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie les sessions calculées et les filtres actifs."
            actionLabel="Réessayer"
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const hasAvailableSessions = data.available_sessions.length > 0
  const hasComparisonSelection = Boolean(sessionA && sessionB)

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Comparaison de sessions"
        subtitle="Sélectionnez deux sessions pour les comparer"
      />

      <div className="space-y-6 p-6">
        {hasAvailableSessions ? (
          <>
            {/* Sélecteurs A / B */}
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">Session A</label>
                <select
                  className="w-full rounded-md border border-border px-3 py-2 text-sm"
                  value={sessionA}
                  onChange={(e) => setSessionA(e.target.value)}
                >
                  <option value="">— Sélectionner —</option>
                  {data.available_sessions.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="mb-1 block text-xs font-medium text-muted-foreground">Session B</label>
                <select
                  className="w-full rounded-md border border-border px-3 py-2 text-sm"
                  value={sessionB}
                  onChange={(e) => setSessionB(e.target.value)}
                >
                  <option value="">— Sélectionner —</option>
                  {data.available_sessions.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Cards résumé */}
            <div className="grid gap-4 sm:grid-cols-2">
              <SessionCard label={data.session_a?.session_label ?? ''} entry={data.session_a} side="A" />
              <SessionCard label={data.session_b?.session_label ?? ''} entry={data.session_b} side="B" />
            </div>

            {hasComparisonSelection ? (
              <>
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Répartition des résultats</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.outcomes_chart ? (
                      <PlotlyChart figure={data.outcomes_chart} />
                    ) : (
                      <EmptyStateNotice
                        title="Répartition indisponible"
                        description="Le comparatif n'a renvoyé aucun donut de résultats pour les sessions choisies."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Radar de performance</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.radar_chart ? (
                      <PlotlyChart figure={data.radar_chart} />
                    ) : (
                      <EmptyStateNotice
                        title="Radar indisponible"
                        description="Aucun radar de performance n'a été généré pour cette paire de sessions."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Résumé des écarts</CardTitle>
                  </CardHeader>
                  <CardContent>
                    {data.metrics.length > 0 && data.session_a && data.session_b ? (
                      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                        {(['kd_ratio', 'win_rate', 'kills_per_match', 'score'] as const).flatMap((key) => {
                          const row = data.metrics.find((m) => m.key === key)
                          if (!row) return []
                          const delta = row.delta ? parseFloat(row.delta) : null
                          return [
                            <DeltaCard
                              key={key}
                              label={row.label}
                              value={row.value_a}
                              delta={delta}
                              lowerIsBetter={false}
                            />,
                          ]
                        })}
                      </div>
                    ) : (
                      <EmptyStateNotice
                        title="Résumé indisponible"
                        description="Aucune métrique résumée n'a été calculée pour les sessions sélectionnées."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Métriques comparées</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.metrics.length > 0 ? (
                      <table className="w-full">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            <th className="py-2 pr-4">Métrique</th>
                            <th className="py-2 pr-4 text-right text-compare-a">Session A</th>
                            <th className="py-2 pr-4 text-right text-compare-b">Session B</th>
                            <th className="py-2 text-right">Delta</th>
                          </tr>
                        </thead>
                        <tbody>
                          {data.metrics.map((row) => (
                            <MetricRow key={row.key} row={row} />
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title="Aucune métrique détaillée"
                        description="Le tableau comparatif est vide pour les sessions choisies."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Progression K/D</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4">
                    {data.kd_progression_chart ? (
                      <PlotlyChart figure={data.kd_progression_chart} />
                    ) : (
                      <EmptyStateNotice
                        title="Progression indisponible"
                        description="La progression K/D n'a pas pu être tracée pour ces sessions."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Par carte</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.maps_table.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            {Object.keys(data.maps_table[0]).map((k) => (
                              <th key={k} className="py-2 pr-4">{k}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {data.maps_table.map((row, i) => (
                            <tr key={i} className="border-b last:border-0">
                              {Object.values(row).map((v, j) => (
                                <td key={j} className="py-1.5 pr-4 text-foreground">
                                  {String(v ?? '-')}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title="Aucune comparaison par carte"
                        description="Aucune ligne par carte n'est disponible pour les sessions sélectionnées."
                      />
                    )}
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-base">Par mode de jeu</CardTitle>
                  </CardHeader>
                  <CardContent className="pb-4 overflow-x-auto">
                    {data.modes_table.length > 0 ? (
                      <table className="w-full text-sm">
                        <thead>
                          <tr className="border-b text-left text-xs text-muted-foreground">
                            {Object.keys(data.modes_table[0]).map((k) => (
                              <th key={k} className="py-2 pr-4">{k}</th>
                            ))}
                          </tr>
                        </thead>
                        <tbody>
                          {data.modes_table.map((row, i) => (
                            <tr key={i} className="border-b last:border-0">
                              {Object.values(row).map((v, j) => (
                                <td key={j} className="py-1.5 pr-4 text-foreground">
                                  {String(v ?? '-')}
                                </td>
                              ))}
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    ) : (
                      <EmptyStateNotice
                        title="Aucune comparaison par mode"
                        description="Le backend n'a renvoyé aucune ventilation par mode pour cette paire de sessions."
                      />
                    )}
                  </CardContent>
                </Card>
              </>
            ) : (
              <EmptyStateCard
                title="Comparaison incomplète"
                description="Sélectionne une session A et une session B pour afficher les graphiques et tableaux comparatifs."
              />
            )}
          </>
        ) : (
          <EmptyStateCard
            title="Aucune session disponible"
            description="Aucune session n'est disponible dans le scope courant. Vérifie le découpage de sessions ou élargis les filtres."
          />
        )}
      </div>
    </div>
  )
}
