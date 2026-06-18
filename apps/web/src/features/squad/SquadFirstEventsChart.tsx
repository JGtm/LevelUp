/**
 * SquadFirstEventsChart — wrapper teammates.17.
 *
 * Légende méta cliquable (rendue côté React, pas via ECharts) : un item par
 * joueur (masque ses frags ET ses morts) + deux items globaux frags / morts
 * (masquent ce type pour tous les joueurs). L'état de masquage est passé au
 * builder qui vide les séries concernées (axes + séparateurs restent stables).
 */
import { useCallback, useMemo, useState } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import type { SquadFirstEvents, SquadFirstEventsRow } from '@/lib/api/types'
import {
  buildSquadFirstEventsOption,
  type SquadFirstEventsOpts,
} from './charts/squadFirstEventsChart'

interface SquadFirstEventsChartProps extends SquadFirstEventsOpts {
  title?: string
  emptyMessage?: string
  data: SquadFirstEvents | null | undefined
  height?: number
}

type EventType = 'frag' | 'death'

function toggleInSet<T>(set: Set<T>, key: T): Set<T> {
  const next = new Set(set)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  return next
}

export function SquadFirstEventsChart({
  data,
  title,
  emptyMessage,
  height = 420,
  ...opts
}: SquadFirstEventsChartProps) {
  const [hiddenPlayers, setHiddenPlayers] = useState<Set<string>>(() => new Set())
  const [hiddenTypes, setHiddenTypes] = useState<Set<EventType>>(() => new Set())

  const players = useMemo(() => (data?.rows ?? []).map((r) => r.player), [data])

  const series = useMemo<ChartSeries<SquadFirstEventsRow>[]>(() => {
    const rows = data?.rows ?? []
    return rows.length > 0 ? [{ key: 'first-events', datapoints: rows }] : []
  }, [data])
  const buildOption = useCallback(
    () => buildSquadFirstEventsOption(data, { ...opts, hiddenPlayers, hiddenTypes }),
    [data, opts, hiddenPlayers, hiddenTypes],
  )

  const typeToggles: Array<{ key: EventType; label: string }> = [
    { key: 'frag', label: opts.fragLabel },
    { key: 'death', label: opts.deathLabel },
  ]

  return (
    <ChartCard title={title} series={series} buildOption={buildOption} height={height} emptyMessage={emptyMessage}>
      {series.length > 0 && (
        <div className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t border-border px-3 py-2">
          {typeToggles.map(({ key, label }) => {
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
                  style={{ backgroundColor: opts.colorByPlayer[p] }}
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
