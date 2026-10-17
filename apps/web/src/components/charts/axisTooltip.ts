/**
 * axisTooltip.ts — contenu des infobulles ECharts à déclenchement par axe (source unique).
 *
 * Le rendu par défaut d'ECharts pose tous les défauts qu'une infobulle ne doit pas avoir : les
 * séries sans valeur au point survolé (joueur masqué, graphe qui ne couvre pas ce match)
 * apparaissent en « - », les nombres sortent en notation anglaise (« 1,234.5 »), les décimales
 * ne suivent pas celles du graphe (un bonus de 4/3 d'assistances s'écrit « 1.3333333333333333 »),
 * et l'ordre des lignes est celui du tableau de séries, pas celui qu'on lit sur le graphe.
 *
 * Contrat : un titre (le repère de l'axe, une seule ligne), puis UNE ligne par série qui a une
 * valeur : pastille de sa couleur, nom, valeur alignée à droite. Les nombres suivent la langue de
 * l'application (`tipNumber`). L'ordre est choisi par l'appelant (`order`).
 */
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { escapeHtml } from './_utils'

/** Signe moins typographique, le même que `formatSignedFixed`. */
const MINUS = '−'

const formats = new Map<string, Intl.NumberFormat>()

function numberFormat(locale: string, decimals: number, signed: boolean): Intl.NumberFormat {
  const key = `${locale}|${decimals}|${signed ? 1 : 0}`
  let nf = formats.get(key)
  if (!nf) {
    nf = new Intl.NumberFormat(locale, {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
      ...(signed ? { signDisplay: 'exceptZero' as const } : {}),
    })
    formats.set(key, nf)
  }
  return nf
}

/**
 * Nombre d'infobulle dans la langue de l'application (« 1 234,5 » / « 1,234.5 »), `decimals`
 * décimales fixes. `signed` : « +3,0 » / « −1,2 » (signe moins typographique), zéro sans signe.
 * Non fini : tiret demi-cadratin.
 */
export function tipNumber(value: number, decimals = 0, signed = false): string {
  if (!Number.isFinite(value)) return '—'
  const text = numberFormat(intlLocale(useAppShellStore.getState().locale), decimals, signed).format(value)
  return text.replace(/^-/, MINUS)
}

/** Un paramètre de série tel que l'ECharts à déclenchement par axe le fournit. */
export interface AxisTipParam {
  seriesName?: string
  seriesIndex?: number
  dataIndex?: number
  axisValueLabel?: string
  axisValue?: string | number
  name?: string
  color?: unknown
  value?: unknown
}

/** Une ligne prête à écrire. */
export interface AxisTipRow {
  name: string
  color: string
  value: number
  seriesIndex: number
  param: AxisTipParam
}

export interface AxisTooltipConfig {
  /** Texte de la valeur d'une ligne (déjà au format de la langue : voir `tipNumber`). */
  value: (row: AxisTipRow) => string
  /**
   * Ordre des lignes : `series` (ordre du tableau de séries, légende), `value-desc` (la plus
   * haute d'abord, comme sur un graphe en courbes), ou un tri propre au graphe.
   */
  order?: 'series' | 'value-desc' | ((a: AxisTipRow, b: AxisTipRow) => number)
  /** Titre : le repère de l'axe par défaut (ses retours à la ligne deviennent « · »). */
  title?: (param: AxisTipParam) => string
  /** Ligne de détail sous une ligne (texte brut, estompée) ; `''` ou absent : aucune. */
  detail?: (row: AxisTipRow) => string
  /** Écarte une ligne (série masquée portant une valeur nulle, barre de zéro) ; `null` toujours écarté. */
  skip?: (row: AxisTipRow) => boolean
  /** Pied de l'infobulle (HTML déjà échappé), ex. une définition. */
  footer?: (rows: AxisTipRow[]) => string
}

/** Valeur numérique d'un paramètre : le nombre, ou le dernier terme d'un couple [x, y]. */
function numericValue(v: unknown): number | null {
  if (typeof v === 'number') return Number.isFinite(v) ? v : null
  if (Array.isArray(v) && v.length > 0) return numericValue(v[v.length - 1])
  return null
}

