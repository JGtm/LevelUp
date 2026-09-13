/**
 * JaugeDoubleForm.tsx — LA JAUGE DOUBLE (artefact 2ec1b8eb, forme `jaugeDouble`).
 *
 * DEUX PISTES SUR UN AXE COMMUN : ma part DANS MON ÉQUIPE, puis DANS LE LOBBY.
 * Chacune porte son propre trait de parité et l'étendue de ses parts match par
 * match. La chute entre les deux pistes EST la taille de mon équipe dans le
 * lobby — c'est ce que la forme existe pour montrer.
 *
 * Le texte de droite dit la part et le nombre de matchs au-dessus de la parité :
 * une part de session ne vaut rien sans savoir si elle vient d'un match ou de
 * tous.
 *
 * L'ÉCHELLE EST COMMUNE aux deux pistes de toutes les lignes, arrondie au palier
 * au-dessus de la plus grande valeur (part ou borne d'étendue) : deux lignes
 * doivent se comparer à l'œil.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { PARITY_INK, SPREAD_INK, TRACK_INK, squadPlayerInk } from '../colors'
import { ValueAxis } from './ValueAxis'
import { niceBound } from '../format'

/** Une piste : une part, sa parité, son étendue, son compte au-dessus. */
export interface JaugeTrack {
  /** Le libellé du dénominateur (« dans mon équipe » / « dans le lobby »). */
  side: string
  pct: number | null
  parity: number | null
  min: number | null
  max: number | null
  above: number
  measured: number
  /** Le détail brut de l'infobulle (« 12 sur 98 »). */
  detail: string
}

export interface JaugeRow {
  key: string
  label: string
  tracks: JaugeTrack[]
}

export interface JaugeDoubleFormProps {
  rows: JaugeRow[]
  formatPct: (v: number) => string
  notMeasuredLabel: string
  aboveFmt: (above: number, total: number) => string
  parityTipFmt: (parity: string) => string
  shareTipFmt: (row: string, side: string, value: string, detail: string) => string
  spreadTipFmt: (row: string, side: string, from: string, to: string) => string
  axisTitle: string
}

const COLUMNS = '196px 84px 1fr 132px'

export function JaugeDoubleForm({
  rows,
  formatPct,
  notMeasuredLabel,
  aboveFmt,
  parityTipFmt,
  shareTipFmt,
  spreadTipFmt,
  axisTitle,
}: JaugeDoubleFormProps) {
  const values = rows.flatMap((r) =>
    r.tracks.flatMap((t) => [t.pct ?? 0, t.max ?? 0, t.parity ?? 0]),
  )
  // La marge de 10 % aère la plus longue barre, mais l'axe d'une PART s'arrête à
  // cent : une graduation à 125 % désignerait une valeur impossible.
  const bound = Math.min(100, niceBound(Math.max(1, ...values) * 1.1))
  return (
    <div className="min-w-[560px]">
      {rows.map((row) => (
        <div
          key={row.key}
          className="mb-3 grid items-center gap-x-3 gap-y-1"
          style={{ gridTemplateColumns: COLUMNS }}
        >
          {row.tracks.map((track, index) => (
            <JaugeLine
              key={track.side}
              row={row}
              track={track}
              showLabel={index === 0}
              bound={bound}
              formatPct={formatPct}
              notMeasuredLabel={notMeasuredLabel}
              aboveFmt={aboveFmt}
              parityTipFmt={parityTipFmt}
              shareTipFmt={shareTipFmt}
              spreadTipFmt={spreadTipFmt}
            />
          ))}
        </div>
      ))}
      <ValueAxis
        columns={COLUMNS}
        min={0}
        max={bound}
        title={axisTitle}
        format={(v) => formatPct(v)}
      />
    </div>
  )
}

interface LineProps {
  row: JaugeRow
  track: JaugeTrack
  showLabel: boolean
  bound: number
  formatPct: (v: number) => string
  notMeasuredLabel: string
  aboveFmt: (above: number, total: number) => string
  parityTipFmt: (parity: string) => string
  shareTipFmt: (row: string, side: string, value: string, detail: string) => string
  spreadTipFmt: (row: string, side: string, from: string, to: string) => string
}

function JaugeLine(p: LineProps) {
  const { track, bound } = p
  const pct = (v: number) => `${Math.min(100, (v / bound) * 100)}%`
  const shareTip =
    track.pct == null
      ? null
      : p.shareTipFmt(p.row.label, track.side, p.formatPct(track.pct), track.detail)
  const spreadTip =
    track.min == null || track.max == null
      ? null
      : p.spreadTipFmt(p.row.label, track.side, p.formatPct(track.min), p.formatPct(track.max))
  return (
    <>
      <div className="truncate text-right text-xs" title={p.showLabel ? p.row.label : undefined}>
        {p.showLabel ? p.row.label : ''}
      </div>
      <div className="text-right text-3xs uppercase tracking-wide text-muted-foreground">
        {track.side}
      </div>
      <div className="relative h-[15px]" style={{ backgroundColor: TRACK_INK }}>
        {spreadTip != null && track.min != null && track.max != null && (
          <div
            className="absolute top-[6px] flex h-[3px]"
            style={{
              left: pct(track.min),
              width: `${Math.max(0.6, ((track.max - track.min) / bound) * 100)}%`,
            }}
          >
            <Tooltip content={spreadTip} className="h-full w-full">
              <span
                className="block h-full w-full"
                style={{ backgroundColor: SPREAD_INK }}
                tabIndex={0}
                role="img"
                aria-label={spreadTip}
              />
            </Tooltip>
          </div>
        )}
        {track.pct != null && shareTip != null && (
          <div className="absolute left-0 top-[2px] flex h-[11px]" style={{ width: pct(track.pct) }}>
            <Tooltip content={shareTip} className="h-full w-full">
              <span
                className="block h-full w-full"
                style={{ backgroundColor: squadPlayerInk(0) }}
                tabIndex={0}
                role="img"
                aria-label={shareTip}
              />
            </Tooltip>
          </div>
        )}
        {track.parity != null && (
          <div
            className="absolute -top-[3px] bottom-[-3px] flex w-[3px]"
            style={{ left: `calc(${Math.min(100, (track.parity / bound) * 100)}% - 1.5px)` }}
          >
            <Tooltip
              content={p.parityTipFmt(p.formatPct(track.parity))}
              className="h-full w-full"
            >
              <span
                className="block h-full w-full"
                style={{ backgroundColor: PARITY_INK }}
                tabIndex={0}
                role="img"
                aria-label={p.parityTipFmt(p.formatPct(track.parity))}
              />
            </Tooltip>
          </div>
        )}
      </div>
      <div className="whitespace-nowrap text-3xs tabular-nums text-muted-foreground">
        {track.pct == null ? (
          p.notMeasuredLabel
        ) : (
          <>
            <b className="text-xs text-foreground">{p.formatPct(track.pct)}</b>
            {'  ·  '}
            {p.aboveFmt(track.above, track.measured)}
          </>
        )}
      </div>
    </>
  )
}
