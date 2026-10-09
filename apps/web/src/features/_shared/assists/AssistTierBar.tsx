/**
 * AssistTierBar — UNE barre d'assistances à trois tons, pour UN sens.
 *
 * Chaque segment est une tranche de part de dégâts de l'assistant : coup de pouce,
 * travail partagé, frag préparé — bornes côté Go (`domain.AssistTier*MaxPct`), libellés
 * d'infobulle dans assistsI18n.ts (seul foyer web des bornes : assistTiers.guard.test.ts). Les tons sont
 * trois CLARTÉS de la couleur du sens (`assist-received` / `assist-given`, cf.
 * `assistTierTone`), jamais une autre teinte (skill color-tokens, famille des stats de combat).
 *
 * Les segments arrivent déjà calculés (assistExchange.ts) : le papillon de la page
 * Relations les prend au VOLUME (échelle log), la tuile de match à la PART
 * (`assistShareSegments`). Ce composant ne connaît que des largeurs.
 *
 * Le papillon en compose deux (gauche renversée, droite), la tuile en pose une seule.
 *
 * `notMeasured` (tuile de match) : la part de la base dont l'assistance n'est pas lue,
 * hachurée au bout EXTÉRIEUR de la piste — jamais dans le vide de la piste, qui dit « non
 * assisté ». Entre les deux reste la piste nue : les frags connus non assistés.
 */
import { NOT_MEASURED_HATCH } from '@/components/charts/notMeasuredHatch'
import { Tooltip } from '@/components/ui/tooltip'
import type { Locale } from '@/lib/i18n/locale'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import type { AssistSegment } from './assistExchange'
import type { AssistsText } from './assistsI18n'
import { assistTierTone } from './assistTierTone'

export const ASSIST_RECEIVED_TOKEN: SemanticToken = 'assist-received'
export const ASSIST_GIVEN_TOKEN: SemanticToken = 'assist-given'

/**
 * card : barre de 12 px, sans piste (carte Binôme).
 * row  : barre de 8 px sur piste (ligne du Noyau dur).
 * tile : barre de 8 px sur piste (tuile de match, sous la barre frags / assistances / décès,
 *        même hauteur qu'elle).
 */
export type AssistTierBarVariant = 'card' | 'row' | 'tile'

/** Part non mesurée de la base : largeur en % de la piste, infobulle déjà rédigée. */
export interface AssistNotMeasuredSegment {
  widthPct: number
  tooltip: string
}

const VARIANT: Record<AssistTierBarVariant, { bar: string; track: boolean }> = {
  card: { bar: 'h-3', track: false },
  row: { bar: 'h-2', track: true },
  tile: { bar: 'h-2', track: true },
}

export function AssistTierBar({
  segments,
  side = 'right',
  color,
  text,
  locale,
  variant,
  testId,
  notMeasured,
}: {
  segments: AssistSegment[]
  /** Sens de lecture : `left` = du centre vers la gauche (demi-barre gauche du papillon). */
  side?: 'left' | 'right'
  color: string
  text: AssistsText
  locale: Locale
  variant: AssistTierBarVariant
  /** Préfixe des `data-testid` de segment (`<testId>-<tier>`, `<testId>-not-measured`). */
  testId: string
  notMeasured?: AssistNotMeasuredSegment | null
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
        // ligne de base et sortirait en partie du cadre rogné de la barre.
        <span key={s.tier} className="flex h-full" style={{ width: `${s.widthPct}%` }}>
          <Tooltip className="h-full w-full" content={text.segment(s.count.toLocaleString(locale), s.tier)}>
            <span
              className="block h-full w-full cursor-help"
              style={{ backgroundColor: assistTierTone(color, s.tier) }}
              data-testid={`${testId}-${s.tier}`}
            />
          </Tooltip>
        </span>
      ))}
      {notMeasured && notMeasured.widthPct > 0 && (
        <span
          className={`flex h-full ${side === 'left' ? 'order-first mr-auto' : 'ml-auto'}`}
          style={{ width: `${notMeasured.widthPct}%` }}
        >
          <Tooltip className="h-full w-full" content={notMeasured.tooltip}>
            <span className="block h-full w-full cursor-help" style={NOT_MEASURED_HATCH} data-testid={`${testId}-not-measured`} />
          </Tooltip>
        </span>
      )}
    </div>
  )
}
