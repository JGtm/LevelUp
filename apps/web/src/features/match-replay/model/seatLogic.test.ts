/**
 * seatLogic.test.ts — le siège suit l'OCCUPANT, et l'occupant vient du DOCUMENT.
 *
 * LES TESTS D'AVANT LE LOT 1.9.14 ONT ÉTÉ REMPLACÉS AVEC LA RÈGLE QU'ILS VERROUILLAIENT.
 * Ils fixaient l'appariement ordinal sur la PARTICIPATION API (`left_in_progress` /
 * `joined_in_progress`, fenêtre de 120 s, départage par indice de film) : ce calcul n'est plus
 * ici — il est fait à la cuisson, sur le film, nommé et compté au registre des replis
 * (`repli_siege_du_remplacant_par_appariement_ordinal`), et le web lit `roster[].seat`. Les
 * garder verts aurait entretenu l'illusion qu'une règle supprimée tient encore.
 *
 * Ce que ces tests-ci tiennent, un par règle :
 *
 *	le siège vient du document (deux entrées de même `seat` = une fiche, deux occupants) ;
 *	entre le départ et l'arrivée, la fiche reste MARQUÉE « a quitté » ;
 *	après l'arrivée, elle porte l'arrivant ;
 *	quand plus aucun successeur ne vient, elle DISPARAÎT ;
 *	un joueur sans aucune vie n'a de fiche à aucune image ;
 *	jamais deux fiches pour un siège.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady, ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import type { ReplayPlayer } from '../../../lib/replay/rosterLogic'
import { buildSeats, groupSeatsByTeam, seatOccupantAt } from './seatLogic'

/** t1 = le partant sort ; t2 = l'arrivant entre. Entre les deux, le siège est en transition. */
const T1 = 400
const T2 = 600
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

/** doc fabrique le document minimal : seul le roster compte pour ces tests. */
function doc(roster: Array<Record<string, unknown>>): ReplayDocumentReady {
  return { frameIntervalMs: 100, frameCount: FIN + 1, originMs: 0, roster } as ReplayDocumentReady
}

