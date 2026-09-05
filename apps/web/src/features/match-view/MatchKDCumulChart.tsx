/**
 * MatchKDCumulChart — match_view.09 (Frags cumulés par équipe)
 *
 * Deux courbes step-line, une par équipe :
 *  - Mon équipe (token `team-ally`) : cumul des kills de l'équipe alliée
 *  - Adversaires (token `team-enemy`) : cumul des kills de l'équipe adverse
 *
 * Pas de courbe de morts : un death = un frag adverse, donc les badges
 * "death" (first_group_death, last_casualty) sont ancrés sur la courbe
 * adverse à l'instant du kill, et leur chip est placé EN DESSOUS de la
 * courbe pour ne pas collisionner avec les chips "kill" placés au-dessus.
 *
 * Source : `combat_tab.highlight_events` filtrés sur `event_type='kill'`,
 * équipe résolue via le scoreboard (team_side du joueur courant = ally team).
 *
 * LA GÉOMÉTRIE VIT DANS `_kdCumul.ts` (registre 2026-09-05, N1) : cumuls, bornes d'axe,
 * placement anti-collision des pastilles et repères de capture y sont purs et testés. Ce
 * fichier ne fait plus que l'HABILLAGE — les couleurs, qui dépendent de la palette
 * d'accessibilité au moment du rendu, les libellés d'axe et les infobulles.
 *
 * L'AXE DES TEMPS EST LA RÉFÉRENCE DE L'ONGLET, et il l'est sans rien faire : `event_time_ms`
 * est déjà recalé sur le début du GAMEPLAY par le serveur (`correctMatchViewEventsT0`), son
 * zéro EST le coup d'envoi. C'est sur cet axe que le bloc « Score dans le temps », juste en
 * dessous, vient se poser — lui vient du film et compte depuis le premier paquet de position,
 * d'où la soustraction de `lib/replay/matchClock` (registre 2026-09-05, P0-7). Rien à
 * convertir ici : toute correction ajoutée à ces abscisses les décalerait de la référence.
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback, useMemo } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, escapeHtml, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase, type EChartsThemeColors } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { formatClockMShort } from '@/lib/formatters'
import type {
  MatchHighlightEvent,
  MatchImpactBadge,
  MatchObjectiveEvent,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { buildKdCumul, type KdCumul, type KdCumulBadge } from './_kdCumul'
import type { MatchViewText } from './i18n'
import { resolveXuidMeta } from './xuidMeta'

interface Props {
  events: MatchHighlightEvent[] | null | undefined
  badges: MatchImpactBadge[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  /** Événements d'objectif (CTF captures…) — overlay de verticales pleine hauteur. */
  objectiveEvents?: MatchObjectiveEvent[] | null
  t: MatchViewText
}

export function MatchKDCumulChart({ events, badges, scoreboard, meXUID, objectiveEvents, t }: Props) {
  // Le chart trace les frags cumulés par équipe : seuls les events `kill` le
  // peuplent. `events` peut être non-vide (médailles, autres faits marquants)
  // sans aucun kill → on force l'EmptyState plutôt qu'un canvas vide titré.
  const hasKillEvents =
    !!events && events.some((e) => (e.event_type ?? '').toLowerCase() === 'kill')
  const series: ChartSeries<MatchHighlightEvent>[] =
    hasKillEvents && events
      ? [{ key: 'match_view.combat.kd_cumul', datapoints: events }]
      : []

  const model = useMemo(
    () =>
      buildKdCumul({
        events,
        badges,
        scoreboard,
        meXUID,
        // Cascade « allie = meme camp que moi » : source unique (xuidMeta.ts).
        xuidMeta: resolveXuidMeta(scoreboard, meXUID),
        objectiveEvents,
      }),
    [events, badges, scoreboard, meXUID, objectiveEvents],
  )

  const buildOption = useCallback(
    (s: ChartSeries<MatchHighlightEvent>[]): EChartsCoreOption => {
      if (s.length === 0 || !model) return { backgroundColor: CHART_BG }
      return kdCumulOption(model, getEChartsThemeColors(), t)
    },
    [model, t],
  )

  return (
    <ChartCard
      title={t.combatKdCumulTitle}
      series={series}
      height={340}
      fluid
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    />
  )
}

