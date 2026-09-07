import { describe, expect, it } from 'vitest'

import type { ReplayTrackReady } from '../../../lib/replay/replayNormalize'
import { drawnSwapAt, equippedWeapons } from './equippedLogic'
import { testReplayDoc as doc } from '../test/testDoc'

/** Une vie couvrant [start, end] sur un slot — même patron que rosterLogic.test.ts. */
function track(slot: number, xuid: string | undefined, start: number, end: number): ReplayTrackReady {
  return {
    slot,
    team: -1,
    xuid,
    startFrame: start,
    endFrame: end,
    points: [
      { t: start, x: 0, y: 0 },
      { t: end, x: 1, y: 1 },
    ],
  }
}

// Un slot, deux armes lues à l'image-clé t=10, et un inventaire dont seul le sélecteur
// d'emplacement varie d'un cas à l'autre : c'est lui qui est sous test. Vie couvrante 0-100 :
// le seul facteur testé reste le sélecteur, jamais la frontière de vie.
function docWithSelector(d?: number) {
  return doc({
    tracks: [track(512, 'A', 0, 100)],
    loadouts: [{ t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
    inventory: [{ t: 10, slot: 512, ...(d === undefined ? {} : { d }) }],
  })
}

describe('equippedWeapons — le sélecteur d’emplacement ordonne la rangée', () => {
  it('D=0 : l’emplacement 0 est EN MAIN, à gauche ; l’autre suit', () => {
    expect(equippedWeapons(docWithSelector(0), 512, 60)).toEqual({
      weapons: [
        { id: '0xAAAA', inHand: true },
        { id: '0xBBBB', inHand: false },
      ],
      order: [0, 1],
      drawn: 0,
      age: 50,
      holstered: false,
      drawnUnread: false,
    })
  })

  it('D=1 : l’emplacement 1 passe devant, marqué en main — et l’ordre publié le dit', () => {
    // `order` est ce que la ligne des MUNITIONS suit : si l'ordre publié divergeait de celui
    // des vignettes, chaque cellule se rattacherait à la mauvaise arme.
    expect(equippedWeapons(docWithSelector(1), 512, 60)).toEqual({
      weapons: [
        { id: '0xBBBB', inHand: true },
        { id: '0xAAAA', inHand: false },
      ],
      order: [1, 0],
      drawn: 1,
      age: 50,
      holstered: false,
      drawnUnread: false,
    })
  })

  it('D=2 : RIEN de dégainé — une mesure, pas une lacune : ordre d’emplacement, aucun marquage', () => {
    // C'est l'état des huit joueurs à la première image-clé, avant le départ.
    expect(equippedWeapons(docWithSelector(2), 512, 60)).toEqual({
      weapons: [
        { id: '0xAAAA', inHand: false },
        { id: '0xBBBB', inHand: false },
      ],
      order: [0, 1],
      drawn: null,
      age: 50,
      holstered: true,
      drawnUnread: false,
    })
  })

  it('sélecteur non lu : ordre d’emplacement, aucun marquage, et ce n’est PAS « rangées »', () => {
    const r = equippedWeapons(docWithSelector(undefined), 512, 60)
    expect(r?.weapons.every((w) => !w.inHand)).toBe(true)
    expect(r?.holstered).toBe(false)
    // La lacune du sélecteur est PUBLIÉE : c'est elle qui fait afficher « dégainée ? » à la
    // ligne des munitions, distinct du « rangées » mesuré.
    expect(r?.drawnUnread).toBe(true)
    expect(r?.drawn).toBeNull()
  })

  it('sélecteur désappariété (D pointe hors du loadout lu) : la mise en valeur s’abstient', () => {
    // Loadout à UNE arme, sélecteur disant « emplacement 1 » : les deux scans ne se
    // recouvrent pas — marquer une main serait afficher une certitude qu'on n'a pas.
    const d = doc({
      tracks: [track(512, 'A', 0, 100)],
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA'] }],
      inventory: [{ t: 10, slot: 512, d: 1 }],
    })
    expect(equippedWeapons(d, 512, 60)).toEqual({
      weapons: [{ id: '0xAAAA', inHand: false }],
      order: [0],
      drawn: null,
      age: 50,
      holstered: false,
      drawnUnread: false,
    })
  })

  it('le sélecteur d’un AUTRE slot ne fuit pas', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 100)],
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [{ t: 10, slot: 513, d: 1 }],
    })
    expect(equippedWeapons(d, 512, 60)?.weapons.every((w) => !w.inHand)).toBe(true)
  })

  it('sans loadout lu, rend null — la rangée affiche sa lacune, pas une liste vide', () => {
    expect(equippedWeapons(doc({ tracks: [track(512, 'A', 0, 100)] }), 512, 60)).toBeNull()
  })

  it('lecture VIDE : aucune arme en main, même si la lecture pleine antérieure en dégainait une', () => {
    // LE DÉFAUT QUE CE CAS FERME (revue adversariale, constat 3). `inventoryAt` substitue la
    // dernière lecture PLEINE pour que la LIGNE D'INVENTAIRE garde un équipement à montrer.
    // Reprendre le sélecteur de cette lecture-là faisait affirmer une arme DÉGAINÉE pour un
    // joueur que l'artefact déclare mort : la rangée d'armes affichait `drawn: 0`, `inHand`.
    const d = doc({
      tracks: [track(512, 'A', 0, 150)],
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA', '0xBBBB'] }],
      inventory: [
        { t: 10, slot: 512, g: [0, 2, 0, 0], d: 0 },
        { t: 100, slot: 512, empty: 'dead' },
      ],
    })
    expect(equippedWeapons(d, 512, 120)).toEqual({
      weapons: [
        { id: '0xAAAA', inHand: false },
        { id: '0xBBBB', inHand: false },
      ],
      order: [0, 1],
      drawn: null,
      age: 110,
      // PAS « rangées » : D=2 est une MESURE (« aucune arme dégainée »), une lecture vide n'en
      // est pas une. L'état est NON AFFIRMÉ, ce que dit `drawnUnread`.
      holstered: false,
      drawnUnread: true,
    })
  })

  it('lecture VIDE inexpliquée, sans lecture pleine antérieure : même abstention', () => {
    const d = doc({
      tracks: [track(512, 'A', 0, 100)],
      loadouts: [{ t: 10, slot: 512, w: ['0xAAAA'] }],
      inventory: [{ t: 20, slot: 512, empty: 'unknown' }],
    })
    const r = equippedWeapons(d, 512, 60)
    expect(r?.drawn).toBeNull()
    expect(r?.drawnUnread).toBe(true)
    expect(r?.holstered).toBe(false)
  })
})

