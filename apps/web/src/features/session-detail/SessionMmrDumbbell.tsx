/**
 * SessionMmrDumbbell — "challenge MMR" par match : pour chaque match, le MMR de
 * votre équipe (●) et celui de l'équipe adverse (◆) reliés par une ligne. La longueur
 * de la ligne = l'écart (le challenge). Adverse plus fort à droite = match en côte.
 *
 * Une instance par session (vue single + drawer côte-à-côte). Y = matchs (#N + carte,
 * ordre chronologique), X = MMR. Markers colorés (équipe vs adverse), ligne de liaison
 * neutre. ChartCard + buildOption custom (scatter ×2 + série custom pour les segments).
 */
import { useMemo } from 'react'
import type { EChartsCoreOption } from 'echarts/core'

import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { useProvidesTeamMmr } from '@/lib/damage/effectiveHp'
import type { SessionDetailMatchRow } from '@/lib/api/types'

import { sessionMatchAxisLabel, useSessionT } from './_shared'
import { log } from './_logger'

interface MmrPoint {
  label: string
  team: number
  enemy: number
}

interface MmrOpts {
  teamLabel: string
  enemyLabel: string
}

// eslint-disable-next-line react-refresh/only-export-components
export function buildSessionMmrDumbbellOption(
  series: ChartSeries<MmrPoint>[],
  opts: MmrOpts,
): EChartsCoreOption {
  const points = series[0]?.datapoints ?? []
  if (points.length === 0) return { backgroundColor: CHART_BG }

  const tc = getEChartsThemeColors()
  const axis = getAxisBase(tc)
  // Couleurs d'équipe choisies par le joueur (mêmes tokens que la match-view Details,
  // overridables via les réglages d'accessibilité) : team-ally = mon équipe, team-enemy
  // = adverse. Cf. MatchScoreboard (tokenCssVar('team-ally'/'team-enemy')).
  const teamColor = resolveToken('team-ally')
  const enemyColor = resolveToken('team-enemy')
  const linkColor = resolveToken('divergent-neutral')

  const labels = points.map((p) => p.label)
  const teamData = points.map((p, i) => [p.team, i])
  const enemyData = points.map((p, i) => [p.enemy, i])
  const linkData = points.map((p, i) => [i, p.team, p.enemy])

  return {
    backgroundColor: CHART_BG,
    grid: { top: 16, bottom: 28, left: 8, right: 24, containLabel: true },
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
        const gap = Math.round(p.team - p.enemy)
        const gapStr = gap >= 0 ? `+${gap}` : `${gap}`
        return [
          `<strong>${p.label.replace('\n', ' · ')}</strong>`,
          `${opts.teamLabel}: <b>${Math.round(p.team)}</b>`,
          `${opts.enemyLabel}: <b>${Math.round(p.enemy)}</b>`,
          `Δ: <b>${gapStr}</b>`,
        ].join('<br/>')
      },
    },
    legend: {
      data: [opts.teamLabel, opts.enemyLabel],
      textStyle: { color: tc.axisLabel },
      bottom: 0,
      itemWidth: 10,
      itemHeight: 10,
    },
    xAxis: { ...axis, type: 'value', scale: true },
    yAxis: {
      ...axis,
      type: 'category',
      inverse: true, // #1 en haut
      data: labels,
    },
    series: [
      // Segments de liaison (sous les markers) — série custom, hors légende.
      {
        type: 'custom',
        silent: true,
        renderItem: (_params: unknown, api: { value: (i: number) => number; coord: (v: number[]) => number[] }) => {
          const idx = api.value(0)
          const a = api.coord([api.value(1), idx])
          const b = api.coord([api.value(2), idx])
          return {
            type: 'line',
            shape: { x1: a[0], y1: a[1], x2: b[0], y2: b[1] },
            style: { stroke: linkColor, lineWidth: 2 },
          }
        },
        data: linkData,
        z: 1,
      },
      {
        name: opts.teamLabel,
        type: 'scatter',
        symbolSize: 11,
        itemStyle: { color: teamColor },
        data: teamData,
        z: 2,
      },
      {
        name: opts.enemyLabel,
        type: 'scatter',
        symbol: 'diamond',
        symbolSize: 11,
        itemStyle: { color: enemyColor },
        data: enemyData,
        z: 2,
      },
    ],
  }
}

interface Props {
  title: string
  matches: SessionDetailMatchRow[]
  height?: number
}

export function SessionMmrDumbbell({ title, matches, height = 280 }: Props) {
  const t = useSessionT()
  // Le warn « MMR vide » ne vise que les sessions social d'un titre QUI fournit le MMR
  // (Infinite). Pour un titre sans team_mmr (Halo 5), le vide est attendu → pas de warn.
  const providesTeamMmr = useProvidesTeamMmr()

  const series = useMemo<ChartSeries<MmrPoint>[]>(() => {
    const sorted = [...matches]
      .filter((m) => m.team_mmr != null && m.enemy_mmr != null)
      .sort((a, b) => a.start_time.localeCompare(b.start_time))
    if (sorted.length === 0) return []
    return [
      {
        key: 'mmr',
        datapoints: sorted.map((m, i) => ({
          label: sessionMatchAxisLabel(i, m.map_name, m.pair_name),
          team: m.team_mmr as number,
          enemy: m.enemy_mmr as number,
        })),
      },
    ]
  }, [matches])

  // Hauteur FIXE (plus de rows*30+60) : une hauteur dynamique désalignerait les cartes
  // A vs B (nombres de matchs différents) et le graphe OC/DR apparié en grille. Aligné
  // sur les autres graphes de la pile (280). Sessions chargées → barres comprimées (ok).
  const rows = series[0]?.datapoints.length ?? 0

  // Observabilité : dumbbell vide alors qu'il y a des matchs = pas de MMR (social ?).
  // Inoffensif si le titre ne fournit pas le MMR (h5) : vide attendu, pas un signal.
  if (providesTeamMmr && matches.length > 0 && rows === 0) {
    log.warn(
      `mmr_missing:${matches[0]?.session_label ?? ''}`,
      'Dumbbell MMR vide : aucun match de la session n\'a de MMR équipe/adverse (session social ?)',
      { matches: matches.length },
    )
  }

  return (
    <ChartCard
      title={title}
      series={series}
      height={height}
      buildOption={(s) =>
        buildSessionMmrDumbbellOption(s, {
          teamLabel: t('session.detail.mmr_team'),
          enemyLabel: t('session.detail.mmr_enemy'),
        })
      }
    />
  )
}