/** Les marques d'une série : les pastilles (deux points) et leur trait de rattachement. */
interface BadgeMarks {
  points: Record<string, unknown>[]
  lines: Array<[Record<string, unknown>, Record<string, unknown>]>
}

/**
 * kdCumulOption — l'option ECharts, extraite du composant pour rester lisible.
 *
 * LES COULEURS SE RÉSOLVENT ICI, dans le builder : leur valeur calculée change avec la
 * palette d'accessibilité, et c'est ce rebuild qui la rafraîchit (cf. ChartCard,
 * dépendance `paletteVersion`).
 */
function kdCumulOption(model: KdCumul, tc: EChartsThemeColors, t: MatchViewText): EChartsCoreOption {
  const colorAlly = resolveToken('team-ally')
  const colorEnemy = resolveToken('team-enemy')
  const accents = { good: resolveToken('success'), bad: resolveToken('warning') }
  const marks = { ally: badgeMarks(model, 'ally', accents, tc), enemy: badgeMarks(model, 'enemy', accents, tc) }

  const axis = getAxisBase(tc)
  return {
    backgroundColor: CHART_BG,
    grid: { left: 50, right: 90, top: 24, bottom: 56, containLabel: true },
    tooltip: {
      ...getTooltipBase(tc),
      trigger: 'axis',
      axisPointer: { type: 'cross', label: { formatter: ({ value }: { value: unknown }) => formatClockMShort(value as number) } },
      formatter: tooltipFormatter(t),
    },
    legend: { ...getLegendBase(tc), data: [t.combatTeamLabel, t.combatEnemyLabel] },
    xAxis: {
      ...axis,
      type: 'value',
      min: 0,
      max: model.totalMs,
      axisLabel: { ...axis.axisLabel, formatter: (v: number) => formatClockMShort(v) },
    },
    yAxis: {
      ...axis,
      type: 'value',
      min: model.yMin,
      max: model.yMax,
    },
    series: [
      teamSeries(t.combatTeamLabel, model.ally, colorAlly, marks.ally),
      teamSeries(t.combatEnemyLabel, model.enemy, colorEnemy, marks.enemy),
      // Série dédiée overlay captures CTF : verticales pleine hauteur.
      // Absente du legend (name hors legend.data). markLine non-silent pour
      // exposer le tooltip scorer au survol ; aucune donnée de ligne propre.
      ...captureSeries(model, { ally: colorAlly, enemy: colorEnemy }, t),
    ],
  }
}

/** Une courbe d'équipe, ses pastilles et leurs traits. */
function teamSeries(
  name: string,
  points: KdCumul['ally'],
  color: string,
  marks: BadgeMarks,
): Record<string, unknown> {
  return {
    name,
    type: 'line',
    showSymbol: false,
    lineStyle: { color, width: 2 },
    itemStyle: { color },
    data: points.map((p) => [p.tMs, p.y]),
    markPoint: { silent: true, data: marks.points },
    markLine: { silent: true, symbol: ['none', 'none'], label: { show: false }, data: marks.lines },
    z: 4,
  }
}

/**
 * badgeMarks habille les pastilles d'un camp : un point plein SUR la courbe, un point
 * invisible qui porte l'étiquette à sa hauteur placée, et le trait pointillé qui les relie.
 */
