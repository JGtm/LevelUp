/**
 * CareerPage — page principale de la carrière du joueur.
 */
import { useState } from 'react'
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { CareerSummaryCard } from './CareerSummaryCard'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerTopMatchesTable } from './CareerTopMatchesTable'
import { CareerEncountersSection } from './CareerEncountersSection'
import { useCareerPage, useCareerTopMatches } from './queries'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { LeaderboardBlock } from '@/features/leaderboard/LeaderboardBlock'
import { useComparePrefetch } from '@/features/compare/queries'

export function CareerPage() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)
  const [showAllTopMatches, setShowAllTopMatches] = useState(false)
  const [compareOpen, setCompareOpen] = useState(false)
  const prefetchCompare = useComparePrefetch(playerSlug)
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
          <Button size="sm" variant="outline" onClick={() => setCompareOpen(true)}>
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
          labels={{
            rankProgressTitle: 'Progression du rang',
            heroProgressTitle: 'Progression Héros',
            xpHistoryTitle: 'Historique XP',
            xpHistoryAxisY: 'XP cumulé',
            lusrRatingTitle: 'Rating LUSR',
            lusrRatingAxisY: 'Rating',
            placeholderUnavailable: 'Graphique indisponible',
            placeholderDescription: 'Aucune donnée exploitable.',
          }}
        />

        {/* Section LUSR */}
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Rating LUSR <InfoTooltip content="Le LUSR (LevelUp Skill Rating) est un rating calculé localement à partir de vos résultats récents. Il reflète votre niveau indépendamment du CSR officiel, en pondérant victoires, KDA et régularité." /></CardTitle>
          </CardHeader>
          <CardContent className="space-y-1">
            {data.lusr ? (
              <>
                <p className="text-sm font-medium text-foreground">
                  Actuel :{' '}
                  <span className="text-primary font-bold">
                    {data.lusr.current_rating ?? '—'}
                  </span>
                  {data.lusr.current_tier_label && (
                    <> · <span className="text-muted-foreground">{data.lusr.current_tier_label}</span></>
                  )}
                  {data.lusr.current_playlist_group && (
                    <span className="ml-2 text-xs text-muted-foreground">
                      ({data.lusr.current_playlist_group})
                    </span>
                  )}
                </p>
                {data.lusr.trend_label && (
                  <p className="text-xs text-muted-foreground">{data.lusr.trend_label}</p>
                )}
              </>
            ) : (
              <EmptyStateNotice
                title="Rating indisponible"
                description="Aucun checkpoint LUSR n'est disponible pour ce joueur ou cette période."
              />
            )}
          </CardContent>
        </Card>

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

        {/* E2.6 : Leaderboard CSR — bloc secondaire sous les KPIs */}
        <LeaderboardBlock
          playerSlug={playerSlug}
          onHoverEntry={prefetchCompare}
        />

        {/* Rencontres fréquentes */}
        <CareerEncountersSection
          playerSlug={playerSlug}
          preview={data.encounters_preview ?? []}
        />
      </div>

      {/* C4.3 : CompareDrawer ouvert depuis l'en-tête Carrière */}
      <CompareDrawer
        playerSlug={playerSlug}
        open={compareOpen}
        onClose={() => setCompareOpen(false)}
      />
    </div>
  )
}
