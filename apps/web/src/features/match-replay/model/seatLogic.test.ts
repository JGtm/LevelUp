/**
 * seatLogic.test.ts — une fiche par PLACE, et la place, l'équipe et la présence viennent du
 * DOCUMENT (règle des places de l'utilisateur, lot M2.4, 2026-09-23).
 *
 * LES TESTS D'AVANT LE LOT M2.4 ONT ÉTÉ RÉÉCRITS AVEC LA RÈGLE QU'ILS VERROUILLAIENT : ils
 * fixaient la tuile « A quitté » entre deux occupants et la règle « présent sans successeur » —
 * celle qui gardait un parti affiché faute de remplaçant sur SA place. Les garder verts aurait
 * entretenu l'illusion qu'une règle supprimée tient encore. Ce que ces tests-ci tiennent :
 *
 *	la place vient du document (deux entrées de même `seat` = une place, deux occupants) ;
 *	un occupant n'est affiché QUE dans sa présence publiée — jamais un parti ;
 *	entre un partant et son remplaçant la place reste VISIBLE et VIDE (Q20) ;
 *	un occupant présent sans corps est « pas encore apparu » (Q21) ;
 *	jamais plus de fiches que de places : une place rend UN occupant, ou vide ;
 *	le témoin au gabarit de `b1ad85eb` : 4 places contre 4, les bons noms aux trois instants ;
 *	un document qui ne publie AUCUNE présence (artefact antérieur) retombe sur l'enveloppe des
 *	vies, le dernier occupant de chaque place la tenant jusqu'à la fin (repli daté) ;
 *	L'ÉQUIPE EST CELLE DU FILM (décision du 2026-10-06) : une entrée dont le film tait l'équipe
 *	ne tient AUCUNE place, et la feuille ne la range nulle part — le témoin `43716616`.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { rosterEntryKey, type ReplayPlayer } from '../../../lib/replay/rosterLogic'
import {
  buildSeats as buildSeatsBrut,
  groupSeatsByTeam,
  seatOccupantAt,
  seatTileAt,
  type ReplaySeat,
} from './seatLogic'

/**
 * buildSeats, avec l'équipe des joueurs de test posée COMME `buildPlayers` la pose : le
 * désignateur de leur entrée de roster, et rien d'autre. Les joueurs d'ici sont fabriqués à la
 * main (vies et lignes de feuille choisies) ; c'est le roster du document qui reste la vérité.
 */
function buildSeats(joueurs: ReplayPlayer[], d: ReplayDocumentReady): ReplaySeat[] {
  const equipes = new Map(d.roster.map((e) => [rosterEntryKey(e), e.team ?? undefined]))
  return buildSeatsBrut(joueurs.map((p) => ({ ...p, team: equipes.get(p.xuid) })), d)
}

/** La dernière image des documents de ces tests. */
const FIN = 999

/** vie fabrique une piste sur l'intervalle d'images donné. */
function vie(debut: number, fin: number): ReplayTrackReady {
  return {
    slot: 1,
    team: 0,
    startFrame: debut,
    endFrame: fin,
    points: [{ t: debut, x: 0, y: 0 }, { t: fin, x: 0, y: 0 }],
  } as unknown as ReplayTrackReady
}

function joueur(xuid: string, side: string | null, lives: ReplayTrackReady[]): ReplayPlayer {
  return {
    xuid,
    filmName: xuid,
    lives,
    board:
      side === null
        ? undefined
        : ({ xuid, gamertag: xuid, team_side: side } as ReplayPlayer['board']),
  }
}

/** Un intervalle de présence tel que le document le publie (`toMax` omis = `to`). */
function pr(from: number, to: number, toMax?: number) {
  return toMax === undefined ? { from, to } : { from, to, toMax }
}

/**
 * doc fabrique le document minimal, DÉJÀ passé par la frontière : seul le roster compte ici, et
 * sa présence est comblée comme la frontière la comble (vide quand l'entrée n'en porte pas).
 */
function doc(roster: Array<Record<string, unknown>>, frameCount = FIN + 1): ReplayDocumentReady {
  return {
    frameIntervalMs: 100,
    frameCount,
    originMs: 0,
    roster: roster.map((e) => ({ presence: [], ...e })),
  } as unknown as ReplayDocumentReady
}

