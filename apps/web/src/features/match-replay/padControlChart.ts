/**
 * padControlChart.ts — LA PROJECTION DU CONTRÔLE DES ARMES SPÉCIALES, une arme par ligne.
 *
 * LE TABLEAU ARMES-EN-COLONNES EST DEVENU UN GRAPHE (2026-09-03, retours utilisateur). La
 * question du bloc — « qui a tenu le lance-roquettes » — se lisait en balayant une colonne de
 * chiffres ; elle se lit maintenant EN UNE LIGNE : le socle à gauche, deux bâtons superposés
 * (un par camp), un segment par joueur du camp.
 *
 * L'ÉCHELLE EST COMMUNE À TOUTES LES LIGNES, et c'est le contraire de la grille des usages.
 * Ici toutes les colonnes portent la MÊME grandeur — un nombre de prises — donc une échelle par
 * ligne mentirait : deux bâtons de même longueur diraient deux fois « ce socle a été pris »,
 * alors que l'un vaut cinq prises et l'autre une. L'axe est entier : une demi-prise n'existe pas.
 *
 * LA TEINTE D'UN JOUEUR EST CELLE DE SON CAMP, ÉCLAIRCIE SELON SON RANG DANS LE CAMP, et ce rang
 * est celui du camp entier (pas celui des seuls preneurs de CE socle) : un joueur garde ainsi la
 * MÊME teinte d'une ligne à l'autre, ce qui est la seule façon de le suivre à l'œil sur dix
 * lignes. Le pas s'adapte à l'effectif — 100 % à 40 % de l'encre du camp, réparti sur les
 * joueurs — plutôt qu'un pas fixe qui écraserait les cinq derniers d'une équipe de huit sur la
 * même nuance.
 *
 * CE QUI N'A PAS DE RAMASSEUR NOMMÉ N'EST VERSÉ À PERSONNE : les occupations sans nom sont
 * annotées à DROITE de la ligne, hors des deux bâtons. Les verser à un camp au hasard rendrait
 * le graphe faux là où il est aujourd'hui seulement incomplet.
 *
 * Pur : aucun React, aucun hex, aucune langue — les noms d'arme, les libellés de camp et les
 * encres arrivent par l'appelant.
 */
import type { PadControl, PadControlTeam } from './padControlLogic'

/** Un joueur dans un bâton : son identité, ses prises, et l'éclaircissement de son encre. */
export interface PadStickSegment {
  xuid: string
  name: string
  count: number
  /** Part de l'encre du camp dans le mélange (100 = l'encre pure). */
  tint: number
  /** L'encre finale, prête à poser (`color-mix` sur le fond de carte). */
  color: string
  /** Part de la borne commune, dans [0, 1]. */
  fraction: number
}

/** Un bâton : un camp sur une ligne d'arme. */
export interface PadStick {
  side: string | null
  label: string
  segments: PadStickSegment[]
}

/** Une ligne : une arme, ses deux bâtons, et ce qui n'a pas de ramasseur nommé. */
export interface PadBarRow {
  weapon: string
  label: string
  sticks: PadStick[]
  /** Occupations de CE socle sans ramasseur nommé. Jamais versées à un camp. */
  unnamed: number
}

/** Le graphe complet : ses lignes, sa borne commune et ses graduations entières. */
export interface PadBarModel {
  rows: PadBarRow[]
  bound: number
  ticks: number[]
}

/** L'encre la plus claire d'un camp, en pourcentage de l'encre pure. */
const MIN_TINT = 40

/**
 * padTicks — les graduations ENTIÈRES d'un axe de prises, dix au plus.
 *
 * Au-delà, le pas s'élargit : un axe qui écrit vingt nombres sur 400 px ne se lit plus, et une
 * graduation sur deux suffit à situer une barre.
 */
export function padTicks(bound: number): number[] {
  const step = Math.max(1, Math.ceil(bound / 10))
  const out: number[] = []
  for (let k = 0; k <= bound; k += step) out.push(k)
  if (out[out.length - 1] !== bound) out.push(bound)
  return out
}

/** La teinte du joueur de rang `rank` dans un camp de `size` joueurs. */
export function padTint(rank: number, size: number): number {
  if (size <= 1) return 100
  return 100 - (rank * (100 - MIN_TINT)) / (size - 1)
}

/** Ce que l'appelant fournit pour habiller le graphe. */
export interface PadBarInput {
  control: PadControl
  /** Le nom d'affichage d'un socle (catalogue du document) — jamais son identifiant brut. */
  weaponLabel: (weapon: string) => string
  teamLabel: (side: string | null) => string
  /** L'encre pleine d'un camp (jetons `team-ally` / `team-enemy`). */
  teamColor: (side: string | null) => string
  /**
   * ORDRE D'AFFICHAGE DES CAMPS : plus petit d'abord (en haut du bâton). L'appelant y met le
   * camp du joueur de la page en premier — c'est sa page, c'est sa ligne du dessus.
   */
  teamRank: (side: string | null) => number
}

/**
 * buildPadControlBars — la projection complète.
 *
 * L'ORDRE DES ARMES EST CELUI DE `padControlLogic` (du socle le plus disputé au moins disputé) :
 * ce module ne le rejoue pas. Un socle dont aucune prise n'est attribuée n'a pas de ligne — il
 * n'a pas de colonne non plus dans le modèle amont, et sa ou ses occupations restent dans la
 * ventilation des manques, en pied de carte.
 */
export function buildPadControlBars(input: PadBarInput): PadBarModel {
  const teams = [...input.control.byTeam].sort(
    (a, b) => input.teamRank(a.side) - input.teamRank(b.side),
  )
  const bound = Math.max(
    1,
    ...input.control.weapons.flatMap((weapon) =>
      teams.map((team) => team.total.byWeapon[weapon] ?? 0),
    ),
  )
  const rows = input.control.weapons.map((weapon) => ({
    weapon,
    label: input.weaponLabel(weapon),
    unnamed: input.control.unnamedByWeapon[weapon] ?? 0,
    sticks: teams.map((team) => ({
      side: team.side,
      label: input.teamLabel(team.side),
      segments: stickSegments(team, weapon, bound, input.teamColor(team.side)),
    })),
  }))
  return { rows, bound, ticks: padTicks(bound) }
}

/** Les segments d'un bâton : les joueurs du camp qui ont pris CE socle, dans l'ordre du camp. */
function stickSegments(
  team: PadControlTeam,
  weapon: string,
  bound: number,
  color: string,
): PadStickSegment[] {
  const size = team.players.length
  return team.players
    .map((p, rank) => {
      const count = p.byWeapon[weapon] ?? 0
      const tint = padTint(rank, size)
      return {
        xuid: p.xuid,
        name: p.name,
        count,
        tint,
        // `color-mix` sur le FOND DE CARTE et non sur du blanc : l'éclaircissement doit tirer
        // vers la surface qui porte le graphe, sinon il vire au pastel en thème sombre.
        color: tint >= 100 ? color : `color-mix(in oklab, ${color} ${tint}%, var(--card))`,
        fraction: count / bound,
      }
    })
    .filter((s) => s.count > 0)
}
