/**
 * RelationsMomentsSection — section « Moments & Rivalités » (Phase 3a).
 *
 * Affichée en permanence : la donnée (heatmap relation × tranche/jour + cartes
 * revanche) est chargée d'office via useRelationsMoments (hérite de la
 * segmentation serveur committed). Heatmap aligné Explorer (cold→hot), frises +
 * WR glissant via wrappers existants. Strings via palmares.toml (FR/EN).
 */
import { Spinner } from '@/components/ui/spinner'
import type { FilterContextInput } from '@/lib/api/types'

import type { PalmaresText } from './i18n'
import { useRelationsMoments } from './queries'
import { RelationsMomentsHeatmap } from './RelationsMomentsHeatmap'
import { RelationsRivalryCards } from './RelationsRivalryCards'

interface Props {
  playerSlug: string
  filterContext: FilterContextInput
  filterHash: string
  text: PalmaresText['relations']['moments']
}

export function RelationsMomentsSection({ playerSlug, filterContext, filterHash, text }: Props) {
  const { data, isLoading, isError } = useRelationsMoments(playerSlug, filterContext, filterHash, true)

  return (
    <section className="flex flex-col gap-3" data-testid="palmares-relations-moments">
      <h2 className="text-base font-semibold text-foreground">{text.sectionTitle}</h2>
      <div className="flex flex-col gap-6">
        {isLoading && (
          <div className="flex items-center justify-center py-12">
            <Spinner size="md" />
          </div>
        )}
        {!isLoading && (isError || !data) && (
          <p className="text-sm text-muted-foreground">{text.unavailable}</p>
        )}
        {!isLoading && data && (
          <>
            <div className="flex flex-col gap-2">
              <RelationsMomentsHeatmap
                cells={data.heatmap ?? []}
                daypartLabels={text.dayparts}
                title={text.heatmapTitle}
                legendLabel={text.heatmapLegend}
                emptyMessage={text.heatmapEmpty}
                matchesLabel={(count) => `${text.heatmapLegend} : ${count}`}
              />
              <p className="text-xs text-muted-foreground">{text.heatmapHelp}</p>
            </div>

            <div className="flex flex-col gap-3">
              <h3 className="text-sm font-semibold text-foreground">{text.rivalriesTitle}</h3>
              <RelationsRivalryCards rivalries={data.rivalries ?? []} t={text} />
            </div>
          </>
        )}
      </div>
    </section>
  )
}
