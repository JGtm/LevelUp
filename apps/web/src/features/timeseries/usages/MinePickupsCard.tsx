/**
 * MinePickupsCard — « Mes prises dans mon camp » (Séries temporelles › Usages, bloc « Mes prises » ;
 * maquette v4, `renderMine`).
 *
 * Par ressource (pastille de sa couleur devant son nom), les objets pris par mon camp, triés par
 * volume de mon camp. Une barre par objet, à l'échelle de l'objet le plus pris ; dedans, ma part
 * (`squad-player-1`) puis celle du reste de mon camp, leurs comptes dans les segments quand ils
 * tiennent (mesure au pixel), « moi n · camp m » au bout dans tous les cas. Les armes de râtelier
 * sont repliées derrière leur intertitre. Dessous, « Bonus perdus » des deux camps en pastilles
 * d'équipe. Une répartition, pas un classement : aucune couleur de valeur.
 */
import { Fragment, useMemo, useRef, useState } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import type { SquadEmpriseObject } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'

import { TEAM_REST_INK, squadPlayerInk } from '@/features/squad/formes/colors'
import type { EmpriseText } from '@/features/squad/emprise/empriseStrings'
import { resourceInk } from '@/features/squad/emprise/resourceColors'
import { TipText } from '@/features/squad/emprise/TipText'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'

import type { MineGroup, MinePickups, MineRow } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

const ME_INK = squadPlayerInk(0)
/** La barre d'un objet très peu pris garde une largeur lisible (maquette : 2 %). */
const MIN_TRACK_PCT = 2

interface Props {
  mine: MinePickups
  itemName: (o: SquadEmpriseObject) => string
  t: EmpriseText
  ut: UsagesCardsText
}

export function MinePickupsCard({ mine, itemName, t, ut }: Props) {
  const ref = useRef<HTMLDivElement | null>(null)
  const [racksOpen, setRacksOpen] = useState(false)
  const hidden = useSegmentLabelFit(ref, [mine, racksOpen])
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={ut.mine.title}
        items={[
          { kind: 'square', label: ut.mine.me, color: ME_INK },
          { kind: 'square', label: ut.mine.rest, color: TEAM_REST_INK },
        ]}
      />
    ),
    [ut],
  )
  return (
    <ObjectifFrame title={ut.mine.title} info={ut.mine.info} legend={legend} testId="usages-mine">
      <div ref={ref} className="flex flex-col gap-[7px]">
        {mine.groups.map((g) => (
          <Fragment key={g.resource}>
            <GroupHead group={g} label={t.resources[g.resource]?.label ?? g.resource} open={racksOpen} onToggle={() => setRacksOpen((o) => !o)} ut={ut} />
            {(!g.folded || racksOpen) &&
              g.rows.map((r) => <MineLine key={r.object.key} row={r} name={itemName(r.object)} max={mine.max} hidden={hidden} ut={ut} />)}
          </Fragment>
        ))}
      </div>
      {mine.losses && mine.losses.us.taken + mine.losses.them.taken > 0 && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground" data-testid="usages-mine-losses">
          <span className="font-semibold text-foreground">{ut.mine.lossesTitle}</span>
          <Loss color={tokenCssVar('team-ally')} lost={mine.losses.us.lost} taken={mine.losses.us.taken} ut={ut} />
          <Loss color={tokenCssVar('team-enemy')} lost={mine.losses.them.lost} taken={mine.losses.them.taken} ut={ut} />
        </div>
      )}
    </ObjectifFrame>
  )
}

