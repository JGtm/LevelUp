/**
 * colors.ts — palette des joueurs pour les charts de la match-view.
 *
 * Quatre rôles distincts (l'équipe alliée vs ennemie reste TOUJOURS lisible) :
 *  - Joueur principal (is_me) → `SQUAD_MAIN_PLAYER_TOKEN` (cohérent avec la
 *    pill de la page Squad — source unique, jamais de token en dur ici).
 *  - Amis (page Escouade — `settings.friend_gamertags`) côté allié → tokens
 *    `SQUAD_TEAMMATE_COLOR_TOKENS` (`squad-player-2` / `-3` / `-4`).
 *    Garantit la cohérence visuelle avec la page Squad.
 *  - Autres coéquipiers (même `team_side` que le main, non amis) → palette
 *    « cool/positive » (`perf-tier-2` / `narrative-encounter-ally-plus` /
 *    `narrative-encounter-recrue` / `chart-series-6`).
 *  - Adversaires (amis ou non) → palette « warm/négative »
 *    (`outcome-loss` / `narrative-debacle` / `narrative-humiliation` / …).
 *    Les amis adverses ne reçoivent PAS la couleur squad : la distinction
 *    équipe l'emporte sur le bonus identité (sinon un ami dans l'équipe
 *    ennemie casserait la lecture team).
 *
 * Le mapping est cyclique à l'intérieur de chaque groupe, ce qui garantit
 * une distinction visuelle même quand un groupe dépasse la palette.
 */
import { resolveToken, type SemanticToken } from '@/lib/accessibility'
import type {
  MatchKillerVictimPair,
  MatchRosterRow,
  MatchScoreboardRow,
} from '@/lib/api/types'
import { SQUAD_MAIN_PLAYER_TOKEN, SQUAD_TEAMMATE_COLOR_TOKENS } from '@/features/squad/colors'
import { normalizeGamertagKey } from '@/lib/players/displayName'

const MAIN_TOKEN: SemanticToken = SQUAD_MAIN_PLAYER_TOKEN

/** Tokens cool « squad-like » réservés aux amis présents en équipe alliée. */
const ALLY_FRIEND_TOKENS: SemanticToken[] = [...SQUAD_TEAMMATE_COLOR_TOKENS]

/** Tokens cool pour les coéquipiers non-amis (différents de ALLY_FRIEND_TOKENS).
 *  Règle : aucun doublon hex avec les autres pools de ce fichier. Deux retraits :
 *   - `narrative-dominant` (identique à `squad-player-4` en Okabe-Ito et en
 *     Cividis) → remplacé par `chart-series-6`, couleur de série générique,
 *     neutre pour un allié non identifié ;
 *   - `narrative-remontada` (identique à `perf-tier-2` en Okabe-Ito, et trop
 *     proche du bleu du joueur principal en palette par défaut) → remplacé par
 *     `narrative-encounter-recrue`, teinte palette-invariante comme
 *     `narrative-encounter-ally-plus` déjà utilisé ici.
 *  Reste hors de portée : Cividis est une rampe de 9 valeurs, les doublons
 *  exacts entre familles y sont structurels. */
const ALLY_OTHER_TOKENS: SemanticToken[] = [
  'perf-tier-2',
  'narrative-encounter-ally-plus',
  'narrative-encounter-recrue',
  'chart-series-6',
]

/** Tokens warm/négatifs pour les adversaires (amis ou non).
 *  Règle : aucun doublon hex — perf-tier-5 retiré car identique à outcome-loss. */
const ENEMY_TOKENS: SemanticToken[] = [
  'outcome-loss',
  'narrative-humiliation',
  'narrative-debacle',
  'narrative-contre-remontada',
  'narrative-encounter-coriace',
  'chart-series-8',
  'perf-tier-4',
]

export interface MatchPlayerColors {
  /** Token sémantique par xuid (utile pour componentColors BarStacked). */
  tokenByXUID: Map<string, SemanticToken>
  /** Hex résolu par xuid (pratique pour TimeseriesLine). */
  hexByXUID: Map<string, string>
  /** Token sémantique par gamertag (le BarStacked agrège par gamertag). */
  tokenByGamertag: Map<string, SemanticToken>
  /** Hex résolu par gamertag (override direct pour BarStacked componentColors). */
  hexByGamertag: Map<string, string>
}

