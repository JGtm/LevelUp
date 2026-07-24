/**
 * UnifiedCitationsPage — page Citations UNIQUE pour tous les titres. La `source` est
 * fixée par la ROUTE (`/citations` → infinite, `/commendations` → native), pas par une
 * détection de slug interne : les routes/nav restent inchangées et la source native ne
 * monte jamais la barre de filtres (zéro fetch inutile). Les deux sources normalisent
 * leurs données vers le même view-model et rendent la même CitationsView — UI identique,
 * seuls le calcul/les données/les images diffèrent (objectif title-agnostic).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { useCitationsPage } from './queries'
import { useCommendationTotals } from '@/features/commendations/queries'
import { useLocalFilterBar } from '@/features/_shared/useLocalFilterBar'
import { CitationsView } from './CitationsView'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { citationsManifest, type CitationsManifestKey } from '@/lib/i18n/generated/citations'
import { useAppShellStore } from '@/stores/appShellStore'
import { normalizeInfinitePage, normalizeNativeTotals } from '@/lib/citations/normalize'
import type { CitationSource, CitationsViewModel } from '@/lib/citations/types'

export function UnifiedCitationsPage({ source }: { source: CitationSource }) {
  // Titre d'onglet : géré par le résolveur global (lib/pageTitle.ts, distingue
  // /career/citations vs /career/commendations par le pathname — même nuance FR
  // Citations / EN Citations|Commendations qu'ici auparavant).
  // Chaque source appelle EXACTEMENT un data-hook → règles des hooks respectées.
  return source === 'native' ? <CitationsNativeSource /> : <CitationsInfiniteSource />
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function masteryHeader(vm: CitationsViewModel, locale: ManifestLocale): string {
  return formatMessage(citationsManifest, 'citations.section.mastery_title', locale, {
    completed: vm.masteredTotal,
    total: vm.itemsTotal,
  })
}

// ─── Source Infinite (moteur de citations dérivé + barre de filtres) ──────────

function CitationsInfiniteSource() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CitationsManifestKey) => formatMessage(citationsManifest, key, locale)

  const { committedFilterContext, committedHash, bar } = useLocalFilterBar({
    playerSlug,
    labels: {
      experience: t('citations.filters.experience'),
      experienceAll: t('citations.filters.experience_all'),
      experienceRanked: t('citations.filters.experience_ranked'),
      experienceUnranked: t('citations.filters.experience_unranked'),
      playlists: t('citations.filters.playlists'),
      modes: t('citations.filters.modes'),
      reset: t('citations.filters.reset'),
    },
  })

  const { data, isLoading, isError, refetch } = useCitationsPage(
    playerSlug,
    { filters: committedFilterContext },
    committedHash,
  )

  if (isLoading) {
    return <div className="flex flex-col gap-6">{bar}</div>
  }
  if (isError) {
    return (
      <div className="flex flex-col gap-6">
        {bar}
        <div className="px-6">
          <Card>
            <CardContent className="py-8 text-center">
              <p className="font-medium text-destructive">{t('citations.errors.load_failed')}</p>
              <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
                {t('citations.errors.retry')}
              </button>
            </CardContent>
          </Card>
        </div>
      </div>
    )
  }
  if (!data) {
    return (
      <div className="flex flex-col gap-6">
        {bar}
        <div className="px-6">
          <EmptyStateCard
            title={t('citations.empty.no_data')}
            description={t('citations.empty.no_data_description')}
            actionLabel={t('citations.errors.retry')}
            onAction={() => refetch()}
          />
        </div>
      </div>
    )
  }

  const vm = normalizeInfinitePage(data)
  return (
    <CitationsView
      vm={vm}
      locale={locale}
      filterBar={bar}
      headerTitle={masteryHeader(vm, locale)}
      completedSuffix={t('citations.category.completed_suffix')}
      emptyTitle={t('citations.empty.no_items')}
      emptyDescription={t('citations.empty.no_items_description')}
    />
  )
}

// ─── Source native (commendations Halo 5, totaux à vie, sans filtres) ─────────

function CitationsNativeSource() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CitationsManifestKey) => formatMessage(citationsManifest, key, locale)

  const { data, isLoading } = useCommendationTotals(playerSlug)

  if (isLoading) {
    return <div className="px-6 py-8 text-sm text-muted-foreground">…</div>
  }

  // Erreur / pas de données → view-model vide → état vide (dégradation gracieuse,
  // comme un titre sans commendations natives).
  const vm: CitationsViewModel = data
    ? normalizeNativeTotals(data)
    : { categories: [], masteredTotal: 0, itemsTotal: 0, source: 'native', hasFilters: false }

  return (
    <CitationsView
      vm={vm}
      locale={locale}
      headerTitle={masteryHeader(vm, locale)}
      completedSuffix={t('citations.category.completed_suffix')}
      emptyTitle={t('citations.empty.no_items')}
      emptyDescription={t('citations.empty.no_items_description')}
    />
  )
}
