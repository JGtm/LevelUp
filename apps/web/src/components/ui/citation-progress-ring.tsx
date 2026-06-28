import { resolveToken } from '@/lib/accessibility'

interface CitationProgressRingProps {
  pct: number
  imageUrl?: string
  /** Maîtrisée (palier final franchi), que ce soit ce match ou avant. */
  isMastered?: boolean
  /** Maîtrisée POUR LA PREMIÈRE FOIS sur ce match (sert au libellé distinctif). */
  isNewlyMastered?: boolean
  size?: number
}

/**
 * Anneau SVG de progression pour une citation (commendation Halo 5).
 * - Arc coloré proportionnel à pct (0–100), couleur via resolveToken
 * - Anneau doré + pastille ✓ dès que la citation est maîtrisée (isMastered ou
 *   isNewlyMastered) — une maîtrise antérieure ne doit PAS s'afficher « en cours ».
 * - Image de la citation centrée (si imageUrl défini)
 */
export function CitationProgressRing({
  pct,
  imageUrl,
  isMastered = false,
  isNewlyMastered = false,
  size = 36,
}: CitationProgressRingProps) {
  const strokeWidth = 2.5
  const radius = (size - strokeWidth) / 2
  const circumference = 2 * Math.PI * radius
  const clampedPct = Math.min(100, Math.max(0, pct))
  const dashOffset = circumference * (1 - clampedPct / 100)

  const mastered = isMastered || isNewlyMastered
  const trackColor = 'var(--border)'
  const arcColor = mastered ? resolveToken('perf-tier-3') : resolveToken('perf-tier-2')
  const imageSize = Math.round(size * 0.58)
  const center = size / 2

  return (
    <div
      className="relative flex items-center justify-center"
      style={{ width: size, height: size }}
    >
      <svg
        width={size}
        height={size}
        viewBox={`0 0 ${size} ${size}`}
        style={{ transform: 'rotate(-90deg)' }}
      >
        {/* Piste de fond */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke={trackColor}
          strokeWidth={strokeWidth}
        />
        {/* Arc de progression */}
        <circle
          cx={center}
          cy={center}
          r={radius}
          fill="none"
          stroke={arcColor}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={`${circumference} ${circumference}`}
          strokeDashoffset={dashOffset}
        />
      </svg>

      {/* Image centrée */}
      {imageUrl ? (
        <img
          src={imageUrl}
          alt=""
          width={imageSize}
          height={imageSize}
          className="absolute object-contain"
          style={{ width: imageSize, height: imageSize }}
          onError={(e) => { (e.currentTarget as HTMLImageElement).style.display = 'none' }}
        />
      ) : (
        <div
          className="absolute rounded-full bg-muted/60"
          style={{ width: imageSize, height: imageSize }}
        />
      )}

      {/* Badge ✓ dès que maîtrisée (nouvellement OU de longue date) — cohérent avec
          l'anneau doré, qui suit la même condition. */}
      {mastered && (
        <div
          className="absolute bottom-0 right-0 flex items-center justify-center rounded-full bg-warning text-warning-foreground"
          style={{ width: 12, height: 12, fontSize: 7, fontWeight: 700 }}
        >
          ✓
        </div>
      )}
    </div>
  )
}
