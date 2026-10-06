/**
 * EquipmentOutcomesCard — « Équipement pris, et ce que j'en ai fait » (Séries temporelles › Usages,
 * bloc « Équipement » ; maquette v4, `renderEquip`, décision D4 du plan).
 *
 * Une ligne par famille, dans l'ordre publié par le Go (non mesurées encadrant les mesurées). Famille
 * mesurée : mes objets servis / gardés sans servir / lâchés (`divergent-pos` / `divergent-neutral` /
 * `divergent-neg`), comptes dans les segments quand ils y tiennent (mesure au pixel), sur une ligne
 * de repli au-dessus sinon — seuls ceux qui ne tiennent pas, à la place de leur part (S2) ;
 * sous-libellé « n objets, dont m pris sur la carte » ; dessous, la barre fine du reste de mon camp
 * et sa ligne de parts. Famille non mesurée (grappin, propulseur) : « Non mesuré » et le compte de
 * mes lâchers. Axe 0-100 % sous les lignes.
 */
import { useMemo, useRef } from 'react'

import { useSegmentLabelFit } from '@/components/charts/segmentLabelFit'
import { Tooltip } from '@/components/ui/tooltip'
import { tokenCssVar, type SemanticToken } from '@/lib/accessibility'

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
const PART_TOKENS: Record<(typeof PARTS)[number], SemanticToken> = {
  used: 'divergent-pos',
  kept: 'divergent-neutral',
  dropped: 'divergent-neg',
}

interface Props {
  rows: EquipmentRow[]
  familyLabel: (family: string) => string
  ut: UsagesCardsText
}

export function EquipmentOutcomesCard({ rows, familyLabel, ut }: Props) {
  const e = ut.equipment
  const ref = useRef<HTMLDivElement | null>(null)
  const hidden = useSegmentLabelFit(ref, rows)
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
          <EquipmentLine key={r.family} row={r} label={familyLabel(r.family)} hidden={hidden} ut={ut} />
        ))}
        <TrackAxis columns={COLUMNS} ticks={TICKS.map((v) => ({ at: v, label: v === 100 ? ut.pctIntFmt(100) : String(v) }))} />
      </div>
    </ObjectifFrame>
  )
}

