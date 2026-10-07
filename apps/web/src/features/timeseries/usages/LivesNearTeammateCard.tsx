/**
 * LivesNearTeammateCard — « Vies à portée d'un coéquipier, vies isolées » (Séries temporelles ›
 * Usages, bloc « Isolement » ; maquette v4, décisions V3 et D5 du plan).
 *
 * Une ligne au gamertag du joueur (`LivesNearTeammateRow`) : la barre ÉPAISSE partage ses vies
 * terminées par une mort entre « à portée d'un coéquipier » et « isolée », la barre FINE ses frags
 * tombés pendant ces vies, et la ligne des frags par vie dessous. Les vies écartées (aucun coéquipier
 * situé, carte sans portée connue, journal des morts non publiable) sont comptées dans l'aide ⓘ.
 * Axe 0-100 %.
 */
import { useMemo, useRef } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'

import { TrackAxis } from '@/features/squad/emprise/PisteCampsForm'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'

import { LIVES_ALONE_INK, LIVES_COLUMNS, LIVES_NEAR_INK } from './livesLayout'
import { LivesNearTeammateRow, type LivesCompact } from './LivesNearTeammateRow'
import type { LivesModel } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

const TICKS = [0, 25, 50, 75, 100] as const

export function LivesNearTeammateCard({ model, player, ut, compact }: { model: LivesModel; player: string; ut: UsagesCardsText; compact?: LivesCompact }) {
  const l = ut.lives
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, model)
  const legend = useMemo(() => <LivesLegend ut={ut} />, [ut])
  return (
    <ObjectifFrame title={l.title} info={l.info(model.excludedUnlocated, model.excludedNoRadar, model.excludedUnpublishable)} legend={legend} testId="usages-lives">
      <div ref={ref} className="flex flex-col gap-3.5">
        <LivesNearTeammateRow model={model} label={player} hidden={hidden} ut={ut} compact={compact} />
        <LivesAxis ut={ut} />
      </div>
    </ObjectifFrame>
  )
}

/** La légende de la carte « Isolement » : à portée, isolée, barre fine des frags. */
export function LivesLegend({ ut }: { ut: UsagesCardsText }) {
  const l = ut.lives
  return (
    <ObjectifLegend
      ariaLabel={l.title}
      items={[
        { kind: 'square', label: l.near, color: LIVES_NEAR_INK },
        { kind: 'square', label: l.alone, color: LIVES_ALONE_INK },
        { kind: 'thin', label: l.thinLegend, color: 'var(--muted-foreground)' }, // color-allow: repère neutre de la légende (maquette)
      ]}
    />
  )
}

/** L'axe 0-100 % sous les lignes. */
export function LivesAxis({ ut }: { ut: UsagesCardsText }) {
  return <TrackAxis columns={LIVES_COLUMNS} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? ut.pctIntFmt(100) : String(v) }))} />
}
