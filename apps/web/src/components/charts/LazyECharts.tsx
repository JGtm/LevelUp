/**
 * LazyECharts — SEUL point de montage de `echarts-for-react` dans l'app.
 *
 * - Import paresseux : le paquet echarts (~600 KB) n'est payé qu'au premier graphe rendu.
 *   L'appelant fournit le `<Suspense>` (chaque wrapper choisit son repli).
 * - Taille : le suivi de `echarts-for-react` (`autoResize`) est remplacé par
 *   `followContainerSize`, qui rattrape aussi un graphe monté pendant que son conteneur
 *   change de largeur (cf. `lib/echarts/followContainerSize.ts`).
 *
 * `echartsReactSingleMount.guard.test.ts` interdit tout autre import de `echarts-for-react`.
 */
import { lazy, useCallback, useEffect, useMemo, useRef } from 'react'
import type { EChartsReactProps } from 'echarts-for-react'

import { sansTexteAriaAnglais } from '@/lib/echarts/ariaSansAnglais'
import { followContainerSize, type ResizableChart } from '@/lib/echarts/followContainerSize'

const ReactECharts = lazy(() =>
  import('echarts-for-react').then((m) => ({ default: m.default ?? m })),
)

export type LazyEChartsProps = Omit<EChartsReactProps, 'autoResize'>

export function LazyECharts({ onChartReady, option, ...props }: LazyEChartsProps) {
  const safeOption = useMemo(() => sansTexteAriaAnglais(option), [option])
  const unfollow = useRef<(() => void) | null>(null)
  useEffect(() => () => unfollow.current?.(), [])

  // Rappelé à chaque (re)création de l'instance (changement de `opts`/`theme`) : le suivi
  // de l'instance précédente est débranché avant de brancher le nouveau.
  const handleReady = useCallback<NonNullable<EChartsReactProps['onChartReady']>>(
    (instance) => {
      unfollow.current?.()
      unfollow.current = followContainerSize(instance as ResizableChart)
      onChartReady?.(instance)
    },
    [onChartReady],
  )

  return <ReactECharts {...props} option={safeOption} autoResize={false} onChartReady={handleReady} />
}
