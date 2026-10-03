/**
 * SessionBarsTrendCard — le wrapper de la frise « une soirée, un bâton ».
 *
 * Hissé avec son module d'option (`sessionBarsTrendChart.ts`) le 2026-09-22 : l'Escouade
 * garde son propre wrapper (il traduit sa `FriseRiposte`), les Séries temporelles montent
 * celui-ci directement avec des séries déjà génériques. Le FICHIER porte « Card » parce
 * que Windows ne distingue pas `SessionBarsTrendChart.tsx` de son module d'option
 * `sessionBarsTrendChart.ts` — deux fichiers voisins ne peuvent pas différer par la casse. Le composant ne fait que brancher
 * l'option sur `ChartCard` — aucune règle de lecture ici.
 *
 * `frameless` par défaut : la frise vit DANS une carte de section, un second cadre ferait
 * un cadre dans un cadre.
 */
import { useCallback, useMemo, type ReactNode } from 'react'

import { ChartCard, type ChartSeries } from './ChartCard'
import {
  buildSessionBarsTrendOption,
  type SessionBarsTrendOpts,
} from './sessionBarsTrendChart'

export interface SessionBarsTrendChartProps extends SessionBarsTrendOpts {
  title?: ReactNode
  emptyMessage?: string
  height?: number
  frameless?: boolean
}

export function SessionBarsTrendChart({
  title,
  emptyMessage,
  height = 300,
  frameless = true,
  labels,
  series: specs,
  yAxisLabel,
  volumeAxis,
  tooltipLines,
}: SessionBarsTrendChartProps) {
  // La série factice porte l'état « il y a quelque chose à peindre » : les données
  // vivent dans la closure de `buildOption`, comme dans `ChartFromOption`.
  const series = useMemo<ChartSeries<number>[]>(
    () => (labels.length > 0 && specs.length > 0 ? [{ key: 'soirees', datapoints: [1] }] : []),
    [labels.length, specs.length],
  )
  const buildOption = useCallback(
    () =>
      buildSessionBarsTrendOption({
        labels,
        series: specs,
        yAxisLabel,
        ...(volumeAxis ? { volumeAxis } : {}),
        ...(tooltipLines ? { tooltipLines } : {}),
      }),
    [labels, specs, yAxisLabel, volumeAxis, tooltipLines],
  )
  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={height}
      emptyMessage={emptyMessage}
      frameless={frameless}
    />
  )
}
