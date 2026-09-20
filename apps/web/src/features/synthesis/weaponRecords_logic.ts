/**
 * weaponRecords_logic — la GÉOMÉTRIE de « la règle » des records de distance, sans React.
 *
 * Plan : .ai/V7.5/PLAN_RECORDS_DISTANCE_2026-09-20.md (rendu A de la maquette
 * MAQUETTE_REGLE_RECORDS_2026-09-20.html). Une ligne graduée du contact vers la longue portée,
 * un losange par arme à son record, et les libellés ÉTAGÉS au-dessus pour qu'aucun n'en
 * recouvre un autre : c'est ce dernier point qui justifie un module pur et testé — un
 * placement glouton se prouve sur des boîtes, pas en regardant un écran.
 *
 * L'ORDRE DU BACKEND EST CONSERVÉ (record croissant, cf. `analysis.WeaponDistanceRecords`) :
 * rien n'est retrié ici. Le placement glouton en dépend (il parcourt de gauche à droite), et
 * deux tris du même fait divergeraient au premier changement de doctrine.
 *
 * Aucun libellé produit ici ne dit « portée de l'arme » : c'est un usage mesuré du joueur.
 */
import type { WeaponDistanceRecordRow } from '@/lib/api/types'
import { intlLocale } from '@/lib/formatters'
import type { ManifestLocale } from '@/lib/i18n/format'

/** Marge horizontale de la règle (les losanges extrêmes ne touchent pas le bord). */
export const RULER_PADDING_X = 16
/** Hauteur d'un rang de libellé : deux lignes de texte (nom + record) à 11 px. */
export const RULER_LABEL_ROW_PX = 26
/** Marge haute au-dessus du premier rang. */
export const RULER_TOP_PX = 8
/** Écart entre la ligne de base du rang le plus bas et l'axe : le record (« 1,3 m », second
 *  texte du libellé, 11 px sous la ligne de base) doit rester au-dessus des losanges
 *  (6 px au-dessus de l'axe) — constaté trop serré à 14 px sur données réelles. */
export const RULER_AXIS_GAP_PX = 24
/** Hauteur réservée sous l'axe pour les graduations. */
export const RULER_AXIS_LABEL_PX = 28
/** Écart minimal entre deux libellés d'un même rang. */
export const RULER_LABEL_MIN_GAP_PX = 14
/** Largeur estimée d'un caractère à 11 px — une estimation suffit à étager, le texte SVG
 *  réel n'est pas mesurable avant le rendu et une erreur d'un ou deux pixels ne fait pas se
 *  chevaucher deux libellés séparés de 14 px. */
export const RULER_CHAR_PX = 6.8
/** Pas des graduations, en mètres. */
export const RULER_TICK_STEP_M = 10
/** Largeur de repli quand le conteneur n'est pas encore mesuré (ou sans ResizeObserver). */
export const RULER_FALLBACK_WIDTH_PX = 960

/** Nom d'affichage d'une arme : libellé de la locale, sinon la clé (jamais un nom inventé). */
export function resolveRecordLabel(
  row: { weapon_key: string; label?: string | null; label_en?: string | null },
  locale: ManifestLocale,
): string {
  const label = locale === 'en' ? row.label_en : row.label
  return label && label.trim() !== '' ? label : row.weapon_key
}

/** « 96,4 m » / « 96.4 m » — une décimale, locale-aware. */
export function formatMeters(m: number, locale: ManifestLocale): string {
  const f = new Intl.NumberFormat(intlLocale(locale), {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  })
  return `${f.format(m)} m`
}

/**
 * Borne haute de l'axe : le record le plus lointain arrondi à la dizaine supérieure, jamais
 * en dessous d'un pas (une règle de 0 à 10 m reste une règle).
 */
export function rulerMaxMeters(rows: readonly { record_m: number }[]): number {
  const max = rows.reduce((acc, r) => Math.max(acc, r.record_m), 0)
  return Math.max(RULER_TICK_STEP_M, Math.ceil(max / RULER_TICK_STEP_M) * RULER_TICK_STEP_M)
}

/** Écart entre le bas de la zone des graduations et la ligne de base du premier rang bas. */
export const RULER_BOTTOM_OFFSET_PX = 10
/** Marge basse sous le dernier rang bas. */
export const RULER_BOTTOM_PAD_PX = 6

/** Le côté de l'axe où un libellé est posé. */
export type RulerSide = 'top' | 'bottom'

/** Premier rang d'un côté où une boîte tient sans toucher la précédente (écart `minGap`). */
function firstFreeRank(ends: number[], x0: number, minGap: number): number {
  let rank = 0
  while (ends[rank] != null && ends[rank] + minGap > x0) rank += 1
  return rank
}

/**
 * staggerLabels — le placement GLOUTON sur UN côté : chaque boîte prend le premier rang où la
 * boîte précédente de ce rang ne la touche pas (écart `minGap`). Les boîtes arrivent triées
 * par `x0` croissant ; le rang 0 est le plus proche de l'axe.
 */
export function staggerLabels(
  boxes: readonly { x0: number; x1: number }[],
  minGap: number = RULER_LABEL_MIN_GAP_PX,
): number[] {
  const ends: number[] = []
  return boxes.map((b) => {
    const rank = firstFreeRank(ends, b.x0, minGap)
    ends[rank] = b.x1
    return rank
  })
}

