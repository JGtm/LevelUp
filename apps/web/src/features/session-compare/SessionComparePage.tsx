/**
 * SessionComparePage — comparaison A/B de deux sessions.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
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
  const color = side === 'A' ? 'text-blue-700' : 'text-orange-700'
  const bg = side === 'A' ? 'bg-blue-50 border-blue-200' : 'bg-orange-50 border-orange-200'

  return (
    <Card className={`border ${bg}`}>
      <CardHeader className="pb-2">
        <CardTitle className={`text-sm ${color}`}>
          Session {side} {label && `— ${label}`}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-1 text-xs text-gray-700">
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
          <p className="text-gray-400 italic">Non sélectionnée</p>
        )}
      </CardContent>
    </Card>
  )
}

function MetricRow({ row }: { row: SessionCompareMetricRow }) {
  const winnerColor =
    row.winner === 'a'
      ? 'text-blue-700 font-semibold'
      : row.winner === 'b'
        ? 'text-orange-700 font-semibold'
        : 'text-gray-700'

  return (
    <tr className="border-b last:border-0 text-sm">
      <td className="py-2 pr-4 text-gray-600">{row.label}</td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'a' ? 'text-blue-700 font-semibold' : 'text-gray-700'}`}>
        {row.value_a}
      </td>
      <td className={`py-2 pr-4 text-right ${row.winner === 'b' ? 'text-orange-700 font-semibold' : 'text-gray-700'}`}>
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
            <p className="font-medium text-red-600">Erreur lors du chargement.</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-purple-600 underline">
              Réessayer
            </button>
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!data) return null

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Comparaison de sessions"
        subtitle="Sélectionnez deux sessions pour les comparer"
      />

      <div className="space-y-6 p-6">
        {/* Sélecteurs A / B */}
        <div className="grid gap-4 sm:grid-cols-2">
          <div>
            <label className="mb-1 block text-xs font-medium text-gray-600">Session A</label>
            <select
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm"
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
            <label className="mb-1 block text-xs font-medium text-gray-600">Session B</label>
            <select
              className="w-full rounded-md border border-gray-200 px-3 py-2 text-sm"
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

        {/* Radar */}
        {data.radar_chart && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Radar de performance</CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <PlotlyChart figure={data.radar_chart} />
            </CardContent>
          </Card>
        )}

        {/* D2 — Delta cards K/D / Win Rate / Kills/match / Score (A vs B) */}
        {data.metrics.length > 0 && data.session_a && data.session_b && (
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
            {(['kd_ratio', 'win_rate', 'kills_per_match', 'score'] as const).flatMap((key) => {
              const row = data.metrics.find((m) => m.key === key)
              if (!row) return []
              const delta = row.delta ? parseFloat(row.delta) : null
              const lowerIsBetter = false
              return [
                <DeltaCard
                  key={key}
                  label={row.label}
                  value={row.value_a}
                  delta={delta}
                  lowerIsBetter={lowerIsBetter}
                />,
              ]
            })}
          </div>
        )}

        {/* Tableau de métriques */}
        {data.metrics.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Métriques comparées</CardTitle>
            </CardHeader>
            <CardContent className="pb-4 overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="border-b text-left text-xs text-gray-500">
                    <th className="py-2 pr-4">Métrique</th>
                    <th className="py-2 pr-4 text-right text-blue-600">Session A</th>
                    <th className="py-2 pr-4 text-right text-orange-600">Session B</th>
                    <th className="py-2 text-right">Delta</th>
                  </tr>
                </thead>
                <tbody>
                  {data.metrics.map((row) => (
                    <MetricRow key={row.key} row={row} />
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}

        {/* Progression K/D */}
        {data.kd_progression_chart && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Progression K/D</CardTitle>
            </CardHeader>
            <CardContent className="pb-4">
              <PlotlyChart figure={data.kd_progression_chart} />
            </CardContent>
          </Card>
        )}

        {/* Résultats par map */}
        {data.maps_table.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Par carte</CardTitle>
            </CardHeader>
            <CardContent className="pb-4 overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-xs text-gray-500">
                    {Object.keys(data.maps_table[0]).map((k) => (
                      <th key={k} className="py-2 pr-4">{k}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {data.maps_table.map((row, i) => (
                    <tr key={i} className="border-b last:border-0">
                      {Object.values(row).map((v, j) => (
                        <td key={j} className="py-1.5 pr-4 text-gray-700">
                          {String(v ?? '-')}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
