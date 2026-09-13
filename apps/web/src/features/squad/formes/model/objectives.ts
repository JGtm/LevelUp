/**
 * objectives.ts — LES OBJECTIFS (artefact 2ec1b8eb, bloc 3).
 *
 * DEUX LECTURES, ET ELLES NE DISENT PAS LA MÊME CHOSE :
 *
 *   - PAR RÔLE (prendre / défendre / tenir) : la part réconcilie des modes qui
 *     n'ont pas les mêmes colonnes — les zones de Bastion et les drapeaux
 *     deviennent comparables, et la parité ne dépend plus du mode. Ce que cette
 *     lecture abandonne : la nature de l'action.
 *   - PAR COLONNE RÉELLE du mode : le geste, pas le profil. Trois retours de
 *     drapeau ne sont pas trois zones sécurisées.
 *
 * LA TABLE RÔLE -> GRANDEURS N'EST PAS ÉCRITE ICI : elle arrive du serveur, qui
 * la dérive de sa source unique (chaque colonne publiée porte son rôle et son
 * unité). Une table de colonnes côté web serait une seconde vérité, et elle
 * dériverait au premier mode ajouté.
 *
 * UNE DURÉE NE SE COMPARE QU'À SA PROPRE PARITÉ : les colonnes « tenir » sont en
 * secondes, les autres en actions. Elles ne s'additionnent jamais entre elles.
 */
import type {
  SquadFormesBlock,
  SquadFormesMatch,
  SquadFormesObjectiveColumn,
} from '@/lib/api/types'

import { average, matchSizes, parityOf, sharePct } from './access'

/** Les trois rôles, dans l'ordre de publication du serveur. */
export const OBJECTIVE_ROLES = ['take', 'defend', 'hold'] as const

export type ObjectiveRole = (typeof OBJECTIVE_ROLES)[number]

/** Les matchs du scope qui portent un objectif. */
export function objectiveMatches(block: SquadFormesBlock): SquadFormesMatch[] {
  return (block.matches ?? []).filter((m) => m.objective != null)
}

/** Les familles de mode à objectif rencontrées, dans l'ordre d'apparition. */
export function objectiveFamilies(block: SquadFormesBlock): string[] {
  const out: string[] = []
  for (const m of objectiveMatches(block)) {
    const fam = m.objective?.family
    if (fam && !out.includes(fam)) out.push(fam)
  }
  return out
}

/** Les matchs d'une famille de mode. */
export function matchesOfFamily(block: SquadFormesBlock, family: string): SquadFormesMatch[] {
  return objectiveMatches(block).filter((m) => m.objective?.family === family)
}

/** Les colonnes publiées par une famille (les mêmes pour tous ses matchs). */
export function columnsOfFamily(
  block: SquadFormesBlock,
  family: string,
): SquadFormesObjectiveColumn[] {
  return matchesOfFamily(block, family)[0]?.objective?.columns ?? []
}

/**
 * objectiveCell — la valeur d'un joueur sur une colonne d'un match, qui peut
 * valoir « non mesuré ».
 *
 * TOUTES LES COLONNES NE SE LISENT PAS PAREIL, ET C'EST LA COLONNE QUI LE DIT.
 * Les grandeurs de `match_objective_stats` viennent du sync : dès qu'un match a une ligne,
 * toutes ses colonnes ont une valeur, et un 0 y est une mesure. Les grandeurs
 * lues du FILM (les prises nettes de drapeau) n'existent que pour les matchs
 * dont l'artefact a été lu — le serveur les marque `optional` et n'écrit alors
 * AUCUNE clé. Les afficher à zéro dirait « il n'a rien pris » là où la vérité
 * est « on n'a pas regardé ».
 *
 * `null` = non mesuré, et la grille le rend en hachure.
 */
