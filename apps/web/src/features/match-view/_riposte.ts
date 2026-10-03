/**
 * _riposte.ts — LA PROJECTION DU BLOC « RIPOSTE », et elle ne rend aucun pixel.
 *
 * Le contrat sert deux listes indépendantes : un JOURNAL de morts (victime, tueur, instant,
 * vengeur, délai) et des COMPTES par joueur. Ce fichier les recoud en ce que les deux
 * graphes affichent : un camp, ses joueurs triés, et pour chacun les couples nommés qui
 * expliquent ses deux comptes.
 *
 * TROIS RÈGLES Y SONT ÉCRITES UNE FOIS, parce que les trois se lisent à l'œil sur l'écran
 * et se cassent en silence :
 *
 *   1. UNE SEULE ÉCHELLE — `scaleMax` est le plus grand compte du MATCH, pas du camp ni de
 *      la ligne. Deux graphes côte à côte avec deux bornes feraient lire « il riposte
 *      autant que l'autre » à un joueur qui riposte deux fois moins.
 *   2. LES DEUX CÔTÉS NE S'ADDITIONNENT PAS — à gauche un événement SUBI (ses morts vengées
 *      par son camp), à droite un événement PORTÉ (les ripostes qu'il a faites). On ne
 *      calcule donc jamais de solde, et le tri se fait sur les ripostes portées.
 *   3. MON CAMP À GAUCHE — l'ordre des camps vient du xuid du joueur de la page ; sans lui
 *      (ou hors du lobby), l'ordre est celui des `team_id`, et aucun camp n'est déclaré
 *      allié (encre neutre côté rendu).
 *
 * Des COMPTES, jamais un taux (décision D21) : rien ici ne divise quoi que ce soit.
 */
import type { MatchRiposteBlock, MatchRiposteDeath } from '@/lib/api/types'

/** Un couple nommé derrière un compte : « vengée par Ronan, 3,1 s ». */
export interface RiposteCouple {
  /** Gamertag de l'autre joueur du couple. Vide si le tableau des scores l'ignore. */
  name: string
  delaiMs: number
}

/** Une ligne de graphe : un joueur, ses deux comptes et les couples qui les expliquent. */
export interface RiposteRow {
  xuid: string
  gamertag: string
  /** Ses morts vengées par son camp (côté gauche, événement SUBI). */
  deathsAvenged: number
  /** Les ripostes qu'il a portées (côté droit, événement PORTÉ). */
  ripostes: number
  /** Qui l'a vengé, lui — une entrée par mort vengée. */
  avengedBy: RiposteCouple[]
  /** Qui il a vengé — une entrée par riposte portée. */
  avengedFor: RiposteCouple[]
}

/** Un camp et ses joueurs. `teamSide` est le format `t{N}` du reste de la page. */
export interface RiposteCamp {
  key: string
  teamSide: string | null
  rows: RiposteRow[]
}

export interface RiposteModel {
  camps: RiposteCamp[]
  /** Borne COMMUNE aux deux graphes et aux deux côtés de l'axe (jamais 0). */
  scaleMax: number
  /** Morts vengées du match — le numérateur du pied de carte. */
  avengedTotal: number
  measuredDeaths: number
  fenetreMs: number
}

/** Lignes de couples affichées avant le repli « +N ». */
export const RIPOSTE_COUPLE_LINES = 5

/** Le camp des joueurs que le tableau des scores ne rattache à aucun `team_id`. */
const CAMP_INCONNU = 'none'

/**
 * buildRiposteModel projette le bloc du contrat en camps prêts à rendre.
 *
 * `meXUID` ne sert qu'à l'ORDRE des camps (le mien d'abord) : il ne filtre rien et
 * n'ajoute aucun joueur.
 */
