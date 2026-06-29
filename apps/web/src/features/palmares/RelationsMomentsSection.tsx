/**
 * RelationsMomentsSection — section « Moments & Rivalités » (Phase 3a).
 *
 * Affichée en permanence : la donnée (heatmap relation × tranche/jour + cartes
 * revanche) est chargée d'office via useRelationsMoments. Heatmap « Quand tu les
 * croises » avec toggle tranche horaire / jour de semaine (#8). Frises + WR
 * glissant via wrappers existants. Strings via palmares.toml (FR/EN).
 */
import { Spinner } from '@/components/ui/spinner'
import type { FilterContextInput } from '@/lib/api/types'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import type { PalmaresText } from './i18n'
import { useRelationsMoments } from './queries'
import { RelationsMomentsHeatmap, type HeatmapBucketCell } from './RelationsMomentsHeatmap'
import { RelationsRivalryCards } from './RelationsRivalryCards'

interface Props {
  playerSlug: string
  filterContext: FilterContextInput
  filterHash: string
  text: PalmaresText['relations']['moments']
}

export function RelationsMomentsSection({ playerSlug, filterContext, filterHash, text }: Props) {
  const { data, isLoading, isError } = useRelationsMoments(playerSlug, filterContext, filterHash, true)
  const mode = useRelationsPrefsStore((s) => s.heatmapMode)
  const setMode = useRelationsPrefsStore((s) => s.setHeatmapMode)

  // Mapping vers la cellule générique (bucket) selon le mode du toggle.
  const cells: HeatmapBucketCell[] =
    mode === 'daypart'
      ? (data?.heatmap ?? []).map((c) => ({ xuid: c.xuid, gamertag: c.gamertag, bucket: c.daypart, count: c.count }))
      : (data?.heatmap_dow ?? []).map((c) => ({
          xuid: c.xuid,
          gamertag: c.gamertag,
          bucket: c.day_of_week,
          count: c.count,
        }))
  const bucketLabels = mode === 'daypart' ? text.dayparts : text.dayLabels

  return (
    <>
      {/* Section « Rythme des rencontres » : heatmap relation × créneau */}
      <section className="flex flex-col gap-3" data-testid="palmares-relations-moments">
        <h2 className="text-base font-semibold text-foreground">{text.sectionTitle}</h2>
        {isLoading && (
          <div className="flex items-center justify-center py-12">
            <Spinner size="md" />
          </div>
        )}
        {!isLoading && (isError || !data) && (
          <p className="text-sm text-muted-foreground">{text.unavailable}</p>
        )}
        {!isLoading && data && (
          <div className="flex flex-col gap-2">
            <div
              className="inline-flex self-start rounded-lg border border-border bg-card p-0.5"
              data-testid="palmares-heatmap-mode"
            >
              {(['daypart', 'day'] as const).map((m) => (
                <button
                  key={m}
                  type="button"
                  aria-pressed={mode === m}
                  onClick={() => setMode(m)}
                  className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                    mode === m ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {m === 'daypart' ? text.heatmapByDaypart : text.heatmapByDay}
                </button>
              ))}
            </div>
            <RelationsMomentsHeatmap
              cells={cells}
              bucketLabels={bucketLabels}
              legendLabel={text.heatmapLegend}
              emptyMessage={text.heatmapEmpty}
              matchesLabel={(count) => `${text.heatmapLegend} : ${count}`}
            />
          </div>
        )}
      </section>

      {/* Section « Rivaux » : cartes de rivaux (sœur de la précédente) */}
      {!isLoading && data && (
        <section className="flex flex-col gap-3">
          <h2 className="text-base font-semibold text-foreground">{text.rivalriesTitle}</h2>
          <RelationsRivalryCards rivalries={data.rivalries ?? []} t={text} />
        </section>
      )}
    </>
  )
}
