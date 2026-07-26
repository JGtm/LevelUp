/**
 * SquadToggleLegendChart — wrapper de sous-chart performance escouade avec
 * légende React compacte cliquable (rendue hors ECharts, pas via `legend`).
 *
 * Deux types (ex. Frags / Morts, HS / Parfaits) + un item par joueur. Cliquer un
 * type le masque pour tous les joueurs ; cliquer un joueur masque ses deux
 * séries. L'état de masquage est passé au builder fourni, qui vide les séries
 * concernées (les axes restent stables). Remplace la légende ECharts N×2
 * scrollable jugée trop dense.
 */
import { useCallback, useState } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'

interface ToggleLegendType {
  /** Clé = label de la série côté builder (testé via hiddenTypes.has(key)). */
  key: string
  label: string
}

interface HiddenState {
  hiddenPlayers: Set<string>
  hiddenTypes: Set<string>
}

interface SquadToggleLegendChartProps {
  title?: string
  /** Série sentinelle pour l'empty-state du ChartCard (datapoints arbitraires). */
  series: ChartSeries<unknown>[]
  /** Joueurs affichés dans la légende (ordre = ordre des séries du builder). */
  players: string[]
  /** gamertag → couleur hex résolue. */
  colorByPlayer: Record<string, string>
  /** Types affichés en tête de légende (2 ou plus). */
  types: ToggleLegendType[]
  /** Types masqués par défaut au montage. */
  initialHiddenTypes?: Set<string>
  /** Builder qui reçoit l'état masqué et renvoie l'option ECharts. */
  buildOption: (hidden: HiddenState) => EChartsCoreOption
  height?: number
  emptyMessage?: string
}

function toggleInSet(set: Set<string>, key: string): Set<string> {
  const next = new Set(set)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  return next
}

export function SquadToggleLegendChart({
  title,
  series,
  players,
  colorByPlayer,
  types,
  initialHiddenTypes,
  buildOption,
  height = 320,
  emptyMessage,
}: SquadToggleLegendChartProps) {
  const [hiddenPlayers, setHiddenPlayers] = useState<Set<string>>(() => new Set())
  const [hiddenTypes, setHiddenTypes] = useState<Set<string>>(() => initialHiddenTypes ?? new Set())

  const build = useCallback(
    () => buildOption({ hiddenPlayers, hiddenTypes }),
    [buildOption, hiddenPlayers, hiddenTypes],
  )

  return (
    <ChartCard title={title} series={series} buildOption={build} height={height} emptyMessage={emptyMessage}>
      {series.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t border-border px-3 py-2">
          {types.map(({ key, label }) => {
            const off = hiddenTypes.has(key)
            return (
              <button
                key={key}
                type="button"
                onClick={() => setHiddenTypes((prev) => toggleInSet(prev, key))}
                aria-pressed={!off}
                className={`text-xs font-medium transition-opacity ${off ? 'opacity-40 line-through' : ''}`}
              >
                {label}
              </button>
            )
          })}

          <span className="h-3.5 w-px bg-border" aria-hidden />

          {players.map((p) => {
            const off = hiddenPlayers.has(p)
            return (
              <button
                key={p}
                type="button"
                onClick={() => setHiddenPlayers((prev) => toggleInSet(prev, p))}
                aria-pressed={!off}
                className={`flex items-center gap-1.5 text-xs transition-opacity ${off ? 'opacity-40' : ''}`}
              >
                <span
                  className="inline-block h-2.5 w-2.5 rounded-sm border border-border"
                  style={{ backgroundColor: colorByPlayer[p] }}
                />
                <span className={off ? 'line-through' : ''}>{p}</span>
              </button>
            )
          })}
        </div>
      )}
    </ChartCard>
  )
}
