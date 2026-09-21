/**
 * TimeseriesNetLivesTrend — « Balance des dégâts cumulée » sur l'onglet Progression
 * des séries temporelles (lot L du PLAN_AJUSTEMENTS_SUPP_PRE_V75_2026-09-21, D21).
 *
 * MÊME GRANDEUR ET MÊME GRAMMAIRE que `SessionNetLivesCumulative` (Sessions) et que
 * `SquadNetLivesChart` (Escouade) : somme CUMULÉE de la balance
 * `netLives = (dégâts infligés − dégâts subis) / PV-pour-tuer`, en vies, match après
 * match. Aire signée divergente ancrée à 0 (`divergentZeroGradient`, PAS de visualMap)
 * + markLine 0 + pastille KPI « avantage net par match ». La même mesure ne doit pas se
 * lire de deux façons : aucune forme n'est inventée ici, tous les calculs délèguent aux
 * helpers canoniques (`netLives`, `cumulativeSigned`, `meanOfValid`).
 *
 * Différences assumées avec l'instance Sessions, toutes portées par la page :
 *   - les `match_rows` arrivent DÉJÀ triés chronologiquement ASC côté service → PAS de
 *     re-tri (comme `TimeseriesFdaGapTrend`, contrairement à Sessions) ;
 *   - étiquette du DERNIER point (`endLabel`) : sur des dizaines/centaines de matchs, la
 *     valeur d'arrivée du cumul est l'information de tête et le survol ne la donne pas ;
 *   - infobulle par match datée (date · carte), l'axe ne portant que `#N` + carte.
 *
 * D5 : un match sans les deux dégâts ne fait PAS avancer le cumul (report de la dernière
 * valeur) mais figure quand même sur l'axe — réserve dite dans l'aide ⓘ du titre.
 * Aucun match exploitable → option `null` → état vide dans la carte titrée.
 * Masqué par `useProvidesDamageTaken()` (capability `damage_taken` — Halo 5 n'a pas de
 * dégâts subis et la carte s'y retire d'elle-même).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import {
  CHART_BG,
  escapeHtml,
  getAxisBase,
  getEChartsThemeColors,
  getTooltipBase,
  hoverRevealSymbol,
} from '@/components/charts/_utils'
import { titleWithInfo } from '@/components/ui/title-with-info'
import { cumulativeSigned, meanOfValid } from '@/lib/charts/cumulativeSeries'
import { divergentZeroGradient } from '@/lib/charts/divergentZeroGradient'
import { netLives } from '@/lib/charts/netLives'
import {
  substituteHpToken,
  useEffectiveHpToKill,
  useProvidesDamageTaken,
} from '@/lib/damage/effectiveHp'
import { formatDate } from '@/lib/formatters/date'
import { useThemeVersion } from '@/lib/echarts/useThemeVersion'
import type { Locale } from '@/lib/i18n/locale'
import type { TimeseriesMatchRow } from '@/lib/api/types'

import { buildMatchCategories } from './matchLabels'
import { ChartFromOption } from './ChartFromOption'

export interface TimeseriesNetLivesPoint {
  /** Étiquette d'axe X du match (`#N` + carte). */
  label: string
  /** En-tête d'infobulle : date · carte (l'axe ne porte pas la date). */
  header: string
  /** Balance du match (vies nettes), `null` si l'un des deux dégâts manque (D5). */
  value: number | null
  /** Balance cumulée (reporte la dernière valeur si le match n'a pas les dégâts). */
  cumulative: number
}

/**
 * Cumul de la balance des dégâts sur les matchs de la page, dans l'ORDRE DU SERVICE
 * (déjà trié ASC — pas de re-tri), délégué au helper générique `cumulativeSigned`
 * (source unique du cumul, CLAUDE.md n°6).
 */
// eslint-disable-next-line react-refresh/only-export-components
export function computeTimeseriesNetLives(
  rows: TimeseriesMatchRow[],
  hp: number,
  locale: Locale,
): TimeseriesNetLivesPoint[] {
  const categories = buildMatchCategories(rows)
  const cum = cumulativeSigned(rows.map((r) => netLives(r.damage_dealt, r.damage_taken, hp)))
  return rows.map((r, i) => {
    const map = r.map_name_fr || r.map_name
    const date = formatDate(r.start_time, locale)
    return {
      label: categories[i] ?? `#${i + 1}`,
      header: map ? `${date} · ${map}` : date,
      value: cum[i].value,
      cumulative: cum[i].cumulative,
    }
  })
}

export interface TimeseriesNetLivesLabels {
  /** Libellé de la série cumulée (infobulle). */
  series: string
  /** Libellé de la balance du match (infobulle). */
  match: string
}

