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

/**
 * Un KILL, indépendamment de tout découpage temporel.
 *
 * POURQUOI CE TYPE EXISTE À PART DE `MomentumKill`. Le binning n'appartient qu'à
 * l'histogramme de la carte « Dominance » : le rejeu 2D, lui, pose les mêmes kills sur
 * l'horloge du film, où un bin n'a aucun sens. Ce qui est COMMUN — ce qui fait qu'un event
 * est un kill, qui l'a porté, et comment son arme voyage — se lit donc ici, une seule fois.
 */
export interface KillEvent {
  tMs: number
  xuid: string
  ally: boolean
  /**
   * Équipe du TUEUR (`actor_team_id`), pour la couleur d'identité — la même que
   * l'en-tête du scoreboard. Null si le backend ne l'a pas résolue (acteur hors
   * scoreboard) : le feed retombe alors sur le couple allié/ennemi.
   */
  teamID: number | null
  /**
   * L'ARME DU KILL, telle que le backend l'a résolue. Les trois champs voyagent
   * ensemble et sont vides ensemble : une source de dégât non identifiée sans
   * ambiguïté ne donne AUCUNE icône, jamais celle d'une autre arme.
   *
   * `weaponLabel` est un nom propre (BR75), pas un libellé traduit. Vide pour les
   * sources sans nom propre (mêlée, grenade) qui gardent pourtant leur icône.
   */
  weaponLabel: string
  weaponImageUrl: string
  weaponTinted: boolean
}

/** Métadonnée minimale par xuid nécessaire au calcul (appartenance équipe). */
type XuidAllyMeta = ReadonlyMap<string, { ally: boolean }>

/**
 * collectKillEvents extrait les kills d'une liste d'events bruts, dans leur ordre d'arrivée.
 *
 * Un event est ignoré s'il n'est pas un `kill`, s'il n'a ni acteur ni horodatage, ou si son
 * acteur est hors scoreboard — les mêmes exclusions qu'avant l'extraction, parce que ce sont
 * celles du binning serveur.
 */
export function collectKillEvents(
  events: MatchHighlightEvent[] | null | undefined,
  xuidMeta: XuidAllyMeta,
): KillEvent[] {
  const out: KillEvent[] = []
  for (const e of events ?? []) {
    if ((e.event_type ?? '').toLowerCase() !== 'kill') continue
    if (!e.actor_xuid || e.event_time_ms == null) continue
    const meta = xuidMeta.get(e.actor_xuid)
    if (!meta) continue
    out.push({
      tMs: e.event_time_ms,
      xuid: e.actor_xuid,
      ally: meta.ally,
      teamID: e.actor_team_id ?? null,
      weaponLabel: e.weapon_label ?? '',
      weaponImageUrl: e.weapon_image_url ?? '',
      weaponTinted: e.weapon_image_tinted ?? false,
    })
  }
  return out
}

/** Un kill affecté à un bin, avec sa position fractionnaire (kill feed / vagues). */
export interface MomentumKill extends KillEvent {
  binIdx: number
  fracInBin: number
}

interface MomentumData {
  momentum: MomentumBin[]
  kills: MomentumKill[]
}

/**
 * Calcule le momentum par bin depuis les events `kill` et les bornes de bins.
 * `xuidMeta` résout l'appartenance d'équipe (allié vs ennemi) d'un acteur.
 * Un event ignoré (mauvais type, xuid/temps manquant, acteur hors scoreboard,
 * temps AVANT le premier bin) ne contribue ni au delta ni aux kills. Un event
 * AU-DELÀ du dernier bin est CLAMPÉ dans le dernier bin — parité stricte avec le
 * binning serveur (`tug_of_war.go` : `if idx >= nBins { idx = nBins - 1 }`), pour
 * que l'histogramme front et le delta net backend comptent les mêmes kills.
 */
export function computeMomentumBins(
  bins: MatchTugOfWarBin[],
  events: MatchHighlightEvent[] | null | undefined,
  xuidMeta: XuidAllyMeta,
): MomentumData {
  const teamKills = new Array<number>(bins.length).fill(0)
  const enemyKills = new Array<number>(bins.length).fill(0)
  const kills: MomentumKill[] = []

  // Les kills sont collectés par le MÊME chemin que le rejeu 2D ; seul le binning est
  // propre à cette carte. Deux collectes, ce serait deux définitions de « ce qui est un
  // kill » sur la même page.
  for (const k of collectKillEvents(events, xuidMeta)) {
    const tSec = k.tMs / 1000
    let idx = bins.findIndex((b) => tSec >= b.bin_start && tSec < b.bin_end)
    if (idx < 0) {
      // Parité avec le clamp backend (tug_of_war.go) : un event au-delà du dernier
      // bin est compté dans le dernier bin ; un temps avant le premier bin est
      // ignoré (le backend saute TimeMS < 0).
      const last = bins.length - 1
      if (last >= 0 && tSec >= bins[last].bin_end) {
        idx = last
      } else {
        continue
      }
    }
    const bin = bins[idx]
    const span = Math.max(1, bin.bin_end - bin.bin_start)
    const frac = Math.min(0.999, Math.max(0, (tSec - bin.bin_start) / span))
    kills.push({ ...k, binIdx: idx, fracInBin: frac })
    if (k.ally) teamKills[idx]++
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
