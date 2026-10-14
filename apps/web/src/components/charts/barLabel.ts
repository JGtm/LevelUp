/**
 * barLabel — L'ÉTIQUETTE ECHARTS POSÉE DANS UN SEGMENT (`position: 'inside'`) : encre de meilleur
 * contraste avec l'aplat (`inkOn`), centrée sur la hauteur. Équivalent ECharts de `SegmentLabel`
 * (pistes HTML) ; une étiquette `inside` à couleur fixe (`tc.text`) est interdite par
 * `barLabel.guard.test.ts`.
 *
 * `fill` : la couleur RÉSOLUE du segment (hex, `rgb()`, `var()`), la même que son `itemStyle.color`. Aplat translucide (`itemStyle.opacity`) :
 * `backdrop` donne le fond (`tc.card`) et l'opacité, l'encre se choisit sur le mélange.
 */
import { inkOn, type InkBackdrop } from '@/lib/accessibility/segmentInk'

export function barLabel<T extends Record<string, unknown>>(fill: string, extra?: T, backdrop?: InkBackdrop) {
  return {
    show: true,
    position: 'inside' as const,
    verticalAlign: 'middle' as const,
    color: inkOn(fill, backdrop),
    ...extra,
  }
}
