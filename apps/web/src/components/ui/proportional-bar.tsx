import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

export interface BarSegment {
  value: number
  color: SemanticToken
}

export function ProportionalBar({ segments }: { segments: BarSegment[] }) {
  const total = segments.reduce((sum, s) => sum + s.value, 0)
  if (total === 0) return null
  const pct = (n: number) => `${(n / total) * 100}%`
  return (
    <div className="flex h-1.5 w-full overflow-hidden rounded-full gap-px">
      {segments.map((s, i) =>
        s.value > 0
          ? <div key={i} style={{ width: pct(s.value), backgroundColor: tokenCssVar(s.color) }} />
          : null
      )}
    </div>
  )
}
