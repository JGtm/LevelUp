import type { CSSProperties } from 'react'
import { tokenCssVar } from '@/lib/accessibility'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

export interface CompareBarProps {
  label: string
  valueA: string
  valueB: string
  rawA: number
  rawB: number
  winner: 'a' | 'b' | 'tie' | null
  ariaLabel?: string
  sampleNote?: string
}

export function CompareBar({ label, valueA, valueB, rawA, rawB, winner, ariaLabel, sampleNote }: CompareBarProps) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  const w = winner ?? 'tie'

  const a = Number.isFinite(rawA) ? rawA : 0
  const b = Number.isFinite(rawB) ? rawB : 0
  const total = a + b
  const ratio = total > 0 ? Math.max(0.05, Math.min(0.95, a / total)) : 0.5
  const pct = `${(ratio * 100).toFixed(1)}%`

  // Gradient CSS : point de rupture proportionnel aux valeurs brutes.
  // Une seule div — aucun positionnement absolu.
  const barStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorA} ${pct}, ${colorB} ${pct})`,
    opacity: w === 'tie' ? 0.85 : 1,
  }

  return (
    <div className="space-y-1 w-full" role="group" aria-label={ariaLabel}>
      <p className="text-center text-xs text-muted-foreground leading-tight">{label}</p>
      <div className="flex items-center gap-2">

        <div className="w-20 shrink-0 flex flex-col items-end">
          <span
            className="text-sm tabular-nums leading-tight"
            style={w === 'a' ? { color: colorA, fontWeight: 600 } : undefined}
          >
            {valueA}
          </span>
        </div>

        <div className="flex-1 h-3 rounded-sm" style={barStyle} data-testid="compare-bar-track" />

        <div className="w-20 shrink-0 flex flex-col items-start">
          <span
            className="text-sm tabular-nums leading-tight"
            style={w === 'b' ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueB}
          </span>
          {sampleNote && (
            <span className="text-[10px] leading-tight text-muted-foreground">{sampleNote}</span>
          )}
        </div>

      </div>
    </div>
  )
}
