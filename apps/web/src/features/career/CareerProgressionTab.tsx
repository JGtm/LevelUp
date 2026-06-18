/**
 * CareerProgressionTab — section Progression du hub Carrière.
 * Sprint 55 : rang, XP, LUSR, charts — sans filtres (API career non filtrable).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { Spinner } from '@/components/ui/spinner'
import { CareerChartsSection } from './CareerChartsSection'
import { CareerRankingBlock } from './CareerRankingBlock'
import { AchievementsCareerSection } from '@/features/achievements/AchievementsCareerSection'
import { useCapability } from '@/lib/capabilities/capabilities'
import { CareerHighlightMatchesSection } from './CareerHighlightMatchesSection'
import { CareerTopEncountersSection } from './CareerTopEncountersSection'
import { CareerRivalsSection } from './CareerRivalsSection'
import { useCareerPage } from './queries'
import { careerManifest } from '@/lib/i18n/generated/career'
import type { ManifestLocale } from '@/lib/i18n/format'
import { useAppShellStore } from '@/stores/appShellStore'

export function CareerProgressionTab() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale
  const t = (key: keyof typeof careerManifest) => careerManifest[key][locale]
  const { data, isLoading, isError, refetch } = useCareerPage(playerSlug)
  // Sidebar Succès Xbox gatée sur `achievements` : on passe `undefined` (et non un
  // FeatureGate rendant null) pour que CareerChartsSection replie sa grille au lieu
  // de réserver une colonne droite vide. NO-OP pour halo_infinite.
  const hasAchievements = useCapability('achievements')

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
      {/* Graphiques carrière (career.01–04) + succès Xbox en colonne droite + Classements à gauche de l'évolution LUSR */}
      <CareerChartsSection
        xpHistory={data.xp_history ?? []}
        lusrCheckpoints={data.lusr?.checkpoints ?? []}
        summary={data.summary}
        heroProgress={data.hero_progress}
        projections={data.projections ?? null}
        friendsXpHistory={data.friends_xp_history}
        rightSlot={
          hasAchievements ? (
            <AchievementsCareerSection playerSlug={playerSlug} layout="sidebar" />
          ) : undefined
        }
        lusrLeftSlot={<CareerRankingBlock playerSlug={playerSlug} lusrData={data.lusr} />}
      />

      {/* Matchs marquants — toggle Best/Worst (15 chacun, format Explorer) */}
      <CareerHighlightMatchesSection />

      {/* Joueurs les plus croisés (hors amis) — top 10 (format Match View encounter) */}
      <CareerTopEncountersSection />

      {/* Top némésis + Top souffre-douleur côte à côte (10 + 10) */}
      <CareerRivalsSection />

    </div>
  )
}
