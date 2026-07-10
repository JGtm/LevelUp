/**
 * SessionDamageComposite — barre composite Dégâts infligés / subis, une par match.
 *
 * Barres horizontales empilées (Y = matchs #N + carte, ordre chronologique) : segment
 * "infligés" (divergent-pos, positif) + segment "subis" (divergent-neg, négatif). La
 * longueur totale = implication aux dégâts du match, le partage = ratio infligés/subis.
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { useAppShellStore } from '@/stores/appShellStore'
import { intlLocale } from '@/lib/formatters'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel } from './_shared'
import { log } from './_logger'

interface DamagePoint {
  label: string
  dealt: number
  taken: number
}

interface DamageOpts {
  dealtLabel: string
  takenLabel: string
  /** false (titre sans damage_taken, ex. Halo 5) → segment + légende « subis » retirés. */
  showTaken?: boolean
  /** Locale BCP-47 pour le formatage des entiers (défaut 'fr-FR' si absent). */
  numLoc?: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionDamageOption(
  series: ChartSeries<DamagePoint>[],
  opts: DamageOpts,
): EChartsCoreOption {
  const showTaken = opts.showTaken ?? true
  const fmtInt = (n: number) => Math.round(n).toLocaleString(opts.numLoc ?? 'fr-FR')
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const dealtColor = resolveToken('divergent-pos')
  // Dégâts subis = couleur "négative" (rouge divergent-neg) pour lire d'un coup d'œil ce
  // qu'on encaisse, vs les dégâts infligés en positif (vert divergent-pos).
  const takenColor = resolveToken('divergent-neg')
  const labels = points.map((p) => p.label)

  const segLabel = {
    show: true,
    position: 'inside' as const,
    color: tc.text,
    fontSize: 10,
    formatter: (p: { value: number }) => (p.value > 0 ? fmtInt(p.value) : ''),
  }

  return {
    backgroundColor: CHART_BG,
    grid: { top: 12, bottom: 28, left: 8, right: 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        if (arr.length === 0) return ''
        const idx = (arr[0] as { dataIndex: number }).dataIndex
        const p = points[idx]
        if (!p) return ''
        const lines = [
          `<strong>${escapeHtml(p.label.replace('\n', ' · '))}</strong>`,
          `${opts.dealtLabel}: <b>${fmtInt(p.dealt)}</b>`,
        ]
        if (showTaken) lines.push(`${opts.takenLabel}: <b>${fmtInt(p.taken)}</b>`)
        return lines.join('<br/>')
      },
    },
    legend: {
      data: showTaken ? [opts.dealtLabel, opts.takenLabel] : [opts.dealtLabel],
      textStyle: { color: tc.axisLabel },
      bottom: 0,
      itemWidth: 10,
      itemHeight: 10,
    },
    xAxis: { ...axis, type: 'value' },
    yAxis: { ...axis, type: 'category', inverse: true, data: labels, axisTick: { show: false } },
    series: [
      {
        name: opts.dealtLabel,
        type: 'bar',
        stack: 'dmg',
        data: points.map((p) => p.dealt),
        itemStyle: { color: dealtColor },
        barMaxWidth: 18,
        label: segLabel,
      },
      // Segment « dégâts subis » retiré si la donnée n'est pas fournie (h5) : sinon
      // toutes les barres « subis » seraient à 0 + une entrée de légende permanente.
      ...(showTaken
        ? [{
            name: opts.takenLabel,
            type: 'bar' as const,
            stack: 'dmg',
            data: points.map((p) => p.taken),
            itemStyle: { color: takenColor },
            barMaxWidth: 18,
            label: segLabel,
          }]
        : []),
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionDamageComposite({ title, matches, height = 280 }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields
  // false (Halo 5) → dégâts subis non fournis : segment + légende « subis » retirés
  // (cf. buildSessionDamageOption). Source unique de masquage via useCapability.
  const providesDamageTaken = useProvidesDamageTaken()
  const locale = useAppShellStore((s) => s.locale)

  const series = useMemo<ChartSeries<DamagePoint>[]>(() => {
    const sorted = [...matches]
      .filter((m) => m.damage_dealt != null || m.damage_taken != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    return [
      {
        key: 'damage',
        datapoints: sorted.map((m, i) => ({
          label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
          dealt: m.damage_dealt ?? 0,
          taken: m.damage_taken ?? 0,
        })),
      },
    ]
  }, [matches])

  // Hauteur FIXE (plus de rows*30+60) : aligne les cartes A vs B et la pile, comme MMR.
  const rows = series[0]?.datapoints.length ?? 0

  // Observabilité : barre dégâts vide alors qu'il y a des matchs = pas de données
  // de dégâts (vieux matchs). Même cause que le nuage OC/DR vide.
  if (matches.length > 0 && rows === 0) {
    log.warn(
      `damage_missing:${matches[0]?.session_label ?? ''}`,
      'Barre dégâts vide : aucun match de la session n\'a de dégâts infligés/subis (vieux matchs ?)',
      { matches: matches.length },
    )
  }

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionDamageOption(s, {
          dealtLabel: fields?.damage_dealt?.label ?? 'damage_dealt',
          takenLabel: fields?.damage_taken?.label ?? 'damage_taken',
          showTaken: providesDamageTaken,
          numLoc: intlLocale(locale),
        })
      }
    />
  )
}
