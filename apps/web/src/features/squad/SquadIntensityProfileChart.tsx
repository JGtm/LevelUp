/**
 * SquadIntensityProfileChart — « Intensité » (onglet Dynamique).
 *
 * Un panneau par joueur : médiane des parts de frags par phase + enveloppe
 * interquartile P25–P75 (irrégularité). Consomme `intensity_profile.rows` par
 * gamertag (JAMAIS la ligne agrégée `all`), couleurs `colorByPlayer`, titres de
 * panneaux = gamertags. Le layout multi-grilles + l'agrégation vivent dans le
 * builder `charts/squadIntensityProfileChart` (échelle Y partagée, repère 10 %).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { resolveToken } from '@/lib/accessibility'
import { phaseShares } from '@/lib/charts/phaseProfile'
import type { SquadIntensityProfile } from '@/lib/api/types'
import {
  buildSquadIntensityProfileOption,
  type IntensityPanelInput,
} from './charts/squadIntensityProfileChart'

interface SquadIntensityProfileChartProps {
  title: string
  /** Sous-titre sous le titre de la carte. */
  subtitle: string
  /** Texte du tooltip d'aide (médiane / enveloppe / repère 10 %). */
  tooltip: string
  medianLabel: string
  envelopeLabel: string
  refLabel: string
  emptyMessage: string
  profile: SquadIntensityProfile
  /** gamertag → couleur hex résolue depuis les semantic tokens. */
  colorByPlayer: Record<string, string>
  /** Ordre d'affichage des joueurs (main player d'abord). Défaut : ordre des options. */
  playerOrder?: string[]
}

const PANEL_ROW_HEIGHT = 230

/** Au moins une manche exploitable (Σ phases > 0) ? (réutilise phaseShares.) */
function hasExploitableMatch(rows: Array<{ phases: number[] | null }>): boolean {
  return rows.some((r) => phaseShares(r.phases) !== null)
}

export function SquadIntensityProfileChart({
  title,
  subtitle,
  tooltip,
  medianLabel,
  envelopeLabel,
  refLabel,
  emptyMessage,
  profile,
  colorByPlayer,
  playerOrder,
}: SquadIntensityProfileChartProps) {
  const panels = useMemo<IntensityPanelInput[]>(() => {
    const labelByKey = new Map(profile.options.map((o) => [o.key, o.label]))
    const order =
      playerOrder && playerOrder.length > 0
        ? playerOrder
        : profile.options.filter((o) => o.key !== 'all').map((o) => o.key)
    const seen = new Set<string>()
    const out: IntensityPanelInput[] = []
    for (const key of order) {
      if (key === 'all' || seen.has(key)) continue
      seen.add(key)
      const rows = profile.rows[key]
      if (!rows || !hasExploitableMatch(rows)) continue
      out.push({
        key,
        label: labelByKey.get(key) ?? key,
        color: colorByPlayer[key] ?? resolveToken('chart-series-1'),
        rows,
      })
    }
    return out
  }, [profile, colorByPlayer, playerOrder])

  // 1 entrée / panneau : pilote l'état vide du ChartCard + sa clé de mémo ; le
  // builder relit `panels` directement (le ChartCard ignore l'argument série).
  const series = useMemo<ChartSeries<IntensityPanelInput>[]>(
    () => panels.map((p) => ({ key: p.key, datapoints: [p] })),
    [panels],
  )

  const buildOption = useCallback(
    () => buildSquadIntensityProfileOption({ panels, medianLabel, envelopeLabel, refLabel }),
    [panels, medianLabel, envelopeLabel, refLabel],
  )

  const rowsCount = panels.length <= 1 ? 1 : Math.ceil(panels.length / 2)
  const height = Math.max(260, rowsCount * PANEL_ROW_HEIGHT)

  return (
    <ChartCard
      title={
        <div className="flex flex-col gap-0.5">
          <span className="flex items-center gap-1.5">
            {title}
            <InfoTooltip content={tooltip} />
          </span>
          <span className="text-xs font-normal text-muted-foreground">{subtitle}</span>
        </div>
      }
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={emptyMessage}
    />
  )
}
