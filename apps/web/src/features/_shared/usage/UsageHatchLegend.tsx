/**
 * UsageHatchLegend — LA LÉGENDE DE LA TEXTURE (D7, 2026-09-21).
 *
 * LE PLEIN ET LA HACHURE ÉTAIENT EXPLIQUÉS DANS UNE INFOBULLE D'EN-TÊTE, donc nulle part :
 * une phrase qui ne s'ouvre qu'au survol d'un titre n'est lue par personne, et le lecteur
 * voyait deux colonnes sur trois rayées sans savoir ce que la rayure disait. La légende est
 * désormais POSÉE SOUS LA FORME qu'elle explique, avec les vraies encres.
 *
 * DEUX VARIANTES, parce que la hachure ne dit pas la même chose selon la forme :
 *   - `gauge` (grilles de jauges) : plein = rapporté à mon équipe, hachuré = rapporté au lobby ;
 *   - `track` (piste du lobby)    : coloré = nous, hachure neutre = les adversaires.
 *
 * Les textures viennent de `usageInks.ts` — jamais d'une seconde définition locale, sinon la
 * légende finit par peindre autre chose que ce qu'elle légende.
 */
import type { CSSProperties } from 'react'

import { ALLY_INK, ENEMY_HATCH, LOBBY_HATCH } from './usageInks'
import type { UsageText } from './usageI18n'

/** Une pastille de légende : l'aplat de « nous », éventuellement recouvert d'une texture. */
function LegendSwatch({ hatch }: { hatch?: CSSProperties }) {
  return (
    <span
      aria-hidden="true"
      className="relative inline-block h-3 w-5 flex-none"
      style={{ backgroundColor: ALLY_INK }}
    >
      {hatch != null && <span className="absolute inset-0" style={hatch} />}
    </span>
  )
}

/** La pastille « eux » : pas d'aplat d'équipe du tout, la hachure neutre seule. */
function EnemySwatch() {
  return <span aria-hidden="true" className="inline-block h-3 w-5 flex-none" style={ENEMY_HATCH} />
}

export function UsageHatchLegend({
  t,
  variant,
  className = '',
}: {
  t: UsageText
  /** `gauge` : plein vs lobby hachuré. `track` : nous coloré vs eux en hachure neutre. */
  variant: 'gauge' | 'track'
  className?: string
}) {
  return (
    <div
      className={`mt-1.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-3xs text-muted-foreground ${className}`}
    >
      <span className="flex items-center gap-1.5">
        <LegendSwatch />
        {t.hatchLegendSolid}
      </span>
      {variant === 'gauge' ? (
        <span className="flex items-center gap-1.5">
          <LegendSwatch hatch={LOBBY_HATCH} />
          {t.hatchLegendLobby}
        </span>
      ) : (
        <span className="flex items-center gap-1.5">
          <EnemySwatch />
          {t.hatchLegendEnemy}
        </span>
      )}
    </div>
  )
}
