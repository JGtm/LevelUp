/**
 * AssistTierBar — UNE barre d'assistances à trois tons, pour UN sens.
 *
 * Chaque segment est une tranche de part de dégâts de l'assistant : coup de pouce,
 * travail partagé, frag préparé — bornes côté Go (`domain.AssistTier*MaxPct`), libellés
 * d'infobulle dans assistsI18n.ts (seul foyer web des bornes : assistTiers.guard.test.ts). Les tons sont
 * des OPACITÉS de la couleur du sens (`assist-received` / `assist-given`), jamais une
 * autre teinte (skill color-tokens, famille des stats de combat).
 *
 * Les segments arrivent déjà calculés (assistExchange.ts) : le papillon de la page
 * Relations les prend au VOLUME (échelle log), la tuile de match à la PART
 * (`assistShareSegments`). Ce composant ne connaît que des largeurs.
 *
 * Extrait de AssistButterflyBar.tsx le 2026-09-18 (lot 3, tuile de match) : le
 * papillon en compose deux (gauche renversée, droite), la tuile en pose une seule.
 */
import { Tooltip } from '@/components/ui/tooltip'
import type { Locale } from '@/lib/i18n/locale'
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'

import type { AssistSegment } from './assistExchange'
import type { AssistTier, AssistsText } from './assistsI18n'

const TIER_OPACITY: Record<AssistTier, number> = { low: 0.35, mid: 0.65, high: 1 }

export const ASSIST_RECEIVED_TOKEN: SemanticToken = 'assist-received'
export const ASSIST_GIVEN_TOKEN: SemanticToken = 'assist-given'

/**
 * card : barre de 12 px, sans piste (carte Binôme).
 * row  : barre de 8 px sur piste (ligne du Noyau dur).
 * tile : barre de 6 px sur piste (tuile de match, sous la barre frags / assistances / décès).
 */
export type AssistTierBarVariant = 'card' | 'row' | 'tile'

const VARIANT: Record<AssistTierBarVariant, { bar: string; track: boolean }> = {
  card: { bar: 'h-3', track: false },
  row: { bar: 'h-2', track: true },
  tile: { bar: 'h-1.5', track: true },
}

export function AssistTierBar({
  segments,
  side = 'right',
  color,
  text,
  locale,
  variant,
  testId,
}: {
  segments: AssistSegment[]
  /** Sens de lecture : `left` = du centre vers la gauche (demi-barre gauche du papillon). */
  side?: 'left' | 'right'
  color: string
  text: AssistsText
  locale: Locale
  variant: AssistTierBarVariant
  /** Préfixe des `data-testid` de segment (`<testId>-<tier>`). */
  testId: string
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
              style={{ backgroundColor: color, opacity: TIER_OPACITY[s.tier] }}
              data-testid={`${testId}-${s.tier}`}
            />
          </Tooltip>
        </span>
      ))}
    </div>
  )
}
