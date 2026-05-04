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
import { Link } from '@tanstack/react-router'
import { Card, CardContent } from '@/components/ui/card'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadMatchHistoryTable } from './SquadMatchHistoryTable'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadIntensityHeatmapChart } from './SquadIntensityHeatmapChart'
import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadWeaponKillsChart } from './SquadWeaponKillsChart'
import { SquadFirstEventsChart } from './SquadFirstEventsChart'
import { getSquadPlayerColors } from './colors'

export function SquadContributionsPage() {
  const { confirmedGamertags, pageData, playerSlug } = useSquadContext()
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

      {synergyRadar.length > 0 && (
        <Card>
          <CardContent className="pt-4 space-y-3">
            <div>
              <h3 className="flex items-center gap-1.5 text-base font-semibold">
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
              </h3>
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
              <h3 className="text-base font-semibold">{t.efficiencySeries.title}</h3>
              <p className="text-sm text-muted-foreground">{t.efficiencySeries.description}</p>
            </div>
            <SquadEfficiencyChart
              rowsByPlayer={performanceSeries}
              playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries[p])}
              colorByPlayer={playerColors}
              labels={t.efficiencySeries}
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
