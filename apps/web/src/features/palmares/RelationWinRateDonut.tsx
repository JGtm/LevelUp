/**
 * RelationWinRateDonut — donut de taux de victoire d'une relation (binôme / bête
 * noire / noyau) avec REPÈRE de la moyenne perso historique posé sur l'anneau.
 *
 * Lecture voulue (option « repère sur l'anneau », validée UX 2026-07-18) :
 *   - l'arc VERT (victoires) part du haut, sens horaire, sur une piste NON REMPLIE
 *     (le reste n'est pas peint en rouge — évite le donut « tout rouge ») ;
 *   - un repère fin (couleur foreground) marque la position de la moyenne perso ;
 *     l'arc dépasse le repère ⇒ sur-performance ; s'arrête avant ⇒ sous-perf.
 *     Reste lisible que la moyenne soit au-dessus OU en-dessous du taux affiché.
 *   - centre : % en BLANC + delta signé « ±N pts » (coloré) vs la moyenne perso ;
 *   - à DROITE : caption de contexte + légende (victoires / défaites / moy. perso).
 *
 * SVG pur (responsive via viewBox, pas d'ECharts) : la couleur d'arc passe par
 * tokenCssVar (outcome-win) ; la piste, le repère et le texte neutre héritent de la
 * couleur `foreground` (currentColor) — pas de hex ni de classe Tailwind de palette
 * (la piste est une couleur structurelle SVG, exception tolérée skill color-tokens).
 */
import { tokenCssVar } from '@/lib/accessibility'
import { formatPercentInt } from '@/lib/formatters'

interface DonutLabels {
  wins: string
  losses: string
  personalAvg: string
  pointsUnit: string
  liftTooltip: string
}

interface Props {
  /** Taux de victoire de la relation (0..1). null ⇒ placeholder « — ». */
  winRate: number | null | undefined
  /** Moyenne perso historique (0..1) — repère sur l'anneau. null ⇒ pas de repère. */
  personalAvg: number | null | undefined
  labels: DonutLabels
  /** Libellé de contexte affiché au-dessus de la légende (ex. « Taux de victoires ensemble »). */
  caption?: string
  /** Diamètre du donut en px (défaut 116). */
  size?: number
}

// Géométrie de l'anneau (unités viewBox 0..100).
const CENTER = 50
const R = 40
const STROKE = 13
const CIRC = 2 * Math.PI * R

const clamp01 = (v: number): number => Math.max(0, Math.min(1, v))

// Point (x, y) sur un cercle de rayon `radius`, à la fraction `frac` (0..1) du
// tour, départ en HAUT (midi) et sens HORAIRE — cohérent avec l'arc SVG.
function polar(frac: number, radius: number): { x: number; y: number } {
  const a = frac * 2 * Math.PI - Math.PI / 2
  return { x: CENTER + radius * Math.cos(a), y: CENTER + radius * Math.sin(a) }
}

