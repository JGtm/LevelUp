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

export function CompositeProgressBar({
  value,
  fillTestId,
}: {
  value?: number | null
  fillTestId?: string
}) {
  const width = clampCompositeProgress(value)

  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-muted-foreground/25">
      <div
        data-testid={fillTestId}
        className="h-full rounded-full bg-sky-500 transition-all duration-300"
        style={{ width: `${width}%` }}
      />
    </div>
  )
}
