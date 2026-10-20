/**
 * objectives.ts — LA LECTURE DU BLOC `formes_retenues` PAR LES CARTES D'OBJECTIF (Escouade ›
 * Contributions, Séries temporelles › Usages) : matchs à objectif, familles de mode, colonnes
 * publiées, valeur d'un joueur, agrégat camp contre lobby.
 *
 * LES RÔLES (prendre / défendre / tenir) réconcilient des modes qui n'ont pas les mêmes colonnes ;
 * LA COLONNE RÉELLE du mode garde le geste. Trois retours de drapeau ne sont pas trois zones
 * sécurisées.
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
 * `null` = grandeur absente du match : elle reste hors de toute somme, jamais un zéro.
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

/** L'agrégat d'un ensemble de colonnes : le total de mon camp, celui du lobby. */
export interface ObjectiveAggregate {
  team: number
  lobby: number
}

/**
 * aggregateColumns — l'agrégat de colonnes sur un ensemble de matchs. Les colonnes passées DOIVENT
 * partager leur unité (des actions, ou des secondes) : additionner un temps de portage à des
 * retours de drapeau ne voudrait rien dire.
 *
 * UN MATCH QUI NE MESURE PAS UNE COLONNE OPTIONNELLE SORT DE L'AGRÉGAT — pour TOUTES les colonnes
 * de l'ensemble. Les grandeurs lues du film n'existent que pour les matchs dont l'artefact a été
 * lu ; les compter à zéro sur les autres ferait passer « on n'a pas regardé » pour « il n'a rien
 * pris ». C'est le faux zéro que le serveur refuse déjà en n'écrivant AUCUNE clé
 * (cf. SquadFormesObjectiveColumn.optional).
 */
export function aggregateColumns(matches: SquadFormesMatch[], columns: SquadFormesObjectiveColumn[]): ObjectiveAggregate {
  let team = 0
  let lobby = 0
  for (const match of matches) {
    if (!matchMeasuresAll(match, columns)) continue
    for (const row of match.objective?.players ?? []) {
      const v = columns.reduce((a, c) => a + (row.values?.[c.key] ?? 0), 0)
      lobby += v
      if (match.player_team != null && row.team_id === match.player_team) team += v
    }
  }
  return { team, lobby }
}

/**
 * matchMeasuresAll — ce match mesure-t-il TOUTES les colonnes optionnelles de l'ensemble ? Une
 * colonne ordinaire est toujours mesurée dès que le match a une feuille d'objectif ; une colonne
 * optionnelle ne l'est que si au moins un joueur en porte la clé.
 */
function matchMeasuresAll(match: SquadFormesMatch, columns: SquadFormesObjectiveColumn[]): boolean {
  const players = match.objective?.players ?? []
  return columns.every((c) => !c.optional || players.some((p) => p.values?.[c.key] != null))
}