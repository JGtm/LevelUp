/**
 * ChartOption — le type de l'option d'un graphe, tel que les builders des pages le produisent.
 *
 * Seul point d'où le reste de l'application obtient ce type : changer de bibliothèque de graphes
 * ne touche que ce fichier et `LazyECharts`, jamais la soixantaine de modules qui construisent
 * des options. `chartsImportBoundary.guard.test.ts` interdit tout autre import de la bibliothèque
 * hors de `components/charts/`.
 */
import type { EChartsCoreOption } from 'echarts/core'

export type ChartOption = EChartsCoreOption
