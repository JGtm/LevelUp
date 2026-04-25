/**
 * TimeseriesScatter — scatter plot multi-types de corrélation.
 *
 * Usage : onglet Distributions — 5 paires de corrélation au choix.
 *         Coloré par outcome (victoire / défaite / inconnu).
 * Construit côté client depuis CorrelationDataPair[].
 */
import { Suspense, lazy, useMemo, useState } from 'react'
import { Spinner } from './spinner'
import { EmptyStateNotice } from './empty-state'
import type { CorrelationDataPair } from '@/lib/api/types'
import { resolveToken, useColorPaletteVersion } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'

const Plot = lazy(() =>
  import('react-plotly.js').then((m) => ({ default: m.default })),
)

const CLEAN_CONFIG: Partial<Plotly.Config> = {
  displaylogo: false,
  modeBarButtonsToRemove: ['toImage', 'sendDataToCloud', 'lasso2d', 'select2d'],
  responsive: true,
}

export interface TimeseriesScatterProps {
  points: CorrelationDataPair[]
  height?: number
}

const BG = '#1d2328'
const GRID = '#2a3038'
const TEXT = '#9ba3af'

// Outcome codes (mirrors Go analysis.OutcomeWin / OutcomeLoss)
const OUTCOME_WIN = 2
const OUTCOME_LOSS = 3

/**
 * Construit le LABEL_CONFIGS dynamique en injectant les libellés canoniques
 * (Phase D plan multi-titres) avec fallback sur les valeurs FR locales.
 *
 * Note : `Durée de vie`, `K/D`, `KDA`, `MMR équipe/adversaires`, `Précision`
 * ne sont pas (encore) tous des FieldKey canoniques — ils restent locaux.
 */
function buildLabelConfigs(
  fieldMappings?: { fields: Record<string, { label: string }> },
): Record<string, { xLabel: string; yLabel: string; title: string }> {
  const labelOf = (key: string, fallback: string): string =>
    fieldMappings?.fields[key]?.label ?? fallback
  const kills = labelOf('kills', 'Kills')
  const deaths = labelOf('deaths', 'Morts')
  const accuracy = labelOf('accuracy', 'Précision')
  const kda = labelOf('kda', 'KDA')
  const kdr = labelOf('kdr', 'K/D')
  const teamMMR = labelOf('team_mmr', 'MMR équipe')
  const enemyMMR = labelOf('enemy_mmr', 'MMR adversaires')
  return {
    kills_vs_kd: { xLabel: kills, yLabel: kdr, title: `${kills} → ${kdr}` },
    lifespan_vs_kills: { xLabel: 'Durée de vie (s)', yLabel: kills, title: `Durée de vie → ${kills}` },
    accuracy_vs_kda: { xLabel: `${accuracy} (%)`, yLabel: kda, title: `${accuracy} → ${kda}` },
    lifespan_vs_deaths: { xLabel: 'Durée de vie (s)', yLabel: deaths, title: `Durée de vie → ${deaths}` },
    kills_vs_deaths: { xLabel: kills, yLabel: deaths, title: `${kills} → ${deaths}` },
    mmr_team_vs_enemy: { xLabel: teamMMR, yLabel: enemyMMR, title: `${teamMMR} vs ${enemyMMR}` },
  }
}

export function TimeseriesScatter({ points, height = 320 }: TimeseriesScatterProps) {
  const availableLabels = useMemo(() => {
    const seen = new Set<string>()
    for (const p of points) seen.add(p.label)
    return Array.from(seen)
  }, [points])

  const [activeLabel, setActiveLabel] = useState<string>(() => availableLabels[0] ?? 'kills_vs_kd')
  const paletteVersion = useColorPaletteVersion()
  const { data: fieldMappings } = useFieldMappings()
  const labelConfigs = useMemo(() => buildLabelConfigs(fieldMappings), [fieldMappings])

  const { traces, layout } = useMemo(() => {
    const filtered = points.filter((p) => p.label === activeLabel)
    const cfg = labelConfigs[activeLabel] ?? { xLabel: 'X', yLabel: 'Y', title: activeLabel }

    const wins = filtered.filter((p) => p.outcome === OUTCOME_WIN)
    const losses = filtered.filter((p) => p.outcome === OUTCOME_LOSS)
    const unknowns = filtered.filter((p) => p.outcome !== OUTCOME_WIN && p.outcome !== OUTCOME_LOSS)

    const makeTrace = (
      pts: CorrelationDataPair[],
      name: string,
      color: string,
    ): Plotly.Data => ({
      type: 'scatter',
      mode: 'markers',
      name,
      x: pts.map((p) => p.x),
      y: pts.map((p) => p.y),
      marker: { color, size: 5, opacity: 0.75 },
      hovertemplate: `${cfg.xLabel} : %{x}<br>${cfg.yLabel} : %{y}<extra>${name}</extra>`,
    })

    const traces: Plotly.Data[] = [
      ...(wins.length > 0 ? [makeTrace(wins, 'Victoire', resolveToken('outcome-win'))] : []),
      ...(losses.length > 0 ? [makeTrace(losses, 'Défaite', resolveToken('outcome-loss'))] : []),
      ...(unknowns.length > 0 ? [makeTrace(unknowns, 'Inconnu', resolveToken('divergent-neutral'))] : []),
    ]

    const layout: Partial<Plotly.Layout> = {
      paper_bgcolor: BG,
      plot_bgcolor: BG,
      margin: { t: 8, b: 48, l: 52, r: 12 },
      height,
      font: { color: TEXT, size: 11 },
      xaxis: {
        showgrid: true,
        gridcolor: GRID,
        zeroline: false,
        title: { text: cfg.xLabel, font: { color: TEXT, size: 10 } },
        tickfont: { color: TEXT, size: 10 },
      },
      yaxis: {
        showgrid: true,
        gridcolor: GRID,
        zeroline: false,
        title: { text: cfg.yLabel, font: { color: TEXT, size: 10 } },
        tickfont: { color: TEXT, size: 10 },
      },
      legend: {
        font: { color: TEXT, size: 10 },
        bgcolor: 'transparent',
        orientation: 'h',
        y: -0.18,
      },
    }

    return { traces, layout }
  }, [points, activeLabel, height, paletteVersion, labelConfigs])

  if (points.length === 0) {
    return (
      <EmptyStateNotice
        title="Données insuffisantes"
        description="Aucune corrélation disponible."
      />
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap gap-1">
        {availableLabels.map((lbl) => (
          <button
            key={lbl}
            onClick={() => setActiveLabel(lbl)}
            className={[
              'rounded px-2 py-0.5 text-xs font-medium transition-colors',
              activeLabel === lbl
                ? 'bg-cyan-500/20 text-cyan-300 ring-1 ring-cyan-500/40'
                : 'bg-white/5 text-gray-400 hover:bg-white/10',
            ].join(' ')}
          >
            {labelConfigs[lbl]?.title ?? lbl}
          </button>
        ))}
      </div>
      <Suspense fallback={<Spinner size="sm" />}>
        <Plot
          data={traces}
          layout={layout}
          config={CLEAN_CONFIG}
          style={{ width: '100%', height: `${height}px` }}
        />
      </Suspense>
    </div>
  )
}
