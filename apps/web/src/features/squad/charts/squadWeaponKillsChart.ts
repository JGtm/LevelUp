/**
 * squadWeaponKillsChart — barres horizontales groupées, une barre par joueur et par ligne :
 * « Outils de destruction » (frags par outil) et « Mécaniques de frag » (Halo 5).
 *
 * Spec d'origine : .ai/charts_specs/teammates/09_weapon_kills_bar_chart.yaml ; forme revue par
 * le lot L2 du plan PLAN_EMPRISE_ET_CARTES_DEPLACEES_2026-09-26 (maquette C3EW, colonne
 * « Proposition ») pour les Outils de destruction :
 *   - le COMPTE de frags au bout de CHAQUE barre non nulle (`valueLabel: 'count'`, défaut) ;
 *     la part du total du joueur passe en infobulle ;
 *   - une pastille à la couleur de la CLASSE devant le nom de la ligne quand elle en porte
 *     une (`cls`), pour la relier à la « Répartition des frags » ;
 *   - la légende des joueurs vit HORS canvas (ChartCard `legend`), posée par l'appelant.
 *
 * `valueLabel: 'share'` garde la lecture historique (part du total du joueur écrite au bout
 * des barres assez larges) pour « Mécaniques de frag », hors du périmètre du lot.
 *
 *   - Y : lignes, la PREMIÈRE en BAS (l'appelant ordonne) ; bandes zébrées par ligne.
 *   - X : frags (caché — les valeurs sont écrites au bout des barres).
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import { fragClassColor } from '@/lib/accessibility/scales'

/** Part (0..1) du total du joueur sous laquelle la part écrite est masquée (mode `share`). */
const MIN_LABEL_SHARE = 0.05

/** Une ligne du graphe : un outil (ou une mécanique) et les frags de chaque joueur. */
export interface SquadBarRow {
  key: string
  label: string
  /** Classe de frag (pastille devant le nom) ; absente = pas de pastille. */
  cls?: string
  killsByPlayer: Record<string, number>
  total: number
}

/** Données du graphe : joueurs (ordre canonique) et lignes, la première en bas. */
export interface SquadBarRows {
  players: string[]
  rows: SquadBarRow[]
}

interface TooltipParam {
  seriesName?: string
  value?: number | null
  marker?: string
  dataIndex?: number
}

export interface SquadWeaponKillsOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
  /** Ce qui s'écrit au bout des barres : le compte (défaut) ou la part du joueur. */
  valueLabel?: 'count' | 'share'
  /** Texte d'infobulle d'une barre (compte + part arrondie en %) ; défaut « <b>n</b> (p %) ». */
  valueText?: (kills: number, sharePct: number) => string
  /**
   * Dénominateur de la part, par joueur (vue compacte de Sessions : TOUS ses frags, quand le graphe
   * n'en montre que les premiers outils) ; défaut : la somme de ses frags sur les lignes affichées.
   */
  shareTotals?: Record<string, number>
  /** Part sous laquelle la part écrite est masquée (mode `share`) ; défaut 5 %. */
  minLabelShare?: number
  /**
   * Graphe d'UN SEUL joueur (Sessions, Vue match) : chaque barre à la couleur de la CLASSE de sa ligne
   * (`fragClassColor`, la couleur de la Répartition des frags) au lieu de l'encre du joueur, et
   * l'infobulle sans le nom du joueur — redondant quand la page n'en montre qu'un.
   */
  soloByClass?: boolean
}

/** Clé de style riche ECharts d'une classe (alphanumérique + souligné seulement). */
const swatchKey = (cls: string) => `c_${cls.replace(/[^A-Za-z0-9_]/g, '_')}`

/** Un nom de ligne ne doit pas casser la syntaxe du texte riche ECharts (`{style|texte}`). */
const richSafe = (s: string) => s.replace(/[{}|]/g, ' ')

