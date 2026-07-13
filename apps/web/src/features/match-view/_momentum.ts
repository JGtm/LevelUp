/**
 * _momentum.ts — logique pure de l'histogramme momentum de la carte « Dominance »
 * (match_view.10, `MatchTugOfWarChart`).
 *
 * Affecte chaque event `kill` à une tranche temporelle (bin) et à une équipe, puis
 * dérive par bin :
 *   - `delta` = kills alliés − kills ennemis (barre signée du momentum) ;
 *   - `teamKills` / `enemyKills` (counts absolus de la tranche) ;
 *   - `cumTeam` / `cumEnemy` (cumuls depuis le début, pour le tooltip) ;
 *   - `trend` (DEC-4) : `'up'` quand le momentum se RENFORCE par rapport à la tranche
 *     précédente (→ opacité pleine au rendu), `'down'` quand il s'essouffle.
 *
 * La liste `kills` (une entrée par kill, avec sa position fractionnaire dans le bin)
 * est renvoyée pour que le composant scatter/vagues n'itère les events qu'une fois.
 *
 * Zéro dépendance React/ECharts : testable en pur (`_momentum.test.ts`).
 *
 * Rappel produit : les `team_kills` / `enemy_kills` du backend (`MatchTugOfWarBin`)
 * sont un delta NET (un seul non-nul par bin) — on recompute ici les deux counts
 * simultanément depuis `highlight_events` + `team_side` du scoreboard.
 */
import type { MatchHighlightEvent, MatchTugOfWarBin } from '@/lib/api/types'

/** Sens du momentum d'une tranche relativement à la précédente (DEC-4). */
type MomentumTrend = 'up' | 'down'

/** Résultat momentum d'une tranche temporelle. */
export interface MomentumBin {
  /** kills alliés − kills ennemis dans la tranche (barre signée). */
  delta: number
  teamKills: number
  enemyKills: number
  /** Cumul kills alliés depuis le début du match (tooltip). */
  cumTeam: number
  /** Cumul kills ennemis depuis le début du match (tooltip). */
  cumEnemy: number
  /** `'up'` = momentum qui se renforce (opacité pleine), `'down'` = essoufflement. */
  trend: MomentumTrend
}

/** Un kill affecté à un bin, avec sa position fractionnaire (scatter/vagues). */
export interface MomentumKill {
  tMs: number
  xuid: string
  ally: boolean
  binIdx: number
  fracInBin: number
}

interface MomentumData {
  momentum: MomentumBin[]
  kills: MomentumKill[]
}

/** Métadonnée minimale par xuid nécessaire au calcul (appartenance équipe). */
type XuidAllyMeta = ReadonlyMap<string, { ally: boolean }>

/**
 * Calcule le momentum par bin depuis les events `kill` et les bornes de bins.
 * `xuidMeta` résout l'appartenance d'équipe (allié vs ennemi) d'un acteur.
 * Un event ignoré (mauvais type, xuid/temps manquant, acteur hors scoreboard,
 * temps hors de toute borne de bin) ne contribue ni au delta ni aux kills.
 */
export function computeMomentumBins(
  bins: MatchTugOfWarBin[],
  events: MatchHighlightEvent[] | null | undefined,
  xuidMeta: XuidAllyMeta,
): MomentumData {
  const teamKills = new Array<number>(bins.length).fill(0)
  const enemyKills = new Array<number>(bins.length).fill(0)
  const kills: MomentumKill[] = []

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
    if (meta.ally) teamKills[idx]++
    else enemyKills[idx]++
  }

  const momentum: MomentumBin[] = []
  let cumTeam = 0
  let cumEnemy = 0
  for (let i = 0; i < bins.length; i++) {
    cumTeam += teamKills[i]
    cumEnemy += enemyKills[i]
    const delta = teamKills[i] - enemyKills[i]
    momentum.push({
      delta,
      teamKills: teamKills[i],
      enemyKills: enemyKills[i],
      cumTeam,
      cumEnemy,
      trend: computeTrend(delta, i > 0 ? momentum[i - 1].delta : 0),
    })
  }

  return { momentum, kills }
}

/**
 * Sens du momentum d'une tranche (DEC-4). `prevDelta` = delta de la tranche
 * précédente (0 avant la première tranche, ce qui rend « up » le premier bin non nul,
 * qu'il penche allié ou ennemi). `'up'` = amplitude signée qui s'accroît dans le sens
 * déjà engagé → opacité pleine ; sinon `'down'` (essoufflement). Un `delta` nul ne
 * produit pas de barre (DEC-5), son `trend` est neutralisé à `'down'`.
 */
function computeTrend(delta: number, prevDelta: number): MomentumTrend {
  if (delta > 0) return delta > prevDelta ? 'up' : 'down'
  if (delta < 0) return delta < prevDelta ? 'up' : 'down'
  return 'down'
}
