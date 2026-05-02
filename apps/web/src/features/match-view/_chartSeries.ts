/**
 * Helpers ChartSeries pour MatchViewPage : KD timeline, Tug-of-war, Cadence.
 *
 * P8.4 (revue 2026-04-29) : extraits de MatchViewPage.tsx (~40L).
 * 2026-05-02 : ajout helpers tugOfWarStackedSeries + cadenceSeriesWithGamertags
 *              pour match_view.10 et match_view.11.
 */
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchScoreboardRow,
  MatchViewCadence,
} from '@/lib/api/types'

export interface KDTimelinePoint {
  time_seconds: number
  kills: number
  deaths: number
}

export function kdTimelineSeries(
  points: KDTimelinePoint[],
  labelOf: (key: string, fallback: string) => string,
): ChartSeries<ChartPoint2D>[] {
  return [
    {
      key: 'match_view.combat.kd_timeline.kills',
      meta: { gamertag: labelOf('kills', 'Kills') },
      datapoints: points.map((p) => ({ x: p.time_seconds, y: p.kills })),
    },
    {
      key: 'match_view.combat.kd_timeline.deaths',
      meta: { gamertag: labelOf('deaths', 'Deaths') },
      datapoints: points.map((p) => ({ x: p.time_seconds, y: p.deaths })),
    },
  ]
}

export interface TugOfWarBin {
  bin_start: number
  bin_end?: number
  team_kills?: number
  enemy_kills?: number
  net_kills: number
}

export function tugOfWarSeries(bins: TugOfWarBin[]): ChartSeries<ChartPoint2D>[] {
  return [
    {
      key: 'match_view.combat.tug_of_war.net_kills',
      meta: { gamertag: 'Kills nets' },
      datapoints: bins.map((b) => ({ x: b.bin_start, y: b.net_kills })),
    },
  ]
}

const TEAM_LABEL = 'Mon équipe'
const ENEMY_LABEL = 'Adverses'

/** Composant tugOfWarStackedSeries — bars divergentes par bin
 *  (Mon équipe positive en haut, Adverses négatives en bas).
 *
 * Catégories formatées en `m:ss`.
 */
export function tugOfWarStackedSeries(
  bins: TugOfWarBin[],
): ChartSeries<ChartPointStacked>[] {
  if (bins.length === 0) return []
  return [
    {
      key: 'match_view.combat.tug_of_war.divergent',
      datapoints: bins.map((b) => ({
        category: formatBinSeconds(b.bin_start),
        components: {
          [TEAM_LABEL]: b.team_kills ?? 0,
          [ENEMY_LABEL]: -(b.enemy_kills ?? 0),
        },
      })),
    },
  ]
}

export const TUG_OF_WAR_LABELS = { team: TEAM_LABEL, enemy: ENEMY_LABEL }

/** Convertit cadence (xuid → kills) en cadence (gamertag → kills) lisible. */
export function cadenceSeriesWithGamertags(
  cadence: MatchViewCadence | null | undefined,
  scoreboard: MatchScoreboardRow[],
): ChartSeries<ChartPointStacked>[] {
  if (!cadence || cadence.datapoints.length === 0) return []
  const xuidToGT = new Map<string, string>()
  for (const r of scoreboard) {
    xuidToGT.set(r.xuid, r.gamertag || r.xuid)
  }
  return [
    {
      key: cadence.key,
      labelKey: cadence.label_key,
      datapoints: cadence.datapoints.map((dp) => {
        const components: Record<string, number> = {}
        for (const [xuid, kills] of Object.entries(dp.components)) {
          if (kills === 0) continue
          components[xuidToGT.get(xuid) ?? xuid] = kills
        }
        return { category: dp.category, components }
      }),
      meta: cadence.meta,
    },
  ]
}

/** Format seconds → "m:ss" (ex 75 → "1:15"). */
export function formatBinSeconds(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.max(0, Math.floor(seconds % 60))
  return `${m}:${s.toString().padStart(2, '0')}`
}

