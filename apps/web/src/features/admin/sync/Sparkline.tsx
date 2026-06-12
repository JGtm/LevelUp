/**
 * Sparkline — mini-tendance SVG inline (flat hard-edge), sans axes ni tooltip.
 * Trait + dot sur la valeur courante (dernier point). Couleur via token
 * sémantique. Géométrie déléguée à sparkline.ts (pur, testé).
 */
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility/semantic-tokens'
import { lastPoint, sparklinePoints } from './sparklineGeometry'

export function Sparkline({
  values,
  token = 'info',
  width = 96,
  height = 24,
  ariaLabel,
}: {
  values: number[]
  token?: SemanticToken
  width?: number
  height?: number
  ariaLabel?: string
}) {
  const points = sparklinePoints(values, width, height)
  if (!points) return null
  const color = tokenCssVar(token)
  const dot = lastPoint(values, width, height)

  return (
    <svg
      width={width}
      height={height}
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={ariaLabel}
      className="flex-none overflow-visible"
    >
      <polyline points={points} fill="none" stroke={color} strokeWidth={1.25} />
      {dot && <circle cx={dot.x} cy={dot.y} r={1.75} fill={color} />}
    </svg>
  )
}
