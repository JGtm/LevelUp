/**
 * CareerPage — page principale de la carrière du joueur.
 */
import { useState } from 'react'
import { useParams, useNavigate } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Spinner } from '@/components/ui/spinner'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { CareerSummaryCard } from './CareerSummaryCard'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerTopMatchesTable } from './CareerTopMatchesTable'
import { CareerEncountersSection } from './CareerEncountersSection'
import { CareerRankingBlock } from './CareerRankingBlock'
import { AchievementsCareerSection } from '@/features/achievements/AchievementsCareerSection'
import { useCareerPage, useCareerTopMatches } from './queries'


export function CareerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const navigate = useNavigate()
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)
  const [showAllTopMatches, setShowAllTopMatches] = useState(false)
  const { data: fullTopMatches, isLoading: loadingTopMatches } = useCareerTopMatches(
    playerSlug,
    showAllTopMatches,
  )

  if (isLoading) return null

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement de la carrière.</p>
            <button
              onClick={() => refetch()}
              className="mt-2 text-sm text-primary underline"
            >
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
          title="Données de carrière indisponibles"
          description="Aucune réponse exploitable n'a été renvoyée pour cette page. Vérifie la source de données ou relance le chargement."
          actionLabel="Réessayer"
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const topMatchesItems =
    showAllTopMatches && fullTopMatches
      ? fullTopMatches.items
      : data.top_matches_preview

  return (
    <div className="flex flex-col">
      <div className="space-y-6 p-6">
        <div className="flex justify-end">
          <Button
            size="sm"
            variant="outline"
            onClick={() => void navigate({ to: '/players/$playerSlug/compare', params: { playerSlug } })}
          >
            Comparer
          </Button>
        </div>
        {/* Résumé + progression */}
        <CareerSummaryCard
          summary={data.summary}
          heroProgress={data.hero_progress}
          projections={data.projections}
        />

        {/* Graphiques ECharts (Phase 2 P2.B — migrés de Plotly) */}
        <CareerChartsSection
          xpHistory={data.xp_history ?? []}
          lusrCheckpoints={data.lusr?.checkpoints ?? []}
          summary={data.summary}
          heroProgress={data.hero_progress}
          projections={data.projections ?? null}
          friendsXpHistory={data.friends_xp_history}
        />

        {/* Succès Xbox — section horizontale (KPI inline + scroll de cartes) */}
        <AchievementsCareerSection playerSlug={playerSlug} />

        {/* Section CSR + LUSR unifiée */}
        <CareerRankingBlock playerSlug={playerSlug} lusrData={data.lusr} />

        {/* Top matchs */}
        {(topMatchesItems?.length ?? 0) > 0 ? (
          <div className="space-y-4">
            {loadingTopMatches ? (
              <div className="flex justify-center py-4">
                <Spinner size="sm" label="Chargement…" />
              </div>
            ) : showAllTopMatches ? (
              <>
                <CareerTopMatchesTable
                  items={topMatchesItems}
                  variant="best"
                  playerSlug={playerSlug}
                />
                <CareerTopMatchesTable
                  items={topMatchesItems}
                  variant="worst"
                  playerSlug={playerSlug}
                />
              </>
            ) : (
              <>
                <CareerTopMatchesTable
                  items={topMatchesItems}
                  title="Meilleurs matchs récents"
                  playerSlug={playerSlug}
                />
                <div className="flex justify-center">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowAllTopMatches(true)}
                  >
                    Voir tous les top matchs
                  </Button>
                </div>
              </>
            )}
          </div>
        ) : (
          <EmptyStateCard
            title="Top matchs indisponibles"
            description="Aucun match distinctif n'a été calculé pour cette page de carrière."
          />
        )}

        {/* Rencontres fréquentes */}
        <CareerEncountersSection
          playerSlug={playerSlug}
          preview={data.encounters_preview ?? []}
        />
      </div>

    </div>
  )
}
