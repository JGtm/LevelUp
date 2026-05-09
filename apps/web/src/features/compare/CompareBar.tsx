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

  // Ratio proportionnel : A occupe rawA/(rawA+rawB) de la barre depuis la gauche.
  const total = rawA + rawB
  const ratio = total === 0 ? 0.5 : Math.max(0.05, Math.min(0.95, rawA / total))
  const aOpacity = w === 'b' ? 0.28 : 1
  const bOpacity = w === 'a' ? 0.28 : 1

  return (
    <div className="space-y-1" role="group" aria-label={ariaLabel}>
      <p className="text-center text-xs text-muted-foreground leading-tight">{label}</p>
      <div className="flex items-center gap-2">

        {/* Valeur A — alignée à droite */}
        <div className="w-20 shrink-0 flex flex-col items-end">
          <span
            className="text-sm tabular-nums leading-tight"
            style={w === 'a' ? { color: colorA, fontWeight: 600 } : undefined}
          >
            {valueA}
          </span>
        </div>

        {/* Barre composite proportionnelle */}
        <div className="relative flex-1 h-3 rounded-sm bg-muted overflow-hidden">
          <div
            className="absolute inset-y-0 left-0"
            style={{ width: `${ratio * 100}%`, background: colorA, opacity: aOpacity }}
          />
          <div
            className="absolute inset-y-0 right-0"
            style={{ width: `${(1 - ratio) * 100}%`, background: colorB, opacity: bOpacity }}
          />
        </div>

        {/* Valeur B — alignée à gauche */}
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
