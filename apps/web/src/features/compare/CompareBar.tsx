import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

export interface CompareBarProps {
  label: string
  valueA: string
  valueB: string
  winner: 'a' | 'b' | 'tie' | null
  ariaLabel?: string
  sampleNote?: string
}

export function CompareBar({ label, valueA, valueB, winner, ariaLabel, sampleNote }: CompareBarProps) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  const w = winner ?? 'tie'

  return (
    <div className="space-y-0.5" role="group" aria-label={ariaLabel}>
      <p className="text-center text-[11px] text-muted-foreground">{label}</p>
      <div className="flex items-center gap-2">
        <span
          className="w-16 shrink-0 text-right text-sm tabular-nums"
          style={w === 'a' ? { color: colorA, fontWeight: 600 } : undefined}
        >
          {valueA}
        </span>

        <div className="relative flex-1 h-2.5 rounded-sm bg-muted overflow-hidden">
          <div className="absolute inset-y-0 left-1/2 w-px bg-border/60 z-10" />
          {w === 'a' && (
            <div
              className="absolute inset-y-0 left-0 w-1/2 rounded-l-sm"
              style={{ background: colorA }}
            />
          )}
          {w === 'b' && (
            <div
              className="absolute inset-y-0 right-0 w-1/2 rounded-r-sm"
              style={{ background: colorB }}
            />
          )}
          {w === 'tie' && (
            <div className="absolute inset-0 rounded-sm bg-muted-foreground/25" />
          )}
        </div>

        <div className="w-16 shrink-0">
          <span
            className="text-sm tabular-nums"
            style={w === 'b' ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueB}
          </span>
          {sampleNote && (
            <p className="text-[10px] leading-tight text-muted-foreground">{sampleNote}</p>
          )}
        </div>
      </div>
    </div>
  )
}
