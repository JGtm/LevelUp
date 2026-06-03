/**
 * ExplorerTargetCadence — bloc « Cadence » du profil cible : DEUX graphes empilés
 * (par match / par minute) reprenant le style « Stats par minute » de Squad
 * Contributions (barres divergentes K/D/A). Rendu à droite du bloc « Sur N matchs
 * joués ensemble » (1/3 de largeur).
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerTargetSampleStats } from '@/lib/api/types'
import { buildCadenceBarsOption, type CadenceBarsLabels } from './explorerCadenceChart'

interface Props {
  sampleStats: ExplorerTargetSampleStats
}

// ChartCard rend l'état "vide" si series.length === 0. Le builder cadence
// n'utilise pas les datapoints (valeurs agrégées passées en closure) → on fournit
// une série sentinelle non vide pour forcer le rendu data.
const SENTINEL: ChartSeries[] = [{ key: 'cadence', datapoints: [{}] }]

// Hauteur MINIMALE par graphe (mode fluid → grandit pour remplir). Volontairement
// basse pour que ce soit la colonne GAUCHE (donut + bilan) qui définisse la hauteur
// de la rangée : les 2 graphes cadence s'y adaptent ensuite via fluid+flex-1. Une
// valeur haute ferait l'inverse (la cadence imposerait la hauteur).
const CHART_HEIGHT = 110

export function ExplorerTargetCadence({ sampleStats }: Props) {
  const appLocale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, appLocale)

  const labels: CadenceBarsLabels = {
    frags: t('explorer.target_profile.cadence_frags'),
    deaths: t('explorer.target_profile.cadence_deaths'),
    assists: t('explorer.target_profile.cadence_assists'),
  }

  const n = sampleStats.sample_size
  const buildPerMatch = useCallback(
    (): EChartsCoreOption =>
      buildCadenceBarsOption(
        {
          kills: n > 0 ? sampleStats.kills / n : 0,
          deaths: n > 0 ? sampleStats.deaths / n : 0,
          assists: n > 0 ? sampleStats.assists / n : 0,
        },
        labels,
        1,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [sampleStats, appLocale],
  )
  const buildPerMinute = useCallback(
    (): EChartsCoreOption =>
      buildCadenceBarsOption(
        {
          kills: sampleStats.kills_per_min ?? 0,
          deaths: sampleStats.deaths_per_min ?? 0,
          assists: sampleStats.assists_per_min ?? 0,
        },
        labels,
        2,
      ),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [sampleStats, appLocale],
  )

  // h-full + 2 graphes `fluid` flex-1 : la colonne s'étire à la hauteur de la
  // rangée (= bloc "Sur N matchs joués ensemble" à gauche) et les 2 graphes se
  // partagent cette hauteur à parts égales. `height` devient le minimum garanti.
  return (
    <div className="flex h-full flex-col gap-4" data-testid="explorer-target-cadence">
      <ChartCard
        title={t('explorer.target_profile.cadence_per_match')}
        series={SENTINEL}
        height={CHART_HEIGHT}
        fluid
        className="flex-1"
        buildOption={buildPerMatch}
      />
      <ChartCard
        title={t('explorer.target_profile.cadence_per_minute')}
        series={SENTINEL}
        height={CHART_HEIGHT}
        fluid
        className="flex-1"
        buildOption={buildPerMinute}
      />
    </div>
  )
}
