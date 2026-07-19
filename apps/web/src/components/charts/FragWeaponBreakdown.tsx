/**
 * FragWeaponBreakdown — breakdown « Frags par arme » en barres horizontales,
 * chaque barre RECOLORÉE par la couleur de sa CLASSE d'arme (fragClassColor) →
 * cohérence visuelle avec le sunburst FragSunburst. Composant PARTAGÉ (Synthesis/
 * Match view/Timeseries/Sessions) — cf. .ai/V7/PLAN_FRAG_DISTRIBUTION_V2.md P1.5.
 *
 * Double encodage : la classe est portée par la couleur, l'arme par le label d'axe,
 * et le tooltip nomme les deux (couleur jamais seule porteuse d'info). Les armes
 * sans classe résolue (registre muet) retombent sur la teinte neutre.
 */
import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { fragClassColor } from '@/lib/accessibility/scales'
import type { SynthesisWeaponKillEntry } from '@/lib/api/types'
import { formatMessage } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

import { ChartCard, type ChartSeries } from './ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from './_utils'

/** Libellés injectés (builder pur, testable sans i18n). */
export interface FragWeaponLabels {
  classLabel: (className: string) => string
  formatValue: (n: number) => string
  killsSuffix: string
}

/** Builder PUR — exporté pour tester l'option ECharts sans monter le React tree. */
// eslint-disable-next-line react-refresh/only-export-components
export function buildFragWeaponBreakdownOption(
  weapons: SynthesisWeaponKillEntry[],
  labels: FragWeaponLabels,
): EChartsCoreOption {
  if (weapons.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  // Barres triées kills desc → afficher la plus grande en HAUT (yAxis inverse via reverse()).
  const ordered = [...weapons].reverse()
  return {
    backgroundColor: CHART_BG,
    grid: { top: 8, bottom: 8, left: 8, right: 80, containLabel: true },
    tooltip: {
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text, fontSize: 11 },
      trigger: 'item',
      formatter: ((p: { name?: string; value?: number; data?: { className?: string } }) => {
        const value = typeof p.value === 'number' ? p.value : 0
        const cls = p.data?.className ? `<br/><span style="opacity:0.7">${escapeHtml(p.data.className)}</span>` : ''
        return `${escapeHtml(p.name ?? '')}<br/><b>${labels.formatValue(value)}</b> ${escapeHtml(labels.killsSuffix)}${cls}`
      }) as unknown as string,
    },
    xAxis: { type: 'value', show: false },
    yAxis: {
      type: 'category',
      data: ordered.map((w) => w.label),
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
      axisTick: { show: false },
      axisLine: { show: false },
    },
    series: [
      {
        type: 'bar',
        barMaxWidth: 20,
        data: ordered.map((w) => ({
          value: w.kills,
          className: w.class ? labels.classLabel(w.class) : undefined,
          itemStyle: { color: fragClassColor(w.class), borderRadius: [0, 3, 3, 0] },
        })),
        label: {
          show: true,
          position: 'right',
          color: tc.axisLabel,
          fontSize: 11,
          formatter: (p: { value: number }) => labels.formatValue(p.value),
        },
      },
    ],
  }
}

export interface FragWeaponBreakdownProps {
  weapons?: SynthesisWeaponKillEntry[]
  title?: string
  height?: number
  fillHeight?: boolean
}

export function FragWeaponBreakdown({ weapons, title, height, fillHeight }: FragWeaponBreakdownProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const numLoc = intlLocale(appLocale)
  const list = weapons ?? []

  const labels: FragWeaponLabels = {
    classLabel: (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale),
    formatValue: (n: number) => n.toLocaleString(numLoc),
    killsSuffix: formatMessage(fragsManifest, 'frags.charts.center_total_label', appLocale).toLowerCase(),
  }

  const buildOption = useCallback(
    (s: ChartSeries<SynthesisWeaponKillEntry>[]) => buildFragWeaponBreakdownOption(s[0]?.datapoints ?? [], labels),
    // labels dérive de appLocale ; on l'inclut plutôt que l'objet (référence neuve à chaque rendu)
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [appLocale],
  )

  // Série VIDE quand aucune arme → ChartCard rend son placeholder (cf. SynthesisWeaponKillsChart).
  const series: ChartSeries<SynthesisWeaponKillEntry>[] = list.length > 0 ? [{ key: 'frag-weapons', datapoints: list }] : []
  const cardTitle = title ?? formatMessage(fragsManifest, 'frags.charts.weapon_breakdown_title', appLocale)
  const emptyMessage = formatMessage(fragsManifest, 'frags.empty.no_data', appLocale)
  const computedHeight = height ?? Math.max(180, list.length * 28 + 16)

  return (
    <ChartCard
      title={cardTitle}
      series={series}
      buildOption={buildOption}
      height={computedHeight}
      emptyMessage={emptyMessage}
      className={fillHeight ? 'flex-1' : ''}
    />
  )
}
