/**
 * BandeForm.tsx — LA BANDE DE RÉGULARITÉ (artefact 2ec1b8eb, forme `bande`).
 *
 * UNE CASE PAR MATCH, dans l'ordre de la période, teintée par l'ÉCART À LA
 * PARITÉ — pas par la valeur. Un écart systématique se voit à la couleur d'une
 * ligne entière ; un accident se voit à une case isolée.
 *
 * L'INTENSITÉ SATURE À TRENTE POINTS : au-delà, deux écarts très différents se
 * ressemblent, et c'est assumé — la bande dit la régularité, pas l'amplitude.
 *
 * « — » EST UN NON MESURÉ, JAMAIS UN ZÉRO : un match sans film décodé et un
 * match où personne n'a touché l'axe portent la même case hachurée, et
 * l'infobulle dit laquelle des deux causes.
 *
 * L'axe du bas nomme chaque match par son heure et sa carte : sans lui, neuf
 * cases ne sont que neuf cases.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { UNMEASURED_HATCH, bandCellInk } from '../colors'

/** Une case : sa valeur (part), ou `null` pour un non mesuré. */
export interface BandeCell {
  key: string
  pct: number | null
  /** L'infobulle complète, déjà rédigée par l'appelant. */
  tooltip: string
}

export interface BandeRow {
  key: string
  label: string
  cells: BandeCell[]
}

export interface BandeFormProps {
  rows: BandeRow[]
  /** Les colonnes : une par match, avec son heure et sa carte. */
  columns: { key: string; time: string; map: string }[]
  /** La parité à laquelle chaque case se compare (50 % entre deux camps). */
  parity: number
  axisTitle: string
}

const COLUMNS = '196px 1fr'

export function BandeForm({ rows, columns, parity, axisTitle }: BandeFormProps) {
  return (
    <div className="min-w-[520px]">
      {rows.map((row) => (
        <div
          key={row.key}
          className="mb-1.5 grid items-center gap-3"
          style={{ gridTemplateColumns: COLUMNS }}
        >
          <div className="truncate text-right text-xs" title={row.label}>
            {row.label}
          </div>
          <div className="grid auto-cols-fr grid-flow-col gap-[2px]">
            {row.cells.map((cell) => (
              <Tooltip key={cell.key} content={cell.tooltip} className="w-full">
                <span
                  className="flex h-[22px] w-full items-center justify-center text-3xs font-semibold"
                  style={
                    cell.pct == null
                      ? UNMEASURED_HATCH
                      : { backgroundColor: bandCellInk(cell.pct - parity), color: 'var(--background)' }
                  }
                  tabIndex={0}
                  role="img"
                  aria-label={cell.tooltip}
                >
                  {cell.pct == null ? '—' : `${Math.round(cell.pct)}%`}
                </span>
              </Tooltip>
            ))}
          </div>
        </div>
      ))}
      <div className="grid gap-3" style={{ gridTemplateColumns: COLUMNS }}>
        <div aria-hidden="true" />
        <div className="mt-1.5 grid auto-cols-fr grid-flow-col gap-[2px] border-t border-border pt-1 text-center text-3xs leading-tight text-muted-foreground">
          {columns.map((col) => (
            <div key={col.key}>
              {col.time}
              <br />
              {col.map}
            </div>
          ))}
        </div>
      </div>
      <div
        className="mt-0.5 text-3xs text-muted-foreground"
        style={{ marginLeft: 'calc(196px + 0.75rem)' }}
      >
        {axisTitle}
      </div>
    </div>
  )
}
