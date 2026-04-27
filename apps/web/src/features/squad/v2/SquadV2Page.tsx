/**
 * SquadV2Page — page de demonstration consommant le payload V2 (chunk S10b).
 *
 * Compose les wrappers ECharts (S10) et les composants tableau/galerie (S10b)
 * sur le payload backend complet (S11). Skeleton minimaliste : la composition
 * finale en plusieurs onglets sera adaptee dans une iteration ulterieure.
 *
 * Usage : `<SquadV2Page playerSlug="..." teammates={["gt1","gt2"]} />`.
 */
import { BarStackedChart } from '@/components/charts/BarStackedChart'
import { BarGroupedChart } from '@/components/charts/BarGroupedChart'
import { TimeseriesLineChart } from '@/components/charts/TimeseriesLineChart'
import { Heatmap2DChart } from '@/components/charts/Heatmap2DChart'
import { RadarChart } from '@/components/charts/RadarChart'

import { HistoryTable } from './components/HistoryTable'
import { WeaponsTable } from './components/WeaponsTable'
import { MedalsGallery } from './components/MedalsGallery'
import { useSquadV2 } from './queries'
import type { SquadPeriod } from './types'

export interface SquadV2PageProps {
  playerSlug: string
  teammates?: string[]
  period?: SquadPeriod
  /** Labels deja localises (le caller resout l'i18n). */
  labels: SquadV2PageLabels
}

export interface SquadV2PageLabels {
  loading: string
  errorPrefix: string
  empty: string
  // Section titles
  synergiesTitle: string
  impactTitle: string
  contributionsTitle: string
  radarTitle: string
  historyTitle: string
  weaponsTitle: string
  medalsTitle: string
  // Chart titles (passes aux wrappers)
  mapBreakdownTitle: string
  timelineTitle: string
  cadenceTitle: string
  intensityTitle: string
  perMinuteTitle: string
  fragsDeathsTitle: string
  // Table labels
  history: {
    date: string
    mode: string
    map: string
    outcome: string
    duration: string
    kdaSuffix: string
  }
  weapons: {
    weapon: string
    total: string
    minKills: (n: number) => string
    grenadeMelee: string
  }
  medals: { emptyMatch: string }
  // Locale ISO pour formatage dates
  locale: string
}

export function SquadV2Page({ playerSlug, teammates, period, labels }: SquadV2PageProps) {
  const { data, isLoading, error } = useSquadV2({ playerSlug, teammates, period })

  if (isLoading) {
    return <div className="p-6 text-center text-muted-foreground">{labels.loading}</div>
  }
  if (error) {
    return (
      <div className="p-6 text-center text-destructive">
        {labels.errorPrefix} {error.message}
      </div>
    )
  }
  if (!data || data.shared_matches_count === 0) {
    return <div className="p-6 text-center text-muted-foreground">{labels.empty}</div>
  }

  const charts = data.charts
  const tables = data.tables
  const squadOrder = [data.main_player, ...data.teammates]

  return (
    <div className="flex flex-col gap-6 p-6" data-testid="squad-v2-page">
      {/* Synergies */}
      {charts && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.synergiesTitle}</h2>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {charts.map_breakdown_lollipop && (
              <BarStackedChart
                title={labels.mapBreakdownTitle}
                series={[charts.map_breakdown_lollipop]}
                orientation="horizontal"
                componentColors={{
                  win: 'outcome-win',
                  loss: 'outcome-loss',
                  tie: 'outcome-draw',
                  dnf: 'outcome-dnf',
                }}
                componentOrder={['win', 'loss', 'tie', 'dnf']}
              />
            )}
            {charts.heatmap_player_map && (
              <Heatmap2DChart
                title={labels.synergiesTitle}
                series={[charts.heatmap_player_map]}
                paletteMode="divergent"
              />
            )}
            {charts.timeline_multi_player && charts.timeline_multi_player.length > 0 && (
              <TimeseriesLineChart
                title={labels.timelineTitle}
                series={charts.timeline_multi_player}
              />
            )}
            {charts.cadence && (
              <BarStackedChart
                title={labels.cadenceTitle}
                series={[charts.cadence]}
              />
            )}
            {charts.intensity_heatmap && (
              <Heatmap2DChart
                title={labels.intensityTitle}
                series={[charts.intensity_heatmap]}
              />
            )}
          </div>
        </section>
      )}

      {/* Contributions */}
      {charts && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.contributionsTitle}</h2>
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            {charts.per_minute_stats && (
              <BarGroupedChart
                title={labels.perMinuteTitle}
                series={[charts.per_minute_stats]}
              />
            )}
            {charts.frags_deaths_combined && (
              <BarGroupedChart
                title={labels.fragsDeathsTitle}
                series={[charts.frags_deaths_combined]}
                componentColors={{
                  kills: 'outcome-win',
                  deaths: 'outcome-loss',
                }}
                componentOrder={['kills', 'deaths']}
              />
            )}
          </div>
        </section>
      )}

      {/* Radar (S8) */}
      {charts?.radar && charts.radar.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.radarTitle}</h2>
          <RadarChart series={charts.radar} />
        </section>
      )}

      {/* Tableaux (S9) */}
      {tables?.history && tables.history.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.historyTitle}</h2>
          <HistoryTable
            rows={tables.history}
            squadOrder={squadOrder}
            locale={labels.locale}
            labels={labels.history}
          />
        </section>
      )}

      {tables?.weapons && tables.weapons.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.weaponsTitle}</h2>
          <WeaponsTable
            rows={tables.weapons}
            squadOrder={squadOrder}
            labels={labels.weapons}
          />
        </section>
      )}

      {tables?.medals && tables.medals.length > 0 && (
        <section>
          <h2 className="mb-3 text-lg font-semibold">{labels.medalsTitle}</h2>
          <MedalsGallery
            entries={tables.medals}
            squadOrder={squadOrder}
            labels={labels.medals}
          />
        </section>
      )}
    </div>
  )
}
