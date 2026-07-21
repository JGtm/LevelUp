import { useCallback } from 'react'
import type { EChartsCoreOption } from 'echarts/core'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getEChartsThemeColors } from '@/components/charts/_utils'
import { fragClassColor } from '@/lib/accessibility/scales'
import type { SynthesisWeaponAccuracyEntry, SynthesisWeaponKillEntry } from '@/lib/api/types'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { fragsManifest } from '@/lib/i18n/generated/frags'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'
import { intlLocale } from '@/lib/formatters'
import { useAppShellStore } from '@/stores/appShellStore'

/** Opacité des armes hors classe survolée (survol lié sunburst ↔ breakdown ↔ précision). */
const DIM_OPACITY = 0.28

interface Props {
  weapons?: SynthesisWeaponAccuracyEntry[]
  /** Kills par arme (label → classe) : recolore les barres par CLASSE, cohérent avec le
   *  sunburst / « Détails des frags ». L'entrée précision de l'API ne porte pas la classe. */
  weaponKills?: SynthesisWeaponKillEntry[]
  height?: number
  fillHeight?: boolean
  /** Survol LIÉ : classe survolée pilotée par un frère (sunburst / breakdown) → estompe les autres. */
  hoveredClass?: string | null
  onClassHover?: (classKey: string | null) => void
}

interface AccuracyPoint {
  label: string
  /** Précision en pourcentage (0..100) — l'API fournit 0..1, converti ici. */
  accuracyPct: number
  shotsFired: number
  shotsLanded: number
  /** Clé de classe d'arme (registre) — recolore la barre + survol lié. undefined si inconnue. */
  classKey?: string
}

function buildWeaponAccuracyOption(
  series: ChartSeries<AccuracyPoint>[],
  locale: ManifestLocale,
  hoveredClass?: string | null,
): EChartsCoreOption {
  const tc = getEChartsThemeColors()
  const numLoc = intlLocale(locale)
  // L'axe catégoriel ECharts empile de bas en haut → reverse pour afficher la meilleure
  // précision EN HAUT (les datapoints arrivent triés desc côté Go).
  const data = [...(series[0]?.datapoints ?? [])].reverse()
  const classLabel = (c?: string) => (c ? formatMessage(fragsManifest, `frags.class.${c}` as never, locale) : '')
  const dimOpacity = (cls?: string) => (hoveredClass && cls !== hoveredClass ? DIM_OPACITY : 1)
  return {
    backgroundColor: CHART_BG,
    // Survol lié : option reconstruite à chaque changement de hoveredClass, animation coupée
    // pour un estompage instantané (aligné sur FragWeaponBreakdown).
    animation: false,
    grid: { top: 8, bottom: 8, left: 8, right: 64, containLabel: true },
    tooltip: {
      backgroundColor: tc.tooltipBg,
      borderColor: tc.tooltipBorder,
      textStyle: { color: tc.text, fontSize: 11 },
      trigger: 'item',
      formatter: ((p: { name?: string; value?: number; data?: { shotsFired?: number; shotsLanded?: number; className?: string } }) => {
        const v = typeof p.value === 'number' ? p.value : 0
        const d = p.data ?? {}
        const shots = `<br/>${(d.shotsLanded ?? 0).toLocaleString(numLoc)} / ${(d.shotsFired ?? 0).toLocaleString(numLoc)} tirs`
        const cls = d.className ? `<br/><span style="opacity:0.7">${escapeHtml(d.className)}</span>` : ''
        return `${escapeHtml(p.name ?? '')}<br/><b>${v.toFixed(1)} %</b>${shots}${cls}`
      }) as unknown as string,
    },
    xAxis: { type: 'value', show: false, max: 100 },
    yAxis: {
      type: 'category',
      data: data.map((d) => d.label),
      axisLabel: { color: tc.axisLabel, fontSize: 11 },
      axisTick: { show: false },
      axisLine: { show: false },
    },
    series: [{
      type: 'bar',
      barMaxWidth: 20,
      data: data.map((d) => ({
        value: d.accuracyPct,
        // classKey brut (survol lié → remontée au parent) + className localisé (tooltip).
        classKey: d.classKey,
        className: d.classKey ? classLabel(d.classKey) : undefined,
        shotsFired: d.shotsFired,
        shotsLanded: d.shotsLanded,
        itemStyle: { color: fragClassColor(d.classKey), borderRadius: [0, 3, 3, 0], opacity: dimOpacity(d.classKey) },
      })),
      label: {
        show: true,
        position: 'right',
        color: tc.axisLabel,
        fontSize: 11,
        formatter: (p: { value: number }) => `${p.value.toFixed(1)} %`,
      },
    }],
  }
}

export function SynthesisWeaponAccuracyChart({ weapons, weaponKills, height, fillHeight, hoveredClass = null, onClassHover }: Props) {
  const locale = useAppShellStore((s) => s.locale)
  const title = formatMessage(synthesisManifest, 'synthesis.charts.weapon_accuracy_title', locale)
  const emptyMessage = formatMessage(synthesisManifest, 'synthesis.empty.no_data', locale)
  const list = weapons ?? []

  // label → classe d'arme (depuis les kills par arme, qui portent la classe du registre) :
  // recolore les barres précision par classe. Armes tirées sans kill = classe inconnue → neutre.
  const classByLabel = new Map<string, string>()
  for (const w of weaponKills ?? []) {
    if (w.class) classByLabel.set(w.label, w.class)
  }

  // Série VIDE (et non datapoints vides) quand aucune arme → ChartCard rend le placeholder.
  const series: ChartSeries<AccuracyPoint>[] =
    list.length > 0
      ? [{
          key: 'weapon-accuracy',
          datapoints: list.map((w) => ({
            label: w.label,
            accuracyPct: w.accuracy * 100,
            shotsFired: w.shots_fired,
            shotsLanded: w.shots_landed,
            classKey: classByLabel.get(w.label),
          })),
        }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<AccuracyPoint>[]) => buildWeaponAccuracyOption(s, locale, hoveredClass),
    // hoveredClass force le rebuild de l'estompage lié.
    [locale, hoveredClass]
  )

  // Survol d'une barre → remonte la classe de l'arme au parent (réciproque du sunburst).
  const onEvents = onClassHover
    ? {
        mouseover: (p: unknown) => onClassHover((p as { data?: { classKey?: string } }).data?.classKey ?? null),
        mouseout: () => onClassHover(null),
      }
    : undefined

  const computedHeight = height ?? Math.max(180, list.length * 28 + 16)

  return (
    <ChartCard
      title={title}
      series={series}
      buildOption={buildOption}
      height={computedHeight}
      emptyMessage={emptyMessage}
      className={fillHeight ? 'flex-1' : ''}
      onEvents={onEvents}
    />
  )
}
