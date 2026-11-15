/**
 * SessionFdaBars — barres Frags / Morts / Assists d'une session, en deux modes :
 *  - mode="game"   : moyenne PAR PARTIE (total / nb de matchs)
 *  - mode="minute" : taux PAR MINUTE (total / minutes jouées) — équivalent "session"
 *    du graphe "Stats par minute" de la page Escouade ; il porte aussi les DÉGÂTS infligés et
 *    subis par minute, en deux barres de plus sur un axe vertical SECONDAIRE à droite (la même
 *    grille, jamais un second graphe), dont la ligne zéro est celle de l'axe de gauche.
 *
 * Barres verticales colorées par stat (Frags→outcome-win, Morts→outcome-loss,
 * Assists→outcome-draw ; dégâts infligés / subis→divergent-pos / divergent-neg, comme le graphe
 * « Dégâts infligés / subis par match »), cohérentes avec le graphe FDA. ChartCard + buildOption
 * custom. (Layout simple plutôt que le layout symétrique multi-joueurs de l'Escouade, plus lisible
 * pour une seule session.)
 */
import { useMemo } from 'react'
import type { ReactNode } from 'react'
import type { ChartOption } from '@/components/charts/chartOption'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { tipNumber } from '@/components/charts/tipNumber'
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import { alignedZeroAxes } from '@/lib/charts/alignedZeroAxes'
import { useProvidesDamageTaken } from '@/lib/damage/effectiveHp'
import { useFieldMappings } from '@/lib/i18n/fieldMappings'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { useSessionT } from './_shared'
import { sessionPerMinuteRates } from './sessionPerMinute.logic'

interface Props {
  title: ReactNode
  matches: SessionDetailMatchRow[]
  mode: 'game' | 'minute'
  height?: number
  /** Domaine Y de gauche [min, max] partagé A/B en mode comparaison (sinon auto-scale aligné). */
  yDomain?: [number, number]
  /** Domaine Y de droite (dégâts par minute) partagé A/B, zéro aligné sur celui de `yDomain`. */
  yDomainRight?: [number, number]
}

type FdaKey = 'frags' | 'deaths' | 'assists' | 'damageDealt' | 'damageTaken'

interface FdaPoint {
  key: FdaKey
  label: string
  value: number
}

const STAT_TOKEN: Record<FdaKey, SemanticToken> = {
  frags: 'stat-kills',
  deaths: 'stat-deaths',
  assists: 'stat-assists',
  damageDealt: 'divergent-pos',
  damageTaken: 'divergent-neg',
}

/** Barres tracées sous l'axe zéro (valeur négative, lue en valeur absolue). */
const BELOW_ZERO: ReadonlySet<FdaKey> = new Set<FdaKey>(['deaths', 'damageTaken'])

/** Barres lues sur l'axe de droite (dégâts par minute). */
const ON_RIGHT_AXIS: ReadonlySet<FdaKey> = new Set<FdaKey>(['damageDealt', 'damageTaken'])

/** Les dégâts sont des centaines : aucune décimale. */
const DAMAGE_DECIMALS = 0

/** Bornes locales des deux axes (zéros alignés) quand la comparaison n'en impose pas. */
function localDomains(points: FdaPoint[]): { left: [number, number]; right: [number, number] } {
  const mag = (keys: FdaKey[]) => Math.max(...points.filter((p) => keys.includes(p.key)).map((p) => p.value), 0)
  return alignedZeroAxes(
    { neg: mag(['deaths']), pos: mag(['frags', 'assists']) },
    { neg: mag(['damageTaken']), pos: mag(['damageDealt']) },
  )
}

interface BuildOpts {
  decimals: number
  yDomain?: [number, number]
  yDomainRight?: [number, number]
  damageAxisName?: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionFdaBarsOption(series: ChartSeries<FdaPoint>[], opts: BuildOpts): ChartOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  const decimalsOf = (key: FdaKey | undefined) =>
    key != null && ON_RIGHT_AXIS.has(key) ? DAMAGE_DECIMALS : opts.decimals
  const fmt = (n: number, d: number) => Number(n.toFixed(d))
  const abs = (v: number, d: number) => tipNumber(Math.abs(v), d)
  const hasRight = points.some((p) => ON_RIGHT_AXIS.has(p.key))

  // Visuel page Escouade : Frags/Assists au-dessus de l'axe zéro (positifs),
  // Morts en dessous (négatifs). Axe X accentué (= ligne zéro), labels en valeur
  // absolue. Couleurs sémantiques par stat plutôt que complément joueur (une seule
  // session → pas de mapping joueur→couleur à préserver). Les dégâts subis suivent la
  // convention des morts : sous le zéro.
  const bar = (p: FdaPoint) => ({
    value: fmt(BELOW_ZERO.has(p.key) ? -p.value : p.value, decimalsOf(p.key)),
    itemStyle: { color: resolveToken(STAT_TOKEN[p.key]) },
  })
  // Une barre n'appartient qu'à UNE des deux séries (gauche / droite) : l'autre porte un trou à
  // sa place, et `barGap: '-100%'` superpose les deux séries sur la même colonne (sans lui, chaque
  // série garderait sa demi-colonne et les barres se décaleraient).
  const leftData = points.map((p) => (ON_RIGHT_AXIS.has(p.key) ? null : bar(p)))
  const rightData = points.map((p) => (ON_RIGHT_AXIS.has(p.key) ? bar(p) : null))
  const barSeries = (data: (ReturnType<typeof bar> | null)[], yAxisIndex: number) => ({
    type: 'bar' as const,
    data,
    yAxisIndex,
    barMaxWidth: 56,
    barGap: '-100%',
    label: {
      show: true,
      position: 'top' as const,
      color: tc.text,
      formatter: (p: { value: number | null; dataIndex: number }) =>
        p.value == null ? '' : abs(p.value, decimalsOf(points[p.dataIndex]?.key)),
    },
  })

