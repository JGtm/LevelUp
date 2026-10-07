/**
 * LivesNearTeammateRow — LA LIGNE D'UN JOUEUR de la carte « Isolement » : la barre ÉPAISSE partage ses
 * vies terminées par une mort entre « à portée d'un coéquipier » (`squad-player-1`) et « isolée »
 * (`extreme`), compte et part dans chaque segment quand ils tiennent, repli au-dessus sinon ; la barre
 * FINE partage ses frags tombés pendant ces vies ; dessous, « frags : n · p % · x par vie … ». Une
 * ligne pour les Séries temporelles et Sessions (le joueur de la page), une par joueur de l'équipe sur
 * la Vue match. La mesure des valeurs qui tiennent (`hidden`) est faite par la carte, sur toutes ses
 * lignes : les clés de mesure sont préfixées par `idPrefix`, unique par ligne.
 */
import type { ReactNode } from 'react'

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
}

export function LivesNearTeammateRow({ model, label, hidden, idPrefix = '', ut, compact }: Props) {
  const l = ut.lives
  const nearPct = model.livesNearShare * 100
  const nearText = compact ? ut.pctIntFmt(nearPct) : `${ut.intFmt(model.near.lives)} · ${ut.pctFmt(nearPct)}`
  const aloneText = compact ? ut.pctIntFmt(100 - nearPct) : `${ut.pctFmt(100 - nearPct)} · ${ut.intFmt(model.alone.lives)}`
  const nearId = `${idPrefix}near`
  const aloneId = `${idPrefix}alone`
  // Seule la valeur qui ne tient pas dans son segment monte au repli (jamais affichée deux fois).
  const nearHidden = hidden.has(nearId)
  const aloneHidden = hidden.has(aloneId)
  return (
    <div className="grid items-center gap-3" style={{ gridTemplateColumns: LIVES_COLUMNS }} data-testid={idPrefix ? `usages-lives-row-${idPrefix}` : undefined}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        {label}
        <small className="block text-[11px] text-muted-foreground" data-testid={`usages-lives-${idPrefix}sub`}>
          {l.rowSub(model.lives)}
        </small>
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        {(nearHidden || aloneHidden) && (
          <div className="flex justify-between gap-2 text-xs tabular-nums text-muted-foreground" data-testid={`usages-lives-${idPrefix}repli`}>
            <span>{nearHidden && nearText}</span>
            <span>{aloneHidden && aloneText}</span>
          </div>
        )}
        <div className="relative h-[22px] rounded-[3px] bg-muted">
          <Part id={nearId} left={0} width={nearPct} color={LIVES_NEAR_INK} text={nearText} hidden={nearHidden} tip={l.tip(l.near, model.near.lives, model.lives, ut.pctFmt(nearPct), 'lives')} align="start" />
          <Part id={aloneId} left={nearPct} width={100 - nearPct} color={LIVES_ALONE_INK} text={aloneText} hidden={aloneHidden} tip={l.tip(l.alone, model.alone.lives, model.lives, ut.pctFmt(100 - nearPct), 'lives')} align="end" />
        </div>
        {model.killsNearShare != null && <KillsLines model={model} idPrefix={idPrefix} ut={ut} compact={compact} />}
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
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div className={`flex h-full w-full cursor-help items-center overflow-hidden ${align === 'start' ? 'justify-start pl-[7px]' : 'justify-end pr-[7px]'}`}>
          {/* `text-white` : écriture posée sur l'aplat, contraste dans le segment et non couleur sémantique. */}
          <span data-fit-label className="whitespace-nowrap text-xs font-medium tabular-nums leading-none text-white" style={{ visibility: hidden ? 'hidden' : 'visible' }}>
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