export function objectiveCell(
  match: SquadFormesMatch,
  xuid: string,
  column: SquadFormesObjectiveColumn,
): number | null {
  const row = (match.objective?.players ?? []).find((p) => p.xuid === xuid)
  const v = row?.values?.[column.key]
  if (v == null) return column.optional ? null : 0
  return v
}

/** L'agrégat d'un ensemble de colonnes : mes totaux, ceux de mon camp, du lobby. */
export interface ObjectiveAggregate {
  me: number
  team: number
  lobby: number
  myShareOfTeamPct: number | null
  myShareOfLobbyPct: number | null
  teamShareOfLobbyPct: number | null
  teamParity: number | null
  lobbyParity: number | null
  /** Étendue et compte au-dessus, par match, sur chacun des deux dénominateurs. */
  teamSpread: Spread
  lobbySpread: Spread
}

/** L'étendue d'une part match par match, et son compte au-dessus de la parité. */
export interface Spread {
  min: number | null
  max: number | null
  above: number
  measured: number
}

function emptySpread(): Spread {
  return { min: null, max: null, above: 0, measured: 0 }
}

function closeSpread(parts: number[], above: number): Spread {
  return {
    min: parts.length > 0 ? Math.min(...parts) : null,
    max: parts.length > 0 ? Math.max(...parts) : null,
    above,
    measured: parts.length,
  }
}

/**
 * aggregateColumns — l'agrégat de colonnes sur un ensemble de matchs. Les
 * colonnes passées DOIVENT partager leur unité (des actions, ou des secondes) :
 * additionner un temps de portage à des retours de drapeau ne voudrait rien dire.
 *
 * UN MATCH QUI NE MESURE PAS UNE COLONNE OPTIONNELLE SORT DE L'AGRÉGAT — de son
 * numérateur ET de son dénominateur. Les grandeurs lues du film n'existent que
 * pour les matchs dont l'artefact a été lu ; les compter à zéro sur les autres
 * ferait passer « on n'a pas regardé » pour « il n'a rien pris », et la part de
 * ce joueur baisserait d'autant. C'est le faux zéro que le serveur refuse déjà
 * en n'écrivant AUCUNE clé (cf. SquadFormesObjectiveColumn.optional).
 */
export function aggregateColumns(
  matches: SquadFormesMatch[],
  mainXuid: string,
  columns: SquadFormesObjectiveColumn[],
): ObjectiveAggregate {
  let me = 0
  let team = 0
  let lobby = 0
  const teamSizes: number[] = []
  const lobbySizes: number[] = []
  const teamParts: number[] = []
  const lobbyParts: number[] = []
  let teamAbove = 0
  let lobbyAbove = 0

  for (const match of matches) {
    if (!matchMeasuresAll(match, columns)) continue
    const sizes = matchSizes(match)
    teamSizes.push(sizes.team)
    lobbySizes.push(sizes.lobby)
    let mMe = 0
    let mTeam = 0
    let mLobby = 0
    for (const row of match.objective?.players ?? []) {
      const v = columns.reduce((a, c) => a + (row.values?.[c.key] ?? 0), 0)
      mLobby += v
      if (match.player_team != null && row.team_id === match.player_team) mTeam += v
      if (row.xuid === mainXuid) mMe += v
    }
    me += mMe
    team += mTeam
    lobby += mLobby
    if (mTeam > 0) {
      const pct = (mMe / mTeam) * 100
      teamParts.push(pct)
      const parity = parityOf(sizes.team)
      if (parity != null && pct >= parity) teamAbove += 1
    }
    if (mLobby > 0) {
      const pct = (mMe / mLobby) * 100
      lobbyParts.push(pct)
      const parity = parityOf(sizes.lobby)
      if (parity != null && pct >= parity) lobbyAbove += 1
    }
  }

  return {
    me,
    team,
    lobby,
    myShareOfTeamPct: sharePct(me, team),
    myShareOfLobbyPct: sharePct(me, lobby),
    teamShareOfLobbyPct: sharePct(team, lobby),
    teamParity: parityOf(average(teamSizes) ?? 0),
    lobbyParity: parityOf(average(lobbySizes) ?? 0),
    teamSpread: teamParts.length > 0 ? closeSpread(teamParts, teamAbove) : emptySpread(),
    lobbySpread: lobbyParts.length > 0 ? closeSpread(lobbyParts, lobbyAbove) : emptySpread(),
  }
}

