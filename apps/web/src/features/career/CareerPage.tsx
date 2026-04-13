/**
 * CareerPage — page principale de la carrière du joueur.
 */
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent } from '@/components/ui/card'
import { CareerSummaryCard } from './CareerSummaryCard'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerTopMatchesTable } from './CareerTopMatchesTable'
import { useCareerPage } from './queries'

export function CareerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner size="lg" label="Chargement de la carrière…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-red-600">Erreur lors du chargement de la carrière.</p>
            <button
              onClick={() => refetch()}
              className="mt-2 text-sm text-purple-600 underline"
            >
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
        title="Carrière"
        subtitle={`Progression de rang et statistiques globales`}
      />

      <div className="space-y-6 p-6">
        {/* Résumé + progression */}
        <CareerSummaryCard
          summary={data.summary}
          heroProgress={data.hero_progress}
          projections={data.projections}
        />

        {/* Graphiques Plotly */}
        <CareerChartsSection charts={data.charts} />

        {/* Section LUSR */}
        {data.lusr && (
          <Card>
            <CardContent className="py-4">
              <p className="text-sm font-medium text-gray-700">
                Rating LUSR actuel :{' '}
                <span className="text-purple-700 font-bold">
                  {data.lusr.current_rating ?? '—'}
                </span>
                {data.lusr.current_tier_label && (
                  <> · {data.lusr.current_tier_label}</>
                )}
              </p>
              {data.lusr.trend_label && (
                <p className="text-xs text-gray-500 mt-1">{data.lusr.trend_label}</p>
              )}
            </CardContent>
          </Card>
        )}

        {/* Top matchs preview */}
        {data.top_matches_preview.length > 0 && (
          <CareerTopMatchesTable items={data.top_matches_preview} />
        )}
      </div>
    </div>
  )
}
