/**
 * SquadFragBreakdownCard — « Répartition des frags » (Escouade › Contributions) : une barre
 * empilée par joueur, un segment par classe d'arme, sur une échelle commune.
 *
 * Maquette C3EW (colonne « Proposition ») et règle S3 du plan
 * PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 :
 *   - le compte de chaque classe s'écrit DANS son segment s'il y tient avec 6 px de marge de
 *     chaque côté (mesure au pixel, `useSegmentLabelFit`) ;
 *   - sinon il part sur une ligne de repli AU-DESSUS de la barre, alignée sur le premier
 *     segment masqué, pastille de la classe + compte — jamais tronqué, jamais seulement en
 *     infobulle ;
 *   - le total au bout de la barre ; la légende des classes en bas, centrée ; le graphe centré
 *     verticalement dans la carte (qui s'étire à la hauteur de sa rangée).
 */
import { useMemo, useRef } from 'react'
import { ChartLegend } from '@/components/charts/ChartLegend'
import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { InfoTooltip } from '@/components/ui/info-tooltip'
import { Tooltip } from '@/components/ui/tooltip'
import { EmptyStateNotice } from '@/components/ui/empty-state'
import { useColorPaletteVersion } from '@/lib/accessibility'
import { fragClassColor, fragClassCssVar } from '@/lib/accessibility/scales'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { FragClassEntry } from '@/lib/api/types'
import {
  buildFragBreakdownRows,
  fragBreakdownClasses,
  repliOffsetPct,
  segmentTextTone,
  type FragBreakdownRow,
  type FragBreakdownSegment,
} from './charts/squadFragBreakdownChart'
import type { SquadText } from './i18n'

interface SquadFragBreakdownCardProps {
  fragClassesByPlayer: Record<string, FragClassEntry[]>
  playerOrder: string[]
  /** gamertag → couleur (getSquadPlayerColors). */
  playerColors: Record<string, string>
  /** Libellé localisé d'une classe (manifeste `frags`). */
  classLabel: (cls: string) => string
  emptyTitle: string
  t: SquadText
  /**
   * Vue compacte du tiroir de comparaison de Sessions (maquette `makeFragbar` avec `cp`) : la part
   * entière de chaque classe dans son segment et au repli (`pctFmt`), le total en sous-libellé du nom
   * (`totalSub`), rien au bout de la barre ; les comptes restent dans l'infobulle.
   */
  compact?: FragBreakdownCompact
}

interface FragBreakdownCompact {
  totalSub: (total: number) => string
  pctFmt: (v: number) => string
}

const fitKey = (player: string, cls: string) => `${player}|${cls}`

export function SquadFragBreakdownCard({
  fragClassesByPlayer,
  playerOrder,
  playerColors,
  classLabel,
  emptyTitle,
  t,
  compact,
}: SquadFragBreakdownCardProps) {
  const rows = useMemo(
    () => buildFragBreakdownRows(fragClassesByPlayer, playerOrder),
    [fragClassesByPlayer, playerOrder],
  )
  // Couleurs résolues (légende, ton de l'écriture) : suivent palette et thème.
  const paletteVersion = useColorPaletteVersion()
  const themeVersion = useThemeVersion()
  const classes = useMemo(
    () => fragBreakdownClasses(fragClassesByPlayer, playerOrder),
    [fragClassesByPlayer, playerOrder],
  )
  const tones = useMemo(
    () => new Map(classes.map((c) => [c, segmentTextTone(fragClassColor(c))])),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [classes, paletteVersion, themeVersion],
  )
  const legendItems = useMemo(
    () => classes.map((c) => ({ key: c, label: classLabel(c), color: fragClassColor(c) })),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [classes, classLabel, paletteVersion, themeVersion],
  )

  const bodyRef = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(bodyRef, rows)
  const title = t.performanceCharts.fragBreakdownTitle

  return (
    <div className="flex h-full flex-col rounded-lg border border-border bg-card" data-testid="squad-frag-breakdown">
      <div className="flex-none border-b border-border px-3 py-2 text-sm font-medium">
        <span className="flex items-center gap-1.5">
          {title}
          <InfoTooltip content={t.performanceCharts.fragBreakdownInfo} />
        </span>
      </div>
      <div className="flex flex-1 flex-col p-3">
        {rows.length === 0 ? (
          <EmptyStateNotice title={emptyTitle} description={t.empty.noBlockData} />
        ) : (
          <div ref={bodyRef} className="my-auto flex flex-col gap-2.5">
            {rows.map((row) => (
              <FragBreakdownBar
                key={row.player}
                row={row}
                color={playerColors[row.player]}
                hidden={hidden}
                tones={tones}
                classLabel={classLabel}
                t={t}
                compact={compact}
              />
            ))}
          </div>
        )}
      </div>
      {rows.length > 0 && (
        <div className="flex-none border-t border-border px-3 py-2" data-testid="chart-card-legend">
          <ChartLegend items={legendItems} ariaLabel={title} />
        </div>
      )}
    </div>
  )
}

interface FragBreakdownBarProps {
  row: FragBreakdownRow
  color: string | undefined
  hidden: ReadonlySet<string>
  tones: Map<string, 'dark' | 'light'>
  classLabel: (cls: string) => string
  t: SquadText
  compact?: FragBreakdownCompact
}