/** Ce qu'une place montre à une image : le nom de l'occupant, ou `vide`. */
function montre(seat: ReplaySeat, frame: number): string {
  const lu = seatOccupantAt(seat, frame)
  return lu.player === null ? lu.kind : `${lu.player.xuid}:${lu.kind}`
}

describe('buildSeats — la place vient du document', () => {
  it('deux entrées de MÊME place font UNE place à deux occupants, dans l’ordre du temps', () => {
    const seats = buildSeats(
      [joueur('A', 't0', [vie(600, FIN)]), joueur('P', 't0', [vie(0, 400)])], // ordre inversé
      doc([
        { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0, presence: [pr(0, 400, 599)] },
        { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0, presence: [pr(600, FIN)] },
      ]),
    )
    expect(seats).toHaveLength(1)
    expect(seats[0].seat).toBe(3)
    expect(seats[0].occupants.map((o) => o.player.xuid)).toEqual(['P', 'A'])
    expect(seats[0].occupants.map((o) => o.deduite)).toEqual([false, true])
  })

  it('une place OUVERTE sous la capacité est une déduction, comme un chaînage', () => {
    const seats = buildSeats(
      [joueur('L', 't0', [vie(645, FIN)])],
      doc([{ xuid: 'L', filmIndex: 23, seat: 23, seatSource: 'ouverte', team: 0, presence: [pr(645, FIN)] }]),
    )
    expect(seats[0].occupants[0].deduite).toBe(true)
  })

  it('l’ordre des fiches est celui des PLACES, pas celui d’arrivée des joueurs', () => {
    const seats = buildSeats(
      [joueur('C', 't0', [vie(0, FIN)]), joueur('A', 't0', [vie(0, FIN)])],
      doc([
        { xuid: 'C', filmIndex: 7, seat: 7, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] },
        { xuid: 'A', filmIndex: 2, seat: 2, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] },
      ]),
    )
    expect(seats.map((s) => s.seat)).toEqual([2, 7])
  })

  it('une entrée SANS PRÉSENCE ne tient aucune place : un parti d’avant le coup d’envoi n’apparaît jamais', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Parti', 't0', [])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] },
        { xuid: 'Parti', filmIndex: 5, seat: 5, seatSource: 'lu', team: 0 },
      ]),
    )
    expect(seats).toHaveLength(1)
    expect(seats[0].occupants[0].player.xuid).toBe('P')
  })

  it('le camp vient du FILM : un joueur sans ligne de feuille rejoint la colonne de son camp', () => {
    // `Z` n'a AUCUNE ligne de feuille — le remplaçant du constat utilisateur — mais le film lui
    // donne le camp 1 : il rejoint la colonne du camp 1. La feuille ne fait que la NOMMER.
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Z', null, [vie(0, FIN)])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 1, presence: [pr(0, FIN)] },
        { xuid: 'Z', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1, presence: [pr(0, FIN)] },
      ]),
    )
    expect(seats.map((s) => s.team)).toEqual([1, 1])
    const groupes = groupSeatsByTeam(seats)
    expect(groupes.map((g) => [g.team, g.side, g.seats.length])).toEqual([[1, 't0', 2]])
  })

  /**
   * LE GARDE-RAIL DU 2026-10-06 (« une section sans équipe n'existe pas ») : une entrée dont le
   * film TAIT l'équipe ne tient aucune place — même quand la feuille la range, même présente —,
   * et n'ouvre aucun groupe. Jamais deux groupes sous le même libellé non plus (constat du
   * 2026-09-19 sur `b1ad85eb` : « trois équipes, dont deux Cobra »).
   */
  it('une entrée SANS ÉQUIPE DU FILM n’a aucune place, et n’ouvre aucun groupe — la feuille ne la range pas', () => {
    const seats = buildSeats(
      [
        joueur('A', 't0', [vie(0, FIN)]),
        joueur('B', 't1', [vie(0, FIN)]),
        joueur('SansFilm0', 't0', [vie(0, FIN)]),
        joueur('SansFilm1', 't1', [vie(0, FIN)]),
      ],
      doc([
        { xuid: 'A', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] },
        { xuid: 'B', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1, presence: [pr(0, FIN)] },
        { xuid: 'SansFilm0', filmIndex: 2, seat: 2, seatSource: 'lu', presence: [pr(0, FIN)] },
        { xuid: 'SansFilm1', filmIndex: 3, seat: 3, seatSource: 'index', presence: [pr(0, FIN)] },
      ]),
    )
    expect(seats.map((s) => s.key)).toEqual(['siege:0:0', 'siege:1:1'])
    const groupes = groupSeatsByTeam(seats)
    expect(groupes.map((g) => [g.team, g.side])).toEqual([
      [0, 't0'],
      [1, 't1'],
    ])
  })

  it('sans aucun camp du film, AUCUNE place : la feuille de match ne regroupe plus', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Q', 't1', [vie(0, FIN)])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', presence: [pr(0, FIN)] },
        { xuid: 'Q', filmIndex: 1, seat: 1, seatSource: 'lu', presence: [pr(0, FIN)] },
      ]),
    )
    expect(seats).toEqual([])
    expect(groupSeatsByTeam(seats)).toEqual([])
  })

  it('une place appartient à UNE équipe : deux équipes ne partagent jamais une tuile', () => {
    // Le même numéro de place porté par deux équipes (source incohérente) : deux places, chacune
    // dans la colonne de son camp — jamais un occupant affiché dans le camp d'un autre.
    const seats = buildSeats(
      [joueur('E', 't0', [vie(0, 400)]), joueur('C', 't1', [vie(500, FIN)])],
      doc([
        { xuid: 'E', filmIndex: 4, seat: 4, seatSource: 'lu', team: 0, presence: [pr(0, 400)] },
        { xuid: 'C', filmIndex: 9, seat: 4, seatSource: 'apparie', team: 1, presence: [pr(500, FIN)] },
      ]),
    )
    expect(groupSeatsByTeam(seats).map((g) => [g.team, g.seats.map((s) => s.occupants[0].player.xuid)])).toEqual([
      [0, ['E']],
      [1, ['C']],
    ])
  })
})

