/**
 * EquipmentOutcomesCard — « Équipement : servi, gardé, lâché » (Séries temporelles › Usages, bloc
 * « Équipement » ; maquette v4, `renderEquip`, décision D4 du plan).
 *
 * Une ligne par famille, dans l'ordre publié par le Go : les objets du joueur servis / gardés sans
 * servir / lâchés (`divergent-pos` / `divergent-neutral` / `divergent-neg`), comptes dans les
 * segments quand ils y tiennent (mesure au pixel), sinon à côté de la barre, juste après son bout,
 * sur la même ligne et centrés sur elle, pastille de la part devant (`BarSides` : la place est
 * réservée à droite de toutes les barres ; jamais au-dessus, jamais seulement en infobulle) ; sous-libellé
 * « n objets, dont m pris sur la carte » ;
 * dessous, la barre fine du reste du camp et sa ligne de parts. Axe 0-100 % sous les lignes.
 */
import { useMemo, useRef } from 'react'

import { BarWithSides, SideValue, sideSlotPx } from '@/components/charts/BarSides'
import { SegmentLabel } from '@/components/charts/SegmentLabel'
import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

import type { FragBreakdownSegment } from '@/features/squad/charts/squadFragBreakdownChart'
import { TrackAxis } from '@/features/squad/emprise/PisteCampsForm'
import { pisteColumns } from '@/features/squad/emprise/pisteLayout'
import { TipText } from '@/features/squad/emprise/TipText'
import { ObjectifFrame, ObjectifLegend } from '@/features/squad/objectif/ObjectifFrame'

import type { EquipmentParts, EquipmentRow } from './usages.logic'
import type { UsagesCardsText } from './usagesCardsText'

/** Colonne des noms (maquette : `--lw:156px`). */
const COLUMNS = pisteColumns(156)
const TICKS = [0, 25, 50, 75, 100] as const
const PARTS = ['used', 'kept', 'dropped'] as const
type Part = (typeof PARTS)[number]
const PART_TOKENS: Record<Part, SemanticToken> = {
  used: 'divergent-pos',
  kept: 'divergent-neutral',
  dropped: 'divergent-neg',
}

/**
 * Vue compacte du tiroir de comparaison de Sessions (maquette `renderEquip` avec `cp`) : parts
 * entières dans les segments, sous-libellé « n objets » (`sub`), ligne du reste réduite à sa part de
 * servis (`restUsed`).
 */
interface EquipmentCompact {
  sub: (objects: number) => string
  restUsed: (pct: string) => string
}

interface Props {
  rows: EquipmentRow[]
  familyLabel: (family: string) => string
  /** Le gamertag du joueur de la page. */
  player: string
  ut: UsagesCardsText
  compact?: EquipmentCompact
}

export function EquipmentOutcomesCard({ rows, familyLabel, player, ut, compact }: Props) {
  const e = ut.equipment
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, rows)
  // La place des valeurs qui débordent, réservée à droite des barres : deux valeurs au plus côte à côte.
  const textOfPart = (s: FragBreakdownSegment) => (compact ? ut.pctIntFmt(s.widthPct) : String(s.kills))
  const rightPx = sideSlotPx(
    rows.map((r) =>
      partSegments(r.me)
        .map(textOfPart)
        .sort((a, b) => b.length - a.length)
        .slice(0, 2)
        .join('  '),
    ),
  )
  const legend = useMemo(
    () => (
      <ObjectifLegend
        ariaLabel={e.title}
        items={[
          { kind: 'square', label: e.used, color: tokenCssVar('divergent-pos') },
          { kind: 'square', label: e.kept, color: tokenCssVar('divergent-neutral') },
          { kind: 'square', label: e.dropped, color: tokenCssVar('divergent-neg') },
          { kind: 'thin', label: e.thinLegend, color: 'var(--muted-foreground)' }, // color-allow: repère neutre de la légende (maquette)
        ]}
      />
    ),
    [e],
  )
  return (
    <ObjectifFrame title={e.title} info={e.info} legend={legend} testId="usages-equipment">
      <div ref={ref} className="flex flex-col gap-3">
        {rows.map((r) => (
          <EquipmentLine key={r.family} row={r} label={familyLabel(r.family)} hidden={hidden} player={player} ut={ut} compact={compact} rightPx={rightPx} />
        ))}
        <TrackAxis columns={COLUMNS} inset={{ left: 0, right: rightPx }} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? ut.pctIntFmt(100) : String(v) }))} />
      </div>
    </ObjectifFrame>
  )
}

