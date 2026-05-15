/**
 * CareerCitationsTab — onglet Citations du hub Carrière.
 * Affiche les citations groupées par catégorie avec barre de progression X/Y — ZZZ%
 * vers le prochain palier. Scope global (citations cumulatives, hors filtre période).
 */
import { useParams } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateCard } from '@/components/ui/empty-state'
import { CitationProgressRing } from '@/components/ui/citation-progress-ring'
import { useCitationsPage } from '@/features/citations/queries'
import { DEFAULT_FILTER_CONTEXT } from '@/stores/createFilterStore'
import { formatMessage } from '@/lib/i18n/format'
import { citationsManifest, type CitationsManifestKey } from '@/lib/i18n/generated/citations'
import { useAppShellStore } from '@/stores/appShellStore'
import type { CitationItem } from '@/lib/api/types'

export function CareerCitationsTab() {
  const { playerSlug } = useParams({ strict: false }) as { playerSlug: string }
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CitationsManifestKey) => formatMessage(citationsManifest, key, locale)

  const { data, isLoading, isError, refetch } = useCitationsPage(
    playerSlug,
    { filters: DEFAULT_FILTER_CONTEXT },
    'hub-all',
  )

  if (isLoading) return null

  if (isError) {
    return (
      <div className="p-6">
        <Card>
          <CardContent className="py-8 text-center">
            <p className="font-medium text-destructive">{t('citations.errors.load_failed')}</p>
            <button onClick={() => refetch()} className="mt-2 text-sm text-primary underline">
              {t('citations.errors.retry')}
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
          title={t('citations.empty.no_data')}
          description={t('citations.empty.no_data_description')}
          actionLabel={t('citations.errors.retry')}
          onAction={() => refetch()}
        />
      </div>
    )
  }

  const byCategory = data.citations_by_category ?? []
  const totalUnique = data.citations.length
  const totalCompleted = byCategory.reduce((acc, g) => acc + g.completed, 0)

  return (
    <div className="space-y-6 p-6">
      <h2 className="text-sm font-semibold">
        {formatMessage(citationsManifest, 'citations.section.mastery_title', locale, { completed: totalCompleted, total: totalUnique })}
      </h2>

      {byCategory.length === 0 ? (
        <EmptyStateCard
          title={t('citations.empty.no_items')}
          description={t('citations.empty.no_items_description')}
        />
      ) : (
        byCategory.map((group) => (
          <div key={group.category} className="rounded-lg border border-border bg-card">
            <div className="border-b border-border px-3 py-2 text-sm font-medium flex items-center justify-between">
              <span>{group.category.charAt(0).toUpperCase() + group.category.slice(1)}</span>
              <span className="text-xs font-normal text-muted-foreground">
                {group.completed} / {group.items.length} {t('citations.category.completed_suffix')}
              </span>
            </div>
            <div className="p-3">
              <div className="flex flex-wrap gap-x-5 gap-y-4">
                {group.items.map((c) => (
                  <CitationCard key={c.name_norm} citation={c} />
                ))}
              </div>
            </div>
          </div>
        ))
      )}
    </div>
  )
}

function CitationCard({ citation }: { citation: CitationItem }) {
  const isMastered = citation.mastery_pct >= 100
  const hasTiers = citation.tier_count > 0
  const tooltip = citation.description
    ? `${citation.name_display} : ${citation.description}`
    : citation.name_display

  return (
    <div title={tooltip} className="flex flex-col items-center gap-1 cursor-default w-[98px]">
      <CitationProgressRing
        pct={citation.mastery_pct}
        imageUrl={citation.image_url ?? undefined}
        isNewlyMastered={isMastered}
        size={68}
      />
      <span className="text-[12px] text-muted-foreground leading-tight text-center w-full truncate">
        {citation.name_display}
      </span>
      {hasTiers && (
        <span className="text-[12px] font-semibold text-foreground/80 leading-none">
          {citation.earned_tiers}/{citation.tier_count}
        </span>
      )}
      {hasTiers && !isMastered && (
        <span className="text-[11px] text-muted-foreground">
          {citation.total}/{citation.next_tier_target}
        </span>
      )}
    </div>
  )
}
