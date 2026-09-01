/**
 * Tests — padControlLogic (le contrôle des armes spéciales, par joueur et par équipe).
 *
 * CE QU'ILS PROTÈGENT, dans l'ordre des pièges du domaine :
 *   - le PONT xuid -> joueur -> équipe : le camp vient du SCOREBOARD, le film n'en porte aucun ;
 *   - une occupation SANS ramasseur nommé n'est comptée POUR PERSONNE — jamais rattrapée ;
 *   - la SOMME BOUCLE : prises affichées + occupations hors tableau = occupations mesurées ;
 *   - le TRI est par total décroissant, camps compris — c'est le sujet du tableau ;
 *   - la DOUBLE PORTE : aucune prise attribuée = rien à rendre (`hasData` faux) ;
 *   - un artefact SANS bloc de datation ne fabrique aucune ventilation.
 *
 * Les fixtures passent par `testReplayDoc`, la seule porte du document de test (garde-rail
 * `testDoc.guard.test.ts`) : elles décrivent un document de TRANSPORT, comme le serveur l'envoie.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'

import {
  buildPadControl,
  padControlGaps,
  padControlMissing,
  type PadControl,
} from './padControlLogic'
import { testReplayDoc } from './test/testDoc'

/** Une vie : le slot, son propriétaire, et deux points pour que la fenêtre existe. */
function vie(slot: number, xuid: string) {
  return {
    slot,
    xuid,
    team: -1,
    startFrame: 0,
    endFrame: 100,
    points: [
      { t: 0, x: 0, y: 0 },
      { t: 100, x: 1, y: 1 },
    ],
  }
}

/** Un socle du calque (les champs que l'agrégation lit). */
function socle(weapon: string) {
  return { weapon, x: 0, y: 0, spawns: [], presence: [] }
}

/** Une occupation achevée : son socle, sa fenêtre, et son ramasseur quand il est nommé. */
function prise(pad: number, xuid: string | null, t = 10) {
  return { pad, t, tLow: t - 5, tHigh: t + 5, xuid }
}

const SNIPER = '0xAAAA1111'
const EPEE = '0xBBBB2222'

const SB: MatchScoreboardRow[] = [
  { xuid: 'a1', gamertag: 'Alpha', team_side: 't0' },
  { xuid: 'a2', gamertag: 'Bravo', team_side: 't0' },
  { xuid: 'b1', gamertag: 'Charlie', team_side: 't1' },
] as MatchScoreboardRow[]

/**
 * LE TÉMOIN. Trois joueurs au scoreboard (deux camps) plus un QUATRIÈME que le film voit vivre
 * et que le scoreboard ignore ; deux socles d'arme et un socle de bonus.
 */
function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha' },
      { filmIndex: 1, xuid: 'a2', name: 'Bravo' },
      { filmIndex: 2, xuid: 'b1', name: 'Charlie' },
      { filmIndex: 3, xuid: 'orphelin', name: 'Delta' },
    ],
    tracks: [vie(1, 'a1'), vie(2, 'a2'), vie(3, 'b1'), vie(4, 'orphelin')],
    weaponPads: [socle(SNIPER), socle(EPEE), socle('powerup_overshield')],
    ...over,
  } as Partial<ReplayDocument>)
}

/** Les lignes de joueur, toutes équipes confondues, indexées par nom d'affichage. */
function parNom(control: PadControl) {
  return new Map(control.byTeam.flatMap((g) => g.players).map((r) => [r.name, r]))
}

describe('buildPadControl — le pont xuid -> joueur -> équipe', () => {
  it('attribue chaque prise nommée à son ramasseur, socle par socle', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [
          prise(0, 'a1', 10),
          prise(0, 'a1', 40),
          prise(1, 'b1', 20),
          prise(0, 'a2', 60),
        ],
      } as Partial<ReplayDocument>),
      SB,
    )
    const lignes = parNom(control)
    expect(lignes.get('Alpha')?.total).toBe(2)
    expect(lignes.get('Alpha')?.byWeapon[SNIPER]).toBe(2)
    expect(lignes.get('Charlie')?.total).toBe(1)
    expect(lignes.get('Charlie')?.byWeapon[EPEE]).toBe(1)
    expect(lignes.get('Bravo')?.total).toBe(1)
    // Delta n'a rien pris, mais il a une ligne : c'est un zéro MESURÉ.
    expect(lignes.get('Delta')?.total).toBe(0)
    expect(control.attributed).toBe(4)
  })

  it('range les joueurs par camp du SCOREBOARD et somme chaque camp', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'a1'), prise(1, 'a2'), prise(0, 'b1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    const t0 = control.byTeam.find((g) => g.side === 't0')
    const t1 = control.byTeam.find((g) => g.side === 't1')
    expect(t0?.total.total).toBe(2)
    expect(t0?.total.byWeapon[SNIPER]).toBe(1)
    expect(t0?.total.byWeapon[EPEE]).toBe(1)
    expect(t1?.total.total).toBe(1)
  })

  it('garde le joueur HORS SCOREBOARD, sans équipe — le trou se montre', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'orphelin')] } as Partial<ReplayDocument>),
      SB,
    )
    const sansEquipe = control.byTeam.find((g) => g.side === null)
    expect(sansEquipe?.players.map((p) => p.name)).toEqual(['Delta'])
    expect(sansEquipe?.total.total).toBe(1)
    // Et il n'a été versé dans AUCUN camp nommé.
    expect(control.byTeam.filter((g) => g.side !== null).reduce((n, g) => n + g.total.total, 0))
      .toBe(0)
  })

  it('sans scoreboard du tout, personne n’a d’équipe et rien n’est deviné', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1')] } as Partial<ReplayDocument>),
      undefined,
    )
    expect(control.byTeam.map((g) => g.side)).toEqual([null])
    expect(control.attributed).toBe(1)
  })
})

