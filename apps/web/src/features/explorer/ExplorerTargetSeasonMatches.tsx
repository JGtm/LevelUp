/**
 * ExplorerTargetSeasonMatches — graphe « matchs par saison » du joueur cible.
 *
 * Barres VERTICALES, une barre par saison du catalogue (S1 → courante), hauteur
 * = total des matchs matchmade de la saison. Quand l'info de rang est disponible,
 * l'image du badge CSR (pic de saison) est affichée au-dessus de chaque barre via
 * un markPoint image ancré au sommet.
 *
 * Données : ExplorerTargetProfile.matches_per_season (mode live = service record
 * filtré par saison ; mode dégradé sans auth = bucketing local). Le split
 * classé/non-classé n'est pas exposé : l'API Waypoint rejette `seasonId + isRanked`
 * (le filtre exige gameVariantCategory → explosion d'appels). On affiche donc le
 * total par saison.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { explorerManifest, type ExplorerManifestKey } from '@/lib/i18n/generated/explorer'
import type { ExplorerLiveSectionStatus, SeasonMatchCount } from '@/lib/api/types'
import type { SemanticToken } from '@/lib/accessibility'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { buildBarStackedOption, type ChartPointStacked } from '@/components/charts/BarStackedChart'
import { ExplorerLiveStatusBadge } from './ExplorerLiveStatusBadge'

interface ExplorerTargetSeasonMatchesProps {
  seasons: SeasonMatchCount[]
  title: string
  /** Statut live de la section (Lot A3) — badge discret dans la barre de titre
   *  quand != "ok" (no_auth avant tout fetch, local_partial en repli bucketing
   *  local — cf. computeSeasonBreakdown côté Go). */
  liveStatus?: ExplorerLiveSectionStatus | null
}

const MATCHES_TOKEN: SemanticToken = 'chart-series-1'

export function ExplorerTargetSeasonMatches({ seasons, title, liveStatus }: ExplorerTargetSeasonMatchesProps) {
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: ExplorerManifestKey) => formatMessage(explorerManifest, key, locale)
  const matchesLabel = t('explorer.target_profile.label_matches')

  const buildOption = useCallback(
    () => buildSeasonMatchesOption(seasons, matchesLabel),
    // matchesLabel dérive de la locale ; seasons et locale suffisent comme deps.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [seasons, locale],
  )

  // Pas de `return null` sur saisons vides : on laisse ChartCard afficher son état
  // « empty » (barre de titre + message centré) pour ne JAMAIS masquer le graphe en
  // silence — un espace blanc laisse croire à un bug. series=[] déclenche l'état
  // empty de ChartCard, qui n'appelle alors pas buildOption (pas de rendu ECharts).
  const series = seasons.length > 0 ? toSeasonSeries(seasons, matchesLabel) : []

  return (
    <div data-testid="explorer-target-season-matches">
      <ChartCard
        title={
          <span className="flex items-center gap-2">
            <span>{title}</span>
            <ExplorerLiveStatusBadge status={liveStatus} />
          </span>
        }
        series={series}
        emptyMessage={t('explorer.target_profile.matches_per_season_empty')}
        height={220}
        buildOption={buildOption}
      />
    </div>
  )
}

/** toSeasonSeries projette les saisons en une série (1 datapoint/saison, total). */
function toSeasonSeries(
  seasons: SeasonMatchCount[],
  matchesLabel: string,
): ChartSeries<ChartPointStacked>[] {
  const datapoints: ChartPointStacked[] = seasons.map((s) => ({
    category: s.season_name,
    components: { [matchesLabel]: s.matches },
  }))
  return [{ key: 'seasons', datapoints }]
}

/**
 * buildSeasonMatchesOption — builder pur (exporté pour test) : barres verticales
 * (total par saison) + badges CSR (markPoint image) au-dessus de chaque barre.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildSeasonMatchesOption(
  seasons: SeasonMatchCount[],
  matchesLabel: string,
): EChartsCoreOption {
  const option = buildBarStackedOption(toSeasonSeries(seasons, matchesLabel), {
    orientation: 'vertical',
    componentColors: { [matchesLabel]: MATCHES_TOKEN },
    componentOrder: [matchesLabel],
    tooltipHideZero: true,
  })

  // Badges CSR : markPoint image ancré au sommet de chaque barre (saison jouée
  // avec un rang). symbolOffset négatif → l'image flotte au-dessus de la barre.
  const markData = seasons.flatMap((s) => {
    if (!s.csr_badge_image_url) return []
    return [
      {
        coord: [s.season_name, s.matches],
        symbol: `image://${s.csr_badge_image_url}`,
        symbolSize: [20, 20],
        symbolOffset: [0, '-65%'],
      },
    ]
  })

  const mut = option as EChartsCoreOption & {
    series?: Array<Record<string, unknown>>
    grid?: Record<string, unknown>
    legend?: Record<string, unknown>
  }
  // Série unique (total par saison) → la légende est redondante : on la masque.
  mut.legend = { show: false }
  if (mut.series && mut.series.length > 0 && markData.length > 0) {
    mut.series[0].markPoint = { silent: true, data: markData, label: { show: false } }
    if (mut.grid) mut.grid.top = 44 // dégage la place pour les badges au-dessus
  }
  return option
}
