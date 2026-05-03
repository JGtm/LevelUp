/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Affiche un
 * radar normalisé du profil de chaque coéquipier sélectionné.
 *
 * Multi-titres : axes du radar dérivés de SQUAD_RADAR_METRICS (FieldKeys
 * canoniques) + filtrage des keys absentes du fields.toml du titre courant.
 * Strings UI via getSquadText.
 *
 * Distingue 3 empty kinds (no_selection / invalid_selection / no_chart_data)
 * comme SquadSynergiesPage pour rendre les défauts diagnosticables.
 */
import { Card, CardContent } from '@/components/ui/card'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useAppShellStore } from '@/stores/appShellStore'
import type { TeammateRow } from '@/lib/api/types'
import { useSquadContext } from './SquadContext'
import { useFieldMappings, type FieldMappingsResponse } from '@/lib/i18n/fieldMappings'
import { getSquadText } from './i18n'
import { SQUAD_RADAR_METRICS, type SquadMetric } from './metrics'
import { RadarChart, type RadarSeriesPayload } from '@/components/charts/RadarChart'
import { SquadMatchHistoryTable } from './SquadMatchHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadIntensityHeatmapChart } from './SquadIntensityHeatmapChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadFirstEventsChart } from './SquadFirstEventsChart'
import { getSquadPlayerColors } from './colors'

function metricLabel(
  m: SquadMetric,
  mappings: FieldMappingsResponse | undefined,
  perGameSuffix: string,
): string {
  const base = mappings?.fields[m.key]?.label ?? m.key
  return m.format === 'per_game' ? `${base}${perGameSuffix}` : base
}

function availableMetrics(
  metrics: readonly SquadMetric[],
  mappings: FieldMappingsResponse | undefined,
): SquadMetric[] {
  if (!mappings) return [...metrics]
  return metrics.filter((m) => !!mappings.fields[m.key])
}

function buildRadarSeries(
  rows: TeammateRow[],
  metrics: SquadMetric[],
  withGamertagLabel: (gt: string) => string,
): RadarSeriesPayload[] {
  return rows.map((row) => ({
    key: row.gamertag,
    axes: metrics.map((m) => {
      const v = m.extract(row.with_kpis) ?? 0
      return { axis: m.key, value: v, raw: v }
    }),
    meta: { gamertag: withGamertagLabel(row.gamertag) },
  }))
}

