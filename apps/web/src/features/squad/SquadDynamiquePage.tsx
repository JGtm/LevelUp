/**
 * SquadDynamiquePage — onglet Dynamique de l'Escouade.
 *
 * Consomme le contexte SquadContext fourni par SquadLayout (même mécanisme de
 * données que Contributions : aucune query key propre). Regroupe les charts
 * d'intensité, de rendement/résistance et la section engagement — déplacés
 * depuis SquadContributionsPage.
 *
 * Multi-titres : strings UI via getSquadText ; SquadEngagementSection gardée
 * derrière FeatureGate capability="engagement".
 */
import { useMemo } from 'react'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { EfficiencyTooltipText } from '@/components/charts/EfficiencyTooltipText'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSquadContext } from './SquadContext'
import { getSquadText } from './i18n'
import { SquadIntensityHeatmapChart } from './SquadIntensityHeatmapChart'
import { SquadEfficiencyChart } from './SquadEfficiencyChart'
import { SquadEngagementSection } from '@/features/engagement/SquadEngagementSection'
import { FeatureGate } from '@/lib/capabilities/FeatureGate'
import type { SquadTeammateEntry } from '@/features/engagement/queries'
import { getSquadPlayerColors } from './colors'

export function SquadDynamiquePage() {
  const { selectedRows, confirmedGamertags, pageData, playerSlug } = useSquadContext()
  const locale = useAppShellStore((s) => s.locale)
  const t = getSquadText(locale)
  const intensityProfile = pageData?.intensity_profile
  const performanceSeries = pageData?.performance_series
  // Le backend renvoie s.gamertag (casse mixte ex "Madina97294") tandis que
  // playerSlug est l'URL param (souvent lowercase). On aligne sur main_player
  // pour que le mapping couleurs matche les clés des séries par joueur.
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

  return (
    <div className="space-y-4">
      <SquadIntensityHeatmapChart
        title={t.intensity.title}
        emptyMessage={t.empty.noBlockData}
        profile={intensityProfileLocalized ?? { options: [], rows: {} }}
        colorByPlayer={playerColors}
        zLabel={t.intensity.zLabel}
      />

      <SquadEfficiencyChart
        infoTooltip={<InfoTooltip content={<EfficiencyTooltipText locale={locale} />} />}
        rowsByPlayer={performanceSeries ?? {}}
        playerOrder={[mainPlayerKey, ...confirmedGamertags].filter((p) => performanceSeries?.[p])}
        colorByPlayer={playerColors}
        labels={t.efficiencySeries}
      />

      <FeatureGate capability="engagement">
        <SquadEngagementSection
          playerSlug={playerSlug}
          matchIds={engagementMatchIds}
          teammates={engagementTeammates}
          colorByPlayer={playerColors}
        />
      </FeatureGate>
    </div>
  )
}
