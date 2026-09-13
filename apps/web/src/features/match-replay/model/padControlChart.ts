/**
 * padControlChart.ts — LA PROJECTION DU CONTRÔLE DES ARMES SPÉCIALES, une arme par ligne.
 *
 * LE TABLEAU ARMES-EN-COLONNES EST DEVENU UN GRAPHE (2026-09-03, retours utilisateur). La
 * question du bloc — « qui a tenu le lance-roquettes » — se lisait en balayant une colonne de
 * chiffres ; elle se lit maintenant EN UNE LIGNE : le socle à gauche, deux bâtons superposés
 * (un par camp), un segment par joueur du camp.
 *
 * UNE ARME, UNE BARRE (2026-09-13). Le graphe empilait UN BÂTON PAR CAMP sur chaque ligne :
 * une arme prise par les deux camps avait deux bâtons, une arme prise par un seul en avait un,
 * et l'utilisateur l'a dit — « pourquoi "Contrôle des armes spéciales" a plusieurs épaisseurs
 * de barres ? ». Désormais une arme = UNE barre, dont le RAIL vaut 100 % des occupations
 * NOMMÉES de ce socle : la question du bloc (« qui a tenu le lance-roquettes ») se lit en part
 * de contrôle. Le compte brut reste écrit dans chaque segment et dans son infobulle, et le
 * TOTAL de la ligne s'écrit à côté du nom de l'arme — sans quoi un 3-1 et un 30-10
 * dessineraient la même barre.
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

/** Un joueur dans la barre : son identité, son camp, ses prises, son encre et sa part. */
export interface PadBarSegment {
  xuid: string
  name: string
  /** Camp du joueur — sert à l'infobulle et au filet qui sépare les deux camps. */
  side: string | null
  /** Libellé du camp, déjà résolu par l'appelant. */
  sideLabel: string
  count: number
  /** Part de l'encre du camp dans le mélange (100 = l'encre pure). */
  tint: number
  /** L'encre finale, prête à poser (`color-mix` sur le fond de carte). */
  color: string
  /** Part du RAIL de la ligne (occupations nommées de ce socle), dans [0, 1]. */
  fraction: number
  /** Vrai sur le PREMIER segment d'un camp qui n'ouvre pas la barre : le filet se pose là. */
  startsSide: boolean
}

/** Une ligne : une arme, sa barre, son total, et ce qui n'a pas de ramasseur nommé. */
export interface PadBarRow {
  weapon: string
  label: string
  /** Prises NOMMÉES de ce socle — le dénominateur du rail. */
  total: number
  segments: PadBarSegment[]
  /** Occupations de CE socle sans ramasseur nommé. Jamais versées à un camp. */
  unnamed: number
}

/** Le graphe complet : ses lignes et les camps de sa légende. */
export interface PadBarModel {
  rows: PadBarRow[]
  /** Les camps présents, dans l'ordre d'affichage — la légende du bloc. */
  teams: { side: string | null; label: string }[]
}

/** L'encre la plus claire d'un camp, en pourcentage de l'encre pure. */
const MIN_TINT = 40

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
  const rows = input.control.weapons.map((weapon) => {
    const total = teams.reduce((sum, team) => sum + (team.total.byWeapon[weapon] ?? 0), 0)
    const segments: PadBarSegment[] = []
    for (const team of teams) {
      const partDuCamp = teamSegments(team, weapon, total, input)
      if (partDuCamp.length === 0) continue
      // Le filet ne se pose qu'ENTRE deux camps : jamais au bord gauche du rail.
      partDuCamp[0].startsSide = segments.length > 0
      segments.push(...partDuCamp)
    }
    return {
      weapon,
      label: input.weaponLabel(weapon),
      total,
      segments,
      unnamed: input.control.unnamedByWeapon[weapon] ?? 0,
    }
  })
  return {
    rows,
    teams: teams.map((team) => ({ side: team.side, label: input.teamLabel(team.side) })),
  }
}

/** Les segments d'un camp : ses joueurs qui ont pris CE socle, dans l'ordre du camp. */
function teamSegments(
  team: PadControlTeam,
  weapon: string,
  total: number,
  input: PadBarInput,
): PadBarSegment[] {
  const color = input.teamColor(team.side)
  const size = team.players.length
  return team.players
    .map((p, rank) => {
      const count = p.byWeapon[weapon] ?? 0
      const tint = padTint(rank, size)
      return {
        xuid: p.xuid,
        name: p.name,
        side: team.side,
        sideLabel: input.teamLabel(team.side),
        count,
        tint,
        // `color-mix` sur le FOND DE CARTE et non sur du blanc : l'éclaircissement doit tirer
        // vers la surface qui porte le graphe, sinon il vire au pastel en thème sombre.
        color: tint >= 100 ? color : `color-mix(in oklab, ${color} ${tint}%, var(--card))`,
        fraction: total > 0 ? count / total : 0,
        startsSide: false,
      }
    })
    .filter((s) => s.count > 0)
}
