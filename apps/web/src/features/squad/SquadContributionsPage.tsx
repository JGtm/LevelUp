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
import { EfficiencyTooltipText } from '@/components/charts/EfficiencyTooltipText'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadIntensityHeatmapChart } from './SquadIntensityHeatmapChart'
import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadKillMechanicsChart } from './SquadKillMechanicsChart'
import { SquadFirstEventsChart } from './SquadFirstEventsChart'
import { SquadEngagementSection } from '@/features/engagement/SquadEngagementSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
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
  // match_ids explicites pour l'engagement : les matchs du scope courant
  // (intersection composition exacte, session/période déjà filtrées). Sans ça,
  // le handler dérive les matchs de la timeseries d'engagement, qui renvoie
  // des bins agrégés (match_id vide) sur un gros historique → session vide.
  // match_history arrive DESC (récent d'abord) — cap à 15 comme le fallback
  // handler, puis .reverse() : GetSquadSession préserve l'ordre du caller et
  // étiquette M1..Mn dans cet ordre → il faut du chronologique ASC (ancien →
  // récent), comme tous les autres charts timeseries.
  const engagementMatchIds = useMemo<string[]>(
    () => (pageData?.match_history ?? []).slice(0, 15).map((m) => m.match_id).reverse(),
    [pageData?.match_history],
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
        />
      </div>

      <SquadIntensityHeatmapChart
        title={t.intensity.title}
        emptyMessage={t.empty.noBlockData}
        profile={intensityProfileLocalized ?? { options: [], rows: {} }}
        colorByPlayer={playerColors}
        zLabel={t.intensity.zLabel}
      />

      <SquadEfficiencyChart
        title={
          <span className="flex items-center gap-1.5">
            {t.efficiencySeries.title}
            <InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />
          </span>
        }
        monoTitle={
          <span className="flex items-center gap-1.5">
            {t.efficiencySeries.rendementTitle}
            <InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />
          </span>
        }
        rowsByPlayer={performanceSeries ?? {}}
        playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries?.[p])}
        colorByPlayer={playerColors}
        labels={t.efficiencySeries}
      />

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

      <FeatureGate capability="engagement">
        <SquadEngagementSection
          playerSlug={playerSlug}
          matchIds={engagementMatchIds}
          teammates={engagementTeammates}
          colorByPlayer={playerColors}
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
