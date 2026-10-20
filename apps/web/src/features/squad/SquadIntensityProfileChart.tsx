/**
 * SquadIntensityProfileChart — « Intensité » (onglet Dynamique).
 *
 * Un panneau par joueur : médiane des parts de frags par phase + enveloppe
 * interquartile P25–P75 (irrégularité). Les PANNEAUX consomment
 * `intensity_profile.rows` par gamertag (jamais les lignes agrégées `team` /
 * `lobby`), couleurs `colorByPlayer`, titres de panneaux = gamertags. Deux
 * courbes de référence neutres se superposent à chaque panneau : LOBBY (tout le
 * match, dès 1 panneau — « le match était-il intense en général ? ») et ÉQUIPE
 * (alliés du joueur principal, à partir de `MIN_PLAYERS_FOR_TEAM_CURVE`
 * joueurs). Le layout multi-grilles + l'agrégation vivent dans le builder
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
  isIntensityOverlayKey,
  type IntensityOverlay,
  type IntensityPanelInput,
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
  /** Texte du tooltip d'aide (courbe / zone d'irrégularité / repère / équipe / lobby). */
  tooltip: string
  medianLabel: string
  envelopeLabel: string
  refLabel: string
  /** Libellé de la courbe de référence ÉQUIPE (3 joueurs et plus). */
  teamLabel: string
  /** Libellé de la courbe de référence LOBBY (dès 1 joueur). */
  lobbyLabel: string
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
  lobbyLabel,
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
        : profile.options.filter((o) => !isIntensityOverlayKey(o.key)).map((o) => o.key)
    const seen = new Set<string>()
    const out: IntensityPanelInput[] = []
    for (const key of order) {
      if (isIntensityOverlayKey(key) || seen.has(key)) continue
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

  // Courbes de référence : `team` (alliés du joueur principal par match, 3 joueurs
  // et plus) et `lobby` (tout le match, dès 1 panneau). Chaque ligne agrégée du
  // payload porte, par match, les frags de toute la population visée ; elle est
  // agrégée par le même helper que les joueurs. Une ligne absente ou sans frag
  // n'est pas montée (aucune courbe plate).
  const overlays = useMemo<IntensityOverlay[]>(() => {
    const out: IntensityOverlay[] = []
    const teamRows = profile.rows['team']
    if (panels.length >= MIN_PLAYERS_FOR_TEAM_CURVE && teamRows && hasExploitableMatch(teamRows)) {
      out.push({ key: 'team', label: teamLabel, rows: teamRows })
    }
    const lobbyRows = profile.rows['lobby']
    if (panels.length >= 1 && lobbyRows && hasExploitableMatch(lobbyRows)) {
      out.push({ key: 'lobby', label: lobbyLabel, rows: lobbyRows })
    }
    return out
  }, [panels.length, profile.rows, teamLabel, lobbyLabel])

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
        overlays,
      }),
    [panels, medianLabel, envelopeLabel, refLabel, axisLabels, overlays],
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