describe('buildPadControl — ce qui n’est PAS attribué', () => {
  it('ne compte pour personne une occupation sans ramasseur nommé', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, null), prise(1, null), prise(0, 'a1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.attributed).toBe(1)
    expect(parNom(control).get('Alpha')?.total).toBe(1)
    expect([...parNom(control).values()].reduce((n, r) => n + r.total, 0)).toBe(1)
  })

  it('compte à part le ramasseur que le film n’a pas vu vivre, et l’index de socle hors bornes', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'inconnu'), prise(99, 'a1'), prise(0, 'a1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.unjoined).toBe(2)
    expect(control.attributed).toBe(1)
  })

  it('la SOMME BOUCLE : prises affichées + manques = occupations mesurées', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'a1'), prise(1, 'b1'), prise(0, null), prise(2, null)],
        coverage: {
          padDating: {
            occupations: 4,
            dated: 3,
            named: 2,
            ambiguous: 1,
            uncovered: 0,
            powerupOccupations: 1,
          },
        },
      } as unknown as Partial<ReplayDocument>),
      SB,
    )
    expect(control.attributed).toBe(2)
    expect(padControlMissing(control)).toBe(2)
    const gaps = padControlGaps(control)
    expect(gaps.reduce((n, g) => n + g.count, 0)).toBe(2)
    expect(gaps.map((g) => g.key)).toEqual(['ambiguous', 'powerup'])
  })

  it('ventile le reste en « datée sans ramasseur nommé » plutôt que de laisser un trou', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'a1'), prise(0, null), prise(1, null)],
        coverage: {
          padDating: {
            occupations: 3,
            dated: 2,
            named: 1,
            ambiguous: 0,
            uncovered: 0,
            powerupOccupations: 0,
          },
        },
      } as unknown as Partial<ReplayDocument>),
      SB,
    )
    expect(padControlGaps(control)).toEqual([{ key: 'unnamed', count: 2 }])
  })

  it('sans bloc de datation, aucune ventilation n’est fabriquée — mais le manque reste vrai', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'a1'), prise(0, null), prise(1, null)],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.coverage.hasStats).toBe(false)
    expect(control.coverage.occupations).toBe(3)
    expect(padControlMissing(control)).toBe(2)
    expect(padControlGaps(control)).toEqual([])
  })
})

describe('buildPadControl — le tri et les colonnes', () => {
  it('trie les joueurs par total décroissant, et les camps aussi', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [
          prise(0, 'b1'),
          prise(1, 'b1'),
          prise(0, 'b1', 30),
          prise(0, 'a2'),
          prise(1, 'a1'),
          prise(0, 'a1', 50),
        ],
      } as Partial<ReplayDocument>),
      SB,
    )
    // t0 fait 3, t1 fait 3 : à égalité, le côté départage (ordre reproductible).
    expect(control.byTeam.map((g) => g.side)).toEqual(['t0', 't1', null])
    const t0 = control.byTeam[0]
    expect(t0.players.map((p) => p.name)).toEqual(['Alpha', 'Bravo'])
    expect(t0.players.map((p) => p.total)).toEqual([2, 1])
  })

  it('ne met en colonne que les socles réellement pris, du plus disputé au moins disputé', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(1, 'a1'), prise(0, 'a1'), prise(1, 'b1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.weapons).toEqual([EPEE, SNIPER])
  })

  it('un socle que personne n’a pris n’a AUCUNE colonne : un zéro n’est pas une mesure', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1')] } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.weapons).toEqual([SNIPER])
  })
})

describe('buildPadControl — la double porte', () => {
  it('aucune prise attribuée = rien à rendre, même avec des socles et des occupations', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, null), prise(1, null), prise(2, null)],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.hasData).toBe(false)
  })

  it('un film sans aucune occupation de socle ne rend rien non plus', () => {
    expect(buildPadControl(temoin(), SB).hasData).toBe(false)
  })

  it('une seule prise attribuée suffit à ouvrir la porte', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1')] } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.hasData).toBe(true)
  })
})