function badgeMarks(
  model: KdCumul,
  team: KdCumulBadge['team'],
  accents: Record<KdCumulBadge['tone'], string>,
  tc: EChartsThemeColors,
): BadgeMarks {
  const chipRich = (border: string) => ({
    backgroundColor: tc.tooltipBg,
    borderColor: border,
    borderWidth: 1,
    borderRadius: 6,
    padding: [5, 9],
    color: tc.text,
    fontSize: 11,
    fontWeight: '600',
    fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif',
    lineHeight: 20,
  })
  const marks: BadgeMarks = { points: [], lines: [] }
  for (const b of model.badges) {
    if (b.team !== team) continue
    const accent = accents[b.tone]
    const toneTag = b.tone === 'good' ? 'g' : 'b'
    marks.points.push({
      coord: [b.tMs, b.yAt],
      symbol: 'circle',
      symbolSize: 9,
      itemStyle: { color: accent, borderColor: '#fff', borderWidth: 1.5 }, // color-allow: structural SVG border on colored dot markers
      label: { show: false },
    })
    marks.points.push({
      coord: [b.tMs, b.yChip],
      symbol: 'circle',
      symbolSize: 1,
      itemStyle: { color: 'transparent', borderColor: 'transparent' },
      label: {
        show: true,
        formatter: `{${toneTag}|${b.label}}`,
        rich: { g: chipRich(accents.good), b: chipRich(accents.bad) },
        align: 'center',
        verticalAlign: 'middle',
      },
    })
    marks.lines.push([
      { coord: [b.tMs, b.yAt], lineStyle: { color: accent, width: 1, type: 'dashed', opacity: 0.5 } },
      { coord: [b.tMs, b.yChip] },
    ])
  }
  return marks
}

/**
 * captureSeries — les captures de drapeau en verticales pleine hauteur.
 *
 * L'axe X est en ms → un markLine `{ xAxis: tMs }` trace une verticale. Vide hors CTF
 * (`extractCtfCaptures` ne rend rien) : la série n'existe alors pas du tout.
 */
function captureSeries(
  model: KdCumul,
  couleurs: { ally: string; enemy: string },
  t: MatchViewText,
): Record<string, unknown>[] {
  if (model.captures.length === 0) return []
  return [
    {
      name: t.combatCtfCaptureLabel,
      type: 'line' as const,
      data: [] as Array<[number, number]>,
      showSymbol: false,
      legendHoverLink: false,
      markLine: {
        silent: false,
        symbol: ['none', 'none'],
        data: model.captures.map((c) => ({
          xAxis: c.tMs,
          lineStyle: {
            color: c.ally ? couleurs.ally : couleurs.enemy,
            width: 1,
            type: 'solid',
            opacity: 0.7,
          },
          label: {
            show: true,
            formatter: t.combatCtfCaptureLabel,
            color: c.ally ? couleurs.ally : couleurs.enemy,
            fontSize: 10,
            fontWeight: 'bold',
            position: 'insideEndTop',
            rotate: 90,
          },
          // Tooltip dédié (item-trigger) : scorer + horodatage de la capture.
          tooltip: {
            show: true,
            trigger: 'item',
            formatter: () => t.combatCtfCaptureTooltip(c.scorer, formatClockMShort(c.tMs)),
          },
        })),
      },
      z: 3,
    },
  ]
}

/**
 * tooltipFormatter écrit l'instant puis une ligne par camp.
 *
 * Les markPoints héritent du nom de leur série et reviennent donc en double dans une
 * infobulle déclenchée par l'axe : on ne garde qu'une entrée par camp.
 */
function tooltipFormatter(t: MatchViewText) {
  return (params: Array<{ axisValue: number; marker: string; seriesName: string; value: [number, number] }>): string => {
    if (!Array.isArray(params) || !params.length) return ''
    const expected = new Set([t.combatTeamLabel, t.combatEnemyLabel])
    const seen = new Map<string, (typeof params)[0]>()
    for (const p of params) {
      if (expected.has(p.seriesName) && !seen.has(p.seriesName)) seen.set(p.seriesName, p)
    }
    if (seen.size === 0) return ''
    const rows = [...seen.values()]
      .map((p) => `${p.marker} ${escapeHtml(p.seriesName ?? '')}: <b>${p.value[1]}</b>`)
      .join('<br/>')
    return `${formatClockMShort(params[0].axisValue)}<br/>${rows}`
  }
}
