/**
 * MatchTugOfWarChart — match_view.10 (Frags par tranche + score cumulé)
 *
 * Stacked bar par tranche temporelle :
 *   - hauteur du segment ally = nb de frags alliés dans la tranche
 *   - hauteur du segment enemy = nb de frags adverses dans la tranche
 *   - hauteur totale du bar = total des frags de la tranche (proxy intensité)
 *
 * Au-dessus de chaque bar : cumul des frags alliés (depuis le début).
 * En-dessous de chaque bar : cumul des frags adverses.
 * L'étiquette du LEADER cumulé à cette tranche est encadrée (background tinté
 * + bordure) avec la couleur de l'équipe — visualise la dynamique de match.
 * Si égalité : aucune des deux n'est encadrée.
 *
 * Layout en 2 grilles ECharts (kill feed dans la grille basse) :
 *   Grille haute (bars + cumul labels + lane alliée + streaks alliés)
 *   Grille basse (lane ennemie + streaks ennemis)
 *
 * Source : `combat_tab.tug_of_war` (counts par bin), `combat_tab.highlight_events`
 * (kill feed scatter + streak detection), scoreboard (team_side resolution).
 */
import type { EChartsCoreOption } from 'echarts/core'
import { useCallback } from 'react'
import { ChartCard, type ChartSeries } from '@/components/charts/ChartCard'
import { CHART_BG, getEChartsThemeColors, getLegendBase, getTooltipBase } from '@/components/charts/_utils'
import { resolveToken } from '@/lib/accessibility'
import { displayPlayerName } from '@/lib/players/displayName'
import type {
  MatchHighlightEvent,
  MatchScoreboardRow,
  MatchTugOfWarBin,
} from '@/lib/api/types'
import type { MatchViewText } from './i18n'

interface Props {
  bins: MatchTugOfWarBin[] | null | undefined
  events: MatchHighlightEvent[] | null | undefined
  scoreboard: MatchScoreboardRow[] | null | undefined
  meXUID: string | null
  t: MatchViewText
}

/** Fenêtre max (ms) entre deux kills consécutifs pour rester dans la même vague. */
const WAVE_WINDOW_MS = 8_000
/** Nombre minimum de kills pour qu'une vague soit considérée notable. */
const WAVE_MIN_KILLS = 3

interface KillEvent { tMs: number; xuid: string; ally: boolean; binIdx: number; fracInBin: number }
interface TeamWave {
  xStart: number; xEnd: number; count: number
  tStartMs: number; tEndMs: number
  waveKills: KillEvent[]
}

/** Détecte les vagues collectives d'une équipe depuis ses kills triés par temps. */
function detectTeamWaves(sorted: KillEvent[]): TeamWave[] {
  const out: TeamWave[] = []
  let i = 0
  while (i < sorted.length) {
    let j = i + 1
    while (j < sorted.length && sorted[j].tMs - sorted[j - 1].tMs <= WAVE_WINDOW_MS) j++
    if (j - i >= WAVE_MIN_KILLS) {
      out.push({
        xStart: sorted[i].binIdx - 0.5 + sorted[i].fracInBin,
        xEnd:   sorted[j - 1].binIdx - 0.5 + sorted[j - 1].fracInBin,
        count:  j - i,
        tStartMs: sorted[i].tMs,
        tEndMs:   sorted[j - 1].tMs,
        waveKills: sorted.slice(i, j),
      })
    }
    i = j
  }
  return out
}

function formatMmSs(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.max(0, Math.floor(seconds % 60))
  return `${m}:${s.toString().padStart(2, '0')}`
}

/**
 * Convertit un hex `#RRGGBB` en `rgba(r,g,b,alpha)`. Utilisé pour générer
 * un fond tinté (alpha 0.18) à partir d'un hex résolu via tokens — pas un
 * choix sémantique, juste un alpha-mix structural pour le badge encadré.
 */