/**
 * LE TÉMOIN `43716616` (4v4, mesure du superviseur du 2026-10-06) : Slowpoke6743 (équipe 0,
 * place 5) part — certain jusqu'à 118, peut-être là jusqu'à 317 ; « 343 Sandwolf [bot] » est
 * déclaré de 248 à 281 SANS équipe (index 8, `seatSource: index`) ; KernelPanic10 (équipe 0)
 * arrive à 318 et prend la place 5. Tant que la source ne donne pas d'équipe au bot, il ne rend
 * RIEN — ni tuile, ni troisième colonne — et la place 5 reste celle d'Eagle.
 */
describe('témoin 43716616 : le bot bouche-trou sans équipe ne rend rien', () => {
  const fin = 2999
  const seats = buildSeats(
    [
      joueur('Slowpoke6743', 't0', [vie(0, 118)]),
      joueur('bot:343 Sandwolf [bot]', null, []),
      joueur('KernelPanic10', 't0', [vie(330, fin)]),
      joueur('Titulaire', 't1', [vie(0, fin)]),
    ],
    doc(
      [
        { xuid: 'Slowpoke6743', filmIndex: 5, seat: 5, seatSource: 'lu', team: 0, presence: [pr(0, 118, 317)] },
        { xuid: '', bot: true, name: '343 Sandwolf [bot]', filmIndex: 8, seat: 8, seatSource: 'index', presence: [pr(248, 281)] },
        { xuid: 'KernelPanic10', filmIndex: 9, seat: 5, seatSource: 'tirs', team: 0, presence: [pr(318, fin)] },
        { xuid: 'Titulaire', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1, presence: [pr(0, fin)] },
      ],
      fin + 1,
    ),
  )

  it('deux colonnes, jamais une troisième ; aucune place 8', () => {
    const groupes = groupSeatsByTeam(seats)
    expect(groupes.map((g) => g.team)).toEqual([0, 1])
    expect(seats.some((s) => s.seat === 8)).toBe(false)
    expect(seats.flatMap((s) => s.occupants.map((o) => o.player.xuid))).not.toContain('bot:343 Sandwolf [bot]')
  })

  it('la place 5 chaîne Slowpoke6743 puis KernelPanic10 ; pendant la déclaration du bot, elle reste à Slowpoke (jusqu’à `toMax`)', () => {
    const place5 = seats.find((s) => s.seat === 5)!
    expect(place5.occupants.map((o) => o.player.xuid)).toEqual(['Slowpoke6743', 'KernelPanic10'])
    expect(montre(place5, 260)).toBe('Slowpoke6743:present')
    expect(montre(place5, 318)).toBe('KernelPanic10:pasEncoreApparu')
    expect(montre(place5, 330)).toBe('KernelPanic10:present')
  })
})

