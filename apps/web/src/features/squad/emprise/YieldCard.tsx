/**
 * YieldCard — « Rendement face à l'adversaire » (Escouade › Emprise, bloc « Prendre, et s'en
 * servir », à droite de « Frags obtenus avec les ressources », même hauteur ; lot L5.2 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26, maquette de l'onglet, `renderRendement`).
 *
 * Par ressource (pastille devant le nom, S8) : l'écart relatif `production[].relative_gap`
 * (notre rendement / le sien − 1, calcul Go) sur un axe commun −50 / +50 %, en barre divergente
 * depuis le zéro — `divergent-pos` au-dessus, `divergent-neg` en dessous, à 80 % sur le fond
 * atténué (maquette) —, la valeur signée AU BOUT de la barre, les rendements bruts (« 3,0
 * contre 2,7 ») de l'autre côté du zéro. Un écart au-delà de ±50 % remplit la demi-piste ; sa
 * valeur s'écrit alors dans le bout de la barre (au-dehors, elle sortirait de la carte).
 * Légende en pied de carte, centrée (S2).
 */
import { useMemo } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

import { MINUS_INK, PLUS_INK } from '../formes/colors'
import { ObjectifFrame, ObjectifLegend } from '../objectif/ObjectifFrame'
import type { EmpriseText } from './empriseStrings'
import { TrackAxis } from './PisteCampsForm'
import { pisteColumns } from './pisteLayout'
import { yieldGeometry, type YieldRow } from './production.logic'
import { resourceInk } from './resourceColors'
import { TipText } from './TipText'
import type { VehicleCoverage } from './vehicles.logic'

const MORE_INK = `color-mix(in oklab, ${PLUS_INK} 80%, var(--muted))`
const LESS_INK = `color-mix(in oklab, ${MINUS_INK} 80%, var(--muted))`
/** Colonne des noms (maquette : `.prow.wide.narrow`, 118 px). */
const COLUMNS = pisteColumns()
/** Graduations de l'axe, en pourcentage de la piste (−50, −25, 0, +25, +50 %). */
const TICK_AT = [0, 25, 50, 75, 100] as const
/** Au-delà de ±40 % d'écart, la valeur s'écrit dans le bout de la barre. */
const INSIDE_FROM_PCT = 40

export function YieldCard({ rows, coverage, t }: { rows: YieldRow[]; coverage?: VehicleCoverage | null; t: EmpriseText }) {
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={t.yield.title}
        items={[
          { kind: 'square', label: t.yield.more, color: MORE_INK },
          { kind: 'square', label: t.yield.less, color: LESS_INK },
        ]}
      />
    ),
    [t],
  )
  return (
    <ObjectifFrame title={t.yield.title} info={t.yield.info} legend={legend} testId="emprise-yield">
      <div className="mt-2 flex flex-col gap-4" aria-label={t.yield.ariaLabel} role="group">
        {rows.map((r) => (
          <YieldLine key={r.resource} row={r} t={t} />
        ))}
        <TrackAxis columns={COLUMNS} ticks={TICK_AT.map((at, i) => ({ at, label: t.yield.axis[i] }))} />
        {coverage && <VehicleNote coverage={coverage} t={t} />}
      </div>
    </ObjectifFrame>
  )
}

function YieldLine({ row, t }: { row: YieldRow; t: EmpriseText }) {
  const res = t.resources[row.resource]
  const { left, width, x } = yieldGeometry(row.gap)
  const up = row.gap >= 0
  const gap = t.yield.gapFmt(row.gap)
  const raw = t.yield.rawFmt(row.us, row.them)
  const inside = Math.abs(row.gap * 100) > INSIDE_FROM_PCT
  // La valeur au bout : dehors, 6 px après le bout ; dedans (écart extrême), 6 px avant.
  const tipStyle = up
    ? inside
      ? { right: `calc(${100 - x}% + 6px)` }
      : { left: `calc(${x}% + 6px)` }
    : inside
      ? { left: `calc(${x}% + 6px)` }
      : { right: `calc(${100 - x}% + 6px)` }
  return (
    <div className="grid items-center gap-3" style={{ gridTemplateColumns: COLUMNS }} data-testid={`emprise-yield-row-${row.resource}`}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        <span className="inline-flex items-center">
          <span className="mr-1.5 inline-block h-[9px] w-[9px] shrink-0 rounded-[2px]" style={{ backgroundColor: resourceInk(row.resource) }} aria-hidden />
          {res.label}
        </span>
        <small className="block text-[11px] text-muted-foreground">{res.yieldSub}</small>
      </div>
      <div className="relative h-[22px] rounded-[3px] bg-muted" role="img" aria-label={`${res.label} : ${gap} (${raw})`}>
        <div className="absolute inset-y-0" style={{ left: `${left}%`, width: `${width}%`, backgroundColor: up ? MORE_INK : LESS_INK }}>
          <Tooltip content={<TipText text={t.yield.tip(res.label, res.yieldSub, row.us, row.them, gap)} />} className="h-full w-full">
            <div className="h-full w-full cursor-help" data-testid={`emprise-yield-bar-${row.resource}`} />
          </Tooltip>
        </div>
        {/* Le zéro : « autant que l'adversaire » (maquette : trait plein 2 px, 3 px au-delà de la piste). */}
        <div className="pointer-events-none absolute -bottom-[3px] -top-[3px] left-1/2 w-0 -translate-x-px border-l-2 border-muted-foreground" aria-hidden />
        <span
          className="pointer-events-none absolute top-1/2 -translate-y-1/2 whitespace-nowrap text-[13px] font-bold tabular-nums text-foreground"
          style={tipStyle}
          data-testid={`emprise-yield-gap-${row.resource}`}
        >
          {gap}
        </span>
        <span
          className="pointer-events-none absolute top-1/2 -translate-y-1/2 whitespace-nowrap text-[11px] tabular-nums text-muted-foreground"
          style={up ? { right: 'calc(50% + 8px)' } : { left: 'calc(50% + 8px)' }}
          data-testid={`emprise-yield-raw-${row.resource}`}
        >
          {raw}
        </span>
      </div>
    </div>
  )
}

/** D9 : la part des frags de véhicule appariés à un passage daté de leur tueur, et les passages de robots ignorés (D10). */
function VehicleNote({ coverage, t }: { coverage: VehicleCoverage; t: EmpriseText }) {
  const v = t.vehicles
  const note = v.pairedNote(coverage.fragsPaired, coverage.fragsTotal, t.pctFmt(coverage.pairedShare * 100))
  return (
    <p className="text-[11px] leading-snug text-muted-foreground" data-testid="emprise-yield-vehicle-note">
      {note}
      {coverage.episodesUnnamed > 0 ? ` ${v.unnamedNote(coverage.episodesUnnamed)}` : ''}
    </p>
  )
}