export function buildRiposteModel(
  block: MatchRiposteBlock,
  meXUID: string | null,
): RiposteModel {
  const deaths = block.deaths ?? []
  const players = block.players ?? []
  const avengedBy = new Map<string, RiposteCouple[]>()
  const avengedFor = new Map<string, RiposteCouple[]>()
  let avengedTotal = 0
  for (const d of deaths) {
    if (!d.avenged) continue
    avengedTotal += 1
    pushCouple(avengedBy, d.victim_xuid, d.avenger_gamertag, d)
    pushCouple(avengedFor, d.avenger_xuid, d.victim_gamertag, d)
  }

  let scaleMax = 1
  const byCamp = new Map<string, RiposteRow[]>()
  let myCamp: string | null = null
  for (const p of players) {
    const xuid = p.xuid ?? ''
    const camp = p.team_id == null ? CAMP_INCONNU : `t${p.team_id}`
    if (xuid !== '' && xuid === meXUID) myCamp = camp
    scaleMax = Math.max(scaleMax, p.deaths_avenged, p.ripostes)
    const rows = byCamp.get(camp) ?? []
    rows.push({
      xuid,
      gamertag: p.gamertag ?? '',
      deathsAvenged: p.deaths_avenged,
      ripostes: p.ripostes,
      avengedBy: avengedBy.get(xuid) ?? [],
      avengedFor: avengedFor.get(xuid) ?? [],
    })
    byCamp.set(camp, rows)
  }

  const camps = Array.from(byCamp.entries())
    .map(([key, rows]) => ({
      key,
      teamSide: key === CAMP_INCONNU ? null : key,
      rows: rows.sort(compareRows),
    }))
    .sort((a, b) => compareCamps(a.key, b.key, myCamp))

  return {
    camps,
    scaleMax,
    avengedTotal,
    measuredDeaths: block.measured_deaths,
    fenetreMs: block.fenetre_ms,
  }
}

/**
 * splitCouples coupe la liste des couples d'une infobulle : les premières lignes, puis le
 * reste en nombre. Une infobulle de trente lignes ne se lit pas et sort de l'écran.
 */
export function splitCouples(
  couples: RiposteCouple[],
  max = RIPOSTE_COUPLE_LINES,
): { shown: RiposteCouple[]; rest: number } {
  if (couples.length <= max) return { shown: couples, rest: 0 }
  return { shown: couples.slice(0, max), rest: couples.length - max }
}

/**
 * Part (0..1) d'un compte sur l'échelle commune. Sortie bornée : une barre ne déborde
 * jamais de sa demi-largeur, même si le contrat servait un compte incohérent.
 */
export function riposteFraction(count: number, scaleMax: number): number {
  if (scaleMax <= 0) return 0
  return Math.max(0, Math.min(1, count / scaleMax))
}

function pushCouple(
  into: Map<string, RiposteCouple[]>,
  xuid: string | undefined,
  name: string | undefined,
  death: MatchRiposteDeath,
): void {
  if (!xuid) return
  const list = into.get(xuid) ?? []
  list.push({ name: name ?? '', delaiMs: death.delai_ms ?? 0 })
  into.set(xuid, list)
}

/** Tri d'un camp : ripostes portées décroissantes, puis morts vengées, puis le nom. */
function compareRows(a: RiposteRow, b: RiposteRow): number {
  if (a.ripostes !== b.ripostes) return b.ripostes - a.ripostes
  if (a.deathsAvenged !== b.deathsAvenged) return b.deathsAvenged - a.deathsAvenged
  return a.gamertag.localeCompare(b.gamertag)
}

/** Mon camp d'abord, le camp inconnu en dernier, les autres par `team_id` croissant. */
function compareCamps(a: string, b: string, myCamp: string | null): number {
  if (a === b) return 0
  if (myCamp != null) {
    if (a === myCamp) return -1
    if (b === myCamp) return 1
  }
  if (a === CAMP_INCONNU) return 1
  if (b === CAMP_INCONNU) return -1
  return a.localeCompare(b, undefined, { numeric: true })
}
