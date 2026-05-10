/**
 * CareerProgressionTab — section Progression du hub Carrière.
 * Sprint 55 : rang, XP, LUSR, charts — sans filtres (API career non filtrable).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerLusrCards } from './CareerLusrCards'
import { AchievementsCareerSection } from '@/features/achievements/AchievementsCareerSection'
import { useCareerPage } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'

export function CareerProgressionTab() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)

  if (isLoading) {
    return (
      <div className="flex h-64 items-center justify-center">
        <Spinner size="lg" label={t('career.errors.loading_progression')} />
      </div>
    )
  }

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('career.errors.load_progression_failed')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('career.errors.retry')}
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
          title={t('career.empty.progression_unavailable')}
          description={t('career.empty.no_response_description')}
          actionLabel={t('career.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  return (
    <div className="space-y-6 p-6">
      {/* Graphiques carrière (career.01–04) + succès Xbox en colonne droite */}
      <CareerChartsSection
        xpHistory={data.xp_history ?? []}
        lusrCheckpoints={data.lusr?.checkpoints ?? []}
        summary={data.summary}
        heroProgress={data.hero_progress}
        projections={data.projections ?? null}
        friendsXpHistory={data.friends_xp_history}
        rightSlot={<AchievementsCareerSection playerSlug={playerSlug} layout="sidebar" />}
      />

      {/* career.11 — grille LUSR par playlist */}
      <CareerLusrCards checkpoints={data.lusr?.checkpoints ?? []} />

    </div>
  )
}
