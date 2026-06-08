/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Affiche les
 * charts de contribution par joueur : K/D/A par minute, synergies radar,
 * intensité, performance, armes, premiers événements, impact scoreboard,
 * historique de matchs.
 *
 * Multi-titres : strings UI via getSquadText.
 */
import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadIntensityHeatmapChart } from './SquadIntensityHeatmapChart'
import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadFirstEventsChart } from './SquadFirstEventsChart'
import { SquadEngagementSection } from '@/features/engagement/SquadEngagementSection'
import type { SquadTeammateEntry } from '@/features/engagement/queries'
import { getSquadPlayerColors } from './colors'

export function SquadContributionsPage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
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
  const engagementTeammates = useMemo<SquadTeammateEntry[]>(
    () =>
      selectedRows
        .filter((r) => confirmedGamertags.includes(r.gamertag) && r.xuid !== null)
        .map((r) => ({ xuid: r.xuid as string, gamertag: r.gamertag })),
    [selectedRows, confirmedGamertags],
  )
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

  return (
    <div className="space-y-4">
      {(perMinuteRows.length > 0 || synergyRadar.length > 0) && (
        <div className="grid gap-4 lg:grid-cols-2">
          {perMinuteRows.length > 0 && (
            <SquadPerMinuteChart
              title={t.perMinute.title}
              rows={perMinuteRows}
              colorByPlayer={playerColors}
              metricLabels={{
                frags: t.perMinute.frags,
                deaths: t.perMinute.deaths,
                assists: t.perMinute.assists,
              }}
              perMinuteSuffix={t.perMinute.suffix}
            />
          )}

          {synergyRadar.length > 0 && (
            <SquadSynergyRadarChart
              title={
                <span className="flex items-center gap-1.5">
                  {t.synergyRadar.title}
                  <InfoTooltip
                    content={
                      <div className="space-y-1">
                        <p><span className="font-medium">{t.synergyRadar.axes.impact}</span> — {t.synergyRadar.tooltip.impact}</p>
                        <p><span className="font-medium">{t.synergyRadar.axes.combat}</span> — {t.synergyRadar.tooltip.combat}</p>
                        <p><span className="font-medium">{t.synergyRadar.axes.survival}</span> — {t.synergyRadar.tooltip.survival}</p>
                        <p><span className="font-medium">{t.synergyRadar.axes.support}</span> — {t.synergyRadar.tooltip.support}</p>
                        <p><span className="font-medium">{t.synergyRadar.axes.score}</span> — {t.synergyRadar.tooltip.score}</p>
                        <p><span className="font-medium">{t.synergyRadar.axes.objective}</span> — {t.synergyRadar.tooltip.objective}</p>
                        <Link to="/help" search={{ tab: 'glossary' }} className="block mt-2 text-primary hover:underline">
                          {t.synergyRadar.tooltip.glossaryLink}
                        </Link>
                      </div>
                    }
                  />
                </span>
              }
              rows={synergyRadar}
              colorByPlayer={playerColors}
              axisLabels={synergyAxisLabels}
            />
          )}
        </div>
      )}

      {intensityProfileLocalized && intensityProfileLocalized.options.length > 0 && (
        <SquadIntensityHeatmapChart
          title={t.intensity.title}
          profile={intensityProfileLocalized}
          colorByPlayer={playerColors}
          zLabel={t.intensity.zLabel}
        />
      )}

      {performanceSeries && Object.keys(performanceSeries).length > 0 && (
        <SquadEfficiencyChart
          title={t.efficiencySeries.title}
          rowsByPlayer={performanceSeries}
          playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries[p])}
          colorByPlayer={playerColors}
          labels={t.efficiencySeries}
        />
      )}

      {performanceSeries && Object.keys(performanceSeries).length > 0 && (
        <section className="space-y-3">
          <h3 className="text-base font-semibold text-foreground">{t.performanceCharts.title}</h3>
          <SquadPerformanceCharts
            rowsByPlayer={performanceSeries}
            playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries[p])}
            colorByPlayer={playerColors}
            labels={t.performanceCharts}
          />
        </section>
      )}

      {weaponKills && weaponKills.bars.length > 0 && (
        <SquadWeaponKillsChart
          title={t.weaponKills.title}
          data={weaponKills}
          colorByPlayer={playerColors}
        />
      )}

      <SquadEngagementSection playerSlug={playerSlug} teammates={engagementTeammates} colorByPlayer={playerColors} />

      {firstEvents && firstEvents.rows.length > 0 && (
        <SquadFirstEventsChart
          title={t.firstEvents.title}
          data={firstEvents}
          colorByPlayer={playerColors}
          fragLabel={t.firstEvents.fragLabel}
          deathLabel={t.firstEvents.deathLabel}
          matchesSuffix={t.firstEvents.matchesSuffix}
        />
      )}

    </div>
  )
}
