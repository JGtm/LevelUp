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
 *	vies, le dernier occupant de chaque place la tenant jusqu'à la fin (repli daté).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { buildSeats, groupSeatsByTeam, seatOccupantAt, seatTileAt, type ReplaySeat } from './seatLogic'

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

  it('le camp vient du FILM quand le document le porte, la feuille reste le repli', () => {
    // `Z` n'a AUCUNE ligne de feuille — le remplaçant « sans équipe » du constat utilisateur —
    // mais le film lui donne le camp 1 : il rejoint la colonne du camp 1.
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Z', null, [vie(0, FIN)])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 1, presence: [pr(0, FIN)] },
        { xuid: 'Z', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1, presence: [pr(0, FIN)] },
      ]),
    )
    expect(new Set(seats.map((s) => s.teamKey))).toEqual(new Set(['f1']))
    const groupes = groupSeatsByTeam(seats)
    expect(groupes).toHaveLength(1)
    expect(groupes[0].side).toBe('t0') // le libellé affiché reste celui de la feuille
    expect(groupes[0].seats).toHaveLength(2)
  })

  /**
   * LE GARDE-RAIL, et il est indépendant du témoin : quel que soit le mélange de sources, deux
   * groupes ne peuvent pas porter le MÊME libellé (constat utilisateur du 2026-09-19 sur
   * `b1ad85eb` : « trois équipes, dont deux Cobra »).
   */
  it('garde-rail : jamais deux groupes sous le même libellé', () => {
    const seats = buildSeats(
      [
        joueur('A', 't0', [vie(0, FIN)]),
        joueur('B', 't1', [vie(0, FIN)]),
        joueur('SansFilm0', 't0', [vie(0, FIN)]),
        joueur('SansFilm1', 't1', [vie(0, FIN)]),
      ],
      doc([
        { xuid: 'A', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0 },
        { xuid: 'B', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1 },
        { xuid: 'SansFilm0', filmIndex: 2, seat: 2, seatSource: 'lu' },
        { xuid: 'SansFilm1', filmIndex: 3, seat: 3, seatSource: 'lu' },
      ]),
    )
    const libelles = groupSeatsByTeam(seats).map((g) => g.side)
    expect(new Set(libelles).size).toBe(libelles.length)
    expect(libelles.sort()).toEqual(['t0', 't1'])
  })

  /**
   * LA TRADUCTION NE SUPPOSE AUCUNE CONVENTION : elle est MESURÉE sur les places que les deux
   * sources nomment. Ici la feuille dit `t0` là où le film dit 1 — l'inverse de l'ordre naïf —
   * et la place muette doit suivre la MESURE, pas l'ordre.
   */
  it('la traduction suit la mesure, pas l’ordre des camps', () => {
    const seats = buildSeats(
      [joueur('A', 't0', [vie(0, FIN)]), joueur('Muet', 't0', [vie(0, FIN)])],
      doc([
        { xuid: 'A', filmIndex: 0, seat: 0, seatSource: 'lu', team: 1 },
        { xuid: 'Muet', filmIndex: 1, seat: 1, seatSource: 'lu' },
      ]),
    )
    expect(new Set(seats.map((s) => s.teamKey))).toEqual(new Set(['f1']))
  })

  it('côté contradictoire : la traduction se retire, le repli de feuille reprend', () => {
    const seats = buildSeats(
      [
        joueur('A', 't0', [vie(0, FIN)]),
        joueur('B', 't0', [vie(0, FIN)]),
        joueur('Muet', 't0', [vie(0, FIN)]),
      ],
      doc([
        { xuid: 'A', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0 },
        { xuid: 'B', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1 },
        { xuid: 'Muet', filmIndex: 2, seat: 2, seatSource: 'lu' },
      ]),
    )
    expect(seats.find((s) => s.seat === 2)!.teamKey).toBe('s:t0')
  })

  it('sans camp du film, le regroupement retombe sur la feuille de match', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Q', 't1', [vie(0, FIN)])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu' },
        { xuid: 'Q', filmIndex: 1, seat: 1, seatSource: 'lu' },
      ]),
    )
    expect(seats.map((s) => s.teamKey)).toEqual(['s:t0', 's:t1'])
    expect(groupSeatsByTeam(seats)).toHaveLength(2)
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
    expect(seats.map((s) => s.key)).toEqual(['siege:0'])
    expect(groupSeatsByTeam(seats)).toHaveLength(1)
  })

  it('un film sans identification (aucun roster) garde sa voie nominale : une place par joueur', () => {
    const seats = buildSeats([joueur('X', 't0', [vie(100, 200)])], doc([]))
    expect(seats.map((s) => s.key)).toEqual(['joueur:X'])
    expect(seatTileAt(seats[0], FIN)?.player?.xuid).toBe('X') // le dernier tient jusqu'à la fin
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
