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
import { CHART_BG, escapeHtml, getEChartsThemeColors, getLegendBase } from './_utils'

/** Libellés injectés (builder pur, testable sans i18n). */
export interface FragWeaponLabels {
  classLabel: (className: string) => string
  formatValue: (n: number) => string
  killsSuffix: string
}

/** Opacité des armes hors classe survolée (survol lié sunburst ↔ breakdown). */
const DIM_OPACITY = 0.28

/**
 * Builder PUR — exporté pour tester l'option ECharts sans monter le React tree.
 * `hoveredClass` (survol lié) : quand renseignée, les armes des AUTRES classes
 * sont estompées ; les armes de la classe survolée restent en pleine opacité.
 */
// eslint-disable-next-line react-refresh/only-export-components
export function buildFragWeaponBreakdownOption(
  weapons: SynthesisWeaponKillEntry[],
  labels: FragWeaponLabels,
  hoveredClass?: string | null,
): EChartsCoreOption {
  if (weapons.length === 0) return { backgroundColor: CHART_BG }
  const tc = getEChartsThemeColors()
  // Barres triées kills desc → afficher la plus grande en HAUT (yAxis inverse via reverse()).
  const ordered = [...weapons].reverse()
  const dimOpacity = (cls?: string) => (hoveredClass && cls !== hoveredClass ? DIM_OPACITY : 1)

  // Légende des CLASSES en bas, centrée (I4, V7.1) : ce graphe est UNE série de barres
  // recolorée PAR DATUM (itemStyle), donc ECharts n'a pas de légende native par catégorie
  // ici (contrairement à un pie / bar multi-séries où chaque nom porte sa propre couleur).
  // On ajoute une série FANTÔME (data vide, `silent`) PAR CLASSE représentée, uniquement
  // pour porter le nom + la couleur au composant `legend` — la vraie série de barres reste
  // sans `name` correspondant, donc non togglable (cliquer une entrée de légende n'affecte
  // jamais les barres réelles ; légende purement informative ici).
  const classOrder: string[] = []
  for (const w of ordered) {
    if (w.class && !classOrder.includes(w.class)) classOrder.push(w.class)
  }
  const legendGhostSeries = classOrder.map((cls) => ({
    name: labels.classLabel(cls),
    type: 'bar' as const,
    data: [] as number[],
    itemStyle: { color: fragClassColor(cls) },
    silent: true,
  }))

  return {
    backgroundColor: CHART_BG,
    // Survol lié : l'option est reconstruite à chaque changement de `hoveredClass` ;
    // animation coupée pour un estompage instantané (pas de re-croissance des barres).
    animation: false,
    // bottom élargi quand une légende est rendue (place réservée, pas de chevauchement
    // avec les barres — même convention que BarStackedChart bottom:40).
    grid: { top: 8, bottom: classOrder.length > 0 ? 40 : 8, left: 8, right: 80, containLabel: true },
    legend:
      classOrder.length > 0
        ? { ...getLegendBase(tc), left: 'center', data: classOrder.map((c) => labels.classLabel(c)) }
        : { show: false },
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
          // classKey brut (clé de classe) porté pour le survol lié → remontée au parent.
          classKey: w.class,
          itemStyle: { color: fragClassColor(w.class), borderRadius: [0, 3, 3, 0], opacity: dimOpacity(w.class) },
        })),
        label: {
          show: true,
          position: 'right',
          color: tc.axisLabel,
          fontSize: 11,
          formatter: (p: { value: number }) => labels.formatValue(p.value),
        },
      },
      // Séries fantômes APRÈS la série réelle (index 0 = données réelles, contrat testé) —
      // portent uniquement les entrées de légende (cf. commentaire plus haut).
      ...legendGhostSeries,
    ],
  }
}

export interface FragWeaponBreakdownProps {
  weapons?: SynthesisWeaponKillEntry[]
  title?: string
  height?: number
  /** Multiplicateur de hauteur (défaut 1) — ex. 1.1 pour +10 % côté match view. */
  heightScale?: number
  fillHeight?: boolean
  /** Classe(s) utilitaire(s) fusionnée(s) sur la ChartCard (ex. `lg:col-span-1`). */
  className?: string
  /**
   * Survol LIÉ (optionnel) : classe survolée pilotée par un composant frère
   * (ex. `FragSunburst` via `MatchFragCard`) → estompe les armes des autres classes.
   * Non fournie → le composant reste autonome (aucun estompage).
   */
  hoveredClass?: string | null
  /** Remonté au parent au survol d'une barre (classe de l'arme, ou null en sortie). */
  onClassHover?: (classKey: string | null) => void
}

export function FragWeaponBreakdown({ weapons, title, height, heightScale = 1, fillHeight, className = '', hoveredClass = null, onClassHover }: FragWeaponBreakdownProps) {
  const appLocale = useAppShellStore((s) => s.locale)
  const numLoc = intlLocale(appLocale)
  const list = weapons ?? []

  const labels: FragWeaponLabels = {
    classLabel: (c: string) => formatMessage(fragsManifest, `frags.class.${c}` as never, appLocale),
    formatValue: (n: number) => n.toLocaleString(numLoc),
    killsSuffix: formatMessage(fragsManifest, 'frags.charts.center_total_label', appLocale).toLowerCase(),
  }

  const buildOption = useCallback(
    (s: ChartSeries<SynthesisWeaponKillEntry>[]) => buildFragWeaponBreakdownOption(s[0]?.datapoints ?? [], labels, hoveredClass),
    // labels dérive de appLocale ; hoveredClass force le rebuild de l'estompage lié.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [appLocale, hoveredClass],
  )

  // Survol d'une barre → remonte la classe de l'arme au parent (réciproque du sunburst).
  const onEvents = onClassHover
    ? {
        mouseover: (p: unknown) => onClassHover((p as { data?: { classKey?: string } }).data?.classKey ?? null),
        mouseout: () => onClassHover(null),
      }
    : undefined

  // Série VIDE quand aucune arme → ChartCard rend son placeholder (pattern ChartCard standard).
  const series: ChartSeries<SynthesisWeaponKillEntry>[] = list.length > 0 ? [{ key: 'frag-weapons', datapoints: list }] : []
  const cardTitle = title ?? formatMessage(fragsManifest, 'frags.charts.weapon_breakdown_title', appLocale)
  const emptyMessage = formatMessage(fragsManifest, 'frags.empty.no_data', appLocale)
  const computedHeight = Math.round((height ?? Math.max(180, list.length * 28 + 16)) * heightScale)

  return (
    <ChartCard
      title={cardTitle}
      series={series}
      buildOption={buildOption}
      height={computedHeight}
      emptyMessage={emptyMessage}
      className={[fillHeight ? 'flex-1' : '', className].filter(Boolean).join(' ')}
      onEvents={onEvents}
    />
  )
}
