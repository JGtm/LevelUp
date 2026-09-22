/**
 * SquadContributionsPage — onglet Contributions de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout. Affiche les
 * charts de contribution par joueur : K/D/A par minute, synergies radar,
 * performance, impact des coéquipiers, médailles, mécaniques de frag. Le
 * « Premier frag / première mort » a rejoint l'onglet Dynamique (chart lanes) ;
 * l'impact et les médailles sont arrivés de Synergies (lot 3, 2026-09-22).
 *
 * Multi-titres : strings UI via getSquadText.
 */
import { useMemo } from 'react'
import { Link } from '@tanstack/react-router'
import { SectionTitle } from '@/components/ui/detail-section'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadPerMinuteChart } from './SquadPerMinuteChart'
import { SquadSynergyRadarChart } from './SquadSynergyRadarChart'
import { SquadPerformanceCharts } from './SquadPerformanceCharts'
import { SquadKillMechanicsChart } from './SquadKillMechanicsChart'
import { SquadImpactScoreboard } from './SquadImpactScoreboard'
import { MedalDigest } from './MedalDigest'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import { getSquadPlayerColors } from './colors'

export function SquadContributionsPage() {
  const { confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  // Tous les dérivés de cette page sont MÉMOÏSÉS : ils descendent en props dans
  // des ChartCard, qui reconstruisent leur option ECharts (et rejouent donc leur
  // animation d'entrée) dès qu'une prop change d'identité — même si sa valeur
  // est la même. Les `?? []` / `?? {}` écrits à la volée en fabriquaient une
  // neuve à chaque rendu.
  const perMinuteRows = useMemo(() => pageData?.per_minute_stats ?? [], [pageData?.per_minute_stats])
  const synergyRadar = useMemo(() => pageData?.synergy_radar ?? [], [pageData?.synergy_radar])
  const performanceSeries = pageData?.performance_series
  const perfSeriesByPlayer = useMemo(() => performanceSeries ?? {}, [performanceSeries])
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que le mapping couleurs matche les clés des SquadPerMinuteEntry.player
  // / SquadSynergyRadarSeries.player etc.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const playerColors = useMemo(
    () => getSquadPlayerColors(mainPlayerKey, confirmedGamertags),
    [mainPlayerKey, confirmedGamertags],
  )
  const playerOrder = useMemo(
    () => [mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries?.[p]),
    [mainPlayerKey, confirmedGamertags, performanceSeries],
  )
  const synergyAxisLabels = useMemo<Record<string, string>>(
    () => ({
      combat: t.synergyRadar.axes.combat,
      survival: t.synergyRadar.axes.survival,
      support: t.synergyRadar.axes.support,
      score: t.synergyRadar.axes.score,
      objective: t.synergyRadar.axes.objective,
      impact: t.synergyRadar.axes.impact,
    }),
    [t],
  )

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
        <SectionTitle>{t.performanceCharts.title}</SectionTitle>
        <SquadPerformanceCharts
          emptyMessage={t.empty.noBlockData}
          rowsByPlayer={perfSeriesByPlayer}
          playerOrder={playerOrder}
          colorByPlayer={playerColors}
          labels={t.performanceCharts}
        />
      </section>

      {/* IMPACT DES COÉQUIPIERS — arrivé de Synergies (lot 3 « sections », 2026-09-22) :
          c'est une contribution par joueur, pas une production de la composition.
          Section non-graphe toujours montée : titre + état vide géré par le composant
          (cadre bordé / carte), au lieu de disparaître. */}
      <section className="space-y-3">
        <SectionTitle>{t.impact.title}</SectionTitle>
        <SquadImpactScoreboard
          matrix={pageData?.impact_matrix ?? { matches: [], players: [], cells: [], badge_ord: [] }}
        />
      </section>

      {/* MÉDAILLES — arrivées de Synergies avec l'impact. Elles restent EN DERNIER de
          leur groupe (décision utilisateur, 2026-09-13) : c'est un palmarès, pas une
          mesure — il se lit après tout ce qui explique le jeu, jamais avant. */}
      <section className="space-y-3">
        <SectionTitle>{t.medals.title}</SectionTitle>
        <MedalDigest
          entries={pageData?.medal_digest ?? []}
          mainPlayer={pageData?.main_player ?? playerSlug}
          t={t.medals}
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
    </div>
  )
}
