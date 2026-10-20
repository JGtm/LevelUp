/**
 * replayCamps.ts — LES CAMPS DU REJEU : qui y entre (le FILM), et comment ils s'appellent (la
 * FEUILLE de match).
 *
 * # L'ÉQUIPE EST CELLE DU FILM, ET D'AUCUNE AUTRE SOURCE
 *
 * Décision du 2026-10-06 (« une section sans équipe n'existe pas ; un joueur ne peut pas ne pas
 * avoir d'équipe dans un match »), dans la ligne de l'ADR 0034 D-9 (« équipe = le film seul ») et
 * de la règle « les vies anonymes n'existent pas ». Le désignateur que le roster publie
 * (`roster[].team`, schéma 57) est posé UNE FOIS sur chaque joueur par `buildPlayers`
 * (`ReplayPlayer.team`) ; `groupByCamp` en tire les camps de TOUS les regroupements du rejeu :
 * colonnes de fiches (`groupSeatsByTeam`), menu de point de vue de la frise et tables de l'onglet
 * Arsenal (`groupByTeam`).
 *
 * UN JOUEUR DONT LE FILM TAIT L'ÉQUIPE N'ENTRE DANS AUCUN CAMP. Ce n'est ni un camp de plus ni
 * un groupe « à part » : c'est un défaut de SOURCE, compté par la cuisson
 * (`coverage.seats.sansEquipe`) et corrigé là-bas. Aucun repli ne le range par la feuille.
 *
 * # LA FEUILLE NE FAIT QUE NOMMER
 *
 * Le côté de feuille (`team_side`, `t{N}`) que portent les membres d'un camp lui donne son NOM
 * (`campLabel` : Eagle, Cobra, ou le `team_name` d'un titre qui le publie), et rien d'autre : il
 * ne décide ni de l'appartenance, ni de l'encre allié / adverse, qui vient elle aussi du film
 * (`filmAllegiance.ts`, 2026-10-06). Ce module est le seul du rejeu à le lire — garde-rail
 * `replayCamps.guard.test.ts`.
 */
import type { MatchScoreboardRow } from '@/lib/api/types'
import { resolveKnownTeamLabel, type TeamNamingText } from '@/lib/halo/teamLabel'

/** Un camp : le désignateur du film qui le DÉCIDE, le côté de feuille qui le NOMME. */
export interface ReplayCamp {
  /** Le désignateur d'équipe que le film écrit (`roster[].team`) : la clé du camp, la seule. */
  team: number
  /**
   * Le côté de feuille (`t{N}`) de ses membres qui ont une ligne — celui de la majorité d'entre
   * eux (`campSideOf`). Il NOMME le camp, rien d'autre ; `null` quand aucun membre n'a de
   * ligne. Il ne décide ni qui est dans le camp, ni de son encre (`filmAllegiance.ts`).
   */
  side: string | null
}

/** Ce que ce module lit d'une ligne de feuille : son côté, et le nom qu'elle donne à l'équipe. */
export type CampSheetRow = Pick<MatchScoreboardRow, 'team_side' | 'team_name'>

/**
 * groupByCamp — range des éléments par camp du FILM.
 *
 * L'ORDRE DES CAMPS EST CELUI DES DÉSIGNATEURS (0, 1, …), stable d'une image et d'une cuisson à
 * l'autre ; dans un camp, l'ordre d'entrée est gardé. Un élément sans désignateur n'entre dans
 * AUCUN camp, et aucun camp n'est créé pour lui (cf. l'en-tête). `rowsOf` rend les lignes de
 * feuille d'un élément — un joueur en a une ou aucune, une place celles de ses occupants — et ne
 * sert qu'à NOMMER le camp (`side`).
 */
export function groupByCamp<T>(
  items: readonly T[],
  teamOf: (item: T) => number | undefined,
  rowsOf: (item: T) => ReadonlyArray<CampSheetRow | undefined>,
): Array<ReplayCamp & { members: T[] }> {
  const parCamp = new Map<number, T[]>()
  for (const item of items) {
    const team = teamOf(item)
    if (team === undefined) continue
    const membres = parCamp.get(team)
    if (membres) membres.push(item)
    else parCamp.set(team, [item])
  }
  return [...parCamp.entries()]
    .sort(([a], [b]) => a - b)
    .map(([team, members]) => ({ team, side: campSideOf(members.flatMap(rowsOf)), members }))
}

/**
 * campSideOf — le côté de feuille d'un camp : celui que porte la MAJORITÉ des lignes données ;
 * à égalité, celui qui a atteint ce compte le premier. `null` quand aucune ligne n'a de côté
 * (une chaîne vide est une absence : le DTO l'écrit pour un camp non résolu).
 *
 * UNE MAJORITÉ, PAS LE PREMIER VENU : la feuille peut contredire le film sur un joueur (la
 * cuisson le compte, `coverage.teams.contradiction`), et ce joueur-là ne doit pas renommer son
 * camp.
 */
export function campSideOf(rows: ReadonlyArray<CampSheetRow | undefined>): string | null {
  const comptes = new Map<string, number>()
  let meilleur: string | null = null
  let max = 0
  for (const row of rows) {
    const side = row?.team_side
    if (!side) continue
    const n = (comptes.get(side) ?? 0) + 1
    comptes.set(side, n)
    if (n > max) {
      max = n
      meilleur = side
    }
  }
  return meilleur
}

/**
 * campLabel — le NOM d'un camp, tel que toutes les surfaces du rejeu l'écrivent.
 *
 * La cascade du dépôt (`resolveKnownTeamLabel`) sur les lignes de feuille DE SON CÔTÉ — le
 * `team_name` publié, sinon le nom officiel du côté (Eagle, Cobra…), sinon « Équipe N » du
 * côté —, et pour PLANCHER « Équipe N » du DÉSIGNATEUR : un camp du film a toujours un numéro,
 * il n'est jamais « sans équipe ». `rows` peut être la feuille entière ou les lignes des seuls
 * membres : seules celles du côté du camp sont lues.
 */
export function campLabel(
  camp: ReplayCamp,
  rows: ReadonlyArray<CampSheetRow | undefined>,
  text: TeamNamingText,
): string {
  const side = camp.side
  const lignes = side
    ? rows.filter((row): row is CampSheetRow => row !== undefined && row.team_side === side)
    : []
  return resolveKnownTeamLabel(lignes, side, camp.team, text)
}
