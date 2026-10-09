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
 * # UNE PLACE APPARTIENT À UNE ÉQUIPE DU FILM (décision du 2026-10-06)
 *
 * « Une section sans équipe n'existe pas ; un joueur ne peut pas ne pas avoir d'équipe dans un
 * match. » L'équipe d'un occupant est le désignateur que le film écrit (`ReplayPlayer.team`,
 * posé par `buildPlayers`) et rien d'autre : une entrée dont le film TAIT l'équipe ne tient
 * AUCUNE place — ni tuile, ni colonne, à aucune image. C'est un défaut de source, compté par la
 * cuisson (`coverage.seats.sansEquipe`) et corrigé là-bas ; la feuille de match ne le comble
 * pas. Les places sont FINIES PAR ÉQUIPE (règle des places) : la clé d'une place porte son
 * équipe, et deux équipes ne partagent jamais une tuile.
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
 * # LES DOCUMENTS À ROSTER QUI NE PUBLIENT PAS DE PRÉSENCE (cf. `presencesParLesVies`)
 *
 * Un artefact antérieur au schéma 69 n'en porte aucune (repli DATÉ) : la présence y est
 * l'enveloppe des vies, et le dernier occupant de chaque place la tient jusqu'à la fin — la règle
 * que la cuisson applique elle-même quand elle n'a lu aucune entité. Le choix se fait au niveau
 * du DOCUMENT, jamais entrée par entrée. Sur ces documents, une place ne rend AUCUNE tuile avant
 * son premier occupant (revue M2-R7) : rien n'y dit qui était là au coup d'envoi, et une place
 * « libre » pendant le préambule mentirait sur un joueur qui n'a simplement pas encore apparu.
 *
 * # UN JOUEUR SANS ENTRÉE DE ROSTER (revue M2-R1)
 *
 * Un joueur que ses vies nomment sans entrée n'a ni place, ni présence, ni équipe publiées : il
 * n'a AUCUNE place, donc aucune tuile ni colonne. La règle des places prime : lui en donner une,
 * même pendant ses seules vies, ferait une fiche de plus que de places (`c75f33b8` : un bot que
 * le kill-feed n'épingle pas, nommé par le relais de la base, montré à côté de la place vide
 * qu'il remplace). La cuisson le compte (`coverage.seats.identitesHorsRoster`, 0 attendu) : c'est
 * un défaut à corriger à la source. Un film SANS IDENTIFICATION (aucun roster) n'écrit donc
 * aucune équipe : sa colonne est vide (`rosterEmpty`).
 */
import type { ReplayDocumentReady, ReplayRosterEntryReady } from '../../../lib/replay/replayNormalize'
import { groupByCamp, type ReplayCamp } from '../../../lib/replay/replayCamps'
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
  /** Clé stable de rendu : `siege:<équipe>:<place>`. */
  key: string
  /** La place telle que le document la publie — l'ordre d'affichage en dépend. */
  seat: number
  /** L'équipe du FILM de ses occupants (`ReplayPlayer.team`) : la colonne où elle se range. */
  team: number
  occupants: SeatOccupant[]
  /**
   * Vrai quand les présences de la place viennent du DOCUMENT (schéma 69 et suivants). Faux sur
   * la voie de l'enveloppe des vies, où la place ne rend rien avant son premier occupant.
   */
  presenceLue: boolean
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
 * seatTileAt — LA TUILE QUE LA PLACE REND À CETTE IMAGE, ou `null` quand elle n'en rend aucune :
 * une place de la voie des vies avant son premier occupant (cf. l'en-tête). Partout ailleurs, la
 * place est TOUJOURS rendue — tenue, « pas encore apparu », ou vide (Q20) : c'est ce qui fait
 * qu'une équipe a un nombre fini de tuiles.
 */
export function seatTileAt(seat: ReplaySeat, frame: number): SeatReading | null {
  const lu = seatOccupantAt(seat, frame)
  if (lu.kind !== 'vide') return lu
  if (!seat.presenceLue && frame < (seat.occupants[0]?.presence[0]?.from ?? 0)) return null
  return lu
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
 * le coup d'envoi, ou jamais présent, ne tient aucune place. Un joueur sans équipe du film — sans
 * entrée de roster, ou dont l'entrée tait l'équipe — n'en tient aucune non plus (cf. l'en-tête).
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
  const places = new Map<string, ReplaySeat>()
  for (const p of players) {
    const e = parIdentite.get(p.xuid)
    // AUCUNE PLACE SANS ÉQUIPE DU FILM (cf. l'en-tête) : ni tuile, ni colonne, à aucune image.
    if (e === undefined || p.team === undefined) continue
    const presence = publiees ? presenceDeLEntree(e) : enveloppeDesVies(p)
    if (presence.length === 0) continue // aucune présence : aucune place, à aucune image
    const numero = e.seat ?? e.filmIndex
    const cle = `siege:${p.team}:${numero}`
    let s = places.get(cle)
    if (!s) {
      s = { key: cle, seat: numero, team: p.team, occupants: [], presenceLue: publiees }
      places.set(cle, s)
    }
    s.occupants.push({ player: p, presence, deduite: estDeduite(e) })
  }
  const out = [...places.values()]
  for (const s of out) s.occupants.sort((a, b) => a.presence[0].from - b.presence[0].from)
  if (!publiees) presencesParLesVies(out, doc.frameCount - 1)
  return out.sort((a, b) => a.seat - b.seat || a.key.localeCompare(b.key))
}

/** Les provenances de place qui sont des DÉDUCTIONS de la cuisson (cf. `SeatOccupant.deduite`). */
const PROVENANCES_DEDUITES: ReadonlySet<string> = new Set(['apparie', 'ouverte'])

function estDeduite(e: ReplayRosterEntryReady): boolean {
  return e.seatSource !== undefined && PROVENANCES_DEDUITES.has(e.seatSource)
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
 * presencesParLesVies — LES DOCUMENTS À ROSTER QUI NE PUBLIENT PAS DE PRÉSENCE (artefacts
 * antérieurs au schéma 69) : chaque occupant est présent sur l'enveloppe de ses vies, et le
 * DERNIER occupant de chaque place la tient jusqu'à la fin — mourir n'est pas partir, et sans
 * présence publiée rien ne dit qui est parti. C'est la règle que la cuisson applique elle-même
 * quand elle n'a lu aucune entité (`repli_presence_par_enveloppe_des_vies`). Un film sans
 * identification n'y passe plus : sans roster, il n'écrit aucune équipe, donc aucune place.
 *
 * KILL-SWITCH DATÉ du repli (modèle `platform/duckdb/shared_reader_legacy.go`) :
 *   - bascule du défaut : 2026-09-23 — depuis le schéma 69, la présence vient du document, et
 *     ce repli ne sert plus aucun artefact qui la publie (cf. `publieDesPresences`) ;
 *   - retrait cible : 2026-12-01 — `buildSeats` ne passera plus par ici pour un document à roster ;
 *   - critère mesurable, sur la SEULE présence et non sur un numéro de schéma (revue M2-R4/R8 : un
 *     schéma publié sans présence rendrait un critère numérique faux) : plus aucun artefact servi
 *     dont le roster est NON VIDE sans qu'une entrée porte `presence`.
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

/** Les places d'un camp DU FILM — le pendant de `groupByTeam`, sur l'unité PLACE. */
export interface ReplaySeatGroup extends ReplayCamp {
  seats: ReplaySeat[]
}

/**
 * groupSeatsByTeam range les places par camp du FILM (`ReplaySeat.team`, `groupByCamp`) : un
 * camp par désignateur, dans l'ordre des désignateurs, et jamais un groupe « sans équipe ».
 *
 * `side` du groupe est le côté de feuille de la majorité de ses occupants qui ont une ligne : il
 * ne sert qu'à NOMMER la colonne (`campLabel`) et à son encre, jamais à la composer.
 */
export function groupSeatsByTeam(seats: readonly ReplaySeat[]): ReplaySeatGroup[] {
  return groupByCamp(
    seats,
    (s) => s.team,
    (s) => s.occupants.map((o) => o.player.board),
  ).map(({ team, side, members }) => ({ team, side, seats: members }))
}