describe('seatOccupantAt — ce que la place montre à l’instant lu', () => {
  // Le partant est CERTAIN jusqu'à 400 et PEUT être là jusqu'à 499 (toMax : l'image-clé
  // suivante) ; son remplaçant arrive à 600 et n'apparaît qu'à 650.
  const siege = buildSeats(
    [joueur('P', 't0', [vie(0, 400)]), joueur('A', 't0', [vie(650, FIN)])],
    doc([
      { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0, presence: [pr(0, 400, 499)] },
      { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0, presence: [pr(600, FIN)] },
    ]),
  )[0]

  it('dans sa présence, le partant tient la place — jusqu’à `toMax`, pas au-delà', () => {
    expect(montre(siege, 0)).toBe('P:present')
    expect(montre(siege, 400)).toBe('P:present')
    expect(montre(siege, 499)).toBe('P:present') // mort, peut-être parti : la fiche le dit
  })

  it('Q20 : entre le partant et son remplaçant, la place est VIDE — jamais le parti', () => {
    for (const f of [500, 550, 599]) expect(montre(siege, f)).toBe('vide')
  })

  it('Q21 : le remplaçant tient la place avant sa première apparition, « pas encore apparu »', () => {
    expect(montre(siege, 600)).toBe('A:pasEncoreApparu')
    expect(montre(siege, 649)).toBe('A:pasEncoreApparu')
    expect(montre(siege, 650)).toBe('A:present')
    expect(montre(siege, FIN)).toBe('A:present')
  })

  it('la règle « présent sans successeur » est supprimée : un parti sans remplaçant n’est PLUS affiché', () => {
    const solo = buildSeats(
      [joueur('P', 't0', [vie(0, 400)])],
      doc([{ xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0, presence: [pr(0, 400, 499)] }]),
    )[0]
    expect(montre(solo, 499)).toBe('P:present')
    expect(montre(solo, 500)).toBe('vide')
    expect(montre(solo, FIN)).toBe('vide')
  })

  it('mourir n’est pas partir : un trou de réapparition est DANS la présence', () => {
    const vivant = buildSeats(
      [joueur('P', 't0', [vie(0, 200), vie(280, FIN)])],
      doc([{ xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] }]),
    )[0]
    expect(montre(vivant, 240)).toBe('P:present')
  })

  it('JAMAIS deux fiches pour une place : la lecture rend UN occupant, ou la place vide', () => {
    for (let f = 0; f <= FIN; f += 7) {
      const lu = seatOccupantAt(siege, f)
      expect(['present', 'pasEncoreApparu', 'vide']).toContain(lu.kind)
      expect(lu.kind === 'vide' ? lu.player === null : lu.player !== null).toBe(true)
    }
  })
})

/**
 * LE TÉMOIN AU GABARIT DE `b1ad85eb` — le roster que la cuisson du schéma 69 publie pour ce
 * film (témoin Go `occupants_temoin_test.go` : mêmes présences, mêmes vies, en images de
 * 100 ms). Il REMPLACE le test qui figeait `[3, 5]` : Eagle avait 3 fiches au coup d'envoi puis
 * 5, Cobra 5 à 6:24. La règle des places en veut 4 contre 4 à chaque image, et les bons
 * occupants aux trois instants que le témoin Go tient (images 227, 1567 et 4067).
 */