/**
 * staggerLabelsTwoSided — le même glouton, sur les DEUX côtés de l'axe (retour utilisateur du
 * 2026-09-20 : tout au-dessus était trop dense). Chaque boîte va du côté où elle tient au
 * rang le plus bas ; à égalité, du côté qui porte le moins de libellés, puis en haut. Deux
 * files indépendantes : un libellé du haut ne contraint jamais un libellé du bas.
 */
export function staggerLabelsTwoSided(
  boxes: readonly { x0: number; x1: number }[],
  minGap: number = RULER_LABEL_MIN_GAP_PX,
): { side: RulerSide; rank: number }[] {
  const ends: Record<RulerSide, number[]> = { top: [], bottom: [] }
  const count: Record<RulerSide, number> = { top: 0, bottom: 0 }
  return boxes.map((b) => {
    const top = firstFreeRank(ends.top, b.x0, minGap)
    const bottom = firstFreeRank(ends.bottom, b.x0, minGap)
    let side: RulerSide = 'top'
    if (bottom < top || (bottom === top && count.bottom < count.top)) side = 'bottom'
    const rank = side === 'top' ? top : bottom
    ends[side][rank] = b.x1
    count[side] += 1
    return { side, rank }
  })
}

export interface RulerItem {
  row: WeaponDistanceRecordRow
  /** Nom affiché (locale, repli clé). */
  label: string
  /** « 52,7 m ». */
  valueText: string
  /** Abscisse du losange, en pixels de la vue. */
  cx: number
  /** Centre du libellé, en pixels — recentré si la boîte sortait de la vue. */
  labelX: number
  /** Côté de l'axe où le libellé est posé. */
  side: RulerSide
  /** Rang d'étagement, 0 = le plus proche de l'axe. */
  rank: number
}

export interface RulerLayout {
  width: number
  height: number
  axisY: number
  /** Nombre de rangs au-dessus et en dessous de l'axe. */
  topRows: number
  bottomRows: number
  maxMeters: number
  ticks: number[]
  items: RulerItem[]
  x: (meters: number) => number
}

/**
 * weaponRecordsLayout — toute la géométrie de la règle pour une largeur donnée.
 *
 * Un libellé dont la boîte sort de la vue est recentré à l'intérieur ; le losange, lui, ne
 * bouge jamais (le trait de rappel relie les deux). La hauteur totale dépend du nombre de
 * rangs qu'exige l'étagement, de chaque côté de l'axe : elle est CALCULÉE, pas fixée — une
 * règle de 30 armes serrées à gauche demande plus de rangs qu'une règle de 6. Sous l'axe, la
 * zone des graduations est réservée avant le premier rang bas.
 */
export function weaponRecordsLayout(
  rows: readonly WeaponDistanceRecordRow[],
  locale: ManifestLocale,
  widthPx: number,
): RulerLayout {
  const width = Math.max(200, widthPx)
  const maxMeters = rulerMaxMeters(rows)
  const x = (m: number) => RULER_PADDING_X + (m / maxMeters) * (width - 2 * RULER_PADDING_X)

  const prepared = rows.map((row) => {
    const label = resolveRecordLabel(row, locale)
    const valueText = formatMeters(row.record_m, locale)
    const tw = Math.max(label.length, valueText.length) * RULER_CHAR_PX + 12
    const cx = x(row.record_m)
    let x0 = Math.max(0, cx - tw / 2)
    let x1 = x0 + tw
    if (x1 > width) {
      x1 = width
      x0 = Math.max(0, width - tw)
    }
    return { row, label, valueText, cx, x0, x1 }
  })
  const placed = staggerLabelsTwoSided(prepared)
  const rowsOf = (side: RulerSide) =>
    placed.reduce((acc, p) => (p.side === side ? Math.max(acc, p.rank + 1) : acc), 0)
  const topRows = rowsOf('top')
  const bottomRows = rowsOf('bottom')
  const axisY = RULER_TOP_PX + topRows * RULER_LABEL_ROW_PX + RULER_AXIS_GAP_PX
  const bottomStart = axisY + RULER_AXIS_LABEL_PX
  const height =
    bottomRows > 0
      ? bottomStart + RULER_BOTTOM_OFFSET_PX + bottomRows * RULER_LABEL_ROW_PX + RULER_BOTTOM_PAD_PX
      : bottomStart

  const ticks: number[] = []
  for (let m = 0; m <= maxMeters; m += RULER_TICK_STEP_M) ticks.push(m)

  return {
    width,
    height,
    axisY,
    topRows,
    bottomRows,
    maxMeters,
    ticks,
    x,
    items: prepared.map((p, i) => ({
      row: p.row,
      label: p.label,
      valueText: p.valueText,
      cx: p.cx,
      labelX: (p.x0 + p.x1) / 2,
      side: placed[i].side,
      rank: placed[i].rank,
    })),
  }
}

/**
 * Ordonnée de la ligne de base du NOM d'un libellé (le record se pose 11 px plus bas). En
 * haut, le rang 0 est juste au-dessus de l'axe ; en bas, le rang 0 est juste sous la zone
 * des graduations.
 */
export function labelBaselineY(
  layout: Pick<RulerLayout, 'axisY'>,
  item: Pick<RulerItem, 'side' | 'rank'>,
): number {
  if (item.side === 'top') {
    return layout.axisY - RULER_AXIS_GAP_PX - item.rank * RULER_LABEL_ROW_PX
  }
  return layout.axisY + RULER_AXIS_LABEL_PX + RULER_BOTTOM_OFFSET_PX + item.rank * RULER_LABEL_ROW_PX
}
