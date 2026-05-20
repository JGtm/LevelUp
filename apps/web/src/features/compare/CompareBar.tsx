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
  /** false = côté A en N/A (donnée absente). Neutralise la barre et le gagnant. */
  availableA?: boolean
  /** false = côté B en N/A. */
  availableB?: boolean
}

export function CompareBar({
  label, valueA, valueB, rawA, rawB, winner, ariaLabel, sampleNote,
  availableA = true, availableB = true,
}: CompareBarProps) {
  const colorA = tokenCssVar('compare-a' as SemanticToken)
  const colorB = tokenCssVar('compare-b' as SemanticToken)
  const bothAvailable = availableA && availableB
  const w = bothAvailable ? (winner ?? 'tie') : 'tie'

  const a = availableA && Number.isFinite(rawA) ? rawA : 0
  const b = availableB && Number.isFinite(rawB) ? rawB : 0
  const total = a + b
  const ratio = bothAvailable && total > 0
    ? Math.max(0.05, Math.min(0.95, a / total))
    : 0.5
  const pct = `${(ratio * 100).toFixed(1)}%`

  // Gradient CSS : point de rupture proportionnel aux valeurs brutes.
  // Une seule div — aucun positionnement absolu.
  const barStyle: CSSProperties = {
    background: `linear-gradient(to right, ${colorA} ${pct}, ${colorB} ${pct})`,
    opacity: bothAvailable ? (w === 'tie' ? 0.85 : 1) : 0.35,
  }

  const naClass = 'text-sm tabular-nums leading-tight text-muted-foreground italic'
  const valueClass = 'text-sm tabular-nums leading-tight'

  return (
    <div className="space-y-1 w-full" role="group" aria-label={ariaLabel}>
      <p className="text-center text-xs text-muted-foreground leading-tight">{label}</p>
      <div className="flex items-center gap-2">

        <div className="w-20 shrink-0 flex flex-col items-end">
          <span
            className={availableA ? valueClass : naClass}
            style={availableA && w === 'a' ? { color: colorA, fontWeight: 600 } : undefined}
          >
            {valueA}
          </span>
        </div>

        <div className="flex-1 h-3 rounded-sm" style={barStyle} data-testid="compare-bar-track" />

        <div className="w-20 shrink-0 flex flex-col items-start">
          <span
            className={availableB ? valueClass : naClass}
            style={availableB && w === 'b' ? { color: colorB, fontWeight: 600 } : undefined}
          >
            {valueB}
          </span>
          {sampleNote && (
            <span className="text-2xs leading-tight text-muted-foreground">{sampleNote}</span>
          )}
        </div>

      </div>
    </div>
  )
}
