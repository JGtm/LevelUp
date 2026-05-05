/**
 * colors.ts — palette des joueurs pour les charts de la match-view.
 *
 * Trois rôles distincts :
 *  - Joueur principal (is_me) → token `compare-a` (cohérent avec la pill
 *    de la page Squad).
 *  - Coéquipiers (même `team_side` que le joueur principal) → palette
 *    « cool/positive » dérivée des tokens utilisés sur Squad
 *    (narrative-dominant / perf-tier-3 / divergent-pos / …).
 *  - Adversaires → palette « warm/négative »
 *    (outcome-loss / narrative-debacle / narrative-humiliation / …).
 *
 * Le mapping est cyclique à l'intérieur de chaque groupe, ce qui garantit
 * une distinction visuelle des joueurs même quand le scoreboard contient
 * 8+ entrées.
 */
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type {
  MatchKillerVictimPair,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'

const MAIN_TOKEN: SemanticToken = 'compare-a'

/** Tokens cool/positifs pour les coéquipiers (palette « squad-like »). */
const ALLY_TOKENS: SemanticToken[] = [
  'narrative-dominant',
  'perf-tier-3',
  'divergent-pos',
  'narrative-encounter-ally-plus',
  'narrative-remontada',
  'perf-tier-2',
]

/** Tokens warm/négatifs pour les adversaires. */
const ENEMY_TOKENS: SemanticToken[] = [
  'outcome-loss',
  'narrative-debacle',
  'narrative-humiliation',
  'perf-tier-4',
  'perf-tier-5',
  'narrative-contre-remontada',
]

export interface MatchPlayerColors {
  /** Token sémantique par xuid (utile pour componentColors BarStacked). */
  tokenByXUID: Map<string, SemanticToken>
  /** Hex résolu par xuid (pratique pour TimeseriesLine). */
  hexByXUID: Map<string, string>
  /** Token sémantique par gamertag (le BarStacked agrège par gamertag). */
  tokenByGamertag: Map<string, SemanticToken>
}

/**
 * Construit la palette joueurs pour un match : main player en `compare-a`,
 * alliés cyclés sur les tokens cool, ennemis cyclés sur les tokens warm.
 *
 * L'ordre par groupe est stable (ordre du scoreboard) pour que le mapping
 * reste cohérent entre les charts qui partagent les mêmes joueurs.
 */
export function buildMatchPlayerColors(
  scoreboard: MatchScoreboardRow[],
  meXUID: string | null,
): MatchPlayerColors {
  const tokenByXUID = new Map<string, SemanticToken>()
  const hexByXUID = new Map<string, string>()
  const tokenByGamertag = new Map<string, SemanticToken>()

  const meRow = meXUID ? scoreboard.find((r) => r.xuid === meXUID) : undefined
  const allyTeam = meRow?.team_side ?? null

  let allyIdx = 0
  let enemyIdx = 0
  for (const row of scoreboard) {
    let token: SemanticToken
    if (row.is_me || row.xuid === meXUID) {
      token = MAIN_TOKEN
    } else if (allyTeam != null && row.team_side === allyTeam) {
      token = ALLY_TOKENS[allyIdx % ALLY_TOKENS.length]
      allyIdx++
    } else {
      token = ENEMY_TOKENS[enemyIdx % ENEMY_TOKENS.length]
      enemyIdx++
    }
    tokenByXUID.set(row.xuid, token)
    hexByXUID.set(row.xuid, resolveToken(token))
    if (row.gamertag) tokenByGamertag.set(row.gamertag, token)
  }
  return { tokenByXUID, hexByXUID, tokenByGamertag }
}

/** Label lisible pour un xuid absent du scoreboard (fallback FragDiff). */
export function unknownPlayerLabel(xuid: string): string {
  const tail = xuid.slice(-4) || xuid
  return `Joueur ${tail}`
}

/**
 * Construit la table `xuid → gamertag` la plus complète possible pour le
 * match courant en cumulant TOUTES les sources serveur ayant déjà résolu
 * les gamertags :
 *
 *  1. Scoreboard (`team_tab.scoreboard`) — joueurs avec stats.
 *  2. Roster (`team_tab.roster`) — inclut les bots et les joueurs sans stats.
 *  3. Killer/Victim pairs (`combat_tab.killer_victim`) — gamertags résolus
 *     côté Go via `buildKillerVictimPairs` (robuste pour les xuids présents
 *     dans `highlight_events` mais absents du scoreboard).
 *
 * La première source non-vide gagne (priorité scoreboard > roster > kvPairs)
 * pour rester cohérent avec ce que l'utilisateur voit dans le scoreboard.
 */
export function buildXUIDToGamertagMap(
  scoreboard: MatchScoreboardRow[],
  kvPairs?: MatchKillerVictimPair[] | null,
  roster?: MatchRosterRow[] | null,
): Map<string, string> {
  const out = new Map<string, string>()
  const insert = (xuid: string | null | undefined, gt: string | null | undefined) => {
    if (!xuid || !gt) return
    if (out.has(xuid)) return
    out.set(xuid, gt)
  }
  for (const r of scoreboard) insert(r.xuid, r.gamertag)
  if (roster) {
    for (const r of roster) insert(r.xuid, r.gamertag)
  }
  if (kvPairs) {
    for (const p of kvPairs) {
      insert(p.killer_xuid, p.killer_gamertag)
      insert(p.victim_xuid, p.victim_gamertag)
    }
  }
  return out
}