export function SquadContributionsPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const { data: mappings } = useFieldMappings()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const matchHistory = pageData?.match_history ?? []
  const impactMatrix = pageData?.impact_matrix
  const perMinuteRows = pageData?.per_minute_stats ?? []
  const synergyRadar = pageData?.synergy_radar ?? []
  const intensityProfile = pageData?.intensity_profile
  const performanceSeries = pageData?.performance_series
  const weaponKills = pageData?.weapon_kills
  const firstEvents = pageData?.first_events
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que le mapping couleurs matche les clés des SquadPerMinuteEntry.player
  // / SquadSynergyRadarSeries.player / SquadFirstEventsRow.player etc.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const playerColors = getSquadPlayerColors(mainPlayerKey, confirmedGamertags)
  const intensityProfileLocalized = intensityProfile
    ? {
        ...intensityProfile,
        // Le backend renvoie "all" comme label brut — on le localise ici.
        options: intensityProfile.options.map((o) =>
          o.key === 'all' ? { ...o, label: t.intensity.allLabel } : o,
        ),
      }
    : undefined
  const synergyAxisLabels: Record<string, string> = {
    combat: t.synergyRadar.axes.combat,
    survival: t.synergyRadar.axes.survival,
    support: t.synergyRadar.axes.support,
    score: t.synergyRadar.axes.score,
    objective: t.synergyRadar.axes.objective,
    impact: t.synergyRadar.axes.impact,
  }

  const metrics = availableMetrics(SQUAD_RADAR_METRICS, mappings)
  const axisLabels = Object.fromEntries(
    metrics.map((m) => [m.key, metricLabel(m, mappings, t.units.perGame)]),
  )

  const hasSelection = confirmedGamertags.length > 0
  const hasRows = selectedRows.length > 0

  const radarSeries = hasRows
    ? buildRadarSeries(selectedRows, metrics, t.table.withTeammate)
    : []

  let emptyContent: React.ReactNode = null
  if (!hasSelection) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.noSelectionTitle}
        description={t.empty.noSelectionDescription}
      />
    )
  } else if (!hasRows) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.invalidSelectionTitle}
        description={t.empty.invalidSelectionDescription}
      />
    )
  } else if (radarSeries.length === 0 || metrics.length === 0) {
    emptyContent = (
      <EmptyStateNotice
        title={t.empty.noChartTitle}
        description={t.empty.noChartDescription}
      />
    )
  }

  return (
    <div className="space-y-4">
      {perMinuteRows.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.perMinute.title}</h3>
              <p className="text-sm text-muted-foreground">{t.perMinute.description}</p>
            </div>
            <SquadPerMinuteChart
              rows={perMinuteRows}
              colorByPlayer={playerColors}
              metricLabels={{
                frags: t.perMinute.frags,
                deaths: t.perMinute.deaths,
                assists: t.perMinute.assists,
              }}
              perMinuteSuffix={t.perMinute.suffix}
            />
          </CardContent>
        </Card>
      )}

      <Card>
        <CardContent className="pt-4">
          {emptyContent ?? (
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">{t.contributions.description}</p>
              <RadarChart
                series={radarSeries}
                height={380}
                axisLabels={axisLabels}
              />
            </div>
          )}
        </CardContent>
      </Card>

      {synergyRadar.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.synergyRadar.title}</h3>
              <p className="text-sm text-muted-foreground">{t.synergyRadar.description}</p>
            </div>
            <SquadSynergyRadarChart
              rows={synergyRadar}
              colorByPlayer={playerColors}
              axisLabels={synergyAxisLabels}
            />
          </CardContent>
        </Card>
      )}

      {intensityProfileLocalized && intensityProfileLocalized.options.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.intensity.title}</h3>
              <p className="text-sm text-muted-foreground">{t.intensity.description}</p>
            </div>
            <SquadIntensityHeatmapChart
              profile={intensityProfileLocalized}
              zLabel={t.intensity.zLabel}
              toggleLabel={t.intensity.toggleLabel}
            />
          </CardContent>
        </Card>
      )}

      {performanceSeries && Object.keys(performanceSeries).length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.performanceCharts.title}</h3>
              <p className="text-sm text-muted-foreground">{t.performanceCharts.description}</p>
            </div>
            <SquadPerformanceCharts
              rowsByPlayer={performanceSeries}
              playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries[p])}
              colorByPlayer={playerColors}
              labels={t.performanceCharts}
            />
          </CardContent>
        </Card>
      )}

      {weaponKills && weaponKills.bars.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.weaponKills.title}</h3>
              <p className="text-sm text-muted-foreground">{t.weaponKills.description}</p>
            </div>
            <SquadWeaponKillsChart
              data={weaponKills}
              colorByPlayer={playerColors}
            />
          </CardContent>
        </Card>
      )}

      {firstEvents && firstEvents.rows.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.firstEvents.title}</h3>
              <p className="text-sm text-muted-foreground">{t.firstEvents.description}</p>
            </div>
            <SquadFirstEventsChart
              data={firstEvents}
              colorByPlayer={playerColors}
              fragLabel={t.firstEvents.fragLabel}
              deathLabel={t.firstEvents.deathLabel}
              matchesSuffix={t.firstEvents.matchesSuffix}
            />
          </CardContent>
        </Card>
      )}

      {impactMatrix && impactMatrix.matches.length > 0 && impactMatrix.players.length > 0 && (
        <SquadImpactScoreboard matrix={impactMatrix} />
      )}

      {matchHistory.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="text-base font-semibold">{t.history.title}</h3>
              <p className="text-sm text-muted-foreground">{t.history.description}</p>
            </div>
            <SquadMatchHistoryTable rows={matchHistory} playerSlug={playerSlug} />
          </CardContent>
        </Card>
      )}
    </div>
  )
}