  const local = opts.yDomain && (!hasRight || opts.yDomainRight) ? null : localDomains(points)
  const leftDomain = opts.yDomain ?? local?.left
  const rightDomain = opts.yDomainRight ?? local?.right

  return {
    backgroundColor: CHART_BG,
    grid: { top: 28, bottom: 36, left: 44, right: hasRight ? 52 : 16, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'shadow' },
      formatter: (params: unknown) => {
        const arr = Array.isArray(params) ? params : []
        // Deux séries se partagent les colonnes : la ligne est celle qui porte la valeur.
        const p = arr.find((x) => typeof (x as { value?: unknown }).value === 'number') as
          | { name: string; value: number; dataIndex?: number }
          | undefined
        if (!p) return ''
        const d = decimalsOf(p.dataIndex == null ? undefined : points[p.dataIndex]?.key)
        return `${escapeHtml(p.name)}: <b>${abs(p.value, d)}</b>`
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      data: points.map((p) => p.label),
      axisLine: { lineStyle: { color: tc.text, width: 2 } }, // ligne zéro accentuée (foreground)
    },
    yAxis: [
      {
        ...axis,
        type: 'value',
        // Domaine figé en mode comparaison (échelle partagée A/B) ; sinon auto-scale (aligné à droite).
        ...(leftDomain ? { min: leftDomain[0], max: leftDomain[1] } : {}),
        axisLabel: { ...(axis.axisLabel as Record<string, unknown>), formatter: (v: number) => abs(v, opts.decimals) },
      },
      ...(hasRight && rightDomain
        ? [
            {
              ...axis,
              type: 'value' as const,
              position: 'right' as const,
              min: rightDomain[0],
              max: rightDomain[1],
              name: opts.damageAxisName,
              nameTextStyle: { color: tc.axisLabel, fontSize: 10, align: 'right' as const },
              splitLine: { show: false }, // la grille est celle de l'axe de gauche
              axisLabel: {
                ...(axis.axisLabel as Record<string, unknown>),
                formatter: (v: number) => abs(v, DAMAGE_DECIMALS),
              },
            },
          ]
        : []),
    ],
    series: [barSeries(leftData, 0), ...(hasRight ? [barSeries(rightData, 1)] : [])],
  }
}

/** Points du graphe : les trois comptes, puis les dégâts quand la session en porte (mode minute). */
function buildPoints(
  matches: SessionDetailMatchRow[],
  mode: Props['mode'],
  withTaken: boolean,
  lbl: (k: string) => string,
): FdaPoint[] {
  if (mode === 'minute') {
    const r = sessionPerMinuteRates(matches, withTaken)
    if (!r) return []
    return [
      { key: 'frags', label: lbl('kills'), value: r.frags },
      { key: 'deaths', label: lbl('deaths'), value: r.deaths },
      { key: 'assists', label: lbl('assists'), value: r.assists },
      ...(r.damageDealt != null ? [{ key: 'damageDealt' as const, label: lbl('damage_dealt'), value: r.damageDealt }] : []),
      ...(r.damageTaken != null ? [{ key: 'damageTaken' as const, label: lbl('damage_taken'), value: r.damageTaken }] : []),
    ]
  }
  if (matches.length === 0) return []
  const sum = (get: (m: SessionDetailMatchRow) => number) => matches.reduce((acc, m) => acc + get(m), 0)
  const n = matches.length
  return [
    { key: 'frags', label: lbl('kills'), value: sum((m) => m.kills) / n },
    { key: 'deaths', label: lbl('deaths'), value: sum((m) => m.deaths) / n },
    { key: 'assists', label: lbl('assists'), value: sum((m) => m.assists) / n },
  ]
}

export function SessionFdaBars({ title, matches, mode, height = 260, yDomain, yDomainRight }: Props) {
  const { data: fieldMappings } = useFieldMappings()
  const fields = fieldMappings?.fields
  const t = useSessionT()
  const withTaken = useProvidesDamageTaken()
  const damageAxisName = t('session.detail.chart_fda_damage_axis')

  const series = useMemo<ChartSeries<FdaPoint>[]>(() => {
    const lbl = (k: string): string => fields?.[k]?.label ?? k
    const datapoints = buildPoints(matches, mode, withTaken, lbl)
    return datapoints.length === 0 ? [] : [{ key: 'fda', datapoints }]
  }, [matches, mode, fields, withTaken])

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionFdaBarsOption(s, {
          decimals: mode === 'minute' ? 2 : 1,
          yDomain,
          yDomainRight,
          damageAxisName,
        })
      }
    />
  )
}