/**
 * Construit la palette joueurs pour un match.
 *
 * `friendGamertags` (typiquement `settings.friend_gamertags`) permet de
 * réutiliser le color scheme de la page Squad pour les amis ALLIÉS. Les amis
 * adverses tombent dans la palette ennemie pour préserver la lecture
 * équipe vs équipe.
 *
 * `roster` est utilisé comme source primaire si disponible (couvre TOUS les
 * joueurs du match — bots inclus, joueurs sans stats inclus), avec le
 * scoreboard en fallback pour le team_side. Les xuids inconnus reçoivent
 * un token "chart-series-X" cyclé pour éviter de tomber sur la couleur
 * principale par défaut.
 */
export function buildMatchPlayerColors(
  scoreboard: MatchScoreboardRow[],
  meXUID: string | null,
  friendGamertags?: readonly string[],
  roster?: MatchRosterRow[],
): MatchPlayerColors {
  const tokenByXUID = new Map<string, SemanticToken>()
  const hexByXUID = new Map<string, string>()
  const tokenByGamertag = new Map<string, SemanticToken>()
  const hexByGamertag = new Map<string, string>()

  // Index team_side : roster prime (couvre les bots et joueurs sans stats),
  // scoreboard en fallback.
  const teamSideByXUID = new Map<string, string | null>()
  const gamertagByXUID = new Map<string, string>()
  for (const r of scoreboard) {
    teamSideByXUID.set(r.xuid, r.team_side ?? null)
    if (r.gamertag) gamertagByXUID.set(r.xuid, r.gamertag)
  }
  if (roster) {
    for (const r of roster) {
      if (!teamSideByXUID.has(r.xuid)) teamSideByXUID.set(r.xuid, r.team_side ?? null)
      if (r.gamertag && !gamertagByXUID.has(r.xuid)) gamertagByXUID.set(r.xuid, r.gamertag)
    }
  }

  // team_side du joueur principal : essayer scoreboard puis roster.
  let allyTeam: string | null = null
  if (meXUID) {
    const sbMe = scoreboard.find((r) => r.xuid === meXUID)
    if (sbMe?.team_side != null) {
      allyTeam = sbMe.team_side
    } else if (roster) {
      const rosterMe = roster.find((r) => r.xuid === meXUID)
      if (rosterMe?.team_side != null) allyTeam = rosterMe.team_side
    }
  }

  const friendSet = new Set<string>()
  for (const gt of friendGamertags ?? []) {
    const norm = normalizeGamertagKey(gt)
    if (norm) friendSet.add(norm)
  }

  // Liste ordonnée de joueurs (scoreboard d'abord pour garder l'ordre attendu,
  // puis roster pour les joueurs absents du scoreboard).
  const seenXUIDs = new Set<string>()
  const orderedXUIDs: string[] = []
  for (const r of scoreboard) {
    if (!seenXUIDs.has(r.xuid)) { orderedXUIDs.push(r.xuid); seenXUIDs.add(r.xuid) }
  }
  if (roster) {
    for (const r of roster) {
      if (!seenXUIDs.has(r.xuid)) { orderedXUIDs.push(r.xuid); seenXUIDs.add(r.xuid) }
    }
  }

  let allyFriendIdx = 0
  let allyOtherIdx = 0
  let enemyIdx = 0
  for (const xuid of orderedXUIDs) {
    let token: SemanticToken
    const isMain = xuid === meXUID
    const teamSide = teamSideByXUID.get(xuid) ?? null
    const isAlly = !isMain && allyTeam != null && teamSide === allyTeam
    const gt = gamertagByXUID.get(xuid)
    const isFriend = !isMain && gt != null && friendSet.has(normalizeGamertagKey(gt))

    if (isMain) {
      token = MAIN_TOKEN
    } else if (isAlly && isFriend) {
      token = ALLY_FRIEND_TOKENS[allyFriendIdx % ALLY_FRIEND_TOKENS.length]
      allyFriendIdx++
    } else if (isAlly) {
      token = ALLY_OTHER_TOKENS[allyOtherIdx % ALLY_OTHER_TOKENS.length]
      allyOtherIdx++
    } else {
      token = ENEMY_TOKENS[enemyIdx % ENEMY_TOKENS.length]
      enemyIdx++
    }
    const hex = resolveToken(token)
    tokenByXUID.set(xuid, token)
    hexByXUID.set(xuid, hex)
    if (gt) {
      tokenByGamertag.set(gt, token)
      hexByGamertag.set(gt, hex)
    }
  }
  return { tokenByXUID, hexByXUID, tokenByGamertag, hexByGamertag }
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
