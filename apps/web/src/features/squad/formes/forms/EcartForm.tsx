/**
 * EcartForm.tsx — L'ÉCART À LA PARITÉ (artefact 2ec1b8eb, forme `ecart`).
 *
 * LE TRAIT AMBRE EST LA PARITÉ, ET LA BARRE PART DE LÀ. À droite : plus que la
 * référence. À gauche : moins. **La longueur EST l'écart**, en points de
 * pourcentage — pas la part elle-même, qui est écrite en clair à droite.
 *
 * C'est ce qui distingue cette forme d'une jauge : une part de 45 % contre une
 * parité de 50 % et une part de 12 % contre une parité de 11,9 % ont des
 * longueurs opposées, alors que leurs jauges se ressembleraient.
 *
 * L'AMPLITUDE EST COMMUNE À TOUTES LES LIGNES d'une même forme, arrondie au
 * palier de cinq points au-dessus du plus grand écart (six points au minimum) :
 * deux lignes de la même carte doivent se comparer à l'œil.
 *
 * Infobulle au survol ET au focus clavier, comme toutes les marques du bloc.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { MINUS_INK, PARITY_INK, PLUS_INK, TRACK_INK } from '../colors'
import { ecartAmplitude } from '../scales'
import { ValueAxis } from './ValueAxis'

/** Une ligne d'écart : sa part, sa parité, et ce qu'en dit l'infobulle. */
export interface EcartRow {
  key: string
  label: string
  /** Sous-libellé (le dénominateur de la ligne, l'aide d'un rôle...). */
  sublabel?: string
  /** La part mesurée. `null` = NON MESURÉ : aucune barre, la mention l'écrit. */
  pct: number | null
  parity: number
  /** Le détail brut de l'infobulle (« 12 sur 98 »). */
  detail?: string
}

export interface EcartFormProps {
  rows: EcartRow[]
  /** Textes : formatage et libellés viennent de l'appelant (aucune langue ici). */
  formatPct: (v: number) => string
  formatSigned: (v: number) => string
  notMeasuredLabel: string
  parityTipFmt: (parity: string) => string
  rowTipFmt: (row: string, value: string, parity: string, gap: string) => string
  pointsFmt: (signed: string) => string
  axisTitle: string
  parityAxisLabel: string
}

const COLUMNS = '196px 1fr 128px'

export function EcartForm({
  rows,
  formatPct,
  formatSigned,
  notMeasuredLabel,
  parityTipFmt,
  rowTipFmt,
  pointsFmt,
  axisTitle,
  parityAxisLabel,
}: EcartFormProps) {
  const amplitude = ecartAmplitude(rows)
  return (
    <div className="min-w-[520px]">
      {rows.map((row) => {
        const gap = row.pct == null ? null : row.pct - row.parity
        const width = gap == null ? 0 : Math.min(50, (Math.abs(gap) / amplitude) * 50)
        const tip =
          gap == null
            ? null
            : rowTipFmt(row.label, formatPct(row.pct as number), formatPct(row.parity), formatSigned(gap)) +
              (row.detail ? ` · ${row.detail}` : '')
        return (
          <div
            key={row.key}
            className="mb-2 grid items-center gap-3"
            style={{ gridTemplateColumns: COLUMNS }}
          >
            <div className="truncate text-right text-xs" title={row.label}>
              {row.label}
              {row.sublabel != null && (
                <small className="block text-3xs text-muted-foreground">{row.sublabel}</small>
              )}
            </div>
            <div className="relative h-5" style={{ backgroundColor: TRACK_INK }}>
              <div
                className="absolute -top-[3px] bottom-[-3px] flex w-[3px]"
                style={{ left: 'calc(50% - 1.5px)' }}
              >
                <Tooltip content={parityTipFmt(formatPct(row.parity))} className="h-full w-full">
                  <span
                    className="block h-full w-full"
                    style={{ backgroundColor: PARITY_INK }}
                    tabIndex={0}
                    role="img"
                    aria-label={parityTipFmt(formatPct(row.parity))}
                  />
                </Tooltip>
              </div>
              {gap != null && tip != null && (
                <div
                  className="absolute top-[3px] flex h-3.5"
                  style={{ width: `${width}%`, left: gap >= 0 ? '50%' : `${50 - width}%` }}
                >
                  <Tooltip content={tip} className="h-full w-full">
                    <span
                      className="block h-full w-full"
                      style={{ backgroundColor: gap >= 0 ? PLUS_INK : MINUS_INK }}
                      tabIndex={0}
                      role="img"
                      aria-label={tip}
                    />
                  </Tooltip>
                </div>
              )}
            </div>
            <div className="text-3xs tabular-nums">
              {row.pct == null ? (
                <span className="text-muted-foreground">{notMeasuredLabel}</span>
              ) : (
                <>
                  <span className="text-foreground">{formatPct(row.pct)}</span>
                  <small className="pl-1.5 text-muted-foreground">
                    {pointsFmt(formatSigned(row.pct - row.parity))}
                  </small>
                </>
              )}
            </div>
          </div>
        )
      })}
      <ValueAxis
        columns={COLUMNS}
        min={-amplitude}
        max={amplitude}
        title={axisTitle}
        format={(v) => (Math.round(v) === 0 ? parityAxisLabel : formatSigned(v))}
      />
    </div>
  )
}
