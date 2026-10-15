/**
 * barLabel — L'ÉTIQUETTE ECHARTS POSÉE DANS UN SEGMENT (`position: 'inside'`) : blanche, cernée d'un
 * contour sombre fin (jetons d'encre de `lib/accessibility/segmentInk.ts`, identiques en clair et en
 * sombre), centrée sur la hauteur. Équivalent ECharts de `SegmentLabel` (pistes HTML) ; une étiquette
 * `inside` à couleur fixe (`tc.text`) est interdite par `segmentInk.guard.test.ts`.
 */
import { segmentInk } from '@/lib/accessibility/segmentInk'

const OUTLINE_WIDTH = 2

export function barLabel<T extends Record<string, unknown>>(extra?: T) {
  const { ink, outline } = segmentInk()
  return {
    show: true,
    position: 'inside' as const,
    verticalAlign: 'middle' as const,
    color: ink,
    textBorderColor: outline,
    textBorderWidth: OUTLINE_WIDTH,
    ...extra,
  }
}
