/**
 * SessionFragDonutCard — carte A de « Frags et usages » : « Répartition des frags » du joueur de la
 * page, en ANNEAU (anneau intérieur : classe d'arme ; anneau extérieur : rôle), le total au centre.
 *
 * La forme d'avant la reconstruction de la page aux formes de l'Emprise, rétablie à la demande du
 * user (2026-10-07) : une barre empilée à un seul joueur ne dit rien de plus qu'un anneau et répète
 * le gamertag, alors que la page ne montre JAMAIS qu'un joueur. Même carte en pleine page et en vue
 * compacte du tiroir de comparaison (l'anneau se lit à toute largeur).
 *
 * Le graphe est `FragSunburst` nu, sa légende des classes `FragClassLegend`, centrée en bas de la
 * carte, sans filet ; survol lié entre les deux. Couleurs : classes de frag (`fragClassColor`).
 */
import { useState } from 'react'

import { FragClassLegend, FragSunburst } from '@/components/charts/FragSunburst'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import type { SessionCompareEntry } from '@/lib/api/types'

/** Hauteur de l'anneau, étiquettes de rôle comprises (viewBox 440 × 300 de `FragSunburst`). */
const DONUT_HEIGHT = 300
/** Largeur maximale de l'anneau : au-delà, il grossit sans rien montrer de plus. */
const DONUT_MAX_WIDTH = 480

interface Props {
  entry: SessionCompareEntry | null
  title: string
  info: string
}

export function SessionFragDonutCard({ entry, title, info }: Props) {
  const [hovered, setHovered] = useState<string | null>(null)
  const distribution = entry?.frag_distribution ?? null
  return (
    <div className="flex h-full min-w-0 flex-col rounded-lg border border-border bg-card" data-testid="session-frag-donut">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        <span className="flex items-center gap-1.5">
          {title}
          <InfoTooltip content={info} />
        </span>
      </div>
      <div className="flex flex-1 flex-col justify-center gap-2 p-3">
        <FragSunburst
          distribution={distribution}
          title={title}
          bare
          hideCenterLabel
          legendSide="none"
          heightPx={DONUT_HEIGHT}
          maxWidthPx={DONUT_MAX_WIDTH}
          externalHoveredClass={hovered}
          onClassHover={setHovered}
        />
        <FragClassLegend distribution={distribution} hoveredClass={hovered} onClassHover={setHovered} />
      </div>
    </div>
  )
}
