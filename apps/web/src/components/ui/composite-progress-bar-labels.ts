/**
 * Helpers purs de CompositeProgressBar — extraits du .tsx pour que le module de
 * composant n'exporte que des composants (react-refresh/only-export-components).
 */

export function clampCompositeProgress(value?: number | null) {
  if (value == null) {
    return 0
  }
  return Math.max(0, Math.min(100, value))
}

function formatXPLabel(value: number, locale: string) {
  return `${Math.max(0, value).toLocaleString(locale)} XP`
}

export function buildCompositeProgressEdgeLabels({
  partialProgress,
  xpPerRank,
  progressPercent,
  locale,
}: {
  partialProgress: number
  xpPerRank?: number | null
  progressPercent: number
  locale: string
}) {
  if (xpPerRank != null && xpPerRank > 0) {
    return {
      current: formatXPLabel(partialProgress, locale),
      target: formatXPLabel(xpPerRank, locale),
    }
  }

  return {
    current: `${progressPercent.toLocaleString(locale, { maximumFractionDigits: 0 })} %`,
    target: '100 %',
  }
}
