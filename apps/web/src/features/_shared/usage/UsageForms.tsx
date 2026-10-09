/**
 * UsageForms — LE RENDU DE LA JAUGE « écart à la parité » et de sa grille (carte « Appui reçu » de
 * Sessions). Les grilles alignées, elles, passent par la primitive partagée
 * `components/charts/ValueGrid`, dont la colonne des libellés et la gouttière s'alignent sur
 * celles d'ici (`LABEL_WIDTH`, `COLUMN_GAP`).
 *
 * DOM ET CSS, PAS ECHARTS — le même choix mesuré que `ValueGrid` : ces formes sont des
 * problèmes de MISE EN PAGE (alignement de rails), sans zoom ni animation. Chaque colonne porte
 * son AXE GRADUÉ et chaque rail est focusable avec son texte en `aria-label`.
 *
 * COULEURS — jetons sémantiques uniquement, tous dans `usageInks.ts` : la tranche = `team-ally`,
 * le trait = `warning` (jeton distinct, jamais une teinte de donnée).
 *
 * Aucun calcul ici : tout vient du modèle (`usageGaugeModel.ts`).
 */
import { Fragment, type CSSProperties } from 'react'

import { Tooltip } from '@/components/ui/tooltip'

import type { UsageGaugeModel, UsageGaugeRowModel } from './usageGaugeModel'
import { ALLY_INK, PARITY_INK } from './usageInks'

/**
 * Largeur de la colonne des libellés de grandeur (alignée sur ValueGrid), partagée avec les
 * grilles voisines qui s'alignent sur elle (CLAUDE.md n°6).
 */
export const LABEL_WIDTH = 152
/**
 * Gouttière entre colonnes (alignée sur ValueGrid). Les rails partent de zéro
 * (`minmax(0, 1fr)`) : aucune largeur mini de colonne, aucun défilement horizontal.
 */
export const COLUMN_GAP = 14

/** Les mêmes largeurs en COLONNE DIVISÉE (drawer de comparaison, `dense`). */
const DENSE_LABEL_WIDTH = 104
const DENSE_COLUMN_GAP = 8

function clampPct(v: number): number {
  return Math.max(0, Math.min(100, v))
}

/** UsageGaugeColumn — UNE COLONNE de la grille de jauges : son en-tête. */
export interface UsageGaugeColumn {
  header: string
}

/**
 * UsageGauge — DEUX cellules de grille (rail, puis texte), jamais un flex local : le rail
 * doit avoir LA MÊME largeur sur toutes les lignes d'une colonne pour que l'axe gradué du
 * pied mesure vraiment les rails qu'il borde. Le compte brut n'est pas écrit dans la cellule :
 * il est dans l'infobulle du rail.
 */
function UsageGauge({ gauge }: { gauge: UsageGaugeModel }) {
  return (
    <>
      <Tooltip content={gauge.tooltip} className="w-full">
        <div
          className="relative h-[11px] w-full min-w-[60px] bg-muted"
          tabIndex={0}
          role="img"
          aria-label={gauge.tooltip}
        >
          {gauge.valuePct != null && (
            <div
              data-outcome-fill=""
              className="absolute left-0 top-0 h-full"
              style={{ width: `${clampPct(gauge.valuePct)}%`, backgroundColor: ALLY_INK }}
            />
          )}
          {gauge.parityPct != null && (
            <div
              data-gauge-parity=""
              className="absolute top-[-2px] h-[15px] w-[2px]"
              style={{ left: `calc(${clampPct(gauge.parityPct)}% - 1px)`, backgroundColor: PARITY_INK }}
            />
          )}
        </div>
      </Tooltip>
      <span className="whitespace-nowrap text-right text-3xs tabular-nums text-foreground">
        {gauge.valueText}
      </span>
    </>
  )
}

/** L'axe gradué 0 · 50 · 100 % d'une colonne de jauges. */
function GaugeAxis() {
  return (
    <div className="relative mt-1 h-[15px] border-t border-border text-3xs text-muted-foreground tabular-nums">
      <span className="absolute left-0 top-0.5">0</span>
      <span className="absolute left-1/2 top-0.5 -translate-x-1/2">50</span>
      <span className="absolute right-0 top-0.5">100 %</span>
    </div>
  )
}

/**
 * UsageGaugeGrid — les jauges de parts : une ligne par grandeur, une colonne par jauge (en-têtes
 * nommés par l'appelant), un axe gradué par colonne. `dense` ne change que les largeurs.
 */
export function UsageGaugeGrid({
  rows,
  columns,
  dense = false,
}: {
  rows: UsageGaugeRowModel[]
  columns: readonly UsageGaugeColumn[]
  /** Colonne divisée : mêmes colonnes, rails et libellés plus étroits. */
  dense?: boolean
}) {
  if (rows.length === 0) return null
  const labelWidth = dense ? DENSE_LABEL_WIDTH : LABEL_WIDTH
  const columnGap = dense ? DENSE_COLUMN_GAP : COLUMN_GAP

  // Chaque colonne de jauge = DEUX sous-colonnes : le rail (élastique, borné) puis le texte
  // (à la largeur du plus long de la colonne). Ainsi tous les rails d'une colonne sont de
  // même largeur, et l'axe gradué du pied (posé dans la sous-colonne rail seulement) mesure
  // exactement ce qu'il borde.
  const gridStyle: CSSProperties = {
    gridTemplateColumns: `${labelWidth}px repeat(${columns.length}, minmax(0, 1fr) max-content)`,
    columnGap,
  }
  return (
    <div className="min-w-0">
      <div className="grid items-center gap-y-[6px]" style={gridStyle}>
        <div aria-hidden="true" />
        {columns.map((column) => (
          <div
            key={column.header}
            // `overflow-hidden` + retour à la ligne en dense : à demi-largeur, un en-tête
            // `nowrap` débordait sur celui de la colonne suivante.
            className={`mb-1 flex items-center justify-between gap-2 overflow-hidden border-b border-border pb-1.5 text-3xs font-semibold uppercase tracking-wider ${
              dense ? 'whitespace-normal leading-tight' : 'whitespace-nowrap'
            }`}
            style={{ gridColumn: 'span 2' }}
          >
            <span title={column.header}>{column.header}</span>
          </div>
        ))}
        {rows.map((row) => (
          <Fragment key={row.key}>
            <div className="overflow-hidden whitespace-nowrap text-xs" title={row.label}>
              <span className="truncate">{row.label}</span>
            </div>
            {columns.map((column, i) => (
              <UsageGauge key={`${row.key}-${column.header}`} gauge={row.gauges[i]} />
            ))}
          </Fragment>
        ))}
        <div aria-hidden="true" />
        {columns.map((column) => (
          <Fragment key={`axis-${column.header}`}>
            <GaugeAxis />
            <div aria-hidden="true" />
          </Fragment>
        ))}
      </div>
    </div>
  )
}