describe('drawnSwapAt — la bascule du sélecteur, datée à l’image-clé', () => {
  // Deux lectures d'inventaire du même slot : la main passe de l'emplacement 0 au 1 à t=200.
  // BORNÉE À LA VIE EN COURS DU SLOT depuis le constat R6 (audit vies anonymes 2026-09-06,
  // même défaut et même remède que `loadoutAt`/`abilityAt`, correctif P0-2) : les fixtures
  // portent désormais une vie couvrante, sans quoi `currentLifeOf` ne rend rien et la
  // fonction s'abstient (`null`).
  const d = doc({
    tracks: [track(512, 'A', 0, 400)],
    inventory: [
      { t: 10, slot: 512, d: 0 },
      { t: 200, slot: 512, d: 1 },
      { t: 205, slot: 513, d: 0 }, // un autre slot ne compte pas
    ],
  })

  it('rend l’âge de la bascule dans la fenêtre d’animation', () => {
    expect(drawnSwapAt(d, 512, 205, 30)).toBe(5)
  })

  it('hors fenêtre, rien — l’animation ne rejoue pas un échange ancien', () => {
    expect(drawnSwapAt(d, 512, 300, 30)).toBeNull()
  })

  it('sans bascule (sélecteur stable), rien', () => {
    const stable = doc({
      tracks: [track(512, 'A', 0, 300)],
      inventory: [
        { t: 10, slot: 512, d: 1 },
        { t: 200, slot: 512, d: 1 },
      ],
    })
    expect(drawnSwapAt(stable, 512, 205, 30)).toBeNull()
  })

  it('un sélecteur non lu entre deux lectures ne fabrique PAS de bascule', () => {
    // d=2 (« rien de dégainé ») et d absent ne participent pas : seule une main lue qui
    // CHANGE d'emplacement est un échange.
    const gaps = doc({
      tracks: [track(512, 'A', 0, 300)],
      inventory: [
        { t: 10, slot: 512, d: 1 },
        { t: 100, slot: 512 },
        { t: 150, slot: 512, d: 2 },
        { t: 200, slot: 512, d: 1 },
      ],
    })
    expect(drawnSwapAt(gaps, 512, 205, 30)).toBeNull()
  })

  it('sans vie couvrante pour ce slot à cette image, aucune bascule n’est datée', () => {
    expect(drawnSwapAt(doc({ inventory: [{ t: 10, slot: 512, d: 0 }, { t: 200, slot: 512, d: 1 }] }), 512, 205, 30))
      .toBeNull()
  })

  /**
   * LE DÉFAUT QUE CE CAS FERME (constat R6, audit vies anonymes 2026-09-06). Un slot est
   * réattribué à chaque réapparition ET entre manches : la dernière lecture de la vie de A
   * (d=0) suivie de la première lecture de la vie de B (d=1) sur le MÊME slot recyclé n'est
   * PAS une bascule d'arme — ce sont deux joueurs différents. Avant le correctif, le balayage
   * ignorait la frontière de vie et datait une bascule fantôme à t=160 (âge 40, dans la
   * fenêtre) ; borné à la vie de B (qui commence à 150), le seul enregistrement qui reste dans
   * la fenêtre est sa toute première lecture — rien à comparer, donc aucune bascule.
   */
  it('BORNE À LA VIE EN COURS DU SLOT : une bascule entre deux vies d’un slot recyclé n’est jamais datée', () => {
    const recycled = doc({
      tracks: [track(512, 'A', 0, 100), track(512, 'B', 150, 300)],
      inventory: [
        { t: 10, slot: 512, d: 0 }, // vie de A
        { t: 90, slot: 512, d: 0 },
        { t: 160, slot: 512, d: 1 }, // vie de B — un AUTRE joueur, pas une bascule de A
      ],
    })
    expect(drawnSwapAt(recycled, 512, 200, 60)).toBeNull()
  })
})