/** Formate une balance : « — » si absente, préfixe « + » au-dessus de zéro. */
function fmtBalance(v: number | null): string {
  if (v == null) return '—'
  return v >= 0 ? `+${v}` : `${v}`
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildTimeseriesNetLivesOption(
  points: TimeseriesNetLivesPoint[],
  labels: TimeseriesNetLivesLabels,
): EChartsCoreOption | null {
  // Aucun match, ou aucun match portant les deux dégâts → état vide (pas une courbe
  // plate à zéro, qui se lirait comme un équilibre parfait).
  if (points.length === 0 || points.every((p) => p.value == null)) return null

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  // Interval ADAPTATIF : la page peut porter des centaines de matchs (labels illisibles
  // à interval:0), aligné sur TimeseriesFdaGapTrend et SessionNetLivesCumulative.
  const interval = points.length > 30 ? Math.floor(points.length / 12) : 0
  const values = points.map((p) => p.cumulative)
  const gradient = divergentZeroGradient(values)

  return {
    backgroundColor: CHART_BG,
    // Marge droite élargie (24 → 56) : l'étiquette du dernier point se pose HORS de la
    // zone de tracé et serait rognée par la bordure de carte.
    grid: { top: 24, bottom: 64, left: 48, right: 56 },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      formatter: (params: Array<{ dataIndex?: number }>) => {
        if (!Array.isArray(params) || params.length === 0) return ''
        const p = points[params[0]?.dataIndex ?? 0]
        if (!p) return ''
        return (
          `<strong>${escapeHtml(p.header)}</strong><br/>` +
          `${escapeHtml(labels.match)}: ${fmtBalance(p.value)}<br/>` +
          `${escapeHtml(labels.series)}: <b>${fmtBalance(p.cumulative)}</b>`
        )
      },
    },
    xAxis: {
      ...axis,
      type: 'category',
      boundaryGap: false,
      data: points.map((p) => p.label),
      axisLabel: { ...(axis.axisLabel as Record<string, unknown>), interval },
    },
    yAxis: { ...axis, type: 'value' },
    series: [
      {
        type: 'line',
        name: labels.series,
        data: values,
        ...hoverRevealSymbol(tc.text), // color-allow: point neutre, l'aire porte le dégradé
        // Ligne + aire divergentes (positif = balance cumulée au-dessus de l'équilibre,
        // négatif en dessous), aire ancrée à 0 (même dégradé, bascule pile sur 0).
        lineStyle: { width: 2, color: gradient },
        areaStyle: { color: gradient, opacity: 0.18, origin: 0 },
        // Valeur d'arrivée du cumul, lisible sans survol.
        endLabel: {
          show: true,
          color: tc.text, // color-allow: encre de texte du thème, pas une teinte de donnée
          fontWeight: 'bold',
          formatter: (p: { value?: unknown }) =>
            fmtBalance(typeof p.value === 'number' ? p.value : null),
        },
        // Ligne de référence à 0 (équilibre : dégâts infligés = dégâts subis).
        markLine: {
          silent: true,
          symbol: 'none',
          lineStyle: { color: tc.axisLabel, type: 'dashed', width: 1 },
          label: { show: false },
          data: [{ yAxis: 0 }],
        },
      },
    ],
  }
}

export interface TimeseriesNetLivesTrendProps {
  rows: TimeseriesMatchRow[]
  locale: Locale
  /** Libellé du titre de carte (FR/EN, manifest `timeseries`). */
  title: string
  /** Aide ⓘ du titre — jeton `{{HP}}` substitué par le barème du titre courant. */
  tooltip: string
  labels: TimeseriesNetLivesLabels
  /** Caption et unité de la pastille KPI « avantage net par match ». */
  avgCaption: string
  avgUnit: string
  emptyMessage?: string
  height?: number
}

export function TimeseriesNetLivesTrend({
  rows,
  locale,
  title,
  tooltip,
  labels,
  avgCaption,
  avgUnit,
  emptyMessage,
  height = 320,
}: TimeseriesNetLivesTrendProps) {
  const providesDamageTaken = useProvidesDamageTaken()
  const hp = useEffectiveHpToKill()
  const themeVersion = useThemeVersion()

  const points = useMemo(
    () => computeTimeseriesNetLives(rows, hp, locale),
    [rows, hp, locale],
  )
  const option = useMemo<EChartsCoreOption | null>(
    () => buildTimeseriesNetLivesOption(points, labels),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [points, labels, themeVersion],
  )
  const mean = useMemo(() => meanOfValid(points.map((p) => p.value)), [points])

  // Titre sans dégâts subis (ex. Halo 5) → masquage silencieux (pas de carte vide).
  if (!providesDamageTaken) return null

  const nf = new Intl.NumberFormat(locale === 'en' ? 'en-US' : 'fr-FR', {
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
    signDisplay: 'always',
  })
  const footer = (
    <div
      className="flex items-center gap-2 border-t border-border px-3 py-2"
      data-testid="timeseries-net-lives-kpi"
    >
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {avgCaption}
      </span>
      <span className="tabular-nums text-xs font-semibold text-foreground">
        {mean == null ? '—' : `${nf.format(mean)} ${avgUnit}`}
      </span>
    </div>
  )

  return (
    <ChartFromOption
      title={titleWithInfo(substituteHpToken(tooltip, hp))(title)}
      option={option}
      height={height}
      emptyMessage={emptyMessage}
      footer={footer}
    />
  )
}