function EquipmentLine({
  row,
  label,
  hidden,
  player,
  ut,
  compact,
  rightPx,
}: {
  row: EquipmentRow
  label: string
  hidden: ReadonlySet<string>
  player: string
  ut: UsagesCardsText
  compact: EquipmentCompact | undefined
  rightPx: number
}) {
  const e = ut.equipment
  const id = row.family
  const meId = `usages-equip-me-${id}`
  const meTotal = row.me[0] + row.me[1] + row.me[2]
  const restTotal = row.rest[0] + row.rest[1] + row.rest[2]
  // Ce qu'écrit un segment : son compte, ou sa part entière en vue compacte.
  const textOf = (s: FragBreakdownSegment) => (compact ? ut.pctIntFmt(s.widthPct) : String(s.kills))
  const sub = compact ? compact.sub(meTotal) : e.sub(meTotal, row.takenMe)
  return (
    <div className="grid items-center gap-3" style={{ gridTemplateColumns: COLUMNS }} data-testid={`usages-equip-row-${id}`}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        {label}
        <small className="block text-[11px] text-muted-foreground" data-testid={`usages-equip-sub-${id}`}>
          {sub}
        </small>
      </div>
      <div className="flex min-w-0 flex-col gap-1">
        {meTotal > 0 ? (
          <ThreeTrack id={meId} parts={row.me} who={`${player} · ${label}`} hidden={hidden} ut={ut} textOf={textOf} rightPx={rightPx} />
        ) : (
          <div style={{ marginRight: rightPx }}>
            <Tooltip content={e.zeroTip(player, label)} className="w-full">
              <div className="h-[22px] w-full rounded-[3px] bg-muted" />
            </Tooltip>
          </div>
        )}
        {restTotal > 0 && <ThreeTrack id={`usages-equip-rest-${id}`} parts={row.rest} who={`${e.rest} · ${label}`} ut={ut} rightPx={rightPx} />}
        <div className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground" data-testid={`usages-equip-restline-${id}`}>
          {restTotal > 0 && compact ? (
            <span>{compact.restUsed(ut.pctIntFmt((row.rest[0] / restTotal) * 100))}</span>
          ) : restTotal > 0 ? (
            <>
              <span>{e.restLine(row.rest[0], row.rest[1], row.rest[2])}</span>
              <span>{e.restUsedShare(ut.pctIntFmt((row.rest[0] / restTotal) * 100))}</span>
            </>
          ) : (
            <span>{e.restNone}</span>
          )}
        </div>
      </div>
    </div>
  )
}

/** Les segments non vides d'une barre à trois parts, bord gauche et largeur en % de la barre. */
function partSegments(parts: EquipmentParts): FragBreakdownSegment[] {
  const total = parts[0] + parts[1] + parts[2]
  const out: FragBreakdownSegment[] = []
  let left = 0
  PARTS.forEach((part, i) => {
    if (parts[i] <= 0) return
    const widthPct = (parts[i] / total) * 100
    out.push({ cls: part, kills: parts[i], leftPct: left, widthPct })
    left += widthPct
  })
  return out
}

/**
 * Trois parts côte à côte (servi, gardé, lâché) ; `hidden` donné : comptes dans les segments (barre
 * épaisse), à côté de la barre quand ils ne tiennent pas (`rightPx` : la place réservée).
 */
function ThreeTrack({
  id,
  parts,
  who,
  hidden,
  ut,
  textOf,
  rightPx,
}: {
  id: string
  parts: EquipmentParts
  who: string
  hidden?: ReadonlySet<string>
  ut: UsagesCardsText
  /** La place réservée à droite de la barre pour les valeurs qui ne tiennent pas. */
  rightPx: number
  /** Ce qu'écrit un segment de la barre épaisse (compte ou part) ; absent sur la barre fine. */
  textOf?: (s: FragBreakdownSegment) => string
}) {
  const labels = hidden != null
  const total = parts[0] + parts[1] + parts[2]
  const side = labels
    ? partSegments(parts)
        .filter((s) => hidden?.has(`${id}-${s.cls}`))
        .map((s) => (
          <SideValue key={s.cls} color={tokenCssVar(PART_TOKENS[s.cls as Part])} testId={`${id}-side-${s.cls}`}>
            <span className="mr-2">{textOf ? textOf(s) : s.kills}</span>
          </SideValue>
        ))
    : null
  const track = (
    <div className={`relative rounded-[3px] bg-muted ${labels ? 'h-[22px]' : 'h-2'}`} role="img" aria-label={who}>
      {partSegments(parts).map((s) => {
        const part = s.cls as Part
        const n = s.kills
        const w = s.widthPct
        const left = s.leftPct
        return (
          <div
            key={part}
            className="absolute inset-y-0 flex"
            style={{ left: `${left}%`, width: `${w}%`, backgroundColor: tokenCssVar(PART_TOKENS[part]) }}
            data-testid={`${id}-${part}`}
            data-fit-key={labels ? `${id}-${part}` : undefined}
          >
            {labels ? (
              <SegmentLabel fill={tokenCssVar(PART_TOKENS[part])} tip={<TipText text={ut.equipment.segTip(who, n, part, total, ut.pctFmt(w))} />} hidden={hidden?.has(`${id}-${part}`)} textClassName="text-[11px] font-semibold">
                {textOf ? textOf(s) : n}
              </SegmentLabel>
            ) : (
              <Tooltip content={<TipText text={ut.equipment.segTip(who, n, part, total, ut.pctFmt(w))} />} className="h-full w-full">
                <div className="h-full w-full cursor-help" />
              </Tooltip>
            )}
          </div>
        )
      })}
    </div>
  )
  // Barre épaisse : valeurs à côté, sur sa ligne ; barre fine : seulement la même marge.
  return labels ? (
    <BarWithSides leftPx={0} rightPx={rightPx} right={<>{side}</>}>
      {track}
    </BarWithSides>
  ) : (
    <div style={{ marginRight: rightPx }}>{track}</div>
  )
}
