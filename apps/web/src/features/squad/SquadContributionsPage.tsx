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
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadKillMechanicsChart } from './SquadKillMechanicsChart'
import { SquadFirstEventsChart } from './SquadFirstEventsChart'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { getSquadPlayerColors } from './colors'

export function SquadContributionsPage() {
  const { confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const perMinuteRows = pageData?.per_minute_stats ?? []
  const synergyRadar = pageData?.synergy_radar ?? []
  const performanceSeries = pageData?.performance_series
  const firstEvents = pageData?.first_events
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que le mapping couleurs matche les clés des SquadPerMinuteEntry.player
  // / SquadSynergyRadarSeries.player / SquadFirstEventsRow.player etc.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const playerColors = getSquadPlayerColors(mainPlayerKey, confirmedGamertags)
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
      {/* Graphes toujours montés : chaque ChartCard affiche son état vide
          (titre + message) au lieu de disparaître quand sa source est vide. */}
      <div className="grid gap-4 lg:grid-cols-2">
        <SquadPerMinuteChart
          title={t.perMinute.title}
          emptyMessage={t.empty.noBlockData}
          rows={perMinuteRows}
          colorByPlayer={playerColors}
          metricLabels={{
            frags: t.perMinute.frags,
            deaths: t.perMinute.deaths,
            assists: t.perMinute.assists,
          }}
          perMinuteSuffix={t.perMinute.suffix}
        />

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
          emptyMessage={t.empty.noBlockData}
          colorByPlayer={playerColors}
          axisLabels={synergyAxisLabels}
          rawLabel={t.synergyRadar.rawLabel}
        />
      </div>

      <section className="space-y-3">
        <h3 className="text-base font-semibold text-foreground">{t.performanceCharts.title}</h3>
        <SquadPerformanceCharts
          emptyMessage={t.empty.noBlockData}
          rowsByPlayer={performanceSeries ?? {}}
          playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries?.[p])}
          colorByPlayer={playerColors}
          labels={t.performanceCharts}
        />
      </section>

      {/* Halo 5 : mécaniques natives par coéquipier (assassinats + compétences
          spartiate). FeatureGate masque hors h5 ; le composant rend null sans données. */}
      <FeatureGate capability="native_kill_mechanics">
        <SquadKillMechanicsChart
          title={t.killMechanics.title}
          emptyMessage={t.empty.noBlockData}
          data={pageData?.native_kill_mechanics}
          colorByPlayer={playerColors}
          labelOf={(m) => t.killMechanics.labels[m as keyof typeof t.killMechanics.labels] ?? m}
        />
      </FeatureGate>

      <SquadFirstEventsChart
        title={t.firstEvents.title}
        emptyMessage={t.empty.noBlockData}
        data={firstEvents}
        colorByPlayer={playerColors}
        fragLabel={t.firstEvents.fragLabel}
        deathLabel={t.firstEvents.deathLabel}
        matchesSuffix={t.firstEvents.matchesSuffix}
      />

    </div>
  )
}