function GroupHead({ group, label, open, onToggle, ut }: { group: MineGroup; label: string; open: boolean; onToggle: () => void; ut: UsagesCardsText }) {
  const dot = (
    <span className="mr-1.5 inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: resourceInk(group.resource) }} aria-hidden />
  )
  if (!group.folded) {
    return (
      <div className="mb-0.5 mt-2 text-[12.5px] font-medium first:mt-0">
        {dot}
        {label}
      </div>
    )
  }
  return (
    <button
      type="button"
      className="mb-0.5 mt-2 cursor-pointer rounded-[3px] text-left text-[12.5px] font-medium text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
      aria-expanded={open}
      onClick={onToggle}
      data-testid="usages-mine-racks-toggle"
    >
      {open ? '▾' : '▸'} {dot}
      {label} <small className="text-[11px] font-normal text-muted-foreground">{ut.mine.foldedFmt(group.rows.length)}</small>
    </button>
  )
}

function MineLine({ row, name, max, hidden, ut }: { row: MineRow; name: string; max: number; hidden: ReadonlySet<string>; ut: UsagesCardsText }) {
  const key = row.object.key
  const mePct = (row.me / row.camp) * 100
  return (
    <div className="grid items-center gap-3 text-xs" style={{ gridTemplateColumns: '150px minmax(0,1fr) auto' }} data-testid={`usages-mine-row-${key}`}>
      <div className="truncate" title={name}>
        {name}
      </div>
      <div className="min-w-0">
        <div
          className="relative h-4 rounded-[3px] bg-muted"
          style={{ width: `${Math.max(MIN_TRACK_PCT, (row.camp / max) * 100)}%` }}
          data-testid={`usages-mine-track-${key}`}
        >
          {row.me > 0 && (
            <Segment id={`usages-mine-me-${key}`} left={0} width={mePct} color={ME_INK} rounded={row.rest > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.meTip(name, row.me, row.camp)} value={row.me} light />
          )}
          {row.rest > 0 && (
            <Segment id={`usages-mine-rest-${key}`} left={mePct} width={100 - mePct} color={TEAM_REST_INK} rounded={row.me > 0 ? 'rounded-r-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.restTip(name, row.rest, row.camp)} value={row.rest} />
          )}
        </div>
      </div>
      <div className="whitespace-nowrap text-[11.5px] tabular-nums text-muted-foreground" data-testid={`usages-mine-value-${key}`}>
        {ut.mine.meWord} <b className="font-semibold text-foreground">{row.me}</b> · {ut.mine.campWord} <b className="font-semibold text-foreground">{row.camp}</b>
      </div>
    </div>
  )
}

function Segment({
  id,
  left,
  width,
  color,
  rounded,
  hidden,
  tip,
  value,
  light,
}: {
  id: string
  left: number
  width: number
  color: string
  rounded: string
  hidden: ReadonlySet<string>
  tip: string
  value: number
  /** Écriture claire sur l'aplat du joueur ; foncée sur celui, pâle, du reste du camp. */
  light?: boolean
}) {
  return (
    <div className={`absolute inset-y-0 ${rounded}`} style={{ left: `${left}%`, width: `${width}%`, backgroundColor: color }} data-fit-key={id} data-testid={id}>
      <Tooltip content={<TipText text={tip} />} className="h-full w-full">
        <div className="flex h-full w-full cursor-help items-center justify-center overflow-hidden">
          {/* `text-white` sur l'aplat du joueur : contraste dans le segment, pas une couleur sémantique. */}
          <span
            data-fit-label
            className={`whitespace-nowrap text-[11px] font-semibold tabular-nums leading-none ${light ? 'text-white' : 'text-foreground'}`}
            style={{ visibility: hidden.has(id) ? 'hidden' : 'visible' }}
          >
            {value}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}

function Loss({ color, lost, taken, ut }: { color: string; lost: number; taken: number; ut: UsagesCardsText }) {
  return (
    <span className="inline-flex items-center gap-1.5 tabular-nums">
      <span className="inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
      <span>
        <b className="font-semibold text-foreground">{ut.mine.lossesFmt(lost, taken)}</b>
        {taken > 0 && ` (${ut.pctIntFmt((lost / taken) * 100)})`}
      </span>
    </span>
  )
}
