/**
 * squadWeaponAccuracyBarsChart — « Précision par rôle » comparatif multi-joueurs (Escouade).
 *
 * Pendant précision de teammates.09 (kills par arme) : MÊME grammaire visuelle — barres
 * horizontales groupées, 1 barre par joueur et par rôle d'arme, longueur = précision %
 * (0..100, axe borné pour que la longueur soit honnête : une barre à 50 % fait la moitié).
 * Les ~30 armes sont agrégées PAR RÔLE côté serveur (precision/automatic/sniper/…) ; précision
 * NATIVE Halo 5 (weapon_accuracy) ; grenades/mêlée/capacités déjà exclues côté serveur.
 *
 *   - Y axis : rôles d'arme (ordre backend, ASC par volume de tirs).
 *   - X axis : précision 0..100 % (borné).
 *   - Couleur = joueur (getSquadPlayerColors, cohérent avec les kills).
 *   - Joueur sans tir sur le rôle = valeur absente (null → pas de barre), distinct d'un 0 %.
 *   - Tooltip : rôle + précision % + tirs par joueur.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadWeaponAccuracy } from '@/lib/api/types'

export interface SquadWeaponAccuracyBarsOpts {
  /** gamertag → couleur hex (cf. getSquadPlayerColors, mêmes couleurs que les kills). */
  colorByPlayer: Record<string, string>
  /** Libellé localisé d'un rôle d'arme (manifeste `frags`, clé frags.role.<role>). */
  roleLabel: (role: string) => string
  /** Libellé « Tirs » (contexte tooltip). */
  shotsLabel: string
}

export function buildSquadWeaponAccuracyBarsOption(
  data: SquadWeaponAccuracy | null | undefined,
  opts: SquadWeaponAccuracyBarsOpts,
): EChartsCoreOption {
  const allBars = data?.bars ?? []
  const players = data?.players ?? []
  if (!data || allBars.length === 0 || players.length === 0) {
    return { backgroundColor: CHART_BG }
  }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)

  // Les rôles sont toujours nommés côté serveur ; on ignore juste un rôle vide s'il arrivait.
  const bars = allBars.filter((b) => (b.role ?? '').trim() !== '')
  if (bars.length === 0) return { backgroundColor: CHART_BG }

  const yLabels = bars.map((b) => opts.roleLabel(b.role))

  // 1 série bar (horizontale, group) par joueur : précision 0..100 % alignée sur yLabels.
  const series = players.map((player) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const values = bars.map((b) => {
      const acc = b.accuracy_by_player[player]
      // Absent = pas de tir sur l'arme → null (pas de barre), distinct d'un vrai 0 %.
      return acc === undefined ? null : Math.round(acc * 1000) / 10
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
        formatter: (p: { value: unknown }) => (typeof p.value === 'number' ? `${p.value} %` : ''),
      },
      barCategoryGap: '35%',
      barGap: '8%',
    }
  })

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 24, left: 8, right: 80, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = (Array.isArray(params) ? params : [params]) as Array<{
          axisValue?: string
          seriesName: string
          value: unknown
          marker: string
          dataIndex: number
        }>
        const lines = arr
          .filter((p) => typeof p.value === 'number')
          .map((p) => {
            const shots = bars[p.dataIndex]?.shots_fired_by_player?.[p.seriesName] ?? 0
            return `${p.marker} ${escapeHtml(p.seriesName)} : <b>${p.value} %</b> (${shots} ${escapeHtml(opts.shotsLabel)})`
          })
        if (lines.length === 0) return ''
        return `${escapeHtml(arr[0].axisValue ?? '')}<br/>${lines.join('<br/>')}`
      },
    },
    // Pas de legend — la pill et le multiselect identifient déjà chaque joueur par sa couleur.
    legend: { show: false },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: 100,
      axisLabel: { color: tc.axisLabel, fontSize: 10, formatter: '{value} %' },
    },
    yAxis: {
      ...axis,
      type: 'category',
      data: yLabels,
      splitArea: {
        show: true,
        areaStyle: { color: [tc.splitAreaA, tc.splitAreaB] },
      },
    },
    series,
  }
}
