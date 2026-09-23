/**
 * seatLogic — LA PLACE : une fiche par place, et la place vient du FILM (lot 1.9.14 ; refondu au
 * lot M2.4 de la campagne « retours rejeu », 2026-09-23, sur la RÈGLE DES PLACES).
 *
 * # LA RÈGLE DES PLACES (utilisateur, 2026-09-23 — elle prime sur toute autre lecture)
 *
 * « quand un joueur part, il libère la place de sa fiche de joueur pour son remplaçant [...] le
 * nombre de joueur dans un match est fini, il y a un maximum. » Une équipe a un nombre FINI de
 * places ; une fiche = une place ; le remplaçant (bot ou humain) prend la place du partant ;
 * jamais plus de fiches que de places ; un joueur parti n'est JAMAIS affiché.
 *
 * # CE QUI VIENT DU DOCUMENT, ET CE QUE CE FICHIER NE CALCULE PAS
 *
 * Depuis le schéma 69, chaque entrée de roster publie sa PLACE (`seat`, et sa provenance
 * `seatSource`), son ÉQUIPE (`team`, celle de son entité `ti=9`) et sa PRÉSENCE (`presence` :
 * des intervalles `{from, to, toMax?}` en images — `to` est la dernière image CERTAINE,
 * `toMax` la dernière où il PEUT encore être là, déjà bornée par la cuisson à la veille de
 * l'arrivée de son successeur). Le chaînage des remplaçants, la borne au successeur et leurs
 * replis se font à la cuisson, nommés et comptés (`coverage.seats`) : ce fichier LIT. Refaire le
 * calcul ici en donnerait une seconde version, sur une autre source, sans compteur.
 *
 * # CE QU'UNE PLACE MONTRE À L'IMAGE LUE (décisions Q20 à Q22)
 *
 *	present          un occupant dont la présence couvre l'image, et qui a déjà un corps — mort
 *	                 ou vivant, c'est la fiche qui le dit : MOURIR N'EST PAS PARTIR, un trou de
 *	                 réapparition est DANS la présence ;
 *	pasEncoreApparu  (Q21) il tient la place, et aucune de ses vies n'a encore commencé ;
 *	vide             (Q20) personne : la place reste VISIBLE, vide — entre un partant et son
 *	                 remplaçant, ou quand aucun remplaçant ne vient. Un parti n'y est jamais
 *	                 affiché, et la grille ne saute pas d'un cran.
 *
 * Un départ pendant la mort (Q22) sort à la dernière image que la présence publiée autorise
 * (`toMax`) : la première image-clé où l'entité n'est plus là, ou l'arrivée du remplaçant.
 *
 * # LES DOCUMENTS QUI NE PUBLIENT PAS DE PRÉSENCE (repli daté, cf. `presencesParLesVies`)
 *
 * Un artefact antérieur au schéma 69 n'en porte aucune : la présence y est l'enveloppe des vies,
 * et le dernier occupant de chaque place la tient jusqu'à la fin — la règle que la cuisson
 * applique elle-même quand elle n'a lu aucune entité. Le choix se fait au niveau du DOCUMENT,
 * jamais entrée par entrée.
 */
