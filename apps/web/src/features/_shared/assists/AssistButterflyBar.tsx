/**
 * AssistButterflyBar — barre papillon des assistances échangées.
 *
 * Gauche : il t'a assisté. Droite : tu l'as assisté. Longueur = VOLUME d'assistances sur
 * une échelle log commune à la page (assistExchange.ts). Chaque demi-barre est une
 * AssistTierBar : trois tons de la couleur du sens, du centre vers l'extérieur — coup de
 * pouce (< 25 % des dégâts), travail partagé (25-50 %), frag préparé (> 50 %), infobulle
 * « N frags assistés · tranche » par segment.
 *
 * Couleurs : `assist-received` (l'autre te sert) et `assist-given` (tu le sers), famille
 * des stats de combat. Les tons sont des clartés de cette couleur (`assistTierTone`), pas
 * d'autres couleurs.
 */
import { tokenCssVar } from '@/lib/accessibility'
import type { AssistTiers, RelationAssists } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'

import { ASSIST_GIVEN_TOKEN, ASSIST_RECEIVED_TOKEN, AssistTierBar } from './AssistTierBar'
import { assistSegments } from './assistExchange'
import type { AssistsText } from './assistsI18n'

type Variant = 'card' | 'row'

const VARIANT: Record<Variant, { axis: string; axisWidth: string }> = {
  // Carte Binôme : axe de 18 px.
  card: { axis: 'h-[18px] bg-foreground/60', axisWidth: '2px' },
  // Ligne du Noyau dur : axe fin.
  row: { axis: 'h-2.5 bg-muted-foreground', axisWidth: '1px' },
}

export function AssistButterflyBar({
  assists,
  volumeMax,
  text,
  locale,
  variant,
}: {
  assists: RelationAssists
  /** Plus gros volume d'un sens sur la page : borne de l'échelle log (assistExchange.ts). */
  volumeMax: number
  text: AssistsText
  locale: Locale
  variant: Variant
}) {
  const v = VARIANT[variant]
  const half = (tiers: AssistTiers) => assistSegments(tiers, volumeMax)
  return (
    <div
      className="grid items-center"
      style={{ gridTemplateColumns: `1fr ${v.axisWidth} 1fr` }}
      data-testid="assist-butterfly"
    >
      <AssistTierBar
        segments={half(assists.received)}
        side="left"
        color={tokenCssVar(ASSIST_RECEIVED_TOKEN)}
        text={text}
        locale={locale}
        variant={variant}
        testId="assist-segment-left"
      />
      <span className={`block w-full ${v.axis}`} aria-hidden="true" />
      <AssistTierBar
        segments={half(assists.given)}
        side="right"
        color={tokenCssVar(ASSIST_GIVEN_TOKEN)}
        text={text}
        locale={locale}
        variant={variant}
        testId="assist-segment-right"
      />
    </div>
  )
}
