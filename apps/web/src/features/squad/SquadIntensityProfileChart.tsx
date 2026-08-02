/**
 * SquadIntensityProfileChart — « Intensité » (onglet Dynamique).
 *
 * Un panneau par joueur : médiane des parts de frags par phase + enveloppe
 * interquartile P25–P75 (irrégularité). Les PANNEAUX consomment
 * `intensity_profile.rows` par gamertag (jamais la ligne agrégée `all`),
 * couleurs `colorByPlayer`, titres de panneaux = gamertags. À partir de
 * `MIN_PLAYERS_FOR_TEAM_CURVE` joueurs, la ligne `all` — et elle seule — sert de
 * courbe de référence d'ÉQUIPE superposée à chaque panneau. Le layout
 * multi-grilles + l'agrégation vivent dans le builder
 * `charts/squadIntensityProfileChart` (échelle Y partagée, repère 10 %).
 */
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { resolveToken } from '@/lib/accessibility'
import { phaseShares } from '@/lib/charts/phaseProfile'
import { useAppShellStore } from '@/stores/appShellStore'
import type { SquadIntensityProfile } from '@/lib/api/types'
import {
  buildSquadIntensityProfileOption,
  intensityAxisLabels,
  type IntensityPanelInput,
  type IntensityTeamOverlay,
} from './charts/squadIntensityProfileChart'

/**
 * Nombre de joueurs à partir duquel la courbe agrégée d'équipe est superposée.
 * En dessous, comparer deux profils entre eux suffit : une 3e courbe n'apporte
 * rien et charge le panneau.
 */
const MIN_PLAYERS_FOR_TEAM_CURVE = 3

interface SquadIntensityProfileChartProps {
  title: string
  /** Sous-titre sous le titre de la carte. */
  subtitle: string
  /** Texte du tooltip d'aide (courbe / zone d'irrégularité / repère / équipe). */
  tooltip: string
  medianLabel: string
  envelopeLabel: string
  refLabel: string
  /** Libellé de la courbe agrégée d'équipe (3 joueurs et plus). */
  teamLabel: string
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
  teamLabel,
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

  // Courbe d'équipe (3 joueurs et plus) : la ligne agrégée `all` du payload porte,
  // pour chaque match, les frags de TOUTE l'escouade — c'est la seule population
  // « équipe » disponible, et elle est agrégée par le même helper que les joueurs.
  const teamOverlay = useMemo<IntensityTeamOverlay | undefined>(() => {
    if (panels.length < MIN_PLAYERS_FOR_TEAM_CURVE) return undefined
    const allRows = profile.rows['all']
    if (!allRows || !hasExploitableMatch(allRows)) return undefined
    return { label: teamLabel, rows: allRows }
  }, [panels.length, profile.rows, teamLabel])

  const locale = useAppShellStore((s) => s.locale)
  const axisLabels = intensityAxisLabels(locale)
  const buildOption = useCallback(
    () =>
      buildSquadIntensityProfileOption({
        panels,
        medianLabel,
        envelopeLabel,
        refLabel,
        axisLabels,
        teamOverlay,
      }),
    [panels, medianLabel, envelopeLabel, refLabel, axisLabels, teamOverlay],
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
      reviewKey="squad.intensity_profile"
    />
  )
}