export function RelationWinRateDonut({ winRate, personalAvg, labels, caption, size = 116 }: Props) {
  const hasWr = winRate != null && Number.isFinite(winRate)
  if (!hasWr) {
    return <span className="font-mono text-2xl font-bold text-muted-foreground">—</span>
  }
  const wr = clamp01(winRate as number)
  const hasAvg = personalAvg != null && Number.isFinite(personalAvg)
  const avg = hasAvg ? clamp01(personalAvg as number) : null

  // Delta signé en points (arrondi) vs la moyenne perso ; couleur au signe.
  const deltaPts = hasAvg ? Math.round(((winRate as number) - (personalAvg as number)) * 100) : null
  const deltaColor =
    deltaPts == null || deltaPts === 0
      ? tokenCssVar('outcome-draw')
      : deltaPts > 0
        ? tokenCssVar('outcome-win')
        : tokenCssVar('outcome-loss')
  const deltaSign = deltaPts != null && deltaPts > 0 ? '+' : deltaPts != null && deltaPts < 0 ? '−' : ''

  const winDash = wr * CIRC
  // Repère : segment radial traversant l'anneau (déborde de 2 unités de chaque côté).
  const tick =
    avg != null
      ? { inner: polar(avg, R - STROKE / 2 - 2), outer: polar(avg, R + STROKE / 2 + 2) }
      : null
  // Arc « déficit » : remplit le vide entre le taux (wr) et la moyenne perso (avg)
  // QUAND on est EN DESSOUS de sa moyenne, en rouge, pour visualiser l'écart. Au-dessus,
  // l'arc vert des victoires dépasse déjà le repère → pas de vide à combler.
  const deficitPath =
    avg != null && wr < avg
      ? (() => {
          const p1 = polar(wr, R)
          const p2 = polar(avg, R)
          const large = avg - wr > 0.5 ? 1 : 0
          return `M ${p1.x} ${p1.y} A ${R} ${R} 0 ${large} 1 ${p2.x} ${p2.y}`
        })()
      : null

  return (
    <div className="flex items-center gap-4">
      <div className="relative shrink-0 text-foreground" style={{ width: size, height: size }}>
        <svg viewBox="0 0 100 100" className="h-full w-full" role="img" aria-label={labels.wins}>
          {/* Piste NON REMPLIE (le reste — défaites — n'est pas peint en rouge) */}
          <circle
            cx={CENTER}
            cy={CENTER}
            r={R}
            fill="none"
            stroke="currentColor"
            strokeOpacity={0.15}
            strokeWidth={STROKE}
          />
          {/* Arc victoires (toujours vert) : part du haut (rotate -90), sens horaire */}
          <circle
            cx={CENTER}
            cy={CENTER}
            r={R}
            fill="none"
            style={{ stroke: tokenCssVar('outcome-win') }}
            strokeWidth={STROKE}
            strokeDasharray={`${winDash} ${CIRC - winDash}`}
            transform={`rotate(-90 ${CENTER} ${CENTER})`}
          />
          {/* Arc déficit (rouge) : écart taux → moyenne perso quand on est en dessous */}
          {deficitPath && (
            <path d={deficitPath} fill="none" style={{ stroke: tokenCssVar('outcome-loss') }} strokeWidth={STROKE} />
          )}
          {/* Repère moyenne perso (couleur foreground via currentColor) */}
          {tick && (
            <line
              x1={tick.inner.x}
              y1={tick.inner.y}
              x2={tick.outer.x}
              y2={tick.outer.y}
              stroke="currentColor"
              strokeWidth={2.4}
              strokeLinecap="round"
            />
          )}
        </svg>
        {/* Centre : % en blanc + delta signé coloré */}
        <div className="absolute inset-0 flex flex-col items-center justify-center">
          <span className="font-mono text-xl font-bold leading-none text-foreground">{formatPercentInt(wr)}</span>
          {deltaPts != null && (
            <span
              className="mt-0.5 font-mono text-2xs font-bold leading-none"
              style={{ color: deltaColor }}
              title={labels.liftTooltip}
            >
              {deltaSign}
              {Math.abs(deltaPts)} {labels.pointsUnit}
            </span>
          )}
        </div>
      </div>

      {/* Colonne de droite : contexte + légende */}
      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        {caption && (
          <p className="text-2xs uppercase tracking-label-md text-muted-foreground">{caption}</p>
        )}
        <ul className="flex flex-col gap-1.5 text-xs">
          <li className="flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: tokenCssVar('outcome-win') }}
              aria-hidden="true"
            />
            <span className="text-muted-foreground">{labels.wins}</span>
            <span className="ml-auto font-mono font-semibold" style={{ color: tokenCssVar('outcome-win') }}>
              {formatPercentInt(wr)}
            </span>
          </li>
          <li className="flex items-center gap-2">
            <span
              className="h-2.5 w-2.5 shrink-0 rounded-full"
              style={{ backgroundColor: tokenCssVar('outcome-loss') }}
              aria-hidden="true"
            />
            <span className="text-muted-foreground">{labels.losses}</span>
            <span className="ml-auto font-mono font-semibold" style={{ color: tokenCssVar('outcome-loss') }}>
              {formatPercentInt(1 - wr)}
            </span>
          </li>
          {avg != null && (
            <li className="flex items-center gap-2 border-t border-border pt-1.5">
              {/* Pastille = repère de l'anneau (barre foreground, pas un rond) */}
              <span className="flex h-2.5 w-2.5 shrink-0 items-center justify-center" aria-hidden="true">
                <span className="h-2.5 w-[2.5px] rounded-full bg-foreground" />
              </span>
              <span className="text-muted-foreground">{labels.personalAvg}</span>
              <span className="ml-auto font-mono font-semibold text-foreground">{formatPercentInt(avg)}</span>
            </li>
          )}
        </ul>
      </div>
    </div>
  )
}
