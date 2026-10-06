/**
 * Tests — padControlLogic (le contrôle des armes spéciales, par joueur et par équipe).
 *
 * CE QU'ILS PROTÈGENT, dans l'ordre des pièges du domaine :
 *   - le PONT xuid -> joueur -> équipe : le camp est le désignateur que le FILM écrit pour l'entrée
 *     du joueur (décision du 2026-10-06), la feuille ne fait que le nommer ;
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
  type PadControl,
} from './padControlLogic'
import { testReplayDoc } from '../test/testDoc'

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
 * LE TÉMOIN. Trois joueurs au scoreboard, un QUATRIÈME que le film voit vivre et que le scoreboard
 * ignore, et un CINQUIÈME au roster du film SANS AUCUNE PISTE, dans deux camps DU FILM (Alpha et
 * Bravo au camp 0, Charlie, Delta et Echo au camp 1) — deux socles d'arme et un socle de bonus.
 *
 * L'ENTRÉE SANS PISTE EST DANS LE TÉMOIN PARTAGÉ À DESSEIN : le filtre « au moins une vie » doit
 * tenir sur tous les scénarios, pas seulement sur celui qui l'éprouve.
 */
function temoin(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameCount: 200,
    frameIntervalMs: 100,
    roster: [
      { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
      { filmIndex: 1, xuid: 'a2', name: 'Bravo', team: 0 },
      { filmIndex: 2, xuid: 'b1', name: 'Charlie', team: 1 },
      { filmIndex: 3, xuid: 'orphelin', name: 'Delta', team: 1 },
      { filmIndex: 4, xuid: 'sansPiste', name: 'Echo', team: 1 },
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

  it('range les joueurs par camp du FILM et somme chaque camp', () => {
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'a1'), prise(1, 'a2'), prise(0, 'b1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    const t0 = control.byTeam.find((g) => g.team === 0)
    const t1 = control.byTeam.find((g) => g.team === 1)
    expect([t0?.side, t1?.side]).toEqual(['t0', 't1'])
    expect(t0?.total.total).toBe(2)
    expect(t0?.total.byWeapon[SNIPER]).toBe(1)
    expect(t0?.total.byWeapon[EPEE]).toBe(1)
    expect(t1?.total.total).toBe(1)
  })

  it('le joueur HORS SCOREBOARD garde sa ligne, dans le camp que le FILM lui donne', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'orphelin')] } as Partial<ReplayDocument>),
      SB,
    )
    // Delta n'a pas de ligne de feuille : sa prise compte pour le camp 1, que la feuille de
    // Charlie nomme. Aucun groupe « sans équipe ».
    expect(control.byTeam.map((g) => g.team)).toEqual([1, 0])
    expect(control.byTeam[0].players.find((p) => p.name === 'Delta')).toMatchObject({ team: 1, side: 't1', total: 1 })
  })

  it('sans scoreboard du tout, les camps restent ceux du FILM — seuls leurs côtés de feuille manquent', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1')] } as Partial<ReplayDocument>),
      undefined,
    )
    expect(control.byTeam.map((g) => [g.team, g.side])).toEqual([
      [0, null],
      [1, null],
    ])
    expect(control.attributed).toBe(1)
  })

  it('un joueur dont le film TAIT l’équipe n’a AUCUNE ligne : sa prise part en non rattachée', () => {
    // Le bot bouche-trou du témoin 43716616 : vivant, une prise nommée — et aucune équipe écrite
    // par le film. Ni ligne, ni groupe à part : la prise compte dans `unjoined`.
    const control = buildPadControl(
      temoin({
        roster: [
          { filmIndex: 0, xuid: 'a1', name: 'Alpha', team: 0 },
          { filmIndex: 8, xuid: 'muet', name: 'Sandwolf' },
        ],
        tracks: [vie(1, 'a1'), vie(8, 'muet')],
        padPickups: [prise(0, 'a1'), prise(0, 'muet', 30)],
      } as Partial<ReplayDocument>),
      // La feuille connaît le muet (côté t0) : elle ne lui donne pas de ligne pour autant.
      [...SB, { xuid: 'muet', gamertag: 'Sandwolf', team_side: 't0' }] as MatchScoreboardRow[],
    )
    expect([...parNom(control).keys()]).toEqual(['Alpha'])
    expect(control.byTeam.map((g) => g.team)).toEqual([0])
    expect(control.unjoined).toBe(1)
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

  it('une entrée de roster SANS PISTE n’a pas de ligne, et sa prise part en hors-film', () => {
    // Le film le nomme au roster mais ne l'a jamais vu vivre : une ligne de zéros le ferait
    // passer pour quelqu'un qui n'a pris aucun socle, et une ligne à 1 pour un joueur du match.
    const control = buildPadControl(
      temoin({
        padPickups: [prise(0, 'sansPiste'), prise(0, 'a1')],
      } as Partial<ReplayDocument>),
      SB,
    )
    expect([...parNom(control).keys()]).not.toContain('Echo')
    expect(control.unjoined).toBe(1)
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
    // La VENTILATION PAR CAUSE n'est plus rendue (pied de carte retiré le 2026-09-13) ; ce qui
    // reste mesuré, et rendu ligne par ligne, c'est ce que le graphe attribue et ce qu'il annote.
    expect(control.attributed).toBe(2)
  })


})

