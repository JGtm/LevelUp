/**
 * _compareScale — bornes d'axe PARTAGÉES entre la session A (colonne principale) et B
 * (drawer) pour rendre les graphes directement comparables en mode comparaison.
 *
 * Calculé une fois au niveau page (drawer ouvert) à partir des matchs/entries de A ET B,
 * puis passé aux DEUX colonnes (même objet). Drawer fermé → undefined (auto-scale, rendu
 * historique). O(nb matchs), trivial → aucun coût perceptible.
 *
 * Les dérivations ci-dessous RÉPLIQUENT (volontairement, en version minimale) celles des
 * graphes concernés ; gardées en phase via le commentaire de renvoi `cf. <Composant>`.
 */
import type { SessionCompareEntry, SessionDetailMatchRow } from '@/lib/api/types'
import { cumulativeSigned } from '@/lib/charts/cumulativeSeries'
import { netLives } from '@/lib/charts/netLives'

import { modalValue } from './SessionPlacementBreakdown'

export interface CompareScale {
  /** [min, max] du solde cumulé (net score), 0 inclus. */
  netScore?: [number, number]
  /** [min, max] de la balance des dégâts cumulée (vies nettes), 0 inclus. */
  netLives?: [number, number]
  /** [min, max] des taux/min FDA (morts en négatif), 0 inclus. */
  fdaMinute?: [number, number]
  /** [min, max] du score d'engagement (résidu centré sur 0), 0 inclus. */
  engagement?: [number, number]
  /** Max du compte de placements (axe Y). */
  placementMaxCount?: number
  /** Nb de placements affichés (axe X #1..N) — commun aux deux sessions. */
  placementAxisMax?: number
  /** Max du compte de modes (axe Y). */
  modeMaxCount?: number
}

/** cf. SessionNetScoreArea : cumul de (kills − deaths), trié par start_time. */
function netCumulatives(matches: SessionDetailMatchRow[]): number[] {
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  let running = 0
  return sorted.map((m) => (running += (m.kills ?? 0) - (m.deaths ?? 0)))
}

/** cf. SessionNetLivesCumulative : cumul de netLives (vies nettes), trié par start_time. */
function netLivesCumulatives(matches: SessionDetailMatchRow[], hp: number): number[] {
  const sorted = [...matches].sort((a, b) => a.start_time.localeCompare(b.start_time))
  return cumulativeSigned(sorted.map((m) => netLives(m.damage_dealt, m.damage_taken, hp))).map(
    (c) => c.cumulative,
  )
}

/** cf. SessionFdaBars mode="minute" : taux frags/morts/assists par minute. */
function fdaMinuteRates(
  matches: SessionDetailMatchRow[],
): { frags: number; deaths: number; assists: number } | null {
  const rows = matches.filter((m) => (m.duration_seconds ?? 0) > 0)
  if (rows.length === 0) return null
  const t = rows.reduce(
    (acc, m) => {
      acc.kills += m.kills
      acc.deaths += m.deaths
      acc.assists += m.assists
      acc.minutes += (m.duration_seconds ?? 0) / 60
      return acc
    },
    { kills: 0, deaths: 0, assists: 0, minutes: 0 },
  )
  if (t.minutes <= 0) return null
  return { frags: t.kills / t.minutes, deaths: t.deaths / t.minutes, assists: t.assists / t.minutes }
}

/** cf. SessionEngagementChart : engagement_score par match (entry.match_series). */
function engagementValues(entry: SessionCompareEntry | null): number[] {
  return (entry?.match_series ?? [])
    .map((p) => p.engagement_score)
    .filter((v): v is number => v != null)
}

/** cf. SessionPlacementBreakdown : compte par placement + borne d'axe (lobby modal). */
function placementBounds(matches: SessionDetailMatchRow[]): { maxCount: number; axisMax: number } | null {
  const placements = matches.map((m) => m.placement).filter((p): p is number => p != null && p > 0)
  if (placements.length === 0) return null
  const lobbySizes = matches.map((m) => m.lobby_size).filter((n): n is number => n != null && n > 0)
  const axisMax = Math.max(modalValue(lobbySizes) ?? 0, Math.max(...placements))
  if (axisMax <= 0) return null
  const counts = new Array<number>(axisMax).fill(0)
  for (const p of placements) if (p >= 1 && p <= axisMax) counts[p - 1] += 1
  return { maxCount: Math.max(...counts), axisMax }
}

/** cf. SessionModeBreakdown : compte de matchs par mode (mode_ui/pair_name). */
function modeMaxCount(matches: SessionDetailMatchRow[]): number {
  const counts = new Map<string, number>()
  for (const m of matches) {
    const mode = (m.mode_ui || m.pair_name || '—').trim()
    counts.set(mode, (counts.get(mode) ?? 0) + 1)
  }
  return counts.size ? Math.max(...counts.values()) : 0
}

/**
 * Combine les sessions A et B en bornes d'axe communes. Chaque champ n'est renseigné que
 * s'il y a de la donnée exploitable des deux côtés (sinon le graphe garde son auto-scale).
 */
export function computeCompareScale(
  aMatches: SessionDetailMatchRow[],
  aEntry: SessionCompareEntry | null,
  bMatches: SessionDetailMatchRow[],
  bEntry: SessionCompareEntry | null,
  hp: number,
): CompareScale {
  const scale: CompareScale = {}

  const netVals = [...netCumulatives(aMatches), ...netCumulatives(bMatches)]
  if (netVals.length > 0) scale.netScore = [Math.min(...netVals, 0), Math.max(...netVals, 0)]

  const netLivesVals = [...netLivesCumulatives(aMatches, hp), ...netLivesCumulatives(bMatches, hp)]
  if (netLivesVals.length > 0)
    scale.netLives = [Math.min(...netLivesVals, 0), Math.max(...netLivesVals, 0)]

  const fdas = [fdaMinuteRates(aMatches), fdaMinuteRates(bMatches)].filter(
    (x): x is NonNullable<typeof x> => x != null,
  )
  if (fdas.length > 0) {
    const posMax = Math.max(...fdas.flatMap((f) => [f.frags, f.assists]), 0)
    const deathMax = Math.max(...fdas.map((f) => f.deaths), 0)
    scale.fdaMinute = [-deathMax, posMax]
  }

  const engVals = [...engagementValues(aEntry), ...engagementValues(bEntry)]
  if (engVals.length > 0) scale.engagement = [Math.min(...engVals, 0), Math.max(...engVals, 0)]

  const ps = [placementBounds(aMatches), placementBounds(bMatches)].filter(
    (x): x is NonNullable<typeof x> => x != null,
  )
  if (ps.length > 0) {
    scale.placementMaxCount = Math.max(...ps.map((p) => p.maxCount))
    scale.placementAxisMax = Math.max(...ps.map((p) => p.axisMax))
  }

  const mMax = Math.max(modeMaxCount(aMatches), modeMaxCount(bMatches))
  if (mMax > 0) scale.modeMaxCount = mMax

  return scale
}