function FragBreakdownBar({ row, color, hidden, tones, classLabel, t, compact }: FragBreakdownBarProps) {
  const isHidden = (cls: string) => hidden.has(fitKey(row.player, cls))
  const offset = repliOffsetPct(row.segments, isHidden)
  // Ce qu'écrit un segment : son compte, ou sa part entière du total du joueur en vue compacte.
  const textOf = (s: FragBreakdownSegment) => (compact ? compact.pctFmt(row.total > 0 ? (s.kills / row.total) * 100 : 0) : String(s.kills))
  return (
    <div
      className={`grid ${compact ? 'grid-cols-[6rem_minmax(0,1fr)]' : 'grid-cols-[7rem_minmax(0,1fr)_2.5rem]'} items-center gap-2.5 text-xs`}
      data-testid={`frag-breakdown-row-${row.player}`}
    >
      <div className="flex min-w-0 items-center gap-1.5">
        <span className="inline-block h-2.5 w-2.5 shrink-0 rounded-full" style={{ backgroundColor: color }} aria-hidden />
        <span className="min-w-0 truncate" title={row.player}>
          {row.player}
          {compact && <small className="block text-[11px] text-muted-foreground">{compact.totalSub(row.total)}</small>}
        </span>
      </div>
      <div className="flex min-w-0 flex-col gap-[3px]">
        {offset != null && (
          <FragBreakdownRepli
            row={row}
            offset={offset}
            folded={row.segments.filter((s) => isHidden(s.cls))}
            classLabel={classLabel}
            textOf={textOf}
          />
        )}
        <div
          className="relative h-[22px]"
          role="img"
          aria-label={t.performanceCharts.fragBreakdownBarAria(row.player, row.total)}
        >
          {row.segments.map((s, i) => (
            <FragBreakdownSeg
              key={s.cls}
              row={row}
              seg={s}
              first={i === 0}
              hidden={isHidden(s.cls)}
              tone={tones.get(s.cls)}
              classLabel={classLabel}
              t={t}
              text={textOf(s)}
            />
          ))}
        </div>
      </div>
      {!compact && (
        <div className="text-right font-semibold tabular-nums" data-testid={`frag-breakdown-total-${row.player}`}>
          {row.total}
        </div>
      )}
    </div>
  )
}

/** La ligne de repli au-dessus de la barre : pastille + compte des segments trop étroits. */
function FragBreakdownRepli({
  row,
  offset,
  folded,
  classLabel,
  textOf,
}: {
  row: FragBreakdownRow
  offset: number
  folded: FragBreakdownSegment[]
  classLabel: (cls: string) => string
  textOf: (s: FragBreakdownSegment) => string
}) {
  return (
    <div
      className="flex gap-2 text-2xs font-semibold tabular-nums"
      style={{ paddingLeft: `${offset}%` }}
      data-testid={`frag-breakdown-repli-${row.player}`}
    >
      {folded.map((s) => (
        <span key={s.cls} className="inline-flex items-center gap-[3px]" title={classLabel(s.cls)}>
          <span
            className="inline-block h-2 w-2 rounded-[2px]"
            style={{ backgroundColor: fragClassCssVar(s.cls) }}
            aria-hidden
          />
          {textOf(s)}
        </span>
      ))}
    </div>
  )
}

/** Un segment de la barre : aplat de la classe, compte écrit dedans s'il y tient, infobulle. */
function FragBreakdownSeg({
  row,
  seg: s,
  first,
  hidden,
  tone,
  classLabel,
  t,
  text,
}: {
  row: FragBreakdownRow
  seg: FragBreakdownSegment
  first: boolean
  hidden: boolean
  tone: 'dark' | 'light' | undefined
  classLabel: (cls: string) => string
  t: SquadText
  /** Ce qui s'écrit dans le segment (compte, ou part en vue compacte). */
  text: string
}) {
  return (
    <div
      className={`absolute inset-y-0 ${first ? 'rounded-l-[3px]' : 'border-l-2 border-card'}`}
      style={{ left: `${s.leftPct}%`, width: `${s.widthPct}%`, backgroundColor: fragClassCssVar(s.cls) }}
      data-fit-key={fitKey(row.player, s.cls)}
      data-testid={`frag-breakdown-seg-${row.player}-${s.cls}`}
    >
      <Tooltip
        content={t.performanceCharts.fragBreakdownSegment(row.player, classLabel(s.cls), s.kills, row.total)}
        className="h-full w-full"
      >
        <div className="flex h-full w-full cursor-help items-center justify-center overflow-hidden">
          {/* `text-white` / `text-black` : une écriture posée SUR un aplat, question de
              contraste dans le segment et non couleur sémantique (même usage que
              StackedTrack). */}
          <span
            data-fit-label
            className={`whitespace-nowrap text-2xs font-semibold tabular-nums ${
              tone === 'dark' ? 'text-black' : 'text-white'
            }`}
            style={{ visibility: hidden ? 'hidden' : 'visible' }}
          >
            {text}
          </span>
        </div>
      </Tooltip>
    </div>
  )
}
