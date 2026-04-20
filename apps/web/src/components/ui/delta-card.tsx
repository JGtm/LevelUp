/**
 * DeltaCard — carte indicateur avec valeur + delta coloré.
 *
 * Transversal D1/D2/D3/D4 NATIVE_COMPONENTS.
 */
import type { ReactNode } from 'react'

export interface DeltaCardProps {
  label: string
  value: ReactNode
  delta?: number | string | null
  unit?: string
  /** Si true, delta négatif = vert (ex: deaths) */
  lowerIsBetter?: boolean
  /** Si true, affiche un avertissement ⚠ (ex: R² < 0.3 = tendance non signif.) */
  warning?: boolean
  /** Texte du warning affiché sous la valeur */
  warningText?: string
}

function formatDelta(delta: number | string | null | undefined, lowerIsBetter: boolean): {
  text: string
  color: string
} {
  if (delta == null) return { text: '', color: '' }
  const num = typeof delta === 'number' ? delta : parseFloat(String(delta))
  if (isNaN(num)) return { text: String(delta), color: 'text-muted-foreground' }
  const isFavorable = lowerIsBetter ? num < 0 : num > 0
  const sign = num > 0 ? '+' : ''
  const text = `${sign}${num.toFixed(typeof delta === 'number' && Math.abs(num) < 1 ? 3 : 1)}`
  const color = isFavorable ? 'text-[#00DC82]' : num === 0 ? 'text-muted-foreground' : 'text-[#FF4B4B]'
  return { text, color }
}

export function DeltaCard({
  label,
  value,
  delta,
  unit,
  lowerIsBetter = false,
  warning = false,
  warningText,
}: DeltaCardProps) {
  const { text: deltaText, color: deltaColor } = formatDelta(delta, lowerIsBetter)

  return (
    <div className="rounded-lg border border-border bg-[#1d2328] px-4 py-3">
      <p className="text-xs text-muted-foreground uppercase tracking-wide mb-1">{label}</p>
      <div className="flex items-baseline gap-1.5">
        <span className="text-xl font-bold text-foreground">{value}</span>
        {unit && <span className="text-xs text-muted-foreground">{unit}</span>}
      </div>
      {deltaText && (
        <p className={`text-xs font-semibold mt-0.5 ${deltaColor}`}>{deltaText}</p>
      )}
      {warning && (
        <p className="text-[10px] text-warning mt-1">⚠ {warningText ?? 'Tendance non significative'}</p>
      )}
    </div>
  )
}
