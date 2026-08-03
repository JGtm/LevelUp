/**
 * RelationsMomentsSection — section « Moments & Rivalités » (Phase 3a).
 *
 * Affichée en permanence : la donnée (heatmap relation × tranche/jour + cartes
 * revanche) est chargée d'office via useRelationsMoments. Heatmap « Quand tu les
 * croises » avec toggle tranche horaire / jour de semaine (#8). Frises + WR
 * glissant via wrappers existants. Strings via palmares.toml (FR/EN).
 */
import { useNavigate } from '@tanstack/react-router'
import { useTitleSlug } from '@/lib/title-routing'

import { Spinner } from '@/components/ui/spinner'
import type { FilterContextInput } from '@/lib/api/types'
import { useRelationsPrefsStore } from '@/stores/relationsPrefsStore'

import type { PalmaresText } from './i18n'
import { useRelationsMoments } from './queries'
import {
  RelationsMomentsHeatmap,
  type HeatmapBucketCell,
  type HeatmapTooltipText,
} from './RelationsMomentsHeatmap'
import { RelationsRivalryCards } from './RelationsRivalryCards'

// Libellés des 24 créneaux horaires (index = heure 0..23). Neutres FR/EN
// (« 00h »…« 23h »), donc hors i18n — aligné sur ExplorerActivityHeatmapChart.
const HOUR_LABELS = Array.from({ length: 24 }, (_, h) => `${String(h).padStart(2, '0')}h`)

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
  const navigate = useNavigate()
  const titleSlug = useTitleSlug()

  // Clic sur un duel de la frise revanche → match view (playerSlug déjà en props).
  const onMatchClick = (matchId: string) => {
    void navigate({
      to: '/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId',
      params: { titleSlug, playerSlug, matchId },
    })
  }

  // Mapping vers la cellule générique (bucket) selon le mode du toggle.
  // Mode jour : le back-end renvoie day_of_week avec 0=dimanche … 6=samedi. On
  // décale en lundi-first pour l'affichage (lundi→0 … dimanche→6) et on fait
  // tourner les labels d'un cran en miroir, pour garder données et étiquettes
  // alignées (text.dayLabels reste en ordre back-end 0=Dim … 6=Sam).
  const cells: HeatmapBucketCell[] =
    mode === 'hour'
      ? (data?.heatmap ?? []).map((c) => ({ xuid: c.xuid, gamertag: c.gamertag, bucket: c.hour, count: c.count }))
      : (data?.heatmap_dow ?? []).map((c) => ({
          xuid: c.xuid,
          gamertag: c.gamertag,
          bucket: (c.day_of_week + 6) % 7,
          count: c.count,
        }))
  const bucketLabels =
    mode === 'hour' ? HOUR_LABELS : [...text.dayLabels.slice(1), text.dayLabels[0]]

  // Libellés du tooltip : l'étiquette du bucket suit le toggle (créneau / jour).
  const tooltipText: HeatmapTooltipText = {
    playerLabel: text.tooltipPlayer,
    bucketTypeLabel: mode === 'hour' ? text.tooltipHourSlot : text.tooltipDaySlot,
    matchesLabel: text.tooltipMatches,
    emptyCell: text.tooltipEmptyCell,
  }

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
              {(['hour', 'day'] as const).map((m) => (
                <button
                  key={m}
                  type="button"
                  aria-pressed={mode === m}
                  onClick={() => setMode(m)}
                  className={`rounded-md px-3 py-1 text-xs font-medium transition-colors ${
                    mode === m ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:text-foreground'
                  }`}
                >
                  {m === 'hour' ? text.heatmapByHour : text.heatmapByDay}
                </button>
              ))}
            </div>
            <RelationsMomentsHeatmap
              cells={cells}
              bucketLabels={bucketLabels}
              legendLabel={text.heatmapLegend}
              emptyMessage={text.heatmapEmpty}
              tooltipText={tooltipText}
            />
          </div>
        )}
      </section>

      {/* Section « Rivaux » : cartes de rivaux (sœur de la précédente) */}
      {!isLoading && data && (
        <section className="flex flex-col gap-3">
          <h2 className="text-base font-semibold text-foreground">{text.rivalriesTitle}</h2>
          <RelationsRivalryCards rivalries={data.rivalries ?? []} t={text} onMatchClick={onMatchClick} />
        </section>
      )}
    </>
  )
}
