/**
 * seatLogic — LE SIÈGE : la fiche suit l'OCCUPANT, pas le joueur (retour user 2026-09-02),
 * et le siège vient du FILM (lot 1.9.14, 2026-09-15).
 *
 * LE CAS QUI A IMPOSÉ CE MODÈLE (match témoin 1b2d9e08) : Winterhawk7676 quitte à 22:31:14,
 * 343 Razzle le remplace À LA MÊME SECONDE. Une fiche par JOUEUR donnait 7 fiches vivantes
 * plus une fiche fantôme « hors film » jusqu'à la fin — alors que la partie reste un 4v4.
 * LE SIÈGE est l'unité stable : un 4v4 a huit sièges, et la fiche d'un siège montre son
 * occupant À L'IMAGE LUE — Winterhawk avant le relais, Razzle après. Et ça chaîne : un
 * même siège peut changer d'occupant des dizaines de fois (A → bot → B → …).
 *
 * # CE QUE LE LOT 1.9.14 CHANGE, ET POURQUOI
 *
 * L'APPARIEMENT NE SE FAIT PLUS ICI. Ce fichier construisait lui-même les relais, par
 * appariement ordinal sur la PARTICIPATION API (`left_in_progress` / `joined_in_progress`
 * et leurs horodatages). Le décodeur lit désormais l'index de film de CHAQUE entité — les
 * remplaçants compris (lot 1.7) — et l'artefact publie le siège de chaque joueur
 * (`roster[].seat`) avec sa provenance (`roster[].seatSource` : `lu` quand le film écrit la
 * reprise, `apparie` quand l'appariement ordinal l'a déduite, côté Go, nommé et compté au
 * registre des replis). Refaire le calcul ici en donnerait une seconde version, sur une autre
 * source, sans compteur — exactement la double vérité que ce chantier ferme ailleurs.
 *
 * LA FICHE N'EXISTE QUE SI QUELQU'UN L'OCCUPE (constat user du 2026-09-15 : « on n'a pas de
 * raison d'afficher les joueurs qui ne jouent pas à l'instant T »). La PRÉSENCE d'un occupant
 * est l'enveloppe de ses vies, du début de la première à la fin de la dernière — un trou de
 * réapparition est DANS la présence, c'est `playerStateAt` qui dit mort ou vivant. Hors de
 * cette enveloppe, le siège n'affiche cet occupant ni avant ni après. Mesure qui l'a commandé :
 * sur `bcb6d393` (CTF 4v4) le document porte ONZE entrées de roster pour HUIT occupants
 * simultanés au plus — trois fiches tenaient l'écran du début à la fin sans jamais jouer
 * ensemble.
 *
 * LE SIÈGE EN TRANSITION GARDE SA FICHE, MARQUÉE. Entre le départ d'un occupant et l'arrivée
 * de son successeur, la fiche reste en place et porte « A quitté » : la vider ferait sauter la
 * grille d'un cran puis revenir. Quand plus AUCUN successeur ne vient, la fiche disparaît — il
 * y a vraiment un joueur de moins.
 */
