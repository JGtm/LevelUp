/**
 * Helpers ChartSeries pour MatchViewPage — onglet Combat (FragDiff,
 * Antagonistes).
 *
 * 2026-05-05 : nettoyage post-refonte combat. Ont été retirés :
 *  - kdTimelineSeries / tugOfWarStackedSeries (charts supprimés)
 *  - cadenceSeriesWithGamertags (chart Cadence remplacé par
 *    EngagementMatchSection — courbe team/attendu/joueur via l'API
 *    /matches/{id}/engagement, branchée comme sur la page Contributions).
 */
import type { ChartPoint2D } from '@/components/charts/TimeseriesLineChart'
import type { ChartPointStacked } from '@/components/charts/BarStackedChart'
import type { ChartSeries } from '@/components/charts/ChartCard'
import type { SemanticToken } from '@/lib/accessibility'
import type {
  MatchHighlightEvent,
  MatchKillerVictimPair,
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