describe('témoin b1ad85eb : quatre places contre quatre, les bons occupants', () => {
  const fin = 5099
  const tout = pr(0, fin)
  const eagle = [
    { xuid: 'MONEY', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0, presence: [tout] },
    { xuid: 'Namikidori', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0, presence: [tout] },
    { xuid: 'DRghie', filmIndex: 6, seat: 6, seatSource: 'lu', team: 0, presence: [tout] },
    { xuid: 'WNBA', filmIndex: 5, seat: 5, seatSource: 'lu', team: 0 }, // parti avant le coup d'envoi
    { xuid: '', name: 'Hundy', bot: true, filmIndex: 8, seat: 5, seatSource: 'apparie', team: 0, presence: [pr(0, 272)] },
    { xuid: 'Hanover', filmIndex: 9, seat: 5, seatSource: 'apparie', team: 0, presence: [pr(412, 662, 773)] },
    { xuid: '', name: 'PardonMy', bot: true, filmIndex: 8, seat: 5, seatSource: 'apparie', team: 0, presence: [pr(774, 830)] },
    { xuid: 'Claudors', filmIndex: 10, seat: 5, seatSource: 'tirs', team: 0, presence: [pr(1013, fin)] },
  ]
  const cobra = [
    { xuid: 'FairyNectar', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1, presence: [pr(0, 3117, 3154)] },
    { xuid: 'Madina', filmIndex: 2, seat: 2, seatSource: 'lu', team: 1, presence: [tout] },
    { xuid: 'Chocoboflor', filmIndex: 4, seat: 4, seatSource: 'lu', team: 1, presence: [tout] },
    { xuid: 'JGtm', filmIndex: 7, seat: 7, seatSource: 'lu', team: 1, presence: [tout] },
    { xuid: '', name: 'BrewDog', bot: true, filmIndex: 8, seat: 1, seatSource: 'apparie', team: 1, presence: [pr(3155, fin)] },
  ]
  const joueurs = [
    ...['MONEY', 'Namikidori', 'DRghie'].map((x) => joueur(x, 't0', [vie(0, fin)])),
    joueur('WNBA', 't0', []),
    joueur('bot:Hundy', 't0', [vie(0, 270)]),
    joueur('Hanover', 't0', [vie(661, 662)]),
    joueur('bot:PardonMy', 't0', [vie(774, 829)]),
    joueur('Claudors', 't0', [vie(1184, fin)]),
    joueur('FairyNectar', 't1', [vie(0, 3117)]),
    ...['Madina', 'Chocoboflor', 'JGtm'].map((x) => joueur(x, 't1', [vie(0, fin)])),
    joueur('bot:BrewDog', 't1', [vie(3236, fin)]),
  ]
  const groupes = groupSeatsByTeam(buildSeats(joueurs, doc([...eagle, ...cobra], fin + 1)))

  /** Les occupants affichés d'un groupe à une image, triés — une place vide n'en donne aucun. */
  function affiches(g: number, frame: number): string[] {
    return groupes[g].seats
      .map((s) => seatOccupantAt(s, frame).player?.xuid)
      .filter((x): x is string => x !== undefined)
      .sort()
  }

  it('deux groupes de QUATRE places — le test d’avant figeait [3, 5]', () => {
    expect(groupes.map((g) => g.seats.length)).toEqual([4, 4])
  })

  it('la place 5 d’Eagle chaîne Hundy, Hanover Cat, PardonMy puis Claudors', () => {
    const place5 = groupes[0].seats.find((s) => s.seat === 5)!
    expect(place5.occupants.map((o) => o.player.xuid)).toEqual([
      'bot:Hundy', 'Hanover', 'bot:PardonMy', 'Claudors',
    ])
    expect(montre(place5, 300)).toBe('vide') // Hundy retiré, Hanover Cat pas encore arrivé
    expect(montre(place5, 500)).toBe('Hanover:pasEncoreApparu')
    expect(montre(place5, 900)).toBe('vide')
    expect(montre(place5, 1100)).toBe('Claudors:pasEncoreApparu')
  })

  it('la place 1 de Cobra : FairyNectar puis Brew Dog, jamais les deux', () => {
    const place1 = groupes[1].seats.find((s) => s.seat === 1)!
    expect(place1.occupants.map((o) => o.player.xuid)).toEqual(['FairyNectar', 'bot:BrewDog'])
    expect(montre(place1, 3154)).toBe('FairyNectar:present')
    expect(montre(place1, 3155)).toBe('bot:BrewDog:pasEncoreApparu')
    expect(montre(place1, 3236)).toBe('bot:BrewDog:present')
  })

  it('à CHAQUE image, au plus quatre occupants par équipe', () => {
    for (let f = 0; f <= fin; f++) {
      expect(affiches(0, f).length, `Eagle à l’image ${f}`).toBeLessThanOrEqual(4)
      expect(affiches(1, f).length, `Cobra à l’image ${f}`).toBeLessThanOrEqual(4)
    }
  })

  it('les bons occupants aux trois instants du témoin', () => {
    expect(affiches(0, 227)).toEqual(['DRghie', 'MONEY', 'Namikidori', 'bot:Hundy'])
    expect(affiches(0, 1567)).toEqual(['Claudors', 'DRghie', 'MONEY', 'Namikidori'])
    expect(affiches(1, 4067)).toEqual(['Chocoboflor', 'JGtm', 'Madina', 'bot:BrewDog'])
  })
})

