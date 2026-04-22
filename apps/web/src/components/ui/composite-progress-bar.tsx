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
    <div
      className="h-3 w-full overflow-hidden rounded-full border border-slate-300/70 bg-slate-200/70"
      style={{
        backgroundImage:
          'repeating-linear-gradient(90deg, rgba(148,163,184,0.18) 0 18px, rgba(255,255,255,0.28) 18px 24px)',
      }}
    >
      <div
        data-testid={fillTestId}
        className="h-full rounded-full bg-sky-500 transition-[width]"
        style={{
          width: `${width}%`,
          backgroundImage:
            'repeating-linear-gradient(90deg, rgba(255,255,255,0.22) 0 18px, rgba(14,165,233,0.92) 18px 24px)',
        }}
      />
    </div>
  )
}
