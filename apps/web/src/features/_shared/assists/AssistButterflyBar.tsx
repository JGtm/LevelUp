/**
 * AssistButterflyBar — barre papillon des assistances échangées.
 *
 * Gauche : il t'a assisté. Droite : tu l'as assisté. Longueur = VOLUME d'assistances sur
 * une échelle log commune à la page (assistExchange.ts). Chaque demi-barre est découpée
 * en trois tons de la couleur du joueur, du centre vers l'extérieur : coup de pouce
 * (< 25 % des dégâts), travail partagé (25-50 %), frag préparé (> 50 %). Chaque segment porte son infobulle « N frags assistés · tranche ».
 *
 * Couleurs : `compare-b` pour l'autre joueur, `compare-a` pour le joueur (même paire que
 * les comparaisons à deux joueurs). Les tons sont des opacités, pas des couleurs.
 */
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'
import type { AssistTiers, RelationAssists } from '@/lib/api/types'
import type { Locale } from '@/lib/i18n/locale'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import { assistSegments, type AssistSegment } from './assistExchange'
import type { AssistTier, AssistsText } from './assistsI18n'

const TIER_OPACITY: Record<AssistTier, number> = { low: 0.35, mid: 0.65, high: 1 }

export const ASSIST_RECEIVED_TOKEN: SemanticToken = 'compare-b'
export const ASSIST_GIVEN_TOKEN: SemanticToken = 'compare-a'

type Variant = 'card' | 'row'

const VARIANT: Record<Variant, { bar: string; axis: string; axisWidth: string; track: boolean }> = {
  // Carte Binôme : barre de 12 px, axe de 18 px, pas de piste.
  card: { bar: 'h-3', axis: 'h-[18px] bg-foreground/60', axisWidth: '2px', track: false },
  // Ligne du Noyau dur : barre de 8 px sur piste, axe fin.
  row: { bar: 'h-2', axis: 'h-2.5 bg-muted-foreground', axisWidth: '1px', track: true },
}

function Half({
  segments,
  side,
  color,
  text,
  locale,
  variant,
}: {
  segments: AssistSegment[]
  side: 'left' | 'right'
  color: string
  text: AssistsText
  locale: Locale
  variant: Variant
}) {
  const v = VARIANT[variant]
  // Du centre vers l'extérieur : à gauche l'ordre se lit donc à l'envers.
  const ordered = side === 'left' ? [...segments].reverse() : segments
  return (
    <div
      className={`flex ${v.bar} overflow-hidden ${side === 'left' ? 'justify-end' : 'justify-start'} ${
        v.track ? 'rounded-sm bg-muted' : ''
      }`}
    >
      {ordered.map((s) => (
        // `flex` (pas `block`) : l'ancre inline-flex du Tooltip s'alignerait sinon sur la
        // ligne de base et sortirait en partie du cadre rogné de la demi-barre.
        <span key={s.tier} className="flex h-full" style={{ width: `${s.widthPct}%` }}>
          <Tooltip className="h-full w-full" content={text.segment(s.count.toLocaleString(locale), s.tier)}>
            <span
              className="block h-full w-full cursor-help"
              style={{ backgroundColor: color, opacity: TIER_OPACITY[s.tier] }}
              data-testid={`assist-segment-${side}-${s.tier}`}
            />
          </Tooltip>
        </span>
      ))}
    </div>
  )
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
      <Half
        segments={half(assists.received)}
        side="left"
        color={tokenCssVar(ASSIST_RECEIVED_TOKEN)}
        text={text}
        locale={locale}
        variant={variant}
      />
      <span className={`block w-full ${v.axis}`} aria-hidden="true" />
      <Half
        segments={half(assists.given)}
        side="right"
        color={tokenCssVar(ASSIST_GIVEN_TOKEN)}
        text={text}
        locale={locale}
        variant={variant}
      />
    </div>
  )
}
