/**
 * RankProgressGauge — jauge arc SVG pour la progression de rang / héros.
 * Utilisé pour C1 (rang XP) et C2 (rang Héros) de la page Carrière.
 *
 * Palette dynamique par seuil de progress_pct :
 *   0–25 %  → rouge   #FF4B4B
 *   25–50 % → orange  #F97316
 *   50–75 % → cyan    #33D6FF
 *   75–100% → vert    #00DC82
 */

interface Props {
  /** Titre affiché sous la valeur centrale (ex: nom du rang) */
  title: string
  /** Progression 0.0–1.0 */
  progressPct: number
  /** Libellé secondaire sous le titre (ex: "12 345 / 50 000 XP") */
  subtitle?: string
  /** Taille SVG en px (largeur = hauteur, défaut 220) */
  size?: number
}

function arcColor(pct: number): string {
  if (pct < 0.25) return '#FF4B4B'
  if (pct < 0.5) return '#F97316'
  if (pct < 0.75) return '#33D6FF'
  return '#00DC82'
}

/** Calcule le path SVG d'un arc partiel dans un cercle de rayon r centré en (cx, cy). */
function describeArc(cx: number, cy: number, r: number, startDeg: number, endDeg: number): string {
  const toRad = (d: number) => (d * Math.PI) / 180
  const x1 = cx + r * Math.cos(toRad(startDeg))
  const y1 = cy + r * Math.sin(toRad(startDeg))
  const x2 = cx + r * Math.cos(toRad(endDeg))
  const y2 = cy + r * Math.sin(toRad(endDeg))
  const largeArc = endDeg - startDeg > 180 ? 1 : 0
  return `M ${x1} ${y1} A ${r} ${r} 0 ${largeArc} 1 ${x2} ${y2}`
}

export function RankProgressGauge({ title, progressPct, subtitle, size = 220 }: Props) {
  const pct = Math.max(0, Math.min(1, progressPct))
  const cx = size / 2
  const cy = size / 2
  const r = size * 0.38
  const strokeWidth = size * 0.072

  // Arc de -200° à +20° (240° total, démarrant en bas-gauche)
  const startAngle = -200
  const totalSweep = 240
  const endAngle = startAngle + totalSweep * pct
  const color = arcColor(pct)

  return (
    <div className="flex flex-col items-center">
      <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} aria-label={title}>
        {/* Piste de fond */}
        <path
          d={describeArc(cx, cy, r, -200, 40)}
          fill="none"
          stroke="#2d3748"
          strokeWidth={strokeWidth}
          strokeLinecap="round"
        />
        {/* Arc de progression */}
        {pct > 0 && (
          <path
            d={describeArc(cx, cy, r, startAngle, pct < 1 ? endAngle : 39.9)}
            fill="none"
            stroke={color}
            strokeWidth={strokeWidth}
            strokeLinecap="round"
            style={{ filter: `drop-shadow(0 0 6px ${color}88)` }}
          />
        )}
        {/* Valeur centrale */}
        <text
          x={cx}
          y={cy - size * 0.04}
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize={size * 0.16}
          fontWeight="700"
          fill={color}
        >
          {Math.round(pct * 100)}%
        </text>
        {/* Titre */}
        <text
          x={cx}
          y={cy + size * 0.14}
          textAnchor="middle"
          dominantBaseline="middle"
          fontSize={size * 0.075}
          fill="#e2e8f0"
        >
          {title.length > 18 ? title.slice(0, 17) + '…' : title}
        </text>
      </svg>
      {subtitle && (
        <p className="mt-1 text-center text-xs text-muted-foreground">{subtitle}</p>
      )}
    </div>
  )
}
