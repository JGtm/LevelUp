/**
 * ResourceGridTable — LA TABLE de la grille « Contrôle des ressources » : une colonne par match
 * (Escouade, `ResourceMatchGridCard`) ou par carte (Séries temporelles), une ligne par ressource puis
 * par objet ; sous les armes spéciales, la ligne des frags obtenus avec (feuille de match) ; les armes
 * de râtelier REPLIÉES derrière un bouton. Une case « 5–2 » (nous – eux) est colorée plus / moins que
 * l'adversaire, saturée à trente points d'écart (`bandCellInk`) ; « — » si l'objet n'était pas là ;
 * « sans film » hachuré (la hachure est réservée à ce cas, S4) ; « camp inconnu » ; « non classé »
 * quand les niveaux de socle ne séparent pas armes spéciales et râteliers. Qui chez nous a pris
 * l'objet : dans l'infobulle. L'en-tête de chaque colonne et son en-tête d'infobulle viennent de la
 * carte qui monte la table.
 */
import { Fragment, useState, type ReactNode } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

import { UNMEASURED_HATCH, bandCellInk } from '../formes/colors'
import { RESOURCE_RACK, type GridCell, type GridRow, type GridSection, type GridWho } from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { resourceInk } from './resourceColors'
import { TipText } from './TipText'

/** Une colonne : sa clé, son en-tête affiché, et l'en-tête des infobulles de ses cases. */
export interface GridColumn {
  key: string
  head: ReactNode
  tipHead: string
}

interface Props {
  columns: GridColumn[]
  sections: GridSection[]
  /** Le nom d'un objet (bonus nommé par le web, arme par le titre). */
  itemName: (row: GridRow) => string
  /** « Moi 3, reste du camp 2 » : qui chez nous a pris l'objet. */
  whoText: (who: GridWho[]) => string
  t: EmpriseText
  /**
   * Vue compacte du tiroir de comparaison de Sessions (maquette `renderGridCompact`) : les seules
   * lignes de ressource (synthèse, frags aux armes spéciales ; aucune ligne d'objet, râteliers en une
   * ligne), la part de notre camp dans la case (« 71 % »), « — » hachuré sans film, « ? » sans
   * niveaux ; colonnes sans largeur minimale, la table tient dans sa demi-largeur.
   */
  compact?: boolean
}

type RowRole = 'summary' | 'item' | 'kills'

export function ResourceGridTable({ columns, sections, itemName, whoText, t, compact = false }: Props) {
  const [racksOpen, setRacksOpen] = useState(false)
  const ctx: RowContext = { tipHeads: columns.map((c) => c.tipHead), whoText, t, compact }
  return (
    <div className={compact ? '' : 'overflow-x-auto'}>
      <div
        className={`grid items-stretch gap-1 ${compact ? '' : 'min-w-[640px]'}`}
        style={{
          gridTemplateColumns: compact
            ? `86px repeat(${columns.length}, minmax(0, 1fr))`
            : `170px repeat(${columns.length}, minmax(72px, 1fr))`,
        }}
        data-testid="emprise-grid-table"
      >
        <div />
        {columns.map((c) => (
          <Fragment key={c.key}>{c.head}</Fragment>
        ))}
        {sections.map((s, si) => (
          <Fragment key={s.resource}>
            {si > 0 && <div className="col-span-full mb-0.5 mt-[5px] border-t border-border" />}
            <SectionRows
              section={s}
              itemName={itemName}
              racksOpen={racksOpen}
              onToggleRacks={() => setRacksOpen((o) => !o)}
              ctx={ctx}
            />
          </Fragment>
        ))}
      </div>
    </div>
  )
}

/** Ce que chaque ligne de la grille partage : l'en-tête d'infobulle de chaque colonne, « qui chez nous ». */
interface RowContext {
  tipHeads: string[]
  whoText: (who: GridWho[]) => string
  t: EmpriseText
  compact: boolean
}

/** Une ressource de la grille : synthèse, objets (repliés pour les râteliers), frags obtenus avec. */
function SectionRows({
  section: s,
  itemName,
  racksOpen,
  onToggleRacks,
  ctx,
}: {
  section: GridSection
  itemName: (row: GridRow) => string
  racksOpen: boolean
  onToggleRacks: () => void
  ctx: RowContext
}) {
  const { t, compact } = ctx
  const res = t.resources[s.resource]
  const racks = s.resource === RESOURCE_RACK
  const summaryLabel = racks && !compact ? (
    <div className="flex flex-col justify-center text-[12.5px] leading-tight">
      <button
        type="button"
        className="cursor-pointer rounded-[3px] text-left text-[12.5px] leading-tight text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
        aria-expanded={racksOpen}
        onClick={onToggleRacks}
        data-testid="emprise-grid-racks-toggle"
      >
        {racksOpen ? '▾' : '▸'} <ResourceDot resource={s.resource} />
        {res.label} <small className="text-[11px] text-muted-foreground">{t.grid.racksCount(s.items.length)}</small>
      </button>
    </div>
  ) : (
    <SummaryLabel resource={s.resource} label={res.label} sub={res.gridSub} />
  )
  return (
    <>
      {s.summary && <GridLine label={summaryLabel} row={s.summary} role="summary" name={res.label} absent={res.absent} ctx={ctx} />}
      {!compact &&
        (!racks || racksOpen) &&
        s.items.map((row) => {
          const name = itemName(row)
          return (
            <GridLine
              key={row.object?.key}
              label={<div className="flex flex-col justify-center pl-3.5 text-xs leading-tight">{name}</div>}
              row={row}
              role="item"
              name={name}
              absent={`${name}${res.itemAbsent}`}
              ctx={ctx}
            />
          )
        })}
      {s.kills && (
        <GridLine
          label={<SummaryLabel resource={s.resource} label={res.label} sub={t.grid.killsSub} />}
          row={s.kills}
          role="kills"
          name={t.grid.killsName}
          absent={t.grid.killsAbsent}
          ctx={ctx}
        />
      )}
    </>
  )
}

