/**
 * ResourceMatchGridCard — « Contrôle des ressources, match par match » (Escouade › Emprise, bloc
 * « Carte par carte » ; lot L5.2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de
 * l'onglet, `renderBand`).
 *
 * Colonnes = les matchs de la soirée : heure, carte, mode, pastille « Victoire 3–0 » (S9) et
 * badge du drapeau de dominance quand il existe. Lignes = une ressource (synthèse, pastille de
 * sa couleur, S8) puis chaque objet ; sous les armes spéciales, la ligne des frags obtenus avec
 * (feuille de match) ; les armes de râtelier REPLIÉES derrière un bouton. Une case « 5–2 » (nous
 * – eux) est colorée plus / moins que l'adversaire, saturée à trente points d'écart
 * (`bandCellInk`) ; « — » si l'objet n'était pas sur la carte ; « sans film » hachuré (la
 * hachure est réservée à ce cas, S4) ; « non classé » quand les niveaux de socle du match ne
 * séparent pas armes spéciales et râteliers. Qui chez nous a pris l'objet : dans l'infobulle.
 */
import { Fragment, useMemo, useState, type ReactNode } from 'react'

import { Tooltip } from '@/components/ui/tooltip'
import type { DominanceValue } from '@/components/charts/outcomeSequence'
import { tokenCssVar } from '@/lib/accessibility'
import type { Locale } from '@/lib/i18n/locale'
import { DOMINANCE_COLOR_TOKENS } from '@/lib/narrative/dominance'

import { MINUS_INK, PLUS_INK, TRACK_INK, UNMEASURED_HATCH, bandCellInk } from '../formes/colors'
import { formatMatchTime } from '../formes/format'
import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import {
  RESOURCE_RACK,
  type EmpriseMatchInfo,
  type GridCell,
  type GridRow,
  type GridSection,
  type GridWho,
  type MatchGrid,
} from './emprise.logic'
import type { EmpriseText } from './empriseStrings'
import { resourceInk } from './resourceColors'
import { TipText } from './TipText'

const OUTCOME_TOKENS = {
  win: 'outcome-win',
  loss: 'outcome-loss',
  tie: 'outcome-draw',
  dnf: 'outcome-dnf',
} as const

interface Props {
  grid: MatchGrid
  /** Le nom d'un objet (bonus nommé par le web, arme par le titre). */
  itemName: (row: GridRow) => string
  /** Le nom d'un joueur de l'escouade par xuid (vide = inconnu). */
  playerName: (xuid: string) => string
  dominanceLabels: Record<DominanceValue, string>
  locale: Locale
  t: EmpriseText
}

type RowRole = 'summary' | 'item' | 'kills'