describe('un document qui ne publie AUCUNE présence (artefact antérieur au schéma 69)', () => {
  const seats = buildSeats(
    [joueur('P', 't0', [vie(0, 400)]), joueur('A', 't0', [vie(600, 800)]), joueur('Q', 't0', [vie(0, 300)])],
    doc([
      { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0 },
      { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0 },
      { xuid: 'Q', filmIndex: 4, seat: 4, seatSource: 'lu', team: 0 },
    ]),
  )

  it('la présence retombe sur l’enveloppe des vies, et la place vide reste entre les deux', () => {
    const place3 = seats.find((s) => s.seat === 3)!
    expect(montre(place3, 400)).toBe('P:present')
    expect(montre(place3, 500)).toBe('vide')
    expect(montre(place3, 600)).toBe('A:present')
  })

  it('le DERNIER occupant de chaque place la tient jusqu’à la fin — mourir n’est pas partir', () => {
    expect(montre(seats.find((s) => s.seat === 3)!, FIN)).toBe('A:present')
    expect(montre(seats.find((s) => s.seat === 4)!, FIN)).toBe('Q:present')
  })
})

describe('seatTileAt — ce qu’une place ne rend pas (revue M2, 2026-09-24)', () => {
  it('M2-R1 : un joueur sans entrée de roster n’a AUCUNE place — ni tuile, ni colonne', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('bot:Robot', null, [vie(300, 400)])],
      doc([{ xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0, presence: [pr(0, FIN)] }]),
    )
    expect(seats.map((s) => s.key)).toEqual(['siege:0:0'])
    expect(groupSeatsByTeam(seats)).toHaveLength(1)
  })

  it('un film sans identification (aucun roster) n’écrit aucune équipe : aucune place', () => {
    // La feuille le nomme et le range, le film ne dit rien : le joueur n'a pas d'équipe du film,
    // donc pas de place (décision du 2026-10-06) — la colonne affiche son constat vide.
    expect(buildSeats([joueur('X', 't0', [vie(100, 200)])], doc([]))).toEqual([])
  })

  it('M2-R7 : document sans présence — aucune tuile avant le premier occupant, la place vide ensuite', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(30, 400)]), joueur('A', 't0', [vie(600, 800)])],
      doc([
        { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0 },
        { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0 },
      ]),
    )
    expect(seatTileAt(seats[0], 10)).toBeNull()
    expect(seatTileAt(seats[0], 30)?.kind).toBe('present')
    expect(seatTileAt(seats[0], 500)?.kind).toBe('vide') // Q20 : entre les deux occupants
  })

  it('document qui publie ses présences : la place attend son premier occupant VIDE (Q20)', () => {
    const seats = buildSeats(
      [joueur('L', 't0', [vie(645, FIN)])],
      doc([{ xuid: 'L', filmIndex: 23, seat: 23, seatSource: 'ouverte', team: 0, presence: [pr(640, FIN)] }]),
    )
    expect(seatTileAt(seats[0], 100)?.kind).toBe('vide')
  })
})
