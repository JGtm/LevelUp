/**
 * Helpers ChartSeries pour MatchViewPage — onglet Combat (FragDiff,
 * Antagonistes, Cadence).
 *
 * 2026-05-05 : nettoyage post-refonte combat. Ont été retirés :
 *  - kdTimelineSeries / tugOfWarStackedSeries (charts supprimés)
 *  - cadenceSeriesWithGamertags (consommait `combat_tab.cadence` côté API,
 *    désormais reconstruit depuis `combat_tab.highlight_events` via
 *    `cadenceSeriesFromEvents` pour rester cohérent avec FragDiff/Antagonistes).
 */
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SemanticToken } from '@/lib/accessibility'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { unknownPlayerLabel } from './colors'

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
 * `meXUID` est mis en première position pour que le wrapper attribue d'abord
 * la couleur principale au joueur. Si `colorByXUID` est fourni, chaque série
 * porte son `colorToken` sémantique (allié vs ennemi) ; sinon le wrapper
 * cycle sur la palette par défaut.
 *
 * `xuidToGamertag` est la table de résolution (cf. `buildXUIDToGamertagMap`)
 * — fallback `Joueur XXXX` (4 derniers chars) uniquement si le xuid n'a
 * été résolu par AUCUNE source serveur.
 */
export function allPlayersFragDiffSeries(
  events: MatchHighlightEvent[],
  xuidToGamertag: Map<string, string>,
  meXUID: string | null,
  colorByXUID?: Map<string, SemanticToken>,
): ChartSeries<ChartPoint2D>[] {
  if (events.length === 0) return []

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
    const gamertag = xuidToGamertag.get(xu) ?? unknownPlayerLabel(xu)
    return {
      key: `match_view.combat.frag_diff.${xu}`,
      colorToken: colorByXUID?.get(xu),
      meta: { gamertag },
      // Insère (0, 0) en tête pour démarrer toutes les courbes au même point.
      datapoints: [{ x: 0, y: 0 }, ...points],
    }
  })
}

/** Construction de la cadence de kills par phase de N secondes, agrégée
 *  par gamertag. Reconstruit côté front depuis `combat_tab.highlight_events`
 *  pour garantir la cohérence avec FragDiff/Antagonistes.
 *
 *  `xuidToGamertag` doit cumuler scoreboard + roster + kvPairs (cf.
 *  `buildXUIDToGamertagMap`) — sinon les xuids présents uniquement dans
 *  `highlight_events` se retrouvent en label `Joueur XXXX`.
 *
 *  Phase par défaut = 60s. Les phases vides (aucun kill) ne sont pas omises
 *  pour préserver la lecture temporelle linéaire — chaque catégorie est un
 *  label `m:ss` du début de phase.
 */
export function cadenceSeriesFromEvents(
  events: MatchHighlightEvent[],
  xuidToGamertag: Map<string, string>,
  phaseSeconds = 60,
): ChartSeries<ChartPointStacked>[] {
  if (events.length === 0 || phaseSeconds <= 0) return []

  const phaseMS = phaseSeconds * 1000
  let maxTime = 0
  const phaseBuckets = new Map<number, Map<string, number>>()

  for (const e of events) {
    if ((e.event_type ?? '').toLowerCase() !== 'kill') continue
    if (e.event_time_ms == null || !e.actor_xuid) continue
    const t = e.event_time_ms
    if (t > maxTime) maxTime = t
    const phase = Math.floor(t / phaseMS)
    const gt = xuidToGamertag.get(e.actor_xuid) ?? unknownPlayerLabel(e.actor_xuid)
    const bucket = phaseBuckets.get(phase) ?? new Map<string, number>()
    bucket.set(gt, (bucket.get(gt) ?? 0) + 1)
    phaseBuckets.set(phase, bucket)
  }

  if (phaseBuckets.size === 0) return []

  const bucketCount = Math.floor(maxTime / phaseMS) + 1
  const dps: ChartPointStacked[] = []
  for (let i = 0; i < bucketCount; i++) {
    const startSec = i * phaseSeconds
    const components: Record<string, number> = {}
    const bucket = phaseBuckets.get(i)
    if (bucket) {
      for (const [gt, kills] of bucket) {
        if (kills > 0) components[gt] = kills
      }
    }
    dps.push({ category: formatBinSeconds(startSec), components })
  }

  return [
    {
      key: 'match_view.combat.cadence',
      datapoints: dps,
      meta: { phase_seconds: phaseSeconds },
    },
  ]
}
