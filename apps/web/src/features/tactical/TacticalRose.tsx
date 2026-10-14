/**
 * TacticalRose — la rose des directions des tueurs : huit bâtons depuis le centre (la victime), le haut
 * du plan en haut, dans le sens horaire. Le secteur k est centré sur 45 k degrés depuis le haut
 * (`coordination.Rose8`) ; la longueur d'un bâton est son compte rapporté au plus fourni. Les comptes
 * viennent du serveur, jamais recalculés ici. Couleur : jeton d'équipe adverse (les tueurs).
 */
import { tokenCssVar } from '@/lib/accessibility/semantic-tokens'

import { batonsDeLaRose } from './mort.logic'

const TAILLE = 120
const CENTRE = TAILLE / 2
const RAYON_MAX = 50
const RAYON_ANNEAU = 52
const RAYON_ANNEAU_INTERIEUR = 26
const EPAISSEUR_BATON = 9

interface TacticalRoseProps {
  rose: readonly number[]
  label: string
}

export function TacticalRose({ rose, label }: TacticalRoseProps) {
  const batons = batonsDeLaRose(rose)
  return (
    <svg
      viewBox={`0 0 ${TAILLE} ${TAILLE}`}
      role="img"
      aria-label={label}
      className="size-[120px] flex-none"
      data-testid="tactical-rose"
    >
      <circle cx={CENTRE} cy={CENTRE} r={RAYON_ANNEAU} fill="none" className="stroke-border" strokeWidth={1} />
      <circle cx={CENTRE} cy={CENTRE} r={RAYON_ANNEAU_INTERIEUR} fill="none" className="stroke-border" strokeWidth={1} opacity={0.6} />
      {batons.map((b) => {
        const rad = (b.angleDeg * Math.PI) / 180
        const longueur = Math.max(b.longueur, 0.12) * RAYON_MAX
        return (
          <line
            key={b.secteur}
            x1={CENTRE}
            y1={CENTRE}
            x2={CENTRE + longueur * Math.sin(rad)}
            y2={CENTRE - longueur * Math.cos(rad)}
            stroke={tokenCssVar('team-enemy')}
            strokeWidth={EPAISSEUR_BATON}
            strokeLinecap="round"
            opacity={0.9}
            data-secteur={b.secteur}
            data-n={b.n}
          />
        )
      })}
    </svg>
  )
}
