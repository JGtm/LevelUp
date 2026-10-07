/**
 * MinePickupsCard — « Part du joueur dans les prises du camp » (Séries temporelles › Usages, bloc
 * « Prises » ; maquette v4, `renderMine`).
 *
 * Par ressource (pastille de sa couleur devant son nom), les objets pris par le camp, triés par
 * volume du camp. Une barre par objet, à l'échelle de l'objet le plus pris ; dedans, la part du
 * joueur (`squad-player-1`, nommé par son gamertag) puis celle du reste du camp, leurs comptes dans
 * les segments quand ils tiennent (mesure au pixel), « JGtm n · camp m » au bout dans tous les cas.
 * Les armes de râtelier sont repliées derrière leur intertitre. Dessous, « Bonus perdus » des deux
 * camps en pastilles d'équipe. Aucune couleur de valeur.
 */
import { Fragment, useMemo, useRef, useState } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import type { SquadEmpriseObject } from '@/lib/api/types'
import { tokenCssVar } from '@/lib/accessibility'

import { repliOffsetPct } from '@/features/squad/charts/squadFragBreakdownChart'
import { SQUAD_MAIN_PLAYER_INK, TEAM_REST_INK } from '@/features/squad/formes/colors'
import type { EmpriseText } from '@/features/squad/emprise/empriseStrings'
import { resourceInk } from '@/features/squad/emprise/resourceColors'
import { TipText } from '@/features/squad/emprise/TipText'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'

import { mineByResource, type MineGroup, type MinePickups, type MineResource, type MineRow } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

const ME_INK = SQUAD_MAIN_PLAYER_INK
/** La barre d'un objet très peu pris garde une largeur lisible (maquette : 2 %). */
const MIN_TRACK_PCT = 2

/**
 * Vue compacte du tiroir de comparaison de Sessions (maquette `renderMineCompact`) : une barre par
 * RESSOURCE, la part du joueur et celle du reste de l’équipe en pourcentage (comptes au survol), une part qui
 * ne tient pas dans son segment sur la ligne de repli au-dessus, bonus perdus en pourcentage ;
 * `resourceSub` est le sous-libellé de chaque ressource.
 */
interface MineCompact {
  resourceSub: string
}

interface Props {
  mine: MinePickups
  itemName: (o: SquadEmpriseObject) => string
  /** Le gamertag du joueur de la page. */
  player: string
  t: EmpriseText
  ut: UsagesCardsText
  compact?: MineCompact
}