describe('buildSeats — le siège vient du document', () => {
  it('deux entrées de MÊME siège font UNE fiche à deux occupants, dans l’ordre du temps', () => {
    const partant = joueur('P', 't0', [vie(0, T1)])
    const arrivant = joueur('A', 't0', [vie(T2, FIN)])
    const seats = buildSeats(
      [arrivant, partant], // ordre d'entrée volontairement inversé
      doc([
        { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0 },
        { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0 },
      ]),
    )
    expect(seats).toHaveLength(1)
    expect(seats[0].seat).toBe(3)
    expect(seats[0].occupants.map((o) => o.player.xuid)).toEqual(['P', 'A'])
    expect(seats[0].occupants[1].apparie).toBe(true)
    expect(seats[0].occupants[0].apparie).toBe(false)
  })

  it('l’ordre des fiches est celui des SIÈGES, pas celui d’arrivée des joueurs', () => {
    const seats = buildSeats(
      [joueur('C', 't0', [vie(0, FIN)]), joueur('A', 't0', [vie(0, FIN)])],
      doc([
        { xuid: 'C', filmIndex: 7, seat: 7, seatSource: 'lu', team: 0 },
        { xuid: 'A', filmIndex: 2, seat: 2, seatSource: 'lu', team: 0 },
      ]),
    )
    expect(seats.map((s) => s.seat)).toEqual([2, 7])
  })

  it('un joueur SANS AUCUNE VIE n’a de fiche à aucune image', () => {
    const seats = buildSeats(
      [joueur('P', 't0', [vie(0, FIN)]), joueur('Z', 't0', [])],
      doc([
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0 },
        { xuid: 'Z', filmIndex: 1, seat: 1, seatSource: 'lu', team: 0 },
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
        { xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 1 },
        { xuid: 'Z', filmIndex: 1, seat: 1, seatSource: 'lu', team: 1 },
      ]),
    )
    expect(new Set(seats.map((s) => s.teamKey))).toEqual(new Set(['f1']))
    const groupes = groupSeatsByTeam(seats)
    expect(groupes).toHaveLength(1)
    expect(groupes[0].side).toBe('t0') // le libellé affiché reste celui de la feuille
    expect(groupes[0].seats).toHaveLength(2)
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

describe('seatOccupantAt — ce que la fiche montre à l’instant lu', () => {
  const seats = buildSeats(
    [joueur('P', 't0', [vie(0, T1)]), joueur('A', 't0', [vie(T2, FIN)])],
    doc([
      { xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0 },
      { xuid: 'A', filmIndex: 9, seat: 3, seatSource: 'apparie', team: 0 },
    ]),
  )
  const siege = seats[0]

  it('avant t1 : le partant tient la fiche', () => {
    expect(seatOccupantAt(siege, 0)).toEqual({ player: expect.anything(), kind: 'present' })
    expect(seatOccupantAt(siege, T1).player?.xuid).toBe('P')
  })

  it('entre t1 et t2 : la fiche est MARQUÉE « a quitté », elle ne disparaît pas', () => {
    const lu = seatOccupantAt(siege, T1 + 1)
    expect(lu.kind).toBe('parti')
    expect(lu.player?.xuid).toBe('P')
    expect(seatOccupantAt(siege, T2 - 1).kind).toBe('parti')
  })

  it('à partir de t2 : la fiche porte l’ARRIVANT', () => {
    expect(seatOccupantAt(siege, T2)).toMatchObject({ kind: 'present' })
    expect(seatOccupantAt(siege, T2).player?.xuid).toBe('A')
    expect(seatOccupantAt(siege, FIN).player?.xuid).toBe('A')
  })

  it('mourir n’est pas partir : sans successeur, la fiche RESTE après la dernière vie', () => {
    // Le joueur meurt a T1 et ne reapparait pas. Le film ne dit pas s’il a quitte ; la fiche
    // le dit deja par « hors film » (retour user du 2026-09-02) et ne disparait pas.
    const solo = buildSeats(
      [joueur('P', 't0', [vie(0, T1)])],
      doc([{ xuid: 'P', filmIndex: 3, seat: 3, seatSource: 'lu', team: 0 }]),
    )[0]
    expect(seatOccupantAt(solo, T1).kind).toBe('present')
    expect(seatOccupantAt(solo, T1 + 1)).toEqual({ player: expect.anything(), kind: 'present' })
    expect(seatOccupantAt(solo, FIN).player?.xuid).toBe('P')
  })

  it('avant l’arrivée du PREMIER occupant, le siège n’a aucune fiche', () => {
    const tardif = buildSeats(
      [joueur('A', 't0', [vie(T2, FIN)])],
      doc([{ xuid: 'A', filmIndex: 9, seat: 9, seatSource: 'lu', team: 0 }]),
    )[0]
    expect(seatOccupantAt(tardif, T2 - 1)).toEqual({ player: null, kind: 'absent' })
    expect(seatOccupantAt(tardif, T2).kind).toBe('present')
  })

  it('JAMAIS deux fiches pour un siège : la lecture rend UN occupant, ou aucun', () => {
    for (let f = 0; f <= FIN; f += 7) {
      const lu = seatOccupantAt(siege, f)
      expect(['present', 'parti', 'absent']).toContain(lu.kind)
      expect(lu.kind === 'absent' ? lu.player === null : lu.player !== null).toBe(true)
    }
  })

  it('un trou de RÉAPPARITION est DANS la présence : la fiche reste', () => {
    // Deux vies du même joueur, séparées de 80 images : il est mort, pas parti.
    const vivant = buildSeats(
      [joueur('P', 't0', [vie(0, 200), vie(280, FIN)])],
      doc([{ xuid: 'P', filmIndex: 0, seat: 0, seatSource: 'lu', team: 0 }]),
    )[0]
    expect(seatOccupantAt(vivant, 240).kind).toBe('present')
  })
})