export function ResourceMatchGridCard({ grid, itemName, playerName, dominanceLabels, locale, t }: Props) {
  const [racksOpen, setRacksOpen] = useState(false)
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.grid.title}
        items={[
          { kind: 'square', label: t.grid.more, color: `color-mix(in oklab, ${PLUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.less, color: `color-mix(in oklab, ${MINUS_INK} 80%, var(--muted))` },
          { kind: 'square', label: t.grid.nothing, color: TRACK_INK },
          { kind: 'hatch', label: t.grid.noFilm },
        ]}
      />
    ),
    [t],
  )
  const ctx: RowContext = {
    head: (m) =>
      t.grid.matchHead(
        formatMatchTime(m.startTime, locale),
        m.map,
        m.mode,
        m.outcome ? `${t.outcomeLower[m.outcome]}${m.score ? ` ${m.score}` : ''}` : null,
      ),
    whoText: (who) =>
      who
        .map((w) => ({ name: w.xuid ? playerName(w.xuid) || t.grid.restLower : t.grid.restLower, n: w.taken }))
        .sort((a, b) => b.n - a.n || a.name.localeCompare(b.name))
        .map((w) => `${w.name} ${w.n}`)
        .join(', '),
    columns: grid.columns,
    t,
  }
  return (
    <ObjectifFrame title={t.grid.title} info={t.grid.info} legend={legend} testId="emprise-grid">
      <div className="overflow-x-auto">
        <div
          className="grid min-w-[640px] items-stretch gap-1"
          style={{ gridTemplateColumns: `170px repeat(${grid.columns.length}, minmax(72px, 1fr))` }}
          data-testid="emprise-grid-table"
        >
          <div />
          {grid.columns.map((m) => (
            <MatchHead key={m.matchId} m={m} dominanceLabels={dominanceLabels} locale={locale} t={t} />
          ))}
          {grid.sections.map((s, si) => (
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
    </ObjectifFrame>
  )
}

/** Ce que chaque ligne de la grille partage : l'en-tête d'infobulle d'un match, « qui chez nous ». */
interface RowContext {
  head: (m: EmpriseMatchInfo) => string
  whoText: (who: GridWho[]) => string
  columns: EmpriseMatchInfo[]
  t: EmpriseText
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
  const { t } = ctx
  const res = t.resources[s.resource]
  const racks = s.resource === RESOURCE_RACK
  const summaryLabel = racks ? (
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
      {(!racks || racksOpen) &&
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

/** Une ligne de la grille : son libellé, puis une case par match. */
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
          tip={cellTip(cell, ctx.head(ctx.columns[ci]), name, absent, ctx.whoText, ctx.t)}
          t={ctx.t}
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

function MatchHead({
  m,
  dominanceLabels,
  locale,
  t,
}: {
  m: EmpriseMatchInfo
  dominanceLabels: Record<DominanceValue, string>
  locale: Locale
  t: EmpriseText
}) {
  const result = m.outcome ? `${t.outcome[m.outcome]}${m.score ? ` ${m.score}` : ''}` : null
  const dom = m.dominance ? dominanceLabels[m.dominance] : null
  const domInk = m.dominance ? tokenCssVar(DOMINANCE_COLOR_TOKENS[m.dominance]) : ''
  return (
    <div className="pb-[3px] text-center text-[11px] leading-tight text-muted-foreground" data-testid={`emprise-grid-head-${m.matchId}`}>
      {formatMatchTime(m.startTime, locale)}
      <b className="block text-xs font-medium text-foreground">{m.map}</b>
      {m.mode}
      {result && m.outcome && (
        <div>
          <span
            className="mt-[3px] inline-block rounded-full px-[7px] text-[10.5px] font-bold text-white"
            style={{ backgroundColor: tokenCssVar(OUTCOME_TOKENS[m.outcome]) }}
            data-testid={`emprise-grid-result-${m.matchId}`}
          >
            {result}
          </span>
        </div>
      )}
      {dom && (
        <div>
          <Tooltip content={<TipText text={t.grid.dominanceTip(dom)} />}>
            <span
              className="mt-[3px] inline-block rounded-full px-[7px] text-[10.5px] font-semibold text-foreground"
              style={{
                backgroundColor: `color-mix(in oklab, ${domInk} 22%, var(--card))`,
                boxShadow: `inset 0 0 0 1.5px ${domInk}`,
              }}
              data-testid={`emprise-grid-dominance-${m.matchId}`}
            >
              {dom}
            </span>
          </Tooltip>
        </div>
      )}
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

function Cell({ cell, role, tip, t }: { cell: GridCell; role: RowRole; tip: string; t: EmpriseText }) {
  const item = role === 'item'
  const base = `grid h-full w-full cursor-default place-items-center rounded-[3px] tabular-nums ${
    item ? 'min-h-6 text-[11.5px]' : 'min-h-[34px] text-[12.5px]'
  }`
  let body: ReactNode
  switch (cell.kind) {
    case 'value':
      body = (
        <div className={base} style={{ backgroundColor: bandCellInk((cell.share - 0.5) * 100) }} data-cell="value">
          {cell.us}–{cell.them}
        </div>
      )
      break
    case 'nofilm':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} style={UNMEASURED_HATCH} data-cell="nofilm">
          {t.grid.noFilmCell}
        </div>
      )
      break
    case 'untiered':
      body = (
        <div className={`${base} bg-muted !text-[11px] text-muted-foreground`} data-cell="untiered">
          {t.grid.untieredCell}
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
