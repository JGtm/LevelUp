/**
 * Piste100Form.tsx — LA PISTE DU LOBBY (artefact 2ec1b8eb, forme `piste100`).
 *
 * LA BARRE ENTIÈRE VAUT LE LOBBY. Coloré : à nous, découpé par joueur. Hachuré :
 * à eux — **compté, jamais nommé** (la référence n'est pas une donnée qu'on
 * affiche, c'est un dénominateur). Le trait ambre est à 50 %, la parité de deux
 * camps.
 *
 * SANS PARITÉ, C'EST UNE AUTRE QUESTION : quand le dénominateur est l'escouade
 * seule, la barre ne dit RIEN de l'adversaire et le trait disparaît. Le laisser
 * ferait lire « à égalité avec l'équipe d'en face » là où la mesure ne parle que
 * du groupe.
 */
import { Tooltip } from '@/components/ui/tooltip'

import { ENEMY_HATCH, PARITY_INK, TRACK_INK, UNMEASURED_HATCH } from '../colors'
import { ValueAxis } from './ValueAxis'

/** Un segment de la piste : un joueur nommé, ou l'adversaire anonyme. */
export interface PisteSegment {
  key: string
  label: string
  value: number
  ink?: string
  /** L'adversaire : hachuré, sans nom de joueur. */
  hatch?: boolean
  /** Le non mesuré (match sans film) : hachure pâle, aucune valeur. */
  unmeasured?: boolean
}

export interface PisteRow {
  key: string
  label: string
  sublabel?: string
  segments: PisteSegment[]
}

export interface Piste100FormProps {
  rows: PisteRow[]
  /** Le trait de parité à 50 %. Faux quand le dénominateur n'est pas le lobby. */
  showParity: boolean
  axisTitle: string
  parityLabel: string
  emptyLabel: string
  formatCount: (v: number) => string
  segmentTipFmt: (name: string, row: string, value: string, total: string) => string
}

const COLUMNS = '196px 1fr'

export function Piste100Form({
  rows,
  showParity,
  axisTitle,
  parityLabel,
  emptyLabel,
  formatCount,
  segmentTipFmt,
}: Piste100FormProps) {
  return (
    <div className="min-w-[520px]">
      {rows.map((row) => {
        const total = row.segments.reduce((a, s) => a + s.value, 0)
        return (
          <div
            key={row.key}
            className="mb-2.5 grid items-center gap-3"
            style={{ gridTemplateColumns: COLUMNS }}
          >
            <div className="truncate text-right text-xs" title={row.label}>
              {row.label}
              {row.sublabel != null && (
                <small className="block text-3xs text-muted-foreground">{row.sublabel}</small>
              )}
            </div>
            <div className="relative flex h-[22px] gap-[2px]">
              {total === 0 ? (
                <div
                  className="flex flex-1 items-center justify-center text-3xs text-muted-foreground"
                  style={{ backgroundColor: TRACK_INK }}
                >
                  {emptyLabel}
                </div>
              ) : (
                row.segments.map((seg) => {
                  if (seg.value <= 0) return null
                  const share = Math.round((seg.value / total) * 100)
                  const tip = segmentTipFmt(
                    seg.label,
                    row.label,
                    formatCount(seg.value),
                    formatCount(total),
                  )
                  return (
                    // La largeur est portée par l'ITEM du flex, jamais par le wrapper du
                    // Tooltip (qui reste `flex-grow: 0` et dimensionnerait le segment à son
                    // texte au lieu de son compte — piège relevé sur la piste du lobby du
                    // bloc d'usage).
                    <div key={seg.key} className="flex h-full" style={{ flex: seg.value }}>
                      <Tooltip content={tip} className="h-full w-full">
                        <span
                          // L'ENCRE DE L'ÉTIQUETTE SUIT LE FOND, PAS LE RÔLE (2026-09-21).
                          // Sur un aplat de joueur, du blanc. Sur la HACHURE de l'adversaire,
                          // le fond reste celui de la carte : le blanc y disparaîtrait en
                          // thème clair, et `muted-foreground` — l'encre précédente — se
                          // confondait avec les rayures, qui sont faites de cette teinte.
                          // `foreground` est l'encre de texte la plus contrastée du thème,
                          // dans les deux thèmes.
                          className={`flex h-full w-full items-center justify-center overflow-hidden whitespace-nowrap text-3xs font-semibold${
                            seg.hatch || seg.unmeasured ? ' text-foreground' : ' text-white'
                          }`}
                          style={
                            seg.unmeasured
                              ? UNMEASURED_HATCH
                              : seg.hatch
                                ? ENEMY_HATCH
                                : { backgroundColor: seg.ink }
                          }
                          tabIndex={0}
                          role="img"
                          aria-label={tip}
                        >
                          {seg.unmeasured ? '' : share >= 12 ? `${share} %` : ''}
                        </span>
                      </Tooltip>
                    </div>
                  )
                })
              )}
              {showParity && total > 0 && (
                <div
                  className="absolute -top-[3px] bottom-[-3px] z-10 flex w-[3px]"
                  style={{ left: 'calc(50% - 1.5px)' }}
                >
                  <Tooltip content={parityLabel} className="h-full w-full">
                    <span
                      className="block h-full w-full"
                      style={{ backgroundColor: PARITY_INK }}
                      tabIndex={0}
                      role="img"
                      aria-label={parityLabel}
                    />
                  </Tooltip>
                </div>
              )}
            </div>
          </div>
        )
      })}
      <ValueAxis columns={COLUMNS} min={0} max={100} title={axisTitle} format={(v) => `${Math.round(v)} %`} />
    </div>
  )
}
