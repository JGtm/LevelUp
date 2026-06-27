/**
 * RelationsMomentsSection — section « Moments & Rivalités » (Phase 3a).
 *
 * Affichée en permanence : la donnée (heatmap relation × tranche/jour + cartes
 * revanche) est chargée d'office via useRelationsMoments. Heatmap « Quand tu les
 * croises » avec toggle tranche horaire / jour de semaine (#8). Frises + WR
 * glissant via wrappers existants. Strings via palmares.toml (FR/EN).
 */
import { useState } from 'react'

import { Spinner } from '@/components/ui/spinner'
import type { FilterContextInput } from '@/lib/api/types'

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

type HeatmapMode = 'daypart' | 'day'

export function RelationsMomentsSection({ playerSlug, filterContext, filterHash, text }: Props) {
  const { data, isLoading, isError } = useRelationsMoments(playerSlug, filterContext, filterHash, true)
  const [mode, setMode] = useState<HeatmapMode>('daypart')

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