/** Une ligne de la grille : son libellé, puis une case par colonne. */
function GridLine({
  label,
  row,
  role,
  name,
  absent,
  ctx,
}: {
  label: ReactNode
  row: GridRow
  role: RowRole
  name: string
  absent: string
  ctx: RowContext
}) {
  return (
    <>
      {label}
      {row.cells.map((cell, ci) => (
        <Cell
          key={ci}
          cell={cell}
          role={role}
          tip={cellTip(cell, ctx.tipHeads[ci], name, absent, ctx.whoText, ctx.t)}
          t={ctx.t}
          compact={ctx.compact}
        />
      ))}
    </>
  )
}

function ResourceDot({ resource }: { resource: string }) {
  return (
    <span
      className="mr-1.5 inline-block h-[9px] w-[9px] rounded-[2px] align-[0px]"
      style={{ backgroundColor: resourceInk(resource) }}
      aria-hidden
    />
  )
}

function SummaryLabel({ resource, label, sub }: { resource: string; label: string; sub: string }) {
  return (
    <div className="flex flex-col justify-center text-[12.5px] leading-tight">
      <span>
        <ResourceDot resource={resource} />
        {label}
      </span>
      <small className="text-[11px] text-muted-foreground">{sub}</small>
    </div>
  )
}

function cellTip(
  cell: GridCell,
  head: string,
  name: string,
  absent: string,
  whoText: (who: GridWho[]) => string,
  t: EmpriseText,
): string {
  switch (cell.kind) {
    case 'nofilm':
      return `${head}\n${t.grid.noFilmTip}`
    case 'noteam':
      return `${head}\n${t.grid.noTeamTip}`
    case 'unmeasured':
      return `${head}\n${t.vehicles.unmeasuredTip}`
    case 'untiered':
      return `${head}\n${cell.tiers === 'unestablished' ? t.grid.unestablishedTip : t.grid.untieredTip}`
    case 'none':
      return `${head}\n${absent}`
    case 'value': {
      const lines = [head, t.grid.cellTip(name, cell.us, cell.them, t.pctFmt(cell.share * 100))]
      if (cell.who.length > 0) lines.push(t.grid.whoFmt(whoText(cell.who)))
      if (cell.padsEmptied) lines.push(t.grid.padsFmt(cell.padsEmptied, cell.us + cell.them))
      return lines.join('\n')
    }
  }
}

/** Le texte court d'une case de la vue compacte (maquette `renderGridCompact`). */
const COMPACT_ABSENT = '—'
const COMPACT_UNKNOWN = '?'

function Cell({ cell, role, tip, t, compact }: { cell: GridCell; role: RowRole; tip: string; t: EmpriseText; compact: boolean }) {
  const item = role === 'item'
  const base = `grid h-full w-full cursor-default place-items-center rounded-[3px] tabular-nums ${
    item ? 'min-h-6 text-[11.5px]' : compact ? 'min-h-7 text-[11.5px]' : 'min-h-[34px] text-[12.5px]'
  }`
  let body: ReactNode
  switch (cell.kind) {
    case 'value':
      body = (
        <div className={base} style={{ backgroundColor: bandCellInk((cell.share - 0.5) * 100) }} data-cell="value">
          {compact ? t.pctIntFmt(cell.share * 100) : `${cell.us}–${cell.them}`}
        </div>
      )
      break
    case 'nofilm':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} style={UNMEASURED_HATCH} data-cell="nofilm">
          {compact ? COMPACT_ABSENT : t.grid.noFilmCell}
        </div>
      )
      break
    case 'unmeasured':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} style={UNMEASURED_HATCH} data-cell="unmeasured">
          {compact ? COMPACT_ABSENT : t.vehicles.unmeasuredCell}
        </div>
      )
      break
    case 'noteam':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} data-cell="noteam">
          {compact ? COMPACT_UNKNOWN : t.grid.noTeamCell}
        </div>
      )
      break
    case 'untiered':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} data-cell="untiered">
          {compact ? COMPACT_UNKNOWN : t.grid.untieredCell}
        </div>
      )
      break
    case 'none':
      body = (
        <div
          className={`${base} text-muted-foreground ${item ? 'border border-dashed border-border' : 'bg-muted'}`}
          data-cell="none"
        >
          —
        </div>
      )
      break
  }
  return (
    <Tooltip content={<TipText text={tip} />} className="h-full w-full">
      {body}
    </Tooltip>
  )
}