function EquipmentLine({ row, label, hidden, ut }: { row: EquipmentRow; label: string; hidden: ReadonlySet<string>; ut: UsagesCardsText }) {
  const e = ut.equipment
  const id = row.family
  const meId = `usages-equip-me-${id}`
  const meTotal = row.measured ? row.me[0] + row.me[1] + row.me[2] : 0
  const restTotal = row.measured ? row.rest[0] + row.rest[1] + row.rest[2] : 0
  return (
    <div className="grid items-center gap-3" style={{ gridTemplateColumns: COLUMNS }} data-testid={`usages-equip-row-${id}`}>
      <div className="min-w-0 text-[12.5px] leading-tight">
        {label}
        <small className="block text-[11px] text-muted-foreground" data-testid={`usages-equip-sub-${id}`}>
          {row.measured ? e.sub(meTotal, row.takenMe) : e.droppedSub(row.droppedMe)}
        </small>
      </div>
      {!row.measured ? (
        <div className="flex h-[22px] items-center rounded-[3px] border border-dashed border-border px-2 text-[11px] text-muted-foreground">
          {e.unmeasured}
        </div>
      ) : (
        <div className="flex min-w-0 flex-col gap-1">
          {meTotal > 0 && <RepliLine id={meId} family={id} parts={row.me} hidden={hidden} />}
          {meTotal > 0 ? (
            <ThreeTrack id={meId} parts={row.me} who={`${e.me} · ${label}`} hidden={hidden} ut={ut} />
          ) : (
            <Tooltip content={e.zeroTip(label)} className="w-full">
              <div className="h-[22px] w-full rounded-[3px] bg-muted" />
            </Tooltip>
          )}
          {restTotal > 0 && <ThreeTrack id={`usages-equip-rest-${id}`} parts={row.rest} who={`${e.rest} · ${label}`} ut={ut} />}
          <div className="flex justify-between gap-2 text-[11px] tabular-nums text-muted-foreground" data-testid={`usages-equip-restline-${id}`}>
            {restTotal > 0 ? (
              <>
                <span>{e.restLine(row.rest[0], row.rest[1], row.rest[2])}</span>
                <span>{e.restUsedShare(ut.pctIntFmt((row.rest[0] / restTotal) * 100))}</span>
              </>
            ) : (
              <span>{e.restNone}</span>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

/**
 * La ligne de repli au-dessus de la barre épaisse : le compte de chaque part qui ne tient pas dans
 * son segment, à sa place (servi à gauche, gardé au centre, lâché à droite) ; absente quand tout
 * tient.
 */
function RepliLine({ id, family, parts, hidden }: { id: string; family: string; parts: EquipmentParts; hidden: ReadonlySet<string> }) {
  const repli = PARTS.map((part, i) => parts[i] > 0 && hidden.has(`${id}-${part}`))
  if (!repli.some(Boolean)) return null
  return (
    <div className="flex justify-between gap-2 text-xs tabular-nums text-muted-foreground" data-testid={`usages-equip-repli-${family}`}>
      {PARTS.map((part, i) => (
        <span key={part} className="inline-flex items-center">
          {repli[i] && (
            <>
              <span className="mr-[5px] inline-block h-[9px] w-[9px] rounded-[2px]" style={{ backgroundColor: tokenCssVar(PART_TOKENS[part]) }} aria-hidden />
              <b className="font-bold text-foreground">{parts[i]}</b>
            </>
          )}
        </span>
      ))}
    </div>
  )
}

/**
 * Trois parts côte à côte (servi, gardé, lâché) ; `hidden` donné : comptes dans les segments (barre
 * épaisse), masqués quand ils ne tiennent pas (la ligne de repli les porte).
 */
function ThreeTrack({ id, parts, who, hidden, ut }: { id: string; parts: EquipmentParts; who: string; hidden?: ReadonlySet<string>; ut: UsagesCardsText }) {
  const labels = hidden != null
  const total = parts[0] + parts[1] + parts[2]
  // Le bord gauche de chaque part : la somme des parts qui la précèdent.
  const lefts = PARTS.map((_, i) => (parts.slice(0, i).reduce((a, b) => a + b, 0) / total) * 100)
  return (
    <div className={`relative rounded-[3px] bg-muted ${labels ? 'h-[22px]' : 'h-2'}`} role="img" aria-label={who}>
      {PARTS.map((part, i) => {
        const n = parts[i]
        if (n <= 0) return null
        const w = (n / total) * 100
        const left = lefts[i]
        return (
          <div
            key={part}
            className="absolute inset-y-0"
            style={{ left: `${left}%`, width: `${w}%`, backgroundColor: tokenCssVar(PART_TOKENS[part]) }}
            data-testid={`${id}-${part}`}
            data-fit-key={labels ? `${id}-${part}` : undefined}
          >
            <Tooltip content={<TipText text={ut.equipment.segTip(who, n, part, total, ut.pctFmt(w))} />} className="h-full w-full">
              <div className="flex h-full w-full cursor-help items-center justify-center overflow-hidden">
                {/* Écriture posée sur l'aplat : contraste dans le segment, pas une couleur sémantique. */}
                {labels && (
                  <span
                    data-fit-label
                    className={`whitespace-nowrap text-[11px] font-semibold tabular-nums leading-none ${part === 'kept' ? 'text-foreground' : 'text-white'}`}
                    style={{ visibility: hidden?.has(`${id}-${part}`) ? 'hidden' : 'visible' }}
                  >
                    {n}
                  </span>
                )}
              </div>
            </Tooltip>
          </div>
        )
      })}
    </div>
  )
}