function colorText(c: unknown): string {
  return typeof c === 'string' ? c : ''
}

/** Une ligne d'infobulle : pastille, nom, valeur à droite. */
export function tipRowHtml(name: string, color: string, valueText: string, detail = ''): string {
  const dot = color
    ? `<span style="width:8px;height:8px;border-radius:50%;background:${color};flex:none"></span>`
    : ''
  return (
    `<div style="display:flex;align-items:center;gap:6px;margin-top:3px">${dot}` +
    `<span>${escapeHtml(name)}</span>` +
    `<span style="margin-left:auto;padding-left:14px;font-weight:600">${escapeHtml(valueText)}</span></div>` +
    (detail ? `<div style="opacity:0.7;padding-left:14px">${escapeHtml(detail)}</div>` : '')
  )
}

/** Le titre d'une infobulle : une ligne en gras. */
export function tipTitleHtml(title: string): string {
  return title ? `<div style="font-weight:600">${escapeHtml(title)}</div>` : ''
}

/**
 * Le `formatter` d'une infobulle par axe. Renvoie `''` (pas d'infobulle) quand aucune série n'a
 * de valeur au point survolé.
 */
export function axisTooltipFormatter(config: AxisTooltipConfig): (params: unknown) => string {
  return (params) => {
    const list = (Array.isArray(params) ? params : [params]) as AxisTipParam[]
    const rows: AxisTipRow[] = []
    for (const p of list) {
      const value = numericValue(p.value)
      if (value === null) continue
      const row: AxisTipRow = {
        name: p.seriesName ?? '',
        color: colorText(p.color),
        value,
        seriesIndex: p.seriesIndex ?? rows.length,
        param: p,
      }
      if (config.skip?.(row)) continue
      rows.push(row)
    }
    if (rows.length === 0) return ''
    const order = config.order ?? 'series'
    if (order === 'value-desc') rows.sort((a, b) => b.value - a.value || a.seriesIndex - b.seriesIndex)
    else if (typeof order === 'function') rows.sort(order)
    else rows.sort((a, b) => a.seriesIndex - b.seriesIndex)
    const first = rows[0].param
    const title = config.title
      ? config.title(first)
      : String(first.axisValueLabel ?? first.name ?? '').replace(/\s*\n\s*/g, ' · ')
    return (
      tipTitleHtml(title) +
      rows.map((r) => tipRowHtml(r.name, r.color, config.value(r), config.detail?.(r))).join('') +
      (config.footer?.(rows) ?? '')
    )
  }
}

/** Vrai pour une valeur sous l'axe, y compris le zéro négatif d'une série tracée en négatif. */
function isBelowAxis(v: number): boolean {
  return v < 0 || Object.is(v, -0)
}

/**
 * Ordre de lecture d'infobulle pour des barres empilées, du haut vers le bas du graphe. Dans chaque
 * pile (même `stack`) : les séries positives, la dernière posée (la plus haute) d'abord ; puis les
 * séries négatives, la plus proche de l'axe d'abord. Les piles gardent leur ordre d'apparition.
 */
export function stackTopFirstOrder(
  series: ReadonlyArray<{ name?: unknown; stack?: unknown }>,
): (a: AxisTipRow, b: AxisTipRow) => number {
  const groupOf = new Map<number, number>()
  const groups = new Map<string, number>()
  series.forEach((s, i) => {
    const key = typeof s.stack === 'string' ? s.stack : `sans-pile:${i}`
    if (!groups.has(key)) groups.set(key, groups.size)
    groupOf.set(i, groups.get(key) ?? 0)
  })
  return (a, b) => {
    const ga = groupOf.get(a.seriesIndex) ?? a.seriesIndex
    const gb = groupOf.get(b.seriesIndex) ?? b.seriesIndex
    if (ga !== gb) return ga - gb
    const na = isBelowAxis(a.value)
    const nb = isBelowAxis(b.value)
    if (na !== nb) return na ? 1 : -1
    return na ? a.seriesIndex - b.seriesIndex : b.seriesIndex - a.seriesIndex
  }
}
