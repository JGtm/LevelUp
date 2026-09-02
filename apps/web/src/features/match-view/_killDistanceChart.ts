/**
 * _killDistanceChart — LA PORTÉE DE CHAQUE ARME DE CE MATCH, en un bâton par arme.
 *
 * Le POC du 2026-08-30 (DEC-8) rendait ces nombres en tableau ; l'utilisateur a rouvert la
 * décision le 2026-09-02 : « un bâton pour chaque arme, plus proche / plus loin, et un
 * indicateur sur la moyenne ». Le bâton court de `min_distance_m` à `max_distance_m`, le
 * losange marque `avg_distance_m` — trois valeurs DÉJÀ au contrat
 * (`combat_tab.kill_distance_by_weapon`), rien n'est recalculé ici.
 *
 * LE BÂTON D'UN SEUL FRAG MESURÉ EST UN POINT (min = max = avg) : c'est exact, pas un
 * défaut de rendu — le losange de moyenne reste le témoin visible. Le NOMBRE de frags
 * mesurés est dans le libellé de l'arme (« ×N ») et dans l'infobulle : l'épaisseur du
 * bâton ne le code pas, une largeur ne se lit pas en nombre.
 *
 * TECHNIQUE DU BÂTON FLOTTANT : deux barres empilées — un socle TRANSPARENT de hauteur
 * `min` (silencieux, hors infobulle) puis la barre visible de hauteur `max − min`. C'est le
 * patron ECharts standard d'un intervalle ; un `custom` renderItem ferait la même chose en
 * plus de code. La moyenne est une série `scatter` posée sur les mêmes catégories.
 */
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getGridBase,
  getTooltipBase,
  type EChartsThemeColors,
} from '@/components/charts/_utils'
import type { MatchKillDistanceWeapon } from '@/lib/api/types'

/** Une arme projetée pour le graphe — libellé résolu, trois distances, compte mesuré. */
export interface KillDistanceBar {
  label: string
  kills: number
  min: number
  max: number
  avg: number
}

/**
 * killDistanceBars — la projection des lignes du contrat vers le graphe.
 *
 * L'ordre est celui du backend (frags mesurés décroissants) et il est CONSERVÉ : l'arme la
 * plus utilisée en tête. ECharts posant la première catégorie en BAS d'un axe Y, l'appelant
 * inverse la liste au montage de l'axe — ici on ne trie rien.
 */
export function killDistanceBars(
  weapons: readonly MatchKillDistanceWeapon[],
  locale: string,
): KillDistanceBar[] {
  return weapons.map((w) => ({
    label: (locale === 'en' ? w.label_en : w.label) || w.weapon_key,
    kills: w.measured_kills,
    min: w.min_distance_m,
    max: w.max_distance_m,
    avg: w.avg_distance_m,
  }))
}

export interface KillDistanceOptionInput {
  bars: readonly KillDistanceBar[]
  tc: EChartsThemeColors
  /** Encre du bâton et du losange de moyenne (tokens résolus par l'appelant). */
  rangeColor: string
  avgColor: string
  /** Formate une distance (« 12,4 m ») — la locale vit chez l'appelant (i18n.ts). */
  fmtDistance: (m: number) => string
  /** Libellés d'infobulle, déjà localisés : frags mesurés, min, moyenne, max. */
  labels: { kills: string; min: string; avg: string; max: string }
}

export function buildKillDistanceOption({
  bars,
  tc,
  rangeColor,
  avgColor,
  fmtDistance,
  labels,
}: KillDistanceOptionInput): EChartsCoreOption {
  // Premier du backend = plus utilisé = EN HAUT : l'axe Y d'ECharts empile du bas vers le
  // haut, donc la liste se lit à l'envers au montage.
  const ordered = [...bars].reverse()
  const categories = ordered.map((b) => `${b.label} ×${b.kills}`)
  const axis = getAxisBase(tc)
  return {
    backgroundColor: CHART_BG,
    grid: getGridBase({ bottom: 24, left: 8 }),
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const list = Array.isArray(params) ? params : [params]
        const first = list[0] as { dataIndex?: number } | undefined
        const bar = first?.dataIndex != null ? ordered[first.dataIndex] : undefined
        if (!bar) return ''
        return [
          `<b>${escapeHtml(bar.label)}</b> — ${labels.kills} : ${bar.kills}`,
          `${labels.min} : ${escapeHtml(fmtDistance(bar.min))}`,
          `${labels.avg} : ${escapeHtml(fmtDistance(bar.avg))}`,
          `${labels.max} : ${escapeHtml(fmtDistance(bar.max))}`,
        ].join('<br/>')
      },
    },
    xAxis: {
      type: 'value',
      ...axis,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => fmtDistance(v) },
    },
    yAxis: { type: 'category', data: categories, ...axis, splitLine: { show: false } },
    series: [
      {
        // Le socle transparent : il PORTE le bâton à `min`, il ne se lit pas.
        type: 'bar',
        stack: 'range',
        silent: true,
        itemStyle: { color: 'transparent' },
        emphasis: { disabled: true },
        data: ordered.map((b) => b.min),
      },
      {
        type: 'bar',
        stack: 'range',
        barWidth: 8,
        itemStyle: { color: rangeColor, borderRadius: 4 },
        data: ordered.map((b) => b.max - b.min),
      },
      {
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 9,
        itemStyle: { color: avgColor },
        data: ordered.map((b, i) => [b.avg, i]),
      },
    ],
  }
}
