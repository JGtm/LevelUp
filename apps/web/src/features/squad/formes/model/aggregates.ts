/**
 * aggregates.ts — LES PARTS ET LEURS PARITÉS (artefact 2ec1b8eb, fonctions
 * `mesure` / `agrege`).
 *
 * Pour un axe (un geste, ou les prises de socle), sur les matchs MESURÉS du
 * scope : ce que j'ai fait, ce que mon camp a fait, ce que le lobby a fait, et
 * les deux parts qui en découlent — dans mon équipe, dans le lobby — chacune
 * avec SA parité, son étendue match par match, et le compte de matchs au-dessus.
 *
 * DEUX DÉNOMINATEURS, TOUJOURS LES DEUX. C'est la correction que l'artefact
 * apporte : « être à la parité de son équipe pendant que l'équipe est sous celle
 * du lobby, ce n'est pas être en défaut, c'est jouer dans une équipe dominée sur
 * cet axe ». Une part sans son parèdre ne répond pas à la question posée.
 *
 * UN DÉNOMINATEUR NUL N'EST PAS UN ZÉRO : la part vaut `null` et la forme écrit
 * « non mesuré ». Un match où personne n'a touché l'axe n'entre ni dans
 * l'étendue ni dans le compte de matchs au-dessus de la parité.
 */
import type { SquadFormesBlock, SquadFormesMatch } from '@/lib/api/types'

import {
  average,
  axisValue,
  isMySide,
  lobbyOf,
  matchSizes,
  measuredMatches,
  parityOf,
  sharePct,
  type ShareAxis,
} from './access'

/** Les trois totaux d'un match sur un axe, et ses deux effectifs. */
export interface MatchMeasure {
  me: number
  team: number
  lobby: number
  teamSize: number
  lobbySize: number
}

/** mesure — les totaux d'un axe sur UN match (les deux camps). */
export function matchMeasure(
  match: SquadFormesMatch,
  mainXuid: string,
  axis: ShareAxis,
): MatchMeasure {
  const sizes = matchSizes(match)
  const out: MatchMeasure = { me: 0, team: 0, lobby: 0, teamSize: sizes.team, lobbySize: sizes.lobby }
  for (const p of lobbyOf(match)) {
    const v = axisValue(p, axis)
    out.lobby += v
    if (isMySide(match, p)) out.team += v
    if (p.xuid === mainXuid) out.me = v
  }
  return out
}

/** Une part, sa parité, son étendue par match et son compte au-dessus. */
export interface ShareSide {
  /** La part du dénominateur, `null` quand le dénominateur est nul. */
  pct: number | null
  /** La parité de ce dénominateur (100 / effectif moyen). */
  parity: number | null
  /** Le numérateur et le dénominateur bruts (infobulle). */
  value: number
  total: number
  /** L'étendue match par match : la plus faible et la plus forte des parts. */
  min: number | null
  max: number | null
  /** Matchs au-dessus de LEUR parité, sur les matchs où l'axe est mesuré. */
  above: number
  measured: number
}

/** L'agrégat d'un axe : les deux dénominateurs, et la part de mon camp. */
export interface AxisAggregate {
  team: ShareSide
  lobby: ShareSide
  /** La part du LOBBY que mon camp prend (le rapport de force). */
  teamShareOfLobbyPct: number | null
  teamTotal: number
  lobbyTotal: number
}

interface SideAccumulator {
  value: number
  total: number
  parts: number[]
  above: number
  sizes: number[]
}

function newSide(): SideAccumulator {
  return { value: 0, total: 0, parts: [], above: 0, sizes: [] }
}

function closeSide(acc: SideAccumulator): ShareSide {
  const parity = parityOf(average(acc.sizes) ?? 0)
  return {
    pct: sharePct(acc.value, acc.total),
    parity,
    value: acc.value,
    total: acc.total,
    min: acc.parts.length > 0 ? Math.min(...acc.parts) : null,
    max: acc.parts.length > 0 ? Math.max(...acc.parts) : null,
    above: acc.above,
    measured: acc.parts.length,
  }
}