/**
 * matchMeasuresAll — ce match mesure-t-il TOUTES les colonnes optionnelles de
 * l'ensemble ? Une colonne ordinaire est toujours mesurée dès que le match a une
 * feuille d'objectif ; une colonne optionnelle ne l'est que si au moins un joueur
 * en porte la clé.
 */
function matchMeasuresAll(
  match: SquadFormesMatch,
  columns: SquadFormesObjectiveColumn[],
): boolean {
  const players = match.objective?.players ?? []
  return columns.every(
    (c) => !c.optional || players.some((p) => p.values?.[c.key] != null),
  )
}

/**
 * aggregateRole — un rôle sur TOUS les matchs à objectif du scope, quelles que
 * soient leurs familles : c'est la réconciliation que permet le rôle.
 *
 * LES GRANDEURS OPTIONNELLES N'Y ENTRENT PAS, ET C'EST UNE DÉCISION. Un rôle
 * agrégé est une SOMME de grandeurs hétérogènes ramenées à une part ; y mêler
 * une grandeur mesurée sur trois matchs quand les autres le sont sur cinq
 * donnerait un total dont le dénominateur change d'un scope à l'autre, sans que
 * rien à l'écran ne le dise. Les prises nettes gardent donc leurs propres
 * cartes — la grille brute et l'écart par colonne, calculés sur les seuls matchs
 * mesurés — et la carte de rôle reste comparable d'une session à l'autre.
 */
export function aggregateRole(block: SquadFormesBlock, role: ObjectiveRole): ObjectiveAggregate {
  const matches = objectiveMatches(block).filter((m) =>
    (m.objective?.columns ?? []).some((c) => c.role === role && !c.optional),
  )
  // Chaque match n'additionne que SES colonnes du rôle : deux familles n'ont pas
  // les mêmes, et la somme se fait match par match dans aggregateColumns.
  const columns = new Map<string, SquadFormesObjectiveColumn>()
  for (const m of matches) {
    for (const c of m.objective?.columns ?? []) {
      if (c.role === role && !c.optional) columns.set(c.key, c)
    }
  }
  return aggregateColumns(matches, block.main_xuid ?? '', [...columns.values()])
}

/** Les segments de la piste du lobby pour un rôle (escouade / reste / adverse). */
export function roleLobbyParts(
  block: SquadFormesBlock,
  role: ObjectiveRole,
  squadXuids: string[],
): { bySquad: Record<string, number>; teamRest: number; opponents: number } {
  const bySquad: Record<string, number> = {}
  for (const x of squadXuids) bySquad[x] = 0
  let teamRest = 0
  let opponents = 0
  for (const match of objectiveMatches(block)) {
    // MÊME RÈGLE QUE aggregateRole : les grandeurs optionnelles restent dehors.
    // La piste du lobby est une répartition à 100 % — y verser une grandeur que
    // certains matchs ne mesurent pas décalerait les segments sans le dire.
    const cols = (match.objective?.columns ?? [])
      .filter((c) => c.role === role && !c.optional)
      .map((c) => c.key)
    if (cols.length === 0) continue
    for (const row of match.objective?.players ?? []) {
      const v = cols.reduce((a, c) => a + (row.values?.[c] ?? 0), 0)
      if (match.player_team == null || row.team_id !== match.player_team) {
        opponents += v
        continue
      }
      if (row.xuid in bySquad) bySquad[row.xuid] += v
      else teamRest += v
    }
  }
  return { bySquad, teamRest, opponents }
}