function hexToRgba(hex: string, alpha: number): string {
  const m = /^#?([0-9a-f]{2})([0-9a-f]{2})([0-9a-f]{2})$/i.exec(hex)
  if (!m) return `rgba(0,0,0,${alpha})`
  return `rgba(${parseInt(m[1], 16)}, ${parseInt(m[2], 16)}, ${parseInt(m[3], 16)}, ${alpha})`
}

export function MatchTugOfWarChart({ bins, events, scoreboard, meXUID, t }: Props) {
  // Les bars de dominance sont recalculées depuis les events `kill` : sans bin
  // ET sans kill, la dominance n'a pas de sens → EmptyState plutôt qu'un canvas
  // vide titré (cas des matchs sans données de combat exploitables).
  const hasKillEvents =
    !!events && events.some((e) => (e.event_type ?? '').toLowerCase() === 'kill')
  const series: ChartSeries<MatchTugOfWarBin>[] =
    bins && bins.length > 0 && hasKillEvents
      ? [{ key: 'match_view.combat.tug_of_war', datapoints: bins }]
      : []

  const buildOption = useCallback(
    (s: ChartSeries<MatchTugOfWarBin>[]): EChartsCoreOption => {
      if (s.length === 0 || !bins || bins.length === 0) return { backgroundColor: CHART_BG }

      const sb = scoreboard ?? []
      const meRow = meXUID ? sb.find((r) => r.xuid === meXUID) : undefined
      const allyTeam = meRow?.team_side ?? null
      const xuidMeta = new Map<string, { gamertag: string; ally: boolean }>()
      for (const r of sb) {
        const ally = r.is_me || (allyTeam != null && r.team_side === allyTeam)
        xuidMeta.set(r.xuid, { gamertag: displayPlayerName(r.gamertag, r.xuid), ally })
      }

      const colorTeam   = resolveToken('team-ally')
      const colorEnemy  = resolveToken('team-enemy')
      const tc = getEChartsThemeColors()

      // ---- Bars : normalisées 0–100 % par tranche (dominance relative) ----
      const categories = bins.map((b) => formatMmSs((b.bin_start + b.bin_end) / 2))

      // Absolus depuis highlight_events + team_side du scoreboard, puis normalisés.
      // bins.team_kills / enemy_kills du backend = delta net (un seul non-nul par
      // bin) — on recompute ici pour les deux équipes simultanément.
      const teamCounts = new Array<number>(bins.length).fill(0)
      const enemyCounts = new Array<number>(bins.length).fill(0)
      const kills: KillEvent[] = []
      for (const e of events ?? []) {
        if ((e.event_type ?? '').toLowerCase() !== 'kill') continue
        if (!e.actor_xuid || e.event_time_ms == null) continue
        const meta = xuidMeta.get(e.actor_xuid)
        if (!meta) continue
        const tSec = e.event_time_ms / 1000
        const idx = bins.findIndex((b) => tSec >= b.bin_start && tSec < b.bin_end)
        if (idx < 0) continue
        const bin = bins[idx]
        const span = Math.max(1, bin.bin_end - bin.bin_start)
        const frac = Math.min(0.999, Math.max(0, (tSec - bin.bin_start) / span))
        kills.push({ tMs: e.event_time_ms, xuid: e.actor_xuid, ally: meta.ally, binIdx: idx, fracInBin: frac })
        if (meta.ally) teamCounts[idx]++
        else enemyCounts[idx]++
      }

      // Normalisation : chaque bin totalise 100 % → dominance relative lisible.
      // Les counts absolus restent disponibles pour les labels cumulés et tooltips.
      const teamPct = bins.map((_, i) => {
        const total = teamCounts[i] + enemyCounts[i]
        return total > 0 ? Math.round((teamCounts[i] / total) * 100) : 0
      })
      const enemyPct = bins.map((_, i) => {
        const total = teamCounts[i] + enemyCounts[i]
        return total > 0 ? Math.round((enemyCounts[i] / total) * 100) : 0
      })

      // Cumuls par tranche pour les markPoints labels
      const teamCum: number[] = []
      const enemyCum: number[] = []
      let tCum = 0
      let eCum = 0
      for (let i = 0; i < bins.length; i++) {
        tCum += teamCounts[i]
        eCum += enemyCounts[i]
        teamCum.push(tCum)
        enemyCum.push(eCum)
      }

      // barMax = 100 fixe (normalisé) — layout hardcodé calé sur le mock
      const teamCumLabelY  = 112
      const allyLaneY      = 130
      const yTopMax        = 145
      const enemyCumLabelY = -12
      const yTopMin        = -18

      // ---- Détection des vagues collectives d'équipe ----------------------
      const allyKillsSorted  = kills.filter((k) =>  k.ally).sort((a, b) => a.tMs - b.tMs)
      const enemyKillsSorted = kills.filter((k) => !k.ally).sort((a, b) => a.tMs - b.tMs)
      const allyWaves  = detectTeamWaves(allyKillsSorted)
      const enemyWaves = detectTeamWaves(enemyKillsSorted)

      // ---- markPoints : cumul scores avec encadrement leader -------------
      type MarkPoint = Record<string, unknown>
      const cumulMarkPoints: MarkPoint[] = []
      for (let i = 0; i < bins.length; i++) {
        const tc_ = teamCum[i]
        const ec_ = enemyCum[i]
        const teamLeads  = tc_ > ec_
        const enemyLeads = ec_ > tc_

        // Largeur cadre adaptée au nombre de chiffres (ex 2 chiffres → 22 px,
        // 3 chiffres → 28 px). Hauteur fixe.
        const widthFor = (n: number) => Math.max(22, 10 + 6 * String(n).length)

        cumulMarkPoints.push({
          coord: [i, teamCumLabelY],
          symbol: 'rect',
          symbolSize: [widthFor(tc_), 14],
          itemStyle: teamLeads
            ? { color: hexToRgba(colorTeam, 0.18), borderColor: colorTeam, borderWidth: 1.5 }
            : { color: 'transparent', borderColor: 'transparent', borderWidth: 0 },
          label: {
            show: true,
            formatter: String(tc_),
            color: colorTeam,
            fontSize: 10,
            fontWeight: 'bold',
          },
        })
        cumulMarkPoints.push({
          coord: [i, enemyCumLabelY],
          symbol: 'rect',
          symbolSize: [widthFor(ec_), 14],
          itemStyle: enemyLeads
            ? { color: hexToRgba(colorEnemy, 0.18), borderColor: colorEnemy, borderWidth: 1.5 }
            : { color: 'transparent', borderColor: 'transparent', borderWidth: 0 },
          label: {
            show: true,
            formatter: String(ec_),
            color: colorEnemy,
            fontSize: 10,
            fontWeight: 'bold',
          },
        })
      }

      // ---- Build series --------------------------------------------------
      const allyKills  = allyKillsSorted
      const enemyKills = enemyKillsSorted

      // Scatter datapoints : étalés dans la largeur du bin (catégorie ECharts
      // est centrée sur l'entier, span ±0.5)
      const allyScatter = allyKills.map((k) => ({
        value: [k.binIdx - 0.5 + k.fracInBin, allyLaneY],
        _tip: `${displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)} — ${formatMmSs(k.tMs / 1000)}`,
      }))
      const enemyScatter = enemyKills.map((k) => ({
        value: [k.binIdx - 0.5 + k.fracInBin, 0],
        _tip: `${displayPlayerName(xuidMeta.get(k.xuid)?.gamertag, k.xuid)} — ${formatMmSs(k.tMs / 1000)}`,
      }))

      const allyLane  = bins.map((_, i) => [i, allyLaneY])
      const enemyLane = bins.map((_, i) => [i, 0])

      // Séries de vague : un segment épais par vague, avec label ×N à droite
      const waveLabel = (color: string, count: number, position: 'top' | 'bottom') => ({
        show: true,
        formatter: `×${count}`,
        color,
        fontSize: 10,
        fontWeight: 'bold' as const,
        position,
        distance: 6,
      })
      const waveTip = (side: string, w: TeamWave): string => {
        const byPlayer = new Map<string, number>()
        for (const k of w.waveKills) byPlayer.set(k.xuid, (byPlayer.get(k.xuid) ?? 0) + 1)
        const contrib = [...byPlayer.entries()]
          .sort((a, b) => b[1] - a[1])
          .map(([xuid, n]) => `${displayPlayerName(xuidMeta.get(xuid)?.gamertag, xuid)} — ${n} kill${n > 1 ? 's' : ''}`)
          .join('<br/>')
        return `<b>Vague ${side} ×${w.count}</b> — ${formatMmSs(w.tStartMs / 1000)} → ${formatMmSs(w.tEndMs / 1000)}<br/>${contrib}`
      }

      const allyWaveSeries = allyWaves.map((w, wi) => ({
        type: 'line' as const,
        name: `wave-ally-${wi}`,
        data: [
          { value: [w.xStart, allyLaneY], _tip: waveTip(t.combatTeamLabel, w) },
          { value: [w.xEnd,   allyLaneY], _tip: waveTip(t.combatTeamLabel, w), label: waveLabel(colorTeam, w.count, 'top') },
        ],
        lineStyle: { color: colorTeam, width: 4, opacity: 0.9 },
        itemStyle: { color: colorTeam, borderColor: 'rgba(255,255,255,0.6)', borderWidth: 1.5 },
        symbol: ['circle', 'circle'] as ['circle', 'circle'],
        symbolSize: 9,
        label: { show: false },
        legendHoverLink: false,
        xAxisIndex: 0,
        yAxisIndex: 0,
        tooltip: { formatter: (p: { data?: { _tip?: string } }) => p.data?._tip ?? '' },
        z: 5,
      }))

      const enemyWaveSeries = enemyWaves.map((w, wi) => ({
        type: 'line' as const,
        name: `wave-enemy-${wi}`,
        data: [
          { value: [w.xStart, 0], _tip: waveTip(t.combatEnemyLabel, w) },
          { value: [w.xEnd,   0], _tip: waveTip(t.combatEnemyLabel, w), label: waveLabel(colorEnemy, w.count, 'bottom') },
        ],
        lineStyle: { color: colorEnemy, width: 4, opacity: 0.9 },
        itemStyle: { color: colorEnemy, borderColor: 'rgba(255,255,255,0.6)', borderWidth: 1.5 },
        symbol: ['circle', 'circle'] as ['circle', 'circle'],
        symbolSize: 9,
        label: { show: false },
        legendHoverLink: false,
        xAxisIndex: 1,
        yAxisIndex: 1,
        tooltip: { formatter: (p: { data?: { _tip?: string } }) => p.data?._tip ?? '' },
        z: 5,
      }))

      const tooltipScatter = (p: { data?: { _tip?: string } }) => p.data?._tip ?? ''

      return {
        backgroundColor: CHART_BG,
        grid: [
          { left: 14, right: 14, top: 8, height: '68%', containLabel: false },
          { left: 14, right: 14, top: '78%', height: '10%', containLabel: false },
        ],
        tooltip: {
          ...getTooltipBase(tc),
          trigger: 'item',
        },
        legend: { ...getLegendBase(tc), bottom: 4, data: [t.combatTeamLabel, t.combatEnemyLabel] },
        xAxis: [
          {
            gridIndex: 0,
            type: 'category',
            data: categories,
            axisLine: { show: true, lineStyle: { color: tc.axisLine } },
            axisTick: { show: true, alignWithLabel: true, length: 4, lineStyle: { color: tc.axisLine } },
            axisLabel: { rotate: 0, fontSize: 9, color: tc.axisLabel, interval: Math.max(0, Math.floor(bins.length / 12)), margin: 6 },
            splitLine: { show: false },
          },
          {
            gridIndex: 1,
            type: 'category',
            data: categories,
            show: false,
            axisLine: { show: false },
            axisTick: { show: false },
            axisLabel: { show: false },
            splitLine: { show: false },
          },
        ],
        yAxis: [
          { gridIndex: 0, type: 'value', min: yTopMin, max: yTopMax, show: false, splitLine: { show: false } },
          { gridIndex: 1, type: 'value', min: -2, max: 2, show: false, splitLine: { show: false } },
        ],
        series: [
          {
            type: 'bar',
            stack: 'total',
            name: t.combatTeamLabel,
            data: teamPct,
            itemStyle: { color: colorTeam, opacity: 0.85 },
            barCategoryGap: '0%',
            xAxisIndex: 0,
            yAxisIndex: 0,
            tooltip: {
              formatter: (p: { name?: string; value?: number; dataIndex?: number }) =>
                `${p.name ?? ''}<br/>${t.combatTeamLabel} : <b>${p.value ?? 0} %</b> (${teamCounts[p.dataIndex ?? 0]} kills)`,
            },
            markPoint: { silent: true, data: cumulMarkPoints },
            markLine: {
              silent: true,
              symbol: ['none', 'none'],
              lineStyle: { color: tc.splitLine, width: 1, type: 'dashed', opacity: 0.8 },
              data: [{ yAxis: 50 }],
              label: { show: true, formatter: '50 %', position: 'insideEndTop', color: tc.text, fontSize: 9, opacity: 0.6 },
            },
          },
          {
            type: 'bar',
            stack: 'total',
            name: t.combatEnemyLabel,
            data: enemyPct,
            itemStyle: { color: colorEnemy, opacity: 0.85 },
            barCategoryGap: '0%',
            xAxisIndex: 0,
            yAxisIndex: 0,
            tooltip: {
              formatter: (p: { name?: string; value?: number; dataIndex?: number }) =>
                `${p.name ?? ''}<br/>${t.combatEnemyLabel} : <b>${p.value ?? 0} %</b> (${enemyCounts[p.dataIndex ?? 0]} kills)`,
            },
          },
          // Lane alliée (top grid) — repère visuel du kill feed
          {
            type: 'line',
            name: 'Lane alliée',
            data: allyLane,
            lineStyle: { color: colorTeam, width: 1.5, opacity: 0.45 },
            showSymbol: false,
            silent: true,
            tooltip: { show: false },
            legendHoverLink: false,
            xAxisIndex: 0,
            yAxisIndex: 0,
            z: 2,
          },
          {
            type: 'scatter',
            name: 'Mes kills',
            data: allyScatter,
            symbol: 'circle',
            symbolSize: 7,
            itemStyle: { color: colorTeam, opacity: 0.7 },
            legendHoverLink: false,
            xAxisIndex: 0,
            yAxisIndex: 0,
            tooltip: { formatter: tooltipScatter },
            z: 3,
          },
          ...allyWaveSeries,
          {
            type: 'line',
            name: 'Lane ennemie',
            data: enemyLane,
            lineStyle: { color: colorEnemy, width: 1.5, opacity: 0.45 },
            showSymbol: false,
            silent: true,
            tooltip: { show: false },
            legendHoverLink: false,
            xAxisIndex: 1,
            yAxisIndex: 1,
            z: 2,
          },
          {
            type: 'scatter',
            name: 'Kills ennemis',
            data: enemyScatter,
            symbol: 'circle',
            symbolSize: 7,
            itemStyle: { color: colorEnemy, opacity: 0.7 },
            legendHoverLink: false,
            xAxisIndex: 1,
            yAxisIndex: 1,
            tooltip: { formatter: tooltipScatter },
            z: 3,
          },
          ...enemyWaveSeries,
        ],
      }
    },
    [bins, events, scoreboard, meXUID, t.combatTeamLabel, t.combatEnemyLabel],
  )

  return (
    <ChartCard
      title={t.combatTugOfWarTitle}
      series={series}
      height={360}
      buildOption={buildOption}
      emptyMessage={t.combatNoData}
    />
  )
}
