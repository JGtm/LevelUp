/**
 * BatonMinMaxForm.tsx — LE BÂTON D'ÉTENDUE (artefact 2ec1b8eb, forme
 * `batonMinMax`).
 *
 * UNE PÉRIODE EN UNE LIGNE PAR GESTE : le segment va du match le plus faible au
 * plus fort, le trait ambre est la MOYENNE PAR MATCH MESURÉ, et les trois
 * chiffres sont écrits à droite.
 *
 * PAR MATCH, JAMAIS PAR MINUTE (décision utilisateur du 2026-09-13) : aucune
 * grandeur n'est divisée par un temps de jeu.
 *
 * CE QU'ELLE ABANDONNE : l'ordre des matchs. On ne voit plus QUAND le pic a eu
 * lieu — c'est la grille par match qui le dit.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { PARITY_INK } from '../colors'
import { niceBound } from '../format'
import { ValueAxis } from './ValueAxis'

/** Une série : un geste, son étendue, sa moyenne. */
export interface BatonSerie {
  key: string
  label: string
  ink: string
  min: number
  max: number
  mean: number
}

export interface BatonMinMaxFormProps {
  series: BatonSerie[]
  formatValue: (v: number) => string
  rangeTipFmt: (row: string, from: string, to: string) => string
  meanTipFmt: (row: string, value: string) => string
  axisTitle: string
}

const COLUMNS = '182px 1fr 152px'

export function BatonMinMaxForm({
  series,
  formatValue,
  rangeTipFmt,
  meanTipFmt,
  axisTitle,
}: BatonMinMaxFormProps) {
  const bound = niceBound(Math.max(1, ...series.map((s) => s.max)))
  return (
    <div className="min-w-[520px]">
      {series.map((serie) => {
        const rangeTip = rangeTipFmt(serie.label, formatValue(serie.min), formatValue(serie.max))
        const meanTip = meanTipFmt(serie.label, formatValue(serie.mean))
        return (
          <div
            key={serie.key}
            className="mb-2.5 grid items-center gap-3"
            style={{ gridTemplateColumns: COLUMNS }}
          >
            <div className="flex items-center justify-end gap-[7px] text-right text-xs">
              {serie.label}
              <i
                className="h-2.5 w-2.5 flex-none"
                style={{ backgroundColor: serie.ink }}
                aria-hidden="true"
              />
            </div>
            <div className="relative h-[18px]">
              <span className="absolute left-0 right-0 top-[8px] h-px bg-border" aria-hidden="true" />
              <div
                className="absolute top-[6px] h-1.5"
                style={{
                  left: `${(serie.min / bound) * 100}%`,
                  width: `${Math.max(0.6, ((serie.max - serie.min) / bound) * 100)}%`,
                }}
              >
                <Tooltip content={rangeTip} className="h-full w-full">
                  <span
                    className="block h-full w-full"
                    style={{ backgroundColor: serie.ink }}
                    tabIndex={0}
                    role="img"
                    aria-label={rangeTip}
                  />
                </Tooltip>
              </div>
              <div
                className="absolute top-[2px] h-3.5 w-[3px]"
                style={{ left: `calc(${(serie.mean / bound) * 100}% - 1.5px)` }}
              >
                <Tooltip content={meanTip} className="h-full w-full">
                  <span
                    className="block h-full w-full"
                    style={{ backgroundColor: PARITY_INK }}
                    tabIndex={0}
                    role="img"
                    aria-label={meanTip}
                  />
                </Tooltip>
              </div>
            </div>
            <div className="text-3xs tabular-nums text-muted-foreground">
              {`${formatValue(serie.min)} … ${formatValue(serie.max)}  ·  ${formatValue(serie.mean)}`}
            </div>
          </div>
        )
      })}
      <ValueAxis columns={COLUMNS} min={0} max={bound} title={axisTitle} format={formatValue} />
    </div>
  )
}