export function MinePickupsCard({ mine, itemName, player, t, ut, compact }: Props) {
  const ref = useRef<HTMLDivElement | null>(null)
  const [racksOpen, setRacksOpen] = useState(false)
  const hidden = useSegmentLabelFit(ref, [mine, racksOpen, compact])
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={ut.mine.title}
        items={[
          { kind: 'square', label: player, color: ME_INK },
          { kind: 'square', label: ut.mine.rest, color: TEAM_REST_INK },
        ]}
      />
    ),
    [ut, player],
  )
  return (
    <ObjectifFrame title={ut.mine.title} info={ut.mine.info} legend={legend} testId="usages-mine">
      <div ref={ref} className={`flex flex-col ${compact ? 'gap-2.5' : 'gap-[7px]'}`}>
        {compact
          ? mineByResource(mine).map((r) => (
              <MineResourceLine key={r.resource} row={r} label={t.resources[r.resource]?.label ?? r.resource} sub={compact.resourceSub} hidden={hidden} player={player} ut={ut} />
            ))
          : mine.groups.map((g) => (
              <Fragment key={g.resource}>
                <GroupHead group={g} label={t.resources[g.resource]?.label ?? g.resource} open={racksOpen} onToggle={() => setRacksOpen((o) => !o)} ut={ut} />
                {(!g.folded || racksOpen) &&
                  g.rows.map((r) => <MineLine key={r.object.key} row={r} name={itemName(r.object)} max={mine.max} hidden={hidden} player={player} ut={ut} />)}
              </Fragment>
            ))}
      </div>
      {mine.losses && mine.losses.us.taken + mine.losses.them.taken > 0 && (
        <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted-foreground" data-testid="usages-mine-losses">
          <span className="font-semibold text-foreground">{ut.mine.lossesTitle}</span>
          <Loss color={tokenCssVar('team-ally')} lost={mine.losses.us.lost} taken={mine.losses.us.taken} ut={ut} pctOnly={!!compact} />
          <Loss color={tokenCssVar('team-enemy')} lost={mine.losses.them.lost} taken={mine.losses.them.taken} ut={ut} pctOnly={!!compact} />
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

function MineLine({
  row,
  name,
  max,
  hidden,
  player,
  ut,
}: {
  row: MineRow
  name: string
  max: number
  hidden: ReadonlySet<string>
  player: string
  ut: UsagesCardsText
}) {
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
            <Segment id={`usages-mine-me-${key}`} left={0} width={mePct} color={ME_INK} rounded={row.rest > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.meTip(player, name, row.me, row.camp)} value={row.me} light />
          )}
          {row.rest > 0 && (
            <Segment id={`usages-mine-rest-${key}`} left={mePct} width={100 - mePct} color={TEAM_REST_INK} rounded={row.me > 0 ? 'rounded-r-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.restTip(name, row.rest, row.camp)} value={row.rest} />
          )}
        </div>
      </div>
      <div className="whitespace-nowrap text-[11.5px] tabular-nums text-muted-foreground" data-testid={`usages-mine-value-${key}`}>
        {player} <b className="font-semibold text-foreground">{row.me}</b> · {ut.mine.campWord} <b className="font-semibold text-foreground">{row.camp}</b>
      </div>
    </div>
  )
}

/** Vue compacte : une ressource, la part du joueur et celle du reste de l’équipe en pourcentage, pleine largeur. */
function MineResourceLine({ row, label, sub, hidden, player, ut }: { row: MineResource; label: string; sub: string; hidden: ReadonlySet<string>; player: string; ut: UsagesCardsText }) {
  const key = row.resource
  const rest = row.camp - row.me
  const mePct = row.camp > 0 ? (row.me / row.camp) * 100 : 0
  const meLabel = ut.pctIntFmt(mePct)
  // Les deux parts somment à 100 : la seconde se déduit de la première ARRONDIE (maquette).
  const restLabel = ut.pctIntFmt(100 - Math.round(mePct))
  return (
    <div className="grid items-center gap-3 text-xs" style={{ gridTemplateColumns: '118px minmax(0,1fr)' }} data-testid={`usages-mine-resource-${key}`}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        <span className="mr-1.5 inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: resourceInk(key) }} aria-hidden />
        {label}
        <small className="block text-[11px] text-muted-foreground">{sub}</small>
      </div>
      <div className="min-w-0">
        <ShareRepli
          id={`usages-mine-repli-${key}`}
          parts={[
            { segId: `usages-mine-me-${key}`, n: row.me, leftPct: 0, widthPct: mePct, color: ME_INK, text: meLabel },
            { segId: `usages-mine-rest-${key}`, n: rest, leftPct: mePct, widthPct: 100 - mePct, color: TEAM_REST_INK, text: restLabel },
          ]}
          hidden={hidden}
        />
        <div className="relative h-4 rounded-[3px] bg-muted">
          {row.me > 0 && (
            <Segment id={`usages-mine-me-${key}`} left={0} width={mePct} color={ME_INK} rounded={rest > 0 ? 'rounded-l-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.meTip(player, label, row.me, row.camp)} value={meLabel} light />
          )}
          {rest > 0 && (
            <Segment id={`usages-mine-rest-${key}`} left={mePct} width={100 - mePct} color={TEAM_REST_INK} rounded={row.me > 0 ? 'rounded-r-[3px]' : 'rounded-[3px]'} hidden={hidden} tip={ut.mine.restTip(label, rest, row.camp)} value={restLabel} />
          )}
        </div>
      </div>
    </div>
  )
}

/** Une part de la barre compacte : son segment, son compte, sa position, son encre, sa part écrite. */
interface SharePart {
  segId: string
  n: number
  leftPct: number
  widthPct: number
  color: string
  text: string
}

/**
 * La ligne de repli au-dessus de la barre compacte : les parts qui ne tiennent pas dans leur segment,
 * alignées sur le début du premier d'entre eux (`repliOffsetPct`, patron de la Répartition des
 * frags) ; absente quand tout tient. La vue compacte n'a pas de colonne de droite : sans elle, une
 * part étroite ne serait lisible qu'au survol.
 */
function ShareRepli({ id, parts, hidden }: { id: string; parts: SharePart[]; hidden: ReadonlySet<string> }) {
  const present = parts.filter((p) => p.n > 0)
  const isHidden = (segId: string) => hidden.has(segId)
  const offset = repliOffsetPct(
    present.map((p) => ({ cls: p.segId, kills: p.n, leftPct: p.leftPct, widthPct: p.widthPct })),
    isHidden,
  )
  if (offset == null) return null
  return (
    <div className="mb-0.5 flex gap-2 whitespace-nowrap text-xs tabular-nums text-muted-foreground" style={{ paddingLeft: `${offset}%` }} data-testid={id}>
      {present
        .filter((p) => isHidden(p.segId))
        .map((p) => (
          <span key={p.segId} className="inline-flex items-center">
            <span className="mr-[5px] inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: p.color }} aria-hidden />
            <b className="font-bold text-foreground">{p.text}</b>
          </span>
        ))}
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
  /** Le compte (pleine page) ou la part écrite (vue compacte). */
  value: number | string
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

/** Les bonus perdus d'un camp : « n sur m (p %) », ou la part seule en vue compacte (compte au survol). */
function Loss({ color, lost, taken, ut, pctOnly }: { color: string; lost: number; taken: number; ut: UsagesCardsText; pctOnly: boolean }) {
  const dot = <span className="inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: color }} aria-hidden />
  if (pctOnly) {
    return (
      <Tooltip content={<TipText text={ut.mine.lossesFmt(lost, taken)} />}>
        <span className="inline-flex cursor-help items-center gap-1.5 tabular-nums">
          {dot}
          <b className="font-semibold text-foreground">{taken > 0 ? ut.pctIntFmt((lost / taken) * 100) : '—'}</b>
        </span>
      </Tooltip>
    )
  }
  return (
    <span className="inline-flex items-center gap-1.5 tabular-nums">
      {dot}
      <span>
        <b className="font-semibold text-foreground">{ut.mine.lossesFmt(lost, taken)}</b>
        {taken > 0 && ` (${ut.pctIntFmt((lost / taken) * 100)})`}
      </span>
    </span>
  )
}