import type { ReplayDocumentReady, ReplayRosterEntryReady } from '../../../lib/replay/replayNormalize'
import { rosterEntryKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { trackWindow } from '../../../lib/replay/replayLogic'

/** Un intervalle de présence, en images : certain jusqu'à `to`, affiché jusqu'à `toMax`. */
export interface PresenceSpan {
  from: number
  to: number
  toMax: number
}

/** Un occupant d'une place : le joueur, et les intervalles pendant lesquels il la tient. */
export interface SeatOccupant {
  player: ReplayPlayer
  /** Ses intervalles de présence, dans l'ordre du temps. */
  presence: PresenceSpan[]
  /**
   * Vrai quand la place de cet occupant vient d'une DÉDUCTION de la cuisson — le chaînage par
   * équipe (`apparie`) ou une place ouverte sous la capacité estimée (`ouverte`) — et non de
   * ce que le film écrit (`lu`, `tirs`). Un rendu qui voudrait les distinguer a de quoi le faire.
   */
  deduite: boolean
}

/** Une place : la suite de ses occupants, dans l'ordre du temps. */
export interface ReplaySeat {
  /** Clé stable de rendu. */
  key: string
  /** La place telle que le document la publie — l'ordre d'affichage en dépend. */
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

/** Ce qu'une place montre à une image (cf. l'en-tête). */
export type SeatStateKind = 'present' | 'pasEncoreApparu' | 'vide'

/** L'état d'une place à une image : son occupant (nul quand elle est vide) et ce qu'il y fait. */
export interface SeatReading {
  player: ReplayPlayer | null
  kind: SeatStateKind
}

/**
 * seatOccupantAt — QUI TIENT CETTE PLACE À CETTE IMAGE : l'occupant dont une présence couvre
 * l'image, et personne d'autre. Les présences d'une même place ne se recouvrent pas (la cuisson
 * les borne au successeur) : la lecture rend UN occupant, ou la place vide.
 *
 * LA RÈGLE « PRÉSENT SANS SUCCESSEUR » N'EXISTE PLUS : c'est elle qui gardait un parti affiché
 * faute de remplaçant sur SA place (`b1ad85eb`, rapport des équipes). Qu'un occupant reste
 * jusqu'au bout se lit dans sa présence, pas dans l'absence d'un suivant.
 */
export function seatOccupantAt(seat: ReplaySeat, frame: number): SeatReading {
  for (const o of seat.occupants) {
    const span = o.presence.find((p) => frame >= p.from && frame <= p.toMax)
    if (span === undefined) continue
    return { player: o.player, kind: dejaApparu(o.player, span, frame) ? 'present' : 'pasEncoreApparu' }
  }
  return { player: null, kind: 'vide' }
}

/**
 * dejaApparu — une vie du joueur a commencé DANS cet intervalle de présence, au plus tard à
 * cette image. Sinon il tient la place sans corps : « pas encore apparu » (Q21) — au coup
 * d'envoi, ou à son arrivée, avant sa première apparition.
 */
function dejaApparu(p: ReplayPlayer, span: PresenceSpan, frame: number): boolean {
  return p.lives.some((life) => {
    const w = trackWindow(life)
    return w.start <= frame && w.end >= span.from
  })
}

/**
 * buildSeats — les places, lues au document, et leurs occupants.
 *
 * UNE ENTRÉE QUE SA PRÉSENCE NE MONTRE À AUCUNE IMAGE N'ENTRE NULLE PART : un joueur parti avant
 * le coup d'envoi, ou jamais présent, ne tient aucune place. Un joueur que le film nomme par ses
 * seules vies, sans entrée de roster, garde sa place propre (`joueur:<xuid>`), présent sur
 * l'enveloppe de ses vies : on ne le chaîne à personne.
 *
 * L'ORDRE DES PLACES EST CELUI DU DOCUMENT (numéro de place croissant) : stable d'une image à
 * l'autre et d'une cuisson à l'autre, ce que l'ordre d'apparition des joueurs n'était pas.
 */
export function buildSeats(players: readonly ReplayPlayer[], doc: ReplayDocumentReady): ReplaySeat[] {
  const parIdentite = new Map<string, ReplayRosterEntryReady>()
  for (const e of doc.roster) {
    const cle = rosterEntryKey(e)
    if (cle) parIdentite.set(cle, e)
  }
  const publiees = publieDesPresences(doc)
  // LA TABLE DE TRADUCTION FEUILLE -> FILM, construite AVANT la boucle (cf. `campsParCote`) :
  // sans elle, une place dont le film tait l'équipe partait dans un espace de clés distinct et
  // formait un SECOND groupe portant le MÊME libellé.
  const parCote = campsParCote(players, parIdentite)
  const places = new Map<string, ReplaySeat>()
  for (const p of players) {
    const e = parIdentite.get(p.xuid)
    const presence = publiees && e ? presenceDeLEntree(e) : enveloppeDesVies(p)
    if (presence.length === 0) continue // aucune présence : aucune place, à aucune image
    const numero = e?.seat ?? e?.filmIndex ?? -1
    const cle = numero >= 0 ? `siege:${numero}` : `joueur:${p.xuid}`
    let s = places.get(cle)
    if (!s) {
      s = { key: cle, seat: numero, side: null, teamKey: '', occupants: [] }
      places.set(cle, s)
    }
    s.occupants.push({ player: p, presence, deduite: estDeduite(e) })
    if (s.side === null) s.side = p.board?.team_side ?? null
    if (s.teamKey === '') s.teamKey = cleDeCamp(e, p, parCote)
  }
  const out = [...places.values()]
  for (const s of out) s.occupants.sort((a, b) => a.presence[0].from - b.presence[0].from)
  if (!publiees) presencesParLesVies(out, doc.frameCount - 1)
  return out.sort((a, b) => {
    const ra = rangDePlace(a)
    const rb = rangDePlace(b)
    return ra !== rb ? ra - rb : a.key.localeCompare(b.key)
  })
}

/** Les provenances de place qui sont des DÉDUCTIONS de la cuisson (cf. `SeatOccupant.deduite`). */
const PROVENANCES_DEDUITES: ReadonlySet<string> = new Set(['apparie', 'ouverte'])

function estDeduite(e: ReplayRosterEntryReady | undefined): boolean {
  return e?.seatSource !== undefined && PROVENANCES_DEDUITES.has(e.seatSource)
}

/**
 * publieDesPresences — le document publie-t-il la présence de ses occupants ? Oui dès qu'UNE
 * entrée en porte une : la cuisson du schéma 69 en donne une à toute entrée qui a une vie, une
 * entité ou une déclaration de bot. Non sur un artefact antérieur, qui n'en porte aucune.
 */
function publieDesPresences(doc: ReplayDocumentReady): boolean {
  return doc.roster.some((e) => e.presence.length > 0)
}

/** La présence publiée d'une entrée (schéma 69), dans l'ordre du temps. */
function presenceDeLEntree(e: ReplayRosterEntryReady): PresenceSpan[] {
  return e.presence
    .map((p) => ({ from: p.from, to: p.to, toMax: Math.max(p.to, p.toMax ?? p.to) }))
    .sort((a, b) => a.from - b.from)
}

/**
 * presencesParLesVies — LE REPLI DES DOCUMENTS QUI NE PUBLIENT PAS DE PRÉSENCE : chaque occupant
 * est présent sur l'enveloppe de ses vies, et le DERNIER occupant de chaque place la tient
 * jusqu'à la fin — mourir n'est pas partir, et sans présence publiée rien ne dit qui est parti.
 * C'est la règle que la cuisson applique elle-même quand elle n'a lu aucune entité
 * (`repli_presence_par_enveloppe_des_vies`), appliquée ici aux artefacts cuits avant elle.
 *
 * KILL-SWITCH DATÉ (modèle `platform/duckdb/shared_reader_legacy.go`) :
 *   - bascule du défaut : 2026-09-23 — depuis le schéma 69, la présence vient du document, et
 *     ce repli ne sert plus aucun artefact qui la publie (cf. `publieDesPresences`) ;
 *   - retrait cible : 2026-12-01 ;
 *   - critère mesurable : plus aucun artefact servi sans `roster[].presence` (schéma < 69),
 *     c'est-à-dire la republication complète du parc (vague D de la campagne).
 */
function presencesParLesVies(places: readonly ReplaySeat[], derniere: number): void {
  for (const s of places) {
    const dernier = s.occupants[s.occupants.length - 1]
    const span = dernier?.presence[dernier.presence.length - 1]
    if (span !== undefined && span.toMax < derniere) span.toMax = derniere
  }
}

/** L'enveloppe des vies d'un joueur — un intervalle, ou aucun quand il n'a pas de vie. */
function enveloppeDesVies(p: ReplayPlayer): PresenceSpan[] {
  if (p.lives.length === 0) return []
  let from = Number.POSITIVE_INFINITY
  let to = Number.NEGATIVE_INFINITY
  for (const life of p.lives) {
    const w = trackWindow(life)
    if (w.start < from) from = w.start
    if (w.end > to) to = w.end
  }
  return [{ from, to, toMax: to }]
}

/**
 * rangDePlace — la place d'une fiche dans la colonne. UNE PLACE NON LUE PASSE EN DERNIER, jamais
 * en tête : `seat` vaut `-1` quand le document ne nomme pas ce joueur, et trier sur ce `-1`
 * mettrait l'inconnu avant tous les autres.
 */
function rangDePlace(s: ReplaySeat): number {
  return s.seat < 0 ? Number.MAX_SAFE_INTEGER : s.seat
}

/**
 * campsParCote — LA TABLE QUI RÉCONCILIE LES DEUX ESPACES DE NOMMAGE DU CAMP.
 *
 * LE DÉFAUT QU'ELLE FERME (constat utilisateur, match `b1ad85eb`, 2026-09-19 : « trois équipes,
 * dont deux Cobra »). Le camp d'une place se lit dans DEUX espaces jamais réconciliés : celui du
 * FILM (`roster[].team`, un entier) et celui de la FEUILLE (`board.team_side`, une chaîne
 * `t0`/`t1`). Quand le film se tait sur une entrée, elle partait sous la clé de la feuille
 * quand les humains de SA propre équipe étaient sous celle du film : deux groupes distincts, un
 * seul et même libellé. (Depuis le schéma 69, l'équipe est celle de l'ENTITÉ de chaque entrée :
 * les bots d'un index partagé ont la leur, et ce repli ne sert plus que les silences restants.)
 *
 * LA TRADUCTION SE MESURE, ELLE NE SE SUPPOSE PAS : un balayage des places dont le film DIT
 * l'équipe ET que la feuille nomme donne l'appariement `t1 -> 1`, `t0 -> 0`. Aucune convention
 * n'est codée en dur — l'ordre des camps de la feuille n'est pas celui du film.
 *
 * UN CÔTÉ CONTRADICTOIRE EST RETIRÉ DE LA TABLE, jamais arbitré : si deux places du même
 * `team_side` portent des camps de film DIFFÉRENTS, la jointure est fausse pour ce côté et le
 * repli de feuille reprend la main. Mieux vaut deux groupes qu'un mauvais regroupement.
 */
function campsParCote(
  players: readonly ReplayPlayer[],
  parIdentite: Map<string, ReplayRosterEntryReady>,
): Map<string, number> {
  const out = new Map<string, number>()
  const douteux = new Set<string>()
  for (const p of players) {
    const team = parIdentite.get(p.xuid)?.team
    const side = p.board?.team_side
    if (team === undefined || team === null || side == null || douteux.has(side)) continue
    const vu = out.get(side)
    if (vu === undefined) out.set(side, team)
    else if (vu !== team) {
      out.delete(side)
      douteux.add(side)
    }
  }
  return out
}

/**
 * cleDeCamp — le camp du FILM d'abord (`roster[].team`) ; quand le film se tait, celui de la
 * feuille TRADUIT vers l'espace du film (`campsParCote`) ; le côté de feuille brut en dernier
 * repli, et rien du tout quand même lui manque.
 *
 * `team` vaut `-1` quand le film dit « aucune équipe » (FFA) : c'est une LECTURE, elle regroupe
 * comme une autre. Le champ ABSENT, lui, est un silence — c'est là que la traduction opère.
 */
function cleDeCamp(
  e: ReplayRosterEntryReady | undefined,
  p: ReplayPlayer,
  parCote: Map<string, number>,
): string {
  if (e?.team !== undefined && e.team !== null) return `f${e.team}`
  const side = p.board?.team_side
  if (side == null) return ''
  const film = parCote.get(side)
  return film !== undefined ? `f${film}` : `s:${side}`
}

/** Les places rangées par camp — le pendant de `groupByTeam`, sur l'unité PLACE. */
export interface ReplaySeatGroup {
  side: string | null
  seats: ReplaySeat[]
}

/**
 * groupSeatsByTeam range les places par camp, LE FILM D'ABORD (cf. `ReplaySeat.teamKey`).
 *
 * `side` du groupe est le premier libellé de feuille qu'une de ses places porte : c'est lui que
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
