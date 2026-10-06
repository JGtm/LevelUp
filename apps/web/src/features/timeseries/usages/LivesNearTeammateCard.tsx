/**
 * LivesNearTeammateCard — « Mes vies : près d'un coéquipier ou seul » (Séries temporelles › Usages,
 * bloc « Près d'un coéquipier ou seul » ; maquette v4, décisions V3 et D5 du plan).
 *
 * Une ligne « Mes vies » : la barre ÉPAISSE partage mes vies terminées par une mort entre « près
 * d'un coéquipier » (`squad-player-1`, distance au plus proche ≤ la portée du radar du match, à
 * l'instant de la mort) et « seul » (`extreme`), compte et part dans chaque segment quand ils
 * tiennent, repli au-dessus sinon ; la barre FINE partage mes frags tombés pendant ces vies ; dessous,
 * « frags : n · p % · x par vie … y par vie · m ». Les vies écartées (aucun coéquipier situé, carte
 * sans portée connue) sont comptées dans l'aide ⓘ. Axe 0-100 %.
 */
import { useMemo, useRef } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar } from '@/lib/accessibility'

import { squadPlayerInk } from '@/features/squad/formes/colors'
import { TrackAxis } from '@/features/squad/emprise/PisteCampsForm'
import { pisteColumns } from '@/features/squad/emprise/pisteLayout'
import { TipText } from '@/features/squad/emprise/TipText'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'

import type { LivesModel } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

const NEAR_INK = squadPlayerInk(0)
const ALONE_INK = tokenCssVar('extreme')
const COLUMNS = pisteColumns(150)
const TICKS = [0, 25, 50, 75, 100] as const

export function LivesNearTeammateCard({ model, ut }: { model: LivesModel; ut: UsagesCardsText }) {
  const l = ut.lives
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, model)
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={l.title}
        items={[
          { kind: 'square', label: l.near, color: NEAR_INK },
          { kind: 'square', label: l.alone, color: ALONE_INK },
          { kind: 'thin', label: l.thinLegend, color: 'var(--muted-foreground)' }, // color-allow: repère neutre de la légende (maquette)
        ]}
      />
    ),
    [l],
  )
  const nearPct = model.livesNearShare * 100
  const nearText = `${ut.intFmt(model.near.lives)} · ${ut.pctFmt(nearPct)}`
  const aloneText = `${ut.pctFmt(100 - nearPct)} · ${ut.intFmt(model.alone.lives)}`
  const repli = hidden.has('near') || hidden.has('alone')
  return (
    <ObjectifFrame title={l.title} info={l.info(model.excludedUnlocated, model.excludedNoRadar)} legend={legend} testId="usages-lives">
      <div ref={ref} className="flex flex-col gap-3.5">
        <div className="grid items-center gap-3" style={{ gridTemplateColumns: COLUMNS }}>
          <div className="min-w-0 text-[12.5px] leading-tight">
            {l.rowLabel}
            <small className="block text-[11px] text-muted-foreground" data-testid="usages-lives-sub">
              {l.rowSub(model.lives)}
            </small>
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            {repli && (
              <div className="flex justify-between gap-2 text-xs tabular-nums text-muted-foreground" data-testid="usages-lives-repli">
                <span>{nearText}</span>
                <span>{aloneText}</span>
              </div>
            )}
            <div className="relative h-[22px] rounded-[3px] bg-muted">
              <Part id="near" left={0} width={nearPct} color={NEAR_INK} text={nearText} hidden={hidden} tip={l.tip(l.near, model.near.lives, model.lives, ut.pctFmt(nearPct), 'lives')} align="start" />
              <Part id="alone" left={nearPct} width={100 - nearPct} color={ALONE_INK} text={aloneText} hidden={hidden} tip={l.tip(l.alone, model.alone.lives, model.lives, ut.pctFmt(100 - nearPct), 'lives')} align="end" />
            </div>
            {model.killsNearShare != null && <KillsLines model={model} ut={ut} />}
          </div>
        </div>
        <TrackAxis columns={COLUMNS} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? ut.pctIntFmt(100) : String(v) }))} />
      </div>
    </ObjectifFrame>
  )
}

function KillsLines({ model, ut }: { model: LivesModel; ut: UsagesCardsText }) {
  const l = ut.lives
  const share = (model.killsNearShare ?? 0) * 100
  const perLife = (v: number | null) => (v == null ? '—' : l.perLifeFmt(v))
  return (
    <>
      <div className="relative h-2 rounded-[3px] bg-muted">
        {model.near.kills > 0 && (
          <ThinPart id="usages-lives-kills-near" left={0} width={share} color={NEAR_INK} tip={l.tip(l.near, model.near.kills, model.kills, ut.pctFmt(share), 'kills')} />
        )}
        {model.alone.kills > 0 && (
          <ThinPart id="usages-lives-kills-alone" left={share} width={100 - share} color={ALONE_INK} tip={l.tip(l.alone, model.alone.kills, model.kills, ut.pctFmt(100 - share), 'kills')} />
        )}
      </div>
      <div className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground" data-testid="usages-lives-kills-line">
        <span>{l.killsLine(model.near.kills, ut.pctFmt(share), perLife(model.perLifeNear))}</span>
        <span>{l.killsLineAlone(perLife(model.perLifeAlone), model.alone.kills)}</span>
      </div>
    </>
  )
}

function Part({
  id,
  left,
  width,
  color,
  text,
  hidden,
  tip,
  align,
}: {
  id: string
  left: number
  width: number
  color: string
  text: string
  hidden: ReadonlySet<string>
  tip: string
  align: 'start' | 'end'
}) {
  if (width <= 0) return null
  return (
    <div className="absolute inset-y-0" style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color }} data-fit-key={id} data-testid={`usages-lives-${id}`}>
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div className={`flex h-full w-full cursor-help items-center overflow-hidden ${align === 'start' ? 'justify-start pl-[7px]' : 'justify-end pr-[7px]'}`}>
          {/* `text-white` : écriture posée sur l'aplat, contraste dans le segment et non couleur sémantique. */}
          <span data-fit-label className="whitespace-nowrap text-xs font-medium tabular-nums leading-none text-white" style={{ visibility: hidden.has(id) ? 'hidden' : 'visible' }}>
            {text}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}

function ThinPart({ id, left, width, color, tip }: { id: string; left: number; width: number; color: string; tip: string }) {
  return (
    <div className="absolute inset-y-0" style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color }} data-testid={id}>
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div className="h-full w-full cursor-help" />
      </Tooltip>
    </div>
  )
}
