/**
 * ChartFromOption — adapte une option ECharts déjà construite en ChartCard.
 *
 * Remplace l'ancien couple `ChartFrame` + `if (!option) return null` : au lieu
 * de faire disparaître le bloc (ou de laisser un bloc titré vide), on rend un
 * `ChartCard` qui gère nativement les 4 états (loading / error / empty / data).
 * Quand `option` est `null` (pas de données), ChartCard affiche `emptyMessage`
 * DANS le même bloc titré, comme partout ailleurs dans l'app.
 *
 * Le wrapper consommateur garde la responsabilité de construire l'option (avec
 * son propre `useMemo` dépendant de `themeVersion`) et signale l'absence de
 * données en passant `option={null}`.
 */
import type { ReactNode } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'

// Sentinelles : ChartCard détecte le vide via `series.length === 0`. On lui
// passe une série factice quand l'option existe (les données réelles vivent
// dans la closure `buildOption`), et une série vide quand option === null.
const NON_EMPTY_SERIES: ChartSeries<number>[] = [{ key: 'data', datapoints: [1] }]
const EMPTY_SERIES: ChartSeries<number>[] = []

export interface ChartFromOptionProps {
  /** Titre affiché en tête du bloc (resté visible même à l'état vide). */
  title?: ReactNode
  /** Option ECharts pré-construite, ou `null` si pas de données. */
  option: EChartsCoreOption | null
  /** Hauteur fixe en pixels. */
  height: number
  /** Message affiché à l'état vide (défaut ChartCard : « Aucune donnée à afficher »). */
  emptyMessage?: string
  /** ClassName optionnel transmis à la ChartCard. */
  className?: string
  /** Handlers d'événements ECharts (ex. legendselectchanged) transmis à ChartCard. */
  onEvents?: Record<string, (params: unknown) => void>
  /** Pied de carte optionnel (rendu sous le graphe, ex. rangée de KPI). */
  footer?: ReactNode
  /** Clé de tournée de revue transmise à ChartCard (badge inerte hors tournée). */
  reviewKey?: string
}

export function ChartFromOption({
  title,
  option,
  height,
  emptyMessage,
  className,
  onEvents,
  footer,
  reviewKey,
}: ChartFromOptionProps) {
  return (
    <ChartCard
      title={title}
      series={option ? NON_EMPTY_SERIES : EMPTY_SERIES}
      buildOption={() => option ?? ({} as EChartsCoreOption)}
      emptyMessage={emptyMessage}
      height={height}
      className={className}
      onEvents={onEvents}
      reviewKey={reviewKey}
    >
      {footer}
    </ChartCard>
  )
}
