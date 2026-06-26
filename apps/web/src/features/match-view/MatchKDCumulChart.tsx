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
 * Stagger 2D anti-collision séparé pour les chips au-dessus (kill) et
 * en dessous (death).
 *
 * Source : `combat_tab.highlight_events` filtrés sur `event_type='kill'`,
 * équipe résolue via le scoreboard (team_side du joueur courant = ally team).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getAxisBase, getEChartsThemeColors, getLegendBase, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import type { MatchHighlightEvent, MatchImpactBadge, MatchScoreboardRow } from '@/lib/api/types'
import { displayPlayerName } from '@/lib/players/displayName'
import type { MatchViewText } from './i18n'

interface Props {
  events: MatchHighlightEvent[] | null | undefined
  badges: MatchImpactBadge[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

interface BadgeSpec {
  emoji: string
  /** kill = ancré sur le killer, chip au-dessus.
   *  death = ancré sur l'équipe adverse du défunt (qui a frag), chip en dessous. */
  eventKind: 'kill' | 'death'
  /** good → token success (vert) / bad → token warning (orange) /
   *  auto → bon si l'event profite à l'équipe alliée, mauvais sinon. */
  tone: 'good' | 'bad' | 'auto'
}

const BADGE_MAP: Record<string, BadgeSpec> = {
  // Kill events
  first_blood:       { emoji: '⚡', eventKind: 'kill',  tone: 'auto' },
  top_gun:           { emoji: '🔫', eventKind: 'kill',  tone: 'good' },
  clutch_finisher:   { emoji: '🎯', eventKind: 'kill',  tone: 'good' },
  last_group_kill:   { emoji: '🐌', eventKind: 'kill',  tone: 'bad'  },
  // kamikaze exclu : badge match-wide, pas de timestamp précis
  // Death events (player_xuid = victime alliée — le frag est porté par l'adverse)
  first_group_death: { emoji: '🪦', eventKind: 'death', tone: 'bad'  },
  last_casualty:     { emoji: '💀', eventKind: 'death', tone: 'good' },
}

interface CurvePt { tMs: number; y: number }

function valueAtMs(curve: CurvePt[], tMs: number): number {
  if (curve.length === 0) return 0
  let last = 0
  for (const p of curve) {
    if (p.tMs <= tMs) last = p.y
    else break
  }
  return last
}

function formatMmSs(valueMs: number): string {
  const m = Math.floor(valueMs / 60000)
  const s = Math.max(0, Math.floor((valueMs % 60000) / 1000))
  return `${m}m${s.toString().padStart(2, '0')}s`
}

export function MatchKDCumulChart({ events, badges, scoreboard, meXUID, t }: Props) {
  // Le chart trace les frags cumulés par équipe : seuls les events `kill` le
  // peuplent. `events` peut être non-vide (médailles, autres faits marquants)
  // sans aucun kill → on force l'EmptyState plutôt qu'un canvas vide titré.
  const hasKillEvents =
    !!events && events.some((e) => (e.event_type ?? '').toLowerCase() === 'kill')
  const series: ChartSeries<MatchHighlightEvent>[] =
    hasKillEvents && events
      ? [{ key: 'match_view.combat.kd_cumul', datapoints: events }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<MatchHighlightEvent>[]): EChartsCoreOption => {
      if (s.length === 0 || !events || events.length === 0) return { backgroundColor: CHART_BG }

      const sb = scoreboard ?? []
      const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
      const allyTeam = meRow?.team_side ?? null
      const xuidMeta = new Map<string, { gamertag: string; ally: boolean }>()
      for (const r of sb) {
        const ally = r.is_me || (allyTeam != null && r.team_side === allyTeam)
        xuidMeta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
      }

      // ---- Cumulatifs par équipe via highlight_events (event_type=kill) ----
      const sortedKills = events
        .filter((e) => (e.event_type ?? '').toLowerCase() === 'kill' && e.actor_xuid && e.event_time_ms != null)
        .sort((a, b) => (a.event_time_ms as number) - (b.event_time_ms as number))

      const allyCurve: CurvePt[] = []
      const enemyCurve: CurvePt[] = []
      let allyCum = 0
      let enemyCum = 0
      for (const ev of sortedKills) {
        const meta = xuidMeta.get(ev.actor_xuid as string)
        if (!meta) continue
        const tMs = ev.event_time_ms as number
        if (meta.ally) {
          allyCum += 1
          allyCurve.push({ tMs, y: allyCum })
        } else {
          enemyCum += 1
          enemyCurve.push({ tMs, y: enemyCum })
        }
      }
      if (allyCurve.length === 0 && enemyCurve.length === 0) return { backgroundColor: CHART_BG }

      const totalMs = Math.max(
        allyCurve.length ? allyCurve[allyCurve.length - 1].tMs : 0,
        enemyCurve.length ? enemyCurve[enemyCurve.length - 1].tMs : 0,
        60_000,
      )
      const yMaxData = Math.max(allyCum, enemyCum, 1)

      // ---- Couleurs (tokens — palette okabe-ito / cividis / etc.) -------
      const tc = getEChartsThemeColors()
      const colorAlly  = resolveToken('team-ally')
      const colorEnemy = resolveToken('team-enemy')
      const accentGood = resolveToken('success')   // vert
      const accentBad  = resolveToken('warning')   // orange
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

      // ---- Placement chips badges avec stagger 2D anti-collision ---------
      const baseOffset = Math.max(2.5, yMaxData * 0.18)
      const chipHeightY = baseOffset * 1.05
      const chipTimeWindow = totalMs * 0.06
      // Listes séparées pour les chips au-dessus (kill) et en dessous (death) :
      // les 2 zones ne peuvent pas se chevaucher — stagger indépendant.
      type Placed = { tMs: number; yChip: number }
      const placedAbove: Placed[] = []
      const placedBelow: Placed[] = []


      type MarkPoint = Record<string, unknown>
      type MarkLineSeg = [Record<string, unknown>, Record<string, unknown>]
      const allyMP: MarkPoint[]  = []
      const allyML: MarkLineSeg[] = []
      const enemyMP: MarkPoint[] = []
      const enemyML: MarkLineSeg[] = []

      const sortedBadges = (badges ?? [])
        .filter((b) => b.time_ms != null && BADGE_MAP[b.key])
        .map((b) => ({ b, spec: BADGE_MAP[b.key], tMs: b.time_ms as number }))
        .sort((a, b) => a.tMs - b.tMs)

      for (const item of sortedBadges) {
        const meta = item.b.player_xuid ? xuidMeta.get(item.b.player_xuid) : undefined
        if (!meta) continue
        // kill   → ancré sur la courbe du killer (player_xuid)
        // death  → ancré sur la courbe de l'équipe adverse du défunt (qui a frag)
        const team: 'ally' | 'enemy' =
          item.spec.eventKind === 'kill'
            ? meta.ally ? 'ally' : 'enemy'
            : meta.ally ? 'enemy' : 'ally'
        const curve = team === 'ally' ? allyCurve : enemyCurve
        const yAt = valueAtMs(curve, item.tMs)
        const placeAbove = item.spec.eventKind === 'kill'

        let yChip = placeAbove ? yAt + baseOffset : yAt - baseOffset
        const sink = placeAbove ? placedAbove : placedBelow
        let safety = 12
        while (
          safety-- > 0 &&
          sink.some(
            (p) =>
              Math.abs(p.tMs - item.tMs) < chipTimeWindow &&
              Math.abs(p.yChip - yChip) < chipHeightY,
          )
        ) {
          yChip += placeAbove ? chipHeightY : -chipHeightY
        }
        sink.push({ tMs: item.tMs, yChip })

        // Tone résolu (auto = good si la frag profite à l'équipe alliée)
        let tone: 'good' | 'bad'
        if (item.spec.tone === 'auto') tone = team === 'ally' ? 'good' : 'bad'
        else tone = item.spec.tone
        const accent = tone === 'good' ? accentGood : accentBad

        const sinkMP = team === 'ally' ? allyMP : enemyMP
        const sinkML = team === 'ally' ? allyML : enemyML
        const toneTag = tone === 'good' ? 'g' : 'b'
        const chipLabel = `${item.spec.emoji} ${meta.gamertag}`
        sinkMP.push({
          coord: [item.tMs, yAt],
          symbol: 'circle',
          symbolSize: 9,
          itemStyle: { color: accent, borderColor: '#fff', borderWidth: 1.5 }, // color-allow: structural SVG border on colored dot markers
          label: { show: false },
        })
        sinkMP.push({
          coord: [item.tMs, yChip],
          symbol: 'circle',
          symbolSize: 1,
          itemStyle: { color: 'transparent', borderColor: 'transparent' },
          label: {
            show: true,
            formatter: `{${toneTag}|${chipLabel}}`,
            rich: {
              g: chipRich(accentGood),
              b: chipRich(accentBad),
            },
            align: 'center',
            verticalAlign: 'middle',
          },
        })
        sinkML.push([
          { coord: [item.tMs, yAt], lineStyle: { color: accent, width: 1, type: 'dashed', opacity: 0.5 } },
          { coord: [item.tMs, yChip] },
        ])
      }

      // Étendre la fenêtre Y haut/bas pour laisser respirer les chips
      const yMaxAxis = Math.max(
        yMaxData * 1.3,
        ...placedAbove.map((p) => p.yChip + chipHeightY),
        yMaxData + 1,
      )
      const yMinAxis = placedBelow.length > 0
        ? Math.min(0, ...placedBelow.map((p) => p.yChip - chipHeightY))
        : 0

      const axis = getAxisBase(tc)
      return {
        backgroundColor: CHART_BG,
        grid: { left: 50, right: 90, top: 24, bottom: 56, containLabel: true },
        tooltip: {
          ...getTooltipBase(tc),
          trigger: 'axis',
          axisPointer: { type: 'cross', label: { formatter: ({ value }: { value: unknown }) => formatMmSs(value as number) } },
          formatter: (params: Array<{ axisValue: number; marker: string; seriesName: string; value: [number, number] }>) => {
            if (!Array.isArray(params) || !params.length) return ''
            // markPoints inherit the parent seriesName and appear as extra entries
            // in axis-triggered tooltips — deduplicate to one entry per series.
            const expected = new Set([t.combatTeamLabel, t.combatEnemyLabel])
            const seen = new Map<string, typeof params[0]>()
            for (const p of params) {
              if (expected.has(p.seriesName) && !seen.has(p.seriesName)) seen.set(p.seriesName, p)
            }
            if (seen.size === 0) return ''
            const rows = [...seen.values()].map(p => `${p.marker} ${p.seriesName}: <b>${p.value[1]}</b>`).join('<br/>')
            return `${formatMmSs(params[0].axisValue)}<br/>${rows}`
          },
        },
        legend: { ...getLegendBase(tc), data: [t.combatTeamLabel, t.combatEnemyLabel] },
        xAxis: {
          ...axis,
          type: 'value',
          min: 0,
          max: totalMs,
          axisLabel: { ...axis.axisLabel, formatter: (v: number) => formatMmSs(v) },
        },
        yAxis: {
          ...axis,
          type: 'value',
          min: Math.floor(yMinAxis),
          max: Math.ceil(yMaxAxis),
        },
        series: [
          {
            name: t.combatTeamLabel,
            type: 'line',
            showSymbol: false,
            lineStyle: { color: colorAlly, width: 2 },
            itemStyle: { color: colorAlly },
            data: allyCurve.map((p) => [p.tMs, p.y]),
            markPoint: { silent: true, data: allyMP },
            markLine: { silent: true, symbol: ['none', 'none'], label: { show: false }, data: allyML },
            z: 4,
          },
          {
            name: t.combatEnemyLabel,
            type: 'line',
            showSymbol: false,
            lineStyle: { color: colorEnemy, width: 2 },
            itemStyle: { color: colorEnemy },
            data: enemyCurve.map((p) => [p.tMs, p.y]),
            markPoint: { silent: true, data: enemyMP },
            markLine: { silent: true, symbol: ['none', 'none'], label: { show: false }, data: enemyML },
            z: 4,
          },
        ],
      }
    },
    [events, badges, scoreboard, meXUID, t.combatTeamLabel, t.combatEnemyLabel],
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
