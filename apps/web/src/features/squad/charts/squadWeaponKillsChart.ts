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
 *   - Label `{value}` sur chaque barre (vide si 0), couleur joueur.
 */
import type { EChartsCoreOption } from 'echarts/core'
import {
  CHART_BG,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
} from '@/components/charts/_utils'
import type { SquadWeaponKills } from '@/lib/api/types'

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

  // 1 série bar (horizontale, group) par joueur, valeurs alignées sur yLabels.
  const series = players.map((player) => {
    const color = opts.colorByPlayer[player] ?? '#888' // color-allow: gris structurel pour joueur sans couleur attribuée
    const values = bars.map((b) => b.kills_by_player[player] ?? 0)
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
        // Le label vide masque la valeur 0 (cohérent avec la spec : `text` "" si 0).
        formatter: (p: { value: unknown }) => {
          const v = typeof p.value === 'number' ? p.value : 0
          return v > 0 ? `${v}` : ''
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