/** agrege — l'agrégat d'un axe sur les matchs mesurés du scope. */
export function aggregateAxis(block: SquadFormesBlock, axis: ShareAxis): AxisAggregate {
  const main = block.main_xuid ?? ''
  const team = newSide()
  const lobby = newSide()
  let teamTotal = 0
  let lobbyTotal = 0
  for (const match of measuredMatches(block)) {
    const m = matchMeasure(match, main, axis)
    teamTotal += m.team
    lobbyTotal += m.lobby
    team.value += m.me
    team.total += m.team
    team.sizes.push(m.teamSize)
    lobby.value += m.me
    lobby.total += m.lobby
    lobby.sizes.push(m.lobbySize)
    if (m.team > 0) {
      const pct = (m.me / m.team) * 100
      team.parts.push(pct)
      const parity = parityOf(m.teamSize)
      if (parity != null && pct >= parity) team.above += 1
    }
    if (m.lobby > 0) {
      const pct = (m.me / m.lobby) * 100
      lobby.parts.push(pct)
      const parity = parityOf(m.lobbySize)
      if (parity != null && pct >= parity) lobby.above += 1
    }
  }
  return {
    team: closeSide(team),
    lobby: closeSide(lobby),
    teamShareOfLobbyPct: sharePct(teamTotal, lobbyTotal),
    teamTotal,
    lobbyTotal,
  }
}

/**
 * La part de MON CAMP dans le lobby sur un match — la valeur de la bande de
 * régularité. `null` quand personne n'a touché l'axe : une case « non mesurée »,
 * jamais une case à zéro pour cent.
 */
export function teamShareOfMatch(
  match: SquadFormesMatch,
  mainXuid: string,
  axis: ShareAxis,
): number | null {
  if (!match.measured) return null
  const m = matchMeasure(match, mainXuid, axis)
  return sharePct(m.team, m.lobby)
}

/** La part du joueur dans le lobby sur un match (même règle de `null`). */
export function myShareOfMatch(
  match: SquadFormesMatch,
  mainXuid: string,
  axis: ShareAxis,
): number | null {
  if (!match.measured) return null
  const m = matchMeasure(match, mainXuid, axis)
  return sharePct(m.me, m.lobby)
}

/** Le total d'un axe pour un joueur sur les matchs mesurés. */
export function playerTotal(block: SquadFormesBlock, xuid: string, axis: ShareAxis): number {
  let total = 0
  for (const match of measuredMatches(block)) {
    for (const p of lobbyOf(match)) {
      if (p.xuid === xuid) total += axisValue(p, axis)
    }
  }
  return total
}

/**
 * L'étendue d'un axe pour le joueur : son match le plus faible, son plus fort,
 * et sa MOYENNE PAR MATCH MESURÉ.
 *
 * PAR MATCH, JAMAIS PAR MINUTE (décision utilisateur du 2026-09-13) : la durée
 * ne normalise rien ici. Un match mesuré sans geste compte comme un zéro — c'est
 * une mesure, pas une absence.
 */
export function playerSpread(
  block: SquadFormesBlock,
  xuid: string,
  axis: ShareAxis,
): { min: number; max: number; mean: number; matches: number } | null {
  const values = measuredMatches(block).map((m) => {
    const row = lobbyOf(m).find((p) => p.xuid === xuid)
    return row ? axisValue(row, axis) : 0
  })
  if (values.length === 0) return null
  const sum = values.reduce((a, v) => a + v, 0)
  return {
    min: Math.min(...values),
    max: Math.max(...values),
    mean: sum / values.length,
    matches: values.length,
  }
}

/**
 * La répartition d'un axe entre l'escouade, le reste de mon camp et l'adversaire
 * — les segments de la piste du lobby.
 *
 * L'ADVERSAIRE EST COMPTÉ, JAMAIS NOMMÉ : il n'a ni couleur d'équipe ni
 * gamertag, seulement une hachure et un total. C'est la règle du bloc.
 */
export interface LobbyParts {
  bySquad: Record<string, number>
  /** Coéquipier de mon camp qui n'est pas dans l'escouade sélectionnée. */
  teamRest: number
  opponents: number
}

export function lobbyParts(
  matches: SquadFormesMatch[],
  squadXuids: string[],
  axis: ShareAxis,
): LobbyParts {
  const bySquad: Record<string, number> = {}
  for (const x of squadXuids) bySquad[x] = 0
  const out: LobbyParts = { bySquad, teamRest: 0, opponents: 0 }
  for (const match of matches) {
    for (const p of lobbyOf(match)) {
      const v = axisValue(p, axis)
      if (!isMySide(match, p)) {
        out.opponents += v
        continue
      }
      if (p.xuid in out.bySquad) out.bySquad[p.xuid] += v
      else out.teamRest += v
    }
  }
  return out
}