function axisLabelStyle(rows: SquadBarRow[], textColor: string) {
  const rich: Record<string, object> = {
    name: { color: textColor, padding: [0, 0, 0, 6] },
  }
  for (const r of rows) {
    if (r.cls && !rich[swatchKey(r.cls)]) {
      rich[swatchKey(r.cls)] = { width: 9, height: 9, borderRadius: 2, backgroundColor: fragClassColor(r.cls) }
    }
  }
  return {
    rich,
    formatter: (_value: string, index: number) => {
      const r = rows[index]
      if (!r) return ''
      return r.cls ? `{${swatchKey(r.cls)}|}{name|${richSafe(r.label)}}` : richSafe(r.label)
    },
  }
}

export function buildSquadWeaponKillsOption(
  data: SquadBarRows | null | undefined,
  opts: SquadWeaponKillsOpts,
): EChartsCoreOption {
  const rows = data?.rows ?? []
  const players = data?.players ?? []
  if (!data || rows.length === 0 || players.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const mode = opts.valueLabel ?? 'count'
  const valueText = opts.valueText ?? ((v: number, pct: number) => `<b>${v}</b> (${pct} %)`)

  // Total du joueur = somme de ses frags sur TOUTES les lignes (dénominateur de la part), sauf
  // dénominateur fourni par l'appelant.
  const playerTotals = new Map<string, number>(
    players.map((p) => [p, opts.shareTotals?.[p] ?? rows.reduce((s, r) => s + (r.killsByPlayer[p] ?? 0), 0)]),
  )
  const minShare = opts.minLabelShare ?? MIN_LABEL_SHARE
  const shareOf = (player: string, value: number): number => {
    const total = playerTotals.get(player) ?? 0
    return total > 0 ? value / total : 0
  }

  const series = players.map((player) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const values = rows.map((r) => {
      const v = r.killsByPlayer[player] ?? 0
      return opts.soloByClass && r.cls ? { value: v, itemStyle: { color: fragClassColor(r.cls) } } : v
    })
    return {
      name: player,
      type: 'bar' as const,
      data: values,
      itemStyle: { color },
      label: {
        show: true,
        position: 'right' as const,
        color: tc.text,
        fontSize: 11,
        fontWeight: 'bold' as const,
        formatter: (p: { value: unknown }) => {
          const v = typeof p.value === 'number' ? p.value : 0
          if (v <= 0) return ''
          if (mode === 'count') return String(v)
          const share = shareOf(player, v)
          return share < minShare ? '' : `${Math.round(share * 100)} %`
        },
      },
      barCategoryGap: '20%',
      barGap: '8%',
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 8, left: 8, right: 40, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (raw: unknown) => {
        const params = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
        if (params.length === 0) return ''
        const dataIndex = params[0]?.dataIndex ?? 0
        const header = escapeHtml(rows[dataIndex]?.label ?? '')
        const lines = params
          .filter((p) => typeof p.value === 'number' && p.value > 0)
          .map((p) => {
            const v = p.value as number
            const player = p.seriesName ?? ''
            const pct = Math.round(shareOf(player, v) * 100)
            const who = opts.soloByClass ? '' : `${escapeHtml(player)}: `
            return `${p.marker ?? ''}${who}${valueText(v, pct)}`
          })
        if (lines.length === 0) return ''
        return `<div style="margin-bottom:4px;font-weight:600">${header}</div>${lines.join('<br/>')}`
      },
    },
    // Légende des joueurs HORS canvas (ChartCard `legend`) quand l'appelant la pose.
    legend: { show: false },
    xAxis: { ...axis, type: 'value', show: false },
    yAxis: {
      ...axis,
      type: 'category',
      data: rows.map((r) => r.key),
      axisLabel: { ...axis.axisLabel, ...axisLabelStyle(rows, tc.text) },
      splitArea: { show: true, areaStyle: { color: [tc.splitAreaA, tc.splitAreaB] } },
    },
    series,
  }
}
