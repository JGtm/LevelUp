/**
 * CareerPage — page principale de la carrière du joueur.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { PageHeader } from '@/components/shell/PageHeader'
import { Spinner } from '@/components/ui/spinner'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { CareerSummaryCard } from './CareerSummaryCard'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerTopMatchesTable } from './CareerTopMatchesTable'
import { CareerEncountersSection } from './CareerEncountersSection'
import { useCareerPage, useCareerTopMatches } from './queries'

export function CareerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)
  const [showAllTopMatches, setShowAllTopMatches] = useState(false)
  const { data: fullTopMatches, isLoading: loadingTopMatches } = useCareerTopMatches(
    playerSlug,
    showAllTopMatches,
  )

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

  const topMatchesItems =
    showAllTopMatches && fullTopMatches
      ? fullTopMatches.items
      : data.top_matches_preview

  return (
    <div className="flex flex-col">
      <PageHeader
        title="Carrière"
        subtitle="Progression de rang et statistiques globales"
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
            <CardHeader>
              <CardTitle className="text-base">Rating LUSR</CardTitle>
            </CardHeader>
            <CardContent className="space-y-1">
              <p className="text-sm font-medium text-gray-700">
                Actuel :{' '}
                <span className="text-purple-700 font-bold">
                  {data.lusr.current_rating ?? '—'}
                </span>
                {data.lusr.current_tier_label && (
                  <> · <span className="text-gray-600">{data.lusr.current_tier_label}</span></>
                )}
                {data.lusr.current_playlist_group && (
                  <span className="ml-2 text-xs text-gray-400">
                    ({data.lusr.current_playlist_group})
                  </span>
                )}
              </p>
              {data.lusr.trend_label && (
                <p className="text-xs text-gray-500">{data.lusr.trend_label}</p>
              )}
            </CardContent>
          </Card>
        )}

        {/* Top matchs */}
        {topMatchesItems.length > 0 && (
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Meilleurs matchs</CardTitle>
            </CardHeader>
            <CardContent>
              {loadingTopMatches ? (
                <div className="flex justify-center py-4">
                  <Spinner size="sm" label="Chargement…" />
                </div>
              ) : (
                <>
                  <CareerTopMatchesTable items={topMatchesItems} />
                  {!showAllTopMatches && (
                    <div className="mt-4 flex justify-center">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => setShowAllTopMatches(true)}
                      >
                        Voir tous les top matchs
                      </Button>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        )}

        {/* Rencontres fréquentes */}
        <CareerEncountersSection
          playerSlug={playerSlug}
          preview={data.encounters_preview}
        />
      </div>
    </div>
  )
}
