/**
 * LivesNearTeammateRow — LA LIGNE D'UN JOUEUR de la carte « Isolement » : la barre ÉPAISSE partage ses
 * vies terminées par une mort entre « à portée d'un coéquipier » (`squad-player-1`) et « isolée »
 * (`extreme`), compte et part dans chaque segment quand ils tiennent, sinon à côté de la barre, du côté de
 * son camp, sur la même ligne (`BarSides`, place réservée : `livesInset`) ; la barre
 * FINE partage ses frags tombés pendant ces vies ; dessous, « frags : n · p % · x par vie … ». Une
 * ligne pour les Séries temporelles et Sessions (le joueur de la page), une par joueur de l'équipe sur
 * la Vue match. La mesure des valeurs qui tiennent (`hidden`) est faite par la carte, sur toutes ses
 * lignes : les clés de mesure sont préfixées par `idPrefix`, unique par ligne.
 */
import type { ReactNode } from 'react'

import { BarWithSides, SideValue, sideSlotPx } from '@/components/charts/BarSides'
import { SegmentLabel } from '@/components/charts/SegmentLabel'
import { Tooltip } from '@/components/ui/tooltip'

import { TipText } from '@/features/squad/emprise/TipText'

import { LIVES_ALONE_INK, LIVES_COLUMNS, LIVES_NEAR_INK } from './livesLayout'
import type { LivesModel } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

/**
 * Vue compacte du tiroir de comparaison de Sessions (maquette `makeLife` avec `cp`) : les parts
 * entières seules dans la barre épaisse et sur la ligne des frags (`killsLine`, `killsLineAlone`).
 */
export interface LivesCompact {
  killsLine: (pct: string, perLife: string) => string
  killsLineAlone: (perLife: string) => string
}

/** Ce que chaque côté de la barre écrit : « compte · part » (à portée) et « part · compte » (isolée), la part seule en compact. */
function livesTexts(model: LivesModel, ut: UsagesCardsText, compact: LivesCompact | undefined): { near: string; alone: string } {
  const nearPct = model.livesNearShare * 100
  return compact
    ? { near: ut.pctIntFmt(nearPct), alone: ut.pctIntFmt(100 - nearPct) }
    : { near: `${ut.intFmt(model.near.lives)} · ${ut.pctFmt(nearPct)}`, alone: `${ut.pctFmt(100 - nearPct)} · ${ut.intFmt(model.alone.lives)}` }
}

/** Les gouttières réservées aux valeurs qui débordent, communes à toutes les lignes d'une carte (et à son axe). */
// eslint-disable-next-line react-refresh/only-export-components
export function livesInset(models: readonly LivesModel[], ut: UsagesCardsText, compact?: LivesCompact): { left: number; right: number } {
  const texts = models.map((m) => livesTexts(m, ut, compact))
  return { left: sideSlotPx(texts.map((x) => x.near)), right: sideSlotPx(texts.map((x) => x.alone)) }
}

interface Props {
  model: LivesModel
  /** Le nom de la ligne (gamertag, avec sa pastille sur la Vue match). */
  label: ReactNode
  /** Les valeurs qui ne tiennent pas dans leur segment (mesure de la carte, clés `${idPrefix}near|alone`). */
  hidden: ReadonlySet<string>
  /** Préfixe des clés de mesure et des identifiants de test ; vide pour la ligne unique. */
  idPrefix?: string
  ut: UsagesCardsText
  compact?: LivesCompact
  /** Les gouttières réservées de chaque côté de la barre (`livesInset`). */
  inset: { left: number; right: number }
}

export function LivesNearTeammateRow({ model, label, hidden, idPrefix = '', ut, compact, inset }: Props) {
  const l = ut.lives
  const nearPct = model.livesNearShare * 100
  const { near: nearText, alone: aloneText } = livesTexts(model, ut, compact)
  const nearId = `${idPrefix}near`
  const aloneId = `${idPrefix}alone`
  // La valeur qui ne tient pas dans son segment s'écrit à côté de la barre, du côté de son camp.
  const nearHidden = hidden.has(nearId)
  const aloneHidden = hidden.has(aloneId)
  return (
    <div className="grid items-center gap-3" style={{ gridTemplateColumns: LIVES_COLUMNS }} data-testid={idPrefix ? `usages-lives-row-${idPrefix}` : undefined}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        {label}
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        <BarWithSides
          leftPx={inset.left}
          rightPx={inset.right}
          left={nearHidden && <SideValue color={LIVES_NEAR_INK} testId={`usages-lives-${idPrefix}side-near`}>{nearText}</SideValue>}
          right={aloneHidden && <SideValue color={LIVES_ALONE_INK} testId={`usages-lives-${idPrefix}side-alone`}>{aloneText}</SideValue>}
        >
        <div className="relative h-[22px] rounded-[3px] bg-muted">
          <Part id={nearId} left={0} width={nearPct} color={LIVES_NEAR_INK} text={nearText} hidden={nearHidden} tip={l.tip(l.near, model.near.lives, model.lives, ut.pctFmt(nearPct), 'lives')} align="start" />
          <Part id={aloneId} left={nearPct} width={100 - nearPct} color={LIVES_ALONE_INK} text={aloneText} hidden={aloneHidden} tip={l.tip(l.alone, model.alone.lives, model.lives, ut.pctFmt(100 - nearPct), 'lives')} align="end" />
        </div>
        </BarWithSides>
        {model.killsNearShare != null && (
          <div style={{ marginLeft: inset.left, marginRight: inset.right }}>
            <KillsLines model={model} idPrefix={idPrefix} ut={ut} compact={compact} />
          </div>
        )}
      </div>
    </div>
  )
}

function KillsLines({ model, idPrefix, ut, compact }: { model: LivesModel; idPrefix: string; ut: UsagesCardsText; compact: LivesCompact | undefined }) {
  const l = ut.lives
  const share = (model.killsNearShare ?? 0) * 100
  const perLife = (v: number | null) => (v == null ? '—' : l.perLifeFmt(v))
  const left = compact ? compact.killsLine(ut.pctIntFmt(share), perLife(model.perLifeNear)) : l.killsLine(model.near.kills, ut.pctFmt(share), perLife(model.perLifeNear))
  const right = compact ? compact.killsLineAlone(perLife(model.perLifeAlone)) : l.killsLineAlone(perLife(model.perLifeAlone), model.alone.kills)
  return (
    <>
      <div className="relative h-2 rounded-[3px] bg-muted">
        {model.near.kills > 0 && (
          <ThinPart id={`usages-lives-${idPrefix}kills-near`} left={0} width={share} color={LIVES_NEAR_INK} tip={l.tip(l.near, model.near.kills, model.kills, ut.pctFmt(share), 'kills')} />
        )}
        {model.alone.kills > 0 && (
          <ThinPart id={`usages-lives-${idPrefix}kills-alone`} left={share} width={100 - share} color={LIVES_ALONE_INK} tip={l.tip(l.alone, model.alone.kills, model.kills, ut.pctFmt(100 - share), 'kills')} />
        )}
      </div>
      <div className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground" data-testid={`usages-lives-${idPrefix}kills-line`}>
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
  hidden: boolean
  tip: string
  align: 'start' | 'end'
}) {
  if (width <= 0) return null
  return (
    <div className="absolute inset-y-0" style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color }} data-fit-key={id} data-testid={`usages-lives-${id}`}>
      <SegmentLabel fill={color} tip={<TipText text={tip} />} align={align} hidden={hidden}>
        {text}
      </SegmentLabel>
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
