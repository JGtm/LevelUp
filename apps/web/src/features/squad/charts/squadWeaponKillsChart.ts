/**
 * squadWeaponKillsChart — teammates.09 : kills par arme, comparatif multi-joueurs.
 *
 * Spec : .ai/charts_specs/teammates/09_weapon_kills_bar_chart.yaml
 *
 *   - Y axis : armes triées ASC par TotalSquad (peu utilisées en haut → réflexe
 *     visuel sur les pics rares en premier).
 *   - X axis : kills (caché — les valeurs sont dans le label de barre).
 *   - 1 série bar par joueur en `barmode: 'group'` (côte à côte).
 *   - Splitarea zebra (alternance bandes claires/transparentes) sur le yAxis
 *     pour aider à suivre la lane d'une arme.
 *   - Pas de légende (la pill+combobox de la page sert de mapping joueur→couleur).
 *   - Label = % du total du joueur sur les barres ASSEZ LARGES (I6, V7.1) — longueurs de
 *     barres INCHANGÉES (valeurs brutes), seul le LABEL affiché change. Sous
 *     MIN_LABEL_SHARE, le label est masqué (bouillie visuelle sur les tout petits
 *     segments) : ces valeurs restent lisibles au survol (tooltip enrichi valeur + %).
 *     Décision utilisateur (plan V7.1 backlog, 2026-07-24).
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadWeaponKills } from '@/lib/api/types'

/** Part (0..1) du total du joueur sous laquelle le label % est masqué sur le segment. */
const MIN_LABEL_SHARE = 0.05

interface TooltipParam {
  seriesName?: string
  value?: number | null
  marker?: string
  dataIndex?: number
}

export interface SquadWeaponKillsOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors). */
  colorByPlayer: Record<string, string>
}

export function buildSquadWeaponKillsOption(
  data: SquadWeaponKills | null | undefined,
  opts: SquadWeaponKillsOpts,
): EChartsCoreOption {
  const allBars = data?.bars ?? []
  const players = data?.players ?? []
  if (!data || allBars.length === 0 || players.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // Masque les armes inconnues : label vide (→ fallback "weapon_<id>") ou label
  // brut de la forme "weapon_-123456". On ne garde que les armes réellement
  // nommées (demande user).
  const bars = allBars.filter((b) => {
    const label = (b.label ?? '').trim()
    return label !== '' && !/^weapon_-?\d+$/i.test(label)
  })
  if (bars.length === 0) return { backgroundColor: CHART_BG }

  // yAxis = labels d'armes (ASC par TotalSquad côté backend).
  const yLabels = bars.map((b) => b.label || `weapon_${b.weapon_id}`)

  // Total du joueur = somme de ses kills sur TOUTES les armes affichées (dénominateur du %
  // affiché sur ses segments, I6). Calculé UNE fois, partagé par le label et le tooltip.
  const playerTotals = new Map<string, number>(
    players.map((p) => [p, bars.reduce((s, b) => s + (b.kills_by_player[p] ?? 0), 0)]),
  )
  const shareOf = (player: string, value: number): number => {
    const total = playerTotals.get(player) ?? 0
    return total > 0 ? value / total : 0
  }

  // 1 série bar (horizontale, group) par joueur, valeurs alignées sur yLabels.
  const series = players.map((player) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const values = bars.map((b) => b.kills_by_player[player] ?? 0)
    const shares = values.map((v) => shareOf(player, v))
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
        // Longueur de barre INCHANGÉE (valeur brute) — seul le LABEL affiché devient le %
        // du total du joueur (I6). Masqué si valeur nulle OU segment trop étroit
        // (< MIN_LABEL_SHARE) pour éviter la bouillie visuelle ; ces petits segments
        // restent lisibles au tooltip (valeur + % enrichis ci-dessous).
        formatter: (p: { value: unknown; dataIndex: number }) => {
          const v = typeof p.value === 'number' ? p.value : 0
          if (v <= 0) return ''
          const share = shares[p.dataIndex] ?? 0
          if (share < MIN_LABEL_SHARE) return ''
          return `${Math.round(share * 100)} %`
        },
      },
      // Bandes de catégorie resserrées (35 %→20 %) → barres un peu plus épaisses (demande user).
      barCategoryGap: '20%',
      barGap: '8%',
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 16, left: 8, right: 80, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      // Enrichi valeur + % du total du joueur (I6) — les petits segments (label masqué sur
      // la barre) restent lisibles ici.
      formatter: (raw: unknown) => {
        const params = (Array.isArray(raw) ? raw : [raw]) as TooltipParam[]
        if (params.length === 0) return ''
        const dataIndex = params[0]?.dataIndex ?? 0
        const header = escapeHtml(yLabels[dataIndex] ?? '')
        const lines = params
          .filter((p) => typeof p.value === 'number' && p.value > 0)
          .map((p) => {
            const v = p.value as number
            const player = p.seriesName ?? ''
            const share = shareOf(player, v)
            return `${p.marker ?? ''}${escapeHtml(player)}: <b>${v}</b> (${Math.round(share * 100)} %)`
          })
        if (lines.length === 0) return ''
        return `<div style="margin-bottom:4px;font-weight:600">${header}</div>${lines.join('<br/>')}`
      },
    },
    // Pas de legend (cf. spec) — la pill et le multiselect identifient déjà
    // chaque joueur par sa couleur.
    legend: { show: false },
    xAxis: {
      ...axis,
      type: 'value',
      show: false, // valeurs dans le label de barre
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      // Bandes zebra pour distinguer les lanes (cf. spec).
      splitArea: {
        show: true,
        areaStyle: { color: [tc.splitAreaA, tc.splitAreaB] },
      },
    },
    series,
  }
}