describe('buildPadControl — le tri et les colonnes', () => {
  it('trie les joueurs par total décroissant, et les camps aussi', () => {
    // TOTAUX TOUS DISTINCTS, ET L'ORDRE ATTENDU N'EST NI CELUI DU ROSTER NI L'ALPHABÉTIQUE :
    // c'est ce qui rend le test sensible. Charlie (camp 1) 5 · Bravo (camp 0) 3 · Alpha (camp 0)
    // 1 · Delta (camp 1) 0. Roster et alphabet donneraient tous deux « Alpha, Bravo » dans le
    // camp 0, et « 0, 1 » pour les camps — retirer l'un ou l'autre des deux tris fait donc tomber
    // cette assertion.
    const control = buildPadControl(
      temoin({
        padPickups: [
          prise(0, 'a1'),
          prise(0, 'a2'),
          prise(1, 'a2', 20),
          prise(0, 'a2', 30),
          prise(1, 'b1'),
          prise(1, 'b1', 20),
          prise(0, 'b1', 30),
          prise(1, 'b1', 40),
          prise(0, 'b1', 50),
        ],
      } as Partial<ReplayDocument>),
      SB,
    )
    // Les camps : le 1 (5) devant le 0 (4) — et aucun troisième.
    expect(control.byTeam.map((g) => g.team)).toEqual([1, 0])
    expect(control.byTeam.map((g) => g.total.total)).toEqual([5, 4])
    // Dans t0, Bravo (3) passe DEVANT Alpha (1) — l'inverse du roster et de l'alphabet.
    const t0 = control.byTeam[1]
    expect(t0.players.map((p) => p.name)).toEqual(['Bravo', 'Alpha'])
    expect(t0.players.map((p) => p.total)).toEqual([3, 1])
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

describe('buildPadControl — la partition « game changers » (plan 2026-09-05)', () => {
  const BR = '0xCCCC3333'

  /**
   * QUATRE SOCLES, ET LE PLUS DISPUTÉ EST REPLIÉ — c'est ce qui rend la partition sensible :
   * un tri global par volume mettrait le Commando (3 prises, voté NON) devant le Sniper
   * (1 prise, élu). La partition passe AVANT le tri, qui survit dans chaque bloc.
   */
  function temoinCatalogue() {
    return temoin({
      weaponLabels: {
        [SNIPER]: { fr: 'S7 Sniper', en: 'S7 Sniper', key: 'hinf_s7_sniper' },
        // Clé CANONIQUE présente mais votée NON : repliée (le cindershot, lui, a été promu).
        [EPEE]: { fr: 'VK78 Commando', en: 'VK78 Commando', key: 'hinf_vk78_commando' },
        // Label SANS clé (artefact ancien) : replié, dégradation voulue (décision D6).
        [BR]: { fr: 'BR75', en: 'BR75' },
      },
      weaponPads: [socle(SNIPER), socle(EPEE), socle('powerup_overshield'), socle(BR)],
      padPickups: [
        prise(0, 'a1', 10),
        prise(1, 'a1', 20),
        prise(1, 'a2', 30),
        prise(1, 'b1', 40),
        prise(3, 'a1', 50),
        prise(3, 'b1', 60),
        prise(2, 'a1', 70),
      ],
    } as unknown as Partial<ReplayDocument>)
  }

  it('ORDONNE AVANT DE TRIER : un socle non élu très disputé ne double jamais un élu', () => {
    // LE VOTE EST UN ORDRE, PLUS UN REPLI (2026-09-13) : toutes les armes sont rendues, les
    // socles décisifs en tête, chaque bloc trié par volume.
    const control = buildPadControl(temoinCatalogue(), SB)
    // Élus : Sniper (1) et socle de bonus (1) — à égalité, l'identifiant départage. Puis les
    // autres, TRIÉS PAR VOLUME : Crémateur (3) devant BR (2).
    expect(control.weapons).toEqual([SNIPER, 'powerup_overshield', EPEE, BR])
  })

  it('un artefact SANS catalogue garde toutes les armes, par volume : clé absente = jamais promu (D6)', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1'), prise(1, 'b1')] } as Partial<ReplayDocument>),
      SB,
    )
    expect(control.weapons).toHaveLength(2)
  })

  it('le TOTAL ne ment pas : lignes, camps et somme attribuée comptent TOUTES les armes', () => {
    const control = buildPadControl(temoinCatalogue(), SB)
    const alpha = parNom(control).get('Alpha')
    // Alpha : Sniper 1 + Crémateur 1 + BR 1 + bonus 1 — les trois repliés comptent.
    expect(alpha?.total).toBe(4)
    expect(alpha?.byWeapon[EPEE]).toBe(1)
    expect(alpha?.byWeapon[BR]).toBe(1)
    expect(control.attributed).toBe(7)
    const t0 = control.byTeam.find((g) => g.side === 't0')
    expect(t0?.total.byWeapon[EPEE]).toBe(2)
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

/**
 * LES NIVEAUX D'ARMES (2026-09-14) — le bloc range ses lignes en base / terrain / puissance /
 * non classé. La règle elle-même est éprouvée chez `weaponTier.test.ts` ; ici on éprouve son
 * RATTACHEMENT aux lignes : un niveau par arme, des sous-totaux qui bouclent sur le total
 * attribué, et les deux drapeaux que l'écran lit pour savoir quoi dire.
 */
describe('buildPadControl — les niveaux', () => {
  const CROISEMENT = {
    catalogN: 5,
    pads: [
      { x: 0, y: 0, pad: 0, family: 'power' },
      { x: 1, y: 0, pad: 1, family: 'rack' },
      { x: 2, y: 0, pad: 2, family: 'powerup' },
    ],
  }

  it('pose un niveau par arme, et le niveau vient de la CARTE', () => {
    const control = buildPadControl(
      temoin({
        mapWeaponPads: CROISEMENT,
        padPickups: [prise(0, 'a1', 10), prise(0, 'b1', 30), prise(1, 'a2', 50)],
      } as unknown as Partial<ReplayDocument>),
      SB,
    )
    expect(control.tierOfWeapon[SNIPER]).toBe('power')
    expect(control.tierOfWeapon[EPEE]).toBe('ground')
    expect(control.tiersMeasured).toBe(true)
    expect(control.randomStarts).toBe(false)
  })

  it('range tout en « non classé » et le SIGNALE quand aucun emplacement ne confirme', () => {
    const control = buildPadControl(
      temoin({ padPickups: [prise(0, 'a1'), prise(1, 'b1')] } as unknown as Partial<ReplayDocument>),
      SB,
    )
    expect(control.tiersMeasured).toBe(false)
    expect(control.tierOfWeapon[SNIPER]).toBe('unclassified')
  })

  it('n’attribue aucun niveau « base » sur un mode à départs aléatoires', () => {
    const loadouts = Array.from({ length: 20 }, (_, i) => ({ t: 0, slot: 512 + i, w: [SNIPER] }))
    const doc = { mapWeaponPads: CROISEMENT, loadouts, padPickups: [prise(0, 'a1')] }
    const regulier = buildPadControl(temoin(doc as unknown as Partial<ReplayDocument>), SB)
    expect(regulier.tierOfWeapon[SNIPER]).toBe('base')
    const aleatoire = buildPadControl(
      temoin({ ...doc, weaponTiers: { randomStarts: true } } as unknown as Partial<ReplayDocument>),
      SB,
    )
    expect(aleatoire.randomStarts).toBe(true)
    expect(aleatoire.tierOfWeapon[SNIPER]).toBe('power')
  })
})