import type { ReplayRosterEntry } from '@/lib/api/types'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { rosterEntryKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { trackWindow } from '../../../lib/replay/replayLogic'

/**
 * EntreeAvecSiege — l'entrée de roster PLUS les deux champs que le lot 1.9.14 publie et que
 * `generated.ts` ne porte pas encore.
 *
 * ELLE DISPARAÎT À LA MONTÉE DE SCHÉMA DE LA VAGUE : `openapi.yaml` et `generated.ts` sont
 * régénérés une seule fois, par le pilote, à la fusion (règle de la vague 2) ; `ReplayRosterEntry`
 * portera alors `seat` et `seatSource`, et cette intersection n'aura plus d'objet. Elle est
 * écrite ici plutôt qu'au contrat zod parce que le contrat valide la FORME du document, pas
 * l'intérieur d'un calque.
 */
type EntreeAvecSiege = ReplayRosterEntry & { seat?: number; seatSource?: string }

/** Un occupant d'un siège : le joueur, et l'enveloppe de sa présence en images. */
export interface SeatOccupant {
  player: ReplayPlayer
  /** Première image de sa présence (début de sa première vie). */
  fromFrame: number
  /** Dernière image de sa présence (fin de sa dernière vie). */
  toFrame: number
  /**
   * Vrai quand le siège de cet occupant vient d'un APPARIEMENT ordinal côté Go, et non de
   * l'index que le film écrit (`seatSource === 'apparie'`). C'est une déduction, et un rendu
   * qui voudrait la distinguer d'une lecture a ici de quoi le faire.
   */
  apparie: boolean
}

/** Un siège : la suite de ses occupants, dans l'ordre du temps. */
export interface ReplaySeat {
  /** Clé stable de rendu. */
  key: string
  /** Le siège tel que le document le publie — l'ordre d'affichage en dépend. */
  seat: number
  /** Libellé de camp de la feuille de match, pour la cascade de `resolveTeamLabel`. */
  side: string | null
  /**
   * Clé de REGROUPEMENT des colonnes. Le camp du FILM (`f<designateur>`) quand le document le
   * porte, celui de la feuille (`s:<team_side>`) sinon — et c'est l'ordre voulu (V4 : le film
   * est la seule source d'équipe du rejeu ; la feuille reste le repli, affiché comme tel par
   * `side`). Un remplaçant que la feuille ne porte pas rejoint ainsi la bonne colonne au lieu
   * de tomber « sans équipe ».
   */
  teamKey: string
  occupants: SeatOccupant[]
}

/** Ce qu'un siège montre à une image. */
export type SeatStateKind = 'present' | 'parti' | 'absent'

/**
 * SeatReading — l'état d'un siège à une image : `present` (l'occupant joue), `parti` (il est
 * sorti, un successeur est attendu — la fiche reste, marquée), `absent` (personne : aucune
 * fiche).
 */
export interface SeatReading {
  player: ReplayPlayer | null
  kind: SeatStateKind
}

/**
 * seatOccupantAt — QUI TIENT CETTE FICHE À CETTE IMAGE.
 *
 * TROIS ÉTATS, ET LEUR FRONTIÈRE EST CE QUE LE LOT A DÛ TRANCHER :
 *
 *	present  une vie de l'occupant couvre l'image ; OU sa présence est finie et AUCUN
 *	         successeur ne vient sur ce siège — il est mort sans réapparaître, ou « hors
 *	         film », et la fiche le dit déjà (retour user du 2026-09-02). Sa fiche RESTE ;
 *	parti    sa présence est finie ET un successeur est attendu : la fiche reste, marquée ;
 *	absent   l'occupant n'est pas encore arrivé, ou son successeur a pris la place.
 *
 * MOURIR N'EST PAS PARTIR, ET LES CONFONDRE ÉTAIT LA FAUTE DE LA PREMIÈRE ÉCRITURE DE CE
 * FICHIER : borner la fiche à la dernière vie faisait disparaître, en fin de partie, tous ceux
 * qui meurent sans revenir — neuf tests d'interface l'ont montré. Le film ne distingue pas un
 * départ d'une mort définitive ; ce qui le distingue ici, c'est qu'un AUTRE occupant prenne le
 * siège. Sans successeur, on ne retire rien.
 *
 * CE QUE LA RÈGLE RETIRE VRAIMENT DE L'ÉCRAN, c'est le joueur qui n'est PAS ENCORE ARRIVÉ
 * (aujourd'hui sa fiche vide tient la grille depuis la première image) et celui qu'un
 * remplaçant a relevé.
 */
export function seatOccupantAt(seat: ReplaySeat, frame: number): SeatReading {
  let sorti = -1
  for (let i = 0; i < seat.occupants.length; i++) {
    const o = seat.occupants[i]
    if (frame < o.fromFrame) break // les occupants suivants ne sont pas encore arrivés
    if (frame <= o.toFrame) return { player: o.player, kind: 'present' }
    sorti = i
  }
  if (sorti < 0) return { player: null, kind: 'absent' }
  const attendu = sorti + 1 < seat.occupants.length
  return { player: seat.occupants[sorti].player, kind: attendu ? 'parti' : 'present' }
}

/**
 * buildSeats — les sièges, lus au document.
 *
 * Les joueurs sans entrée de roster (le film les nomme par leurs seules vies) gardent chacun
 * leur siège, sous une clé qui leur est propre : on ne les chaîne à personne. Un joueur SANS
 * AUCUNE VIE n'entre nulle part — il n'occupe de siège à aucune image, et lui en donner un
 * remettrait à l'écran la fiche vide que ce lot retire.
 *
 * L'ORDRE DES SIÈGES EST CELUI DU DOCUMENT (numéro de siège croissant) : stable d'une image à
 * l'autre et d'une cuisson à l'autre, ce que l'ordre d'apparition des joueurs n'était pas.
 */
export function buildSeats(players: readonly ReplayPlayer[], doc: ReplayDocumentReady): ReplaySeat[] {
  const parIdentite = new Map<string, EntreeAvecSiege>()
  for (const e of doc.roster ?? []) {
    const cle = rosterEntryKey(e)
    if (cle) parIdentite.set(cle, e as EntreeAvecSiege)
  }
  const sieges = new Map<string, ReplaySeat>()
  for (const p of players) {
    const presence = enveloppeDePresence(p)
    if (!presence) continue // aucune vie : aucune fiche, à aucune image
    const e = parIdentite.get(p.xuid)
    const numero = e?.seat ?? e?.filmIndex ?? -1
    const cle = numero >= 0 ? `siege:${numero}` : `joueur:${p.xuid}`
    let s = sieges.get(cle)
    if (!s) {
      s = { key: cle, seat: numero, side: null, teamKey: '', occupants: [] }
      sieges.set(cle, s)
    }
    s.occupants.push({ ...presence, player: p, apparie: e?.seatSource === 'apparie' })
    if (s.side === null) s.side = p.board?.team_side ?? null
    if (s.teamKey === '') s.teamKey = cleDeCamp(e, p)
  }
  const out = [...sieges.values()]
  for (const s of out) s.occupants.sort((a, b) => a.fromFrame - b.fromFrame)
  return out.sort((a, b) => {
    const ra = rangDeSiege(a)
    const rb = rangDeSiege(b)
    return ra !== rb ? ra - rb : a.key.localeCompare(b.key)
  })
}

/**
 * rangDeSiege — la place d'un siège dans la colonne. UN SIÈGE NON LU PASSE EN DERNIER, jamais
 * en tête : `seat` vaut `-1` quand le document ne nomme pas ce joueur, et trier sur ce `-1`
 * mettrait l'inconnu avant tous les autres.
 */
function rangDeSiege(s: ReplaySeat): number {
  return s.seat < 0 ? Number.MAX_SAFE_INTEGER : s.seat
}

/**
 * cleDeCamp — le camp du FILM d'abord, celui de la feuille ensuite, rien en dernier.
 *
 * `team` vaut `-1` quand le film dit « aucune équipe » (FFA) : c'est une LECTURE, elle
 * regroupe comme une autre. Le champ ABSENT, lui, est un silence — on retombe alors sur la
 * feuille.
 */
function cleDeCamp(e: EntreeAvecSiege | undefined, p: ReplayPlayer): string {
  if (e?.team !== undefined && e.team !== null) return `f${e.team}`
  const side = p.board?.team_side
  return side != null ? `s:${side}` : ''
}

/** L'enveloppe des vies d'un joueur, ou `null` quand il n'en a aucune. */
function enveloppeDePresence(p: ReplayPlayer): { fromFrame: number; toFrame: number } | null {
  if (p.lives.length === 0) return null
  let from = Number.POSITIVE_INFINITY
  let to = Number.NEGATIVE_INFINITY
  for (const life of p.lives) {
    const w = trackWindow(life)
    if (w.start < from) from = w.start
    if (w.end > to) to = w.end
  }
  return { fromFrame: from, toFrame: to }
}

/** Les sièges rangés par camp — le pendant de `groupByTeam`, sur l'unité SIÈGE. */
export interface ReplaySeatGroup {
  side: string | null
  seats: ReplaySeat[]
}

/**
 * groupSeatsByTeam range les sièges par camp, LE FILM D'ABORD (cf. `ReplaySeat.teamKey`).
 *
 * `side` du groupe est le premier libellé de feuille qu'un de ses sièges porte : c'est lui que
 * `resolveTeamLabel` consomme. Un camp que la feuille ne nomme pas garde `null` — le libellé
 * numéroté d'`i18n` s'en charge, et aucun camp n'est inventé.
 */
export function groupSeatsByTeam(seats: readonly ReplaySeat[]): ReplaySeatGroup[] {
  const groups = new Map<string, { side: string | null; seats: ReplaySeat[]; rang: number }>()
  for (const s of seats) {
    let g = groups.get(s.teamKey)
    if (!g) {
      g = { side: s.side, seats: [], rang: groups.size }
      groups.set(s.teamKey, g)
    }
    if (g.side === null) g.side = s.side
    g.seats.push(s)
  }
  return [...groups.entries()]
    .sort(([ka, a], [kb, b]) => ordreDesCamps(ka, a.side, a.rang, kb, b.side, b.rang))
    .map(([, g]) => ({ side: g.side, seats: g.seats }))
}

/**
 * ordreDesCamps — l'ordre des colonnes, STABLE d'une image à l'autre.
 *
 * Par le libellé de feuille quand les deux camps en ont un (l'ordre d'avant ce lot, celui que
 * l'utilisateur connaît) ; par la clé de camp sinon ; un camp sans clé passe en dernier.
 */
function ordreDesCamps(
  ka: string,
  sa: string | null,
  ra: number,
  kb: string,
  sb: string | null,
  rb: number,
): number {
  if (ka === '' !== (kb === '')) return ka === '' ? 1 : -1
  if (sa !== null && sb !== null && sa !== sb) return sa.localeCompare(sb)
  if (ka !== kb) return ka.localeCompare(kb)
  return ra - rb
}
