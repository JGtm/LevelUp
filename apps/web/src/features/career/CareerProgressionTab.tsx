/**
 * CareerProgressionTab — onglet Progression du hub Carrière.
 * Sprint 55 : extrait de CareerPage, contient rang, XP, LUSR, charts, XP history.
 * Exclut top_matches et encounters (migrés vers Synthèse).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { EmptyStateCard, EmptyStateNotice } from '@/components/ui/empty-state'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { Spinner } from '@/components/ui/spinner'
import { CareerSummaryCard } from './CareerSummaryCard'
import { CareerChartsSection } from './CareerChartsSection'
import { useCareerPage } from './queries'
import { formatMessage } from '@/lib/i18n/format'
import { careerManifest, type CareerManifestKey } from '@/lib/i18n/generated/career'
import { useAppShellStore } from '@/stores/appShellStore'
import { CompareDrawer } from '@/features/compare/CompareDrawer'
import { LeaderboardBlock } from '@/features/leaderboard/LeaderboardBlock'
import { useComparePrefetch } from '@/features/compare/queries'
import { useState } from 'react'

function isCareerLocale(loc: string | undefined): loc is 'fr' | 'en' {
  return loc === 'fr' || loc === 'en'
}

export function CareerProgressionTab() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)
  const [compareOpen, setCompareOpen] = useState(false)
  const prefetchCompare = useComparePrefetch(playerSlug)
  const rawLocale = useAppShellStore((s) => s.locale)
  const locale = isCareerLocale(rawLocale) ? rawLocale : 'fr'
  const t = (key: CareerManifestKey) => formatMessage(careerManifest, key, locale)

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Spinner size="lg" label="Chargement de la progression…" />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">Erreur lors du chargement de la progression.</p>
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
          title="Données de progression indisponibles"
          description="Aucune réponse exploitable n'a été renvoyée pour cette page."
          actionLabel="Réessayer"
          onAction={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      <div className="flex justify-end">
        <Button size="sm" variant="outline" onClick={() => setCompareOpen(true)}>
          Comparer
        </Button>
      </div>

      {/* Résumé + progression rang */}
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
          rankProgressTitle: t('career.charts.rank_progress'),
          heroProgressTitle: t('career.charts.hero_progress'),
          xpHistoryTitle: t('career.charts.xp_history_title'),
          xpHistoryAxisY: t('career.charts.xp_history_axis_y'),
          lusrRatingTitle: t('career.charts.lusr_rating_title'),
          lusrRatingAxisY: t('career.charts.lusr_rating_axis_y'),
          placeholderUnavailable: t('career.charts.placeholder_unavailable'),
          placeholderDescription: t('career.charts.placeholder_description'),
        }}
      />

      {/* Section LUSR */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">
            Rating LUSR{' '}
            <InfoTooltip content="Le LUSR (LevelUp Skill Rating) est un rating calculé localement à partir de vos résultats récents. Il reflète votre niveau indépendamment du CSR officiel, en pondérant victoires, KDA et régularité." />
          </CardTitle>
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
                  <>
                    {' '}
                    · <span className="text-muted-foreground">{data.lusr.current_tier_label}</span>
                  </>
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

      {/* Leaderboard CSR */}
      <LeaderboardBlock playerSlug={playerSlug} onHoverEntry={prefetchCompare} />

      <CompareDrawer
        playerSlug={playerSlug}
        open={compareOpen}
        onClose={() => setCompareOpen(false)}
      />
    </div>
  )
}
