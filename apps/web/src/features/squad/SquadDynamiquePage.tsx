/**
 * SquadDynamiquePage — onglet Dynamique de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout (même mécanisme de
 * données que Contributions : aucune query key propre). Regroupe les charts
 * d'intensité, de rendement/résistance, le « Premier frag / première mort » et
 * la section engagement — déplacés depuis SquadContributionsPage.
 *
 * Multi-titres : strings UI via getSquadText ; SquadEngagementSection gardée
 * derrière FeatureGate capability="engagement".
 */
import { useMemo } from 'react'
import { FirstBloodLanes } from '@/components/charts/FirstBloodLanes'
import { intensityTooltipText } from '@/components/charts/intensityTooltipText'
import { firstBloodMaxSec, toFirstBloodSeries } from '@/features/_shared/firstBlood'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadIntensityProfileChart } from './SquadIntensityProfileChart'
import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { SquadNetLivesChart } from './SquadNetLivesChart'
import { SquadEngagementGapChart } from './SquadEngagementGapChart'
import { SquadEngagementSection } from '@/features/engagement/SquadEngagementSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import type { SquadTeammateEntry } from '@/features/engagement/queries'
import { getSquadPlayerColors } from './colors'

export function SquadDynamiquePage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const performanceSeries = pageData?.performance_series
  // Dérivés MÉMOÏSÉS : ils descendent en props dans des ChartCard, qui
  // reconstruisent leur option ECharts (et rejouent leur animation d'entrée) dès
  // qu'une prop change d'identité, même à valeur égale. Les objets/tableaux
  // fabriqués à la volée dans le JSX (`?? {}`, `[main, ...co].filter(...)`)
  // étaient neufs à chaque rendu.
  const intensityProfile = useMemo(
    () => pageData?.intensity_profile ?? { options: [], rows: {} },
    [pageData?.intensity_profile],
  )
  const perfSeriesByPlayer = useMemo(() => performanceSeries ?? {}, [performanceSeries])
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que le mapping couleurs matche les clés des séries par joueur.
  const mainPlayerKey = pageData?.main_player ?? playerSlug
  const playerColors = useMemo(
    () => getSquadPlayerColors(mainPlayerKey, confirmedGamertags),
    [mainPlayerKey, confirmedGamertags],
  )
  /** Roster complet — ordre des bandes du profil d'intensité. */
  const roster = useMemo(
    () => [mainPlayerKey, ...confirmedGamertags],
    [mainPlayerKey, confirmedGamertags],
  )
  /** Roster restreint aux joueurs ayant une série de performance. */
  const playerOrder = useMemo(
    () => roster.filter((p) => performanceSeries?.[p]),
    [roster, performanceSeries],
  )
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
  // « Premier frag / première mort » : une bande par joueur de l'escouade, valeurs
  // par match servies telles quelles par l'API (aucun bucketing serveur).
  const firstBlood = useMemo(
    () => toFirstBloodSeries(pageData?.first_blood),
    [pageData?.first_blood],
  )
  return (
    <div className="space-y-4">
      <SquadIntensityProfileChart
        title={t.intensity.title}
        subtitle={t.intensity.subtitle}
        tooltip={intensityTooltipText(locale, { withTeam: true })}
        medianLabel={t.intensity.medianLabel}
        envelopeLabel={t.intensity.envelopeLabel}
        refLabel={t.intensity.refLabel}
        teamLabel={t.intensity.teamLabel}
        lobbyLabel={t.intensity.lobbyLabel}
        emptyMessage={t.empty.noBlockData}
        profile={intensityProfile}
        colorByPlayer={playerColors}
        playerOrder={roster}
      />

      {/* Premier frag / première mort — bandes par joueur (2 à 4 joueurs).
          Le titre et l'état vide viennent du manifest partagé first_blood :
          même vocabulaire ici, sur Timeseries et sur Sessions. */}
      <FirstBloodLanes
        data={firstBlood}
        maxSec={firstBloodMaxSec(firstBlood)}
        emptyMessage={t.empty.noBlockData}
      />

      {/* Aide PROPRE à cette surface (portée par les deux titres de carte) : ces
          cartes tracent le TAUX « une vie » des indicateurs canoniques, quand
          l'explication partagée EfficiencyTooltipText décrit les dégâts bruts
          des charts Timeseries/Session. */}
      <SquadEfficiencyChart
        rowsByPlayer={perfSeriesByPlayer}
        playerOrder={playerOrder}
        colorByPlayer={playerColors}
        labels={t.efficiencySeries}
      />

      <SquadNetLivesChart
        rowsByPlayer={perfSeriesByPlayer}
        playerOrder={playerOrder}
        colorByPlayer={playerColors}
        t={t}
        emptyMessage={t.empty.noBlockData}
      />

      <FeatureGate capability="engagement">
        {/* Engagement + Écart d'engagement cumulé côte à côte sur desktop (empilés
            en mobile). */}
        <div className="grid gap-4 md:grid-cols-2">
          <SquadEngagementSection
            playerSlug={playerSlug}
            matchIds={engagementMatchIds}
            teammates={engagementTeammates}
            colorByPlayer={playerColors}
          />
          <SquadEngagementGapChart
            playerSlug={playerSlug}
            matchIds={engagementMatchIds}
            teammates={engagementTeammates}
            colorByPlayer={playerColors}
            t={t}
            emptyMessage={t.empty.noBlockData}
          />
        </div>
      </FeatureGate>
    </div>
  )
}