/** Construction des séries du chart match_view.18 (Antagonistes — qui a tué qui).
 *
 * Format `BarStackedChart` horizontal :
 *  - 1 catégorie par tueur (ordonnée par total décroissant)
 *  - composants = victimes, valeur = kills
 *
 * On reçoit déjà les paires agrégées (`killer_victim` du backend).
 */
export function antagonistStackedSeries(
  pairs: MatchKillerVictimPair[],
): ChartSeries<ChartPointStacked>[] {
  if (pairs.length === 0) return []

  const killerTotals = new Map<string, { gamertag: string; total: number }>()
  for (const p of pairs) {
    const acc = killerTotals.get(p.killer_xuid) ?? { gamertag: p.killer_gamertag, total: 0 }
    acc.total += p.kill_count
    killerTotals.set(p.killer_xuid, acc)
  }

  const orderedKillers = Array.from(killerTotals.entries()).sort(
    ([, a], [, b]) => b.total - a.total,
  )

  const datapoints: ChartPointStacked[] = orderedKillers.map(
    ([killerXUID, { gamertag }]) => {
      const components: Record<string, number> = {}
      for (const p of pairs) {
        if (p.killer_xuid !== killerXUID) continue
        const key = p.victim_gamertag || p.victim_xuid
        components[key] = (components[key] ?? 0) + p.kill_count
      }
      return { category: gamertag || killerXUID, components }
    },
  )

  return [
    {
      key: 'match_view.combat.antagonists',
      datapoints,
    },
  ]
}

/** Point de la série "frags différentiel cumulé" pour un joueur. */
export interface FragDiffPoint extends ChartPoint2D {
  x: number
  y: number
}

/** Construction des séries pour match_view.13 (Frags différentiel cumulé — tous les joueurs).
 *
 * Pour chaque joueur, on calcule (cumKills - cumDeaths) après chaque event.
 * Une série par joueur, X = secondes depuis le début, Y = différentiel.
 * On insère un point initial (0, 0) pour aligner toutes les courbes.
 *
 * `meXUID` est mis en première position pour que le mapping de couleurs (cyclées
 * dans le wrapper) lui assigne la couleur "principale".
 */
export function allPlayersFragDiffSeries(
  events: MatchHighlightEvent[],
  scoreboard: MatchScoreboardRow[],
  meXUID: string | null,
): ChartSeries<ChartPoint2D>[] {
  if (events.length === 0) return []

  const xuidToGT = new Map<string, string>()
  for (const r of scoreboard) {
    xuidToGT.set(r.xuid, r.gamertag || r.xuid)
  }

  // Tri chronologique stable
  const sorted = [...events]
    .filter((e) => e.event_time_ms != null && e.actor_xuid)
    .sort((a, b) => (a.event_time_ms ?? 0) - (b.event_time_ms ?? 0))

  const playerSeries = new Map<string, FragDiffPoint[]>()
  const cumDiff = new Map<string, number>()

  for (const e of sorted) {
    const xu = e.actor_xuid as string
    const t = e.event_time_ms ?? 0
    const etype = (e.event_type ?? '').toLowerCase()
    if (etype !== 'kill' && etype !== 'death') continue
    const cur = (cumDiff.get(xu) ?? 0) + (etype === 'kill' ? 1 : -1)
    cumDiff.set(xu, cur)
    const list = playerSeries.get(xu) ?? []
    list.push({ x: Math.floor(t / 1000), y: cur })
    playerSeries.set(xu, list)
  }

  if (playerSeries.size === 0) return []

  // Ordre : joueur principal d'abord, puis par nb d'events décroissant.
  const xuids = Array.from(playerSeries.keys())
  xuids.sort((a, b) => {
    if (a === meXUID) return -1
    if (b === meXUID) return 1
    return (playerSeries.get(b)?.length ?? 0) - (playerSeries.get(a)?.length ?? 0)
  })

  return xuids.map((xu) => {
    const points = playerSeries.get(xu) ?? []
    return {
      key: `match_view.combat.frag_diff.${xu}`,
      meta: { gamertag: xuidToGT.get(xu) ?? xu },
      // Insère (0, 0) en tête pour démarrer toutes les courbes au même point.
      datapoints: [{ x: 0, y: 0 }, ...points],
    }
  })
}
