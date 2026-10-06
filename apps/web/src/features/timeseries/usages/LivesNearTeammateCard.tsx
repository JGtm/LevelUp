/**
 * LivesNearTeammateCard — « Vies à portée d'un coéquipier, vies isolées » (Séries temporelles ›
 * Usages, bloc « Isolement » ; maquette v4, décisions V3 et D5 du plan).
 *
 * Une ligne au gamertag du joueur : la barre ÉPAISSE partage ses vies terminées par une mort entre
 * « à portée d'un coéquipier » (`squad-player-1`, distance au plus proche ≤ la portée du radar du
 * match, à l'instant de la mort) et « isolée » (`extreme`), compte et part dans chaque segment quand
 * ils tiennent, repli au-dessus sinon ; la barre FINE partage ses frags tombés pendant ces vies ; dessous,
 * « frags : n · p % · x par vie … y par vie · m ». Les vies écartées (aucun coéquipier situé, carte
 * sans portée connue, journal des morts non publiable) sont comptées dans l'aide ⓘ. Axe 0-100 %.
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

/**
 * Vue compacte du tiroir de comparaison de Sessions (maquette `makeLife` avec `cp`) : les parts
 * entières seules dans la barre épaisse et sur la ligne des frags (`killsLine`, `killsLineAlone`).
 */
interface LivesCompact {
  killsLine: (pct: string, perLife: string) => string
  killsLineAlone: (perLife: string) => string
}

export function LivesNearTeammateCard({ model, player, ut, compact }: { model: LivesModel; player: string; ut: UsagesCardsText; compact?: LivesCompact }) {
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
  const nearText = compact ? ut.pctIntFmt(nearPct) : `${ut.intFmt(model.near.lives)} · ${ut.pctFmt(nearPct)}`
  const aloneText = compact ? ut.pctIntFmt(100 - nearPct) : `${ut.pctFmt(100 - nearPct)} · ${ut.intFmt(model.alone.lives)}`
  // Seule la valeur qui ne tient pas dans son segment monte au repli (jamais affichée deux fois).
  const nearHidden = hidden.has('near')
  const aloneHidden = hidden.has('alone')
  return (
    <ObjectifFrame title={l.title} info={l.info(model.excludedUnlocated, model.excludedNoRadar, model.excludedUnpublishable)} legend={legend} testId="usages-lives">
      <div ref={ref} className="flex flex-col gap-3.5">
        <div className="grid items-center gap-3" style={{ gridTemplateColumns: COLUMNS }}>
          <div className="min-w-0 text-[12.5px] leading-tight">
            {player}
            <small className="block text-[11px] text-muted-foreground" data-testid="usages-lives-sub">
              {l.rowSub(model.lives)}
            </small>
          </div>
          <div className="flex min-w-0 flex-col gap-1">
            {(nearHidden || aloneHidden) && (
              <div className="flex justify-between gap-2 text-xs tabular-nums text-muted-foreground" data-testid="usages-lives-repli">
                <span>{nearHidden && nearText}</span>
                <span>{aloneHidden && aloneText}</span>
              </div>
            )}
            <div className="relative h-[22px] rounded-[3px] bg-muted">
              <Part id="near" left={0} width={nearPct} color={NEAR_INK} text={nearText} hidden={hidden} tip={l.tip(l.near, model.near.lives, model.lives, ut.pctFmt(nearPct), 'lives')} align="start" />
              <Part id="alone" left={nearPct} width={100 - nearPct} color={ALONE_INK} text={aloneText} hidden={hidden} tip={l.tip(l.alone, model.alone.lives, model.lives, ut.pctFmt(100 - nearPct), 'lives')} align="end" />
            </div>
            {model.killsNearShare != null && <KillsLines model={model} ut={ut} compact={compact} />}
          </div>
        </div>
        <TrackAxis columns={COLUMNS} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? ut.pctIntFmt(100) : String(v) }))} />
      </div>
    </ObjectifFrame>
  )
}

function KillsLines({ model, ut, compact }: { model: LivesModel; ut: UsagesCardsText; compact: LivesCompact | undefined }) {
  const l = ut.lives
  const share = (model.killsNearShare ?? 0) * 100
  const perLife = (v: number | null) => (v == null ? '—' : l.perLifeFmt(v))
  const left = compact ? compact.killsLine(ut.pctIntFmt(share), perLife(model.perLifeNear)) : l.killsLine(model.near.kills, ut.pctFmt(share), perLife(model.perLifeNear))
  const right = compact ? compact.killsLineAlone(perLife(model.perLifeAlone)) : l.killsLineAlone(perLife(model.perLifeAlone), model.alone.kills)
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
        <span>{left}</span>
        <span>{right}</span>
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
