/**
 * focusFilter.test.ts — ce que les joueurs masqués font à chaque genre d'élément qui n'est pas un pion.
 *
 * Le joueur A tient le slot 1 (images 0-50) puis le slot 3 (60-100) ; B tient le slot 2 (0-100) ; C tient le slot 4.
 * La règle est celle de `ownershipShown` (DU-3 : un élément se dessine dès qu'un de ses joueurs est affiché ; testée
 * à part) : ici, on prouve QUELLES parties chaque élément déclare. Avec A et B seuls au roster, « Seul A » est
 * « B masqué » et « Masquer A » est « A masqué » ; la dernière suite ajoute C pour les éléments à plusieurs joueurs.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { buildFocusFilter, type FocusFilter } from './focusFilter'
import { testReplayDoc } from '../test/testDoc'

const tracks: NonNullable<ReplayDocument['tracks']> = [
  { slot: 1, team: 0, xuid: 'A', startFrame: 0, endFrame: 50, points: [{ t: 0, x: 0, y: 0 }, { t: 50, x: 1, y: 0 }] },
  { slot: 3, team: 0, xuid: 'A', startFrame: 60, endFrame: 100, points: [{ t: 60, x: 0, y: 0 }, { t: 100, x: 1, y: 0 }] },
  { slot: 2, team: 1, xuid: 'B', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 5, y: 5 }, { t: 100, x: 6, y: 5 }] },
  { slot: 4, team: 1, xuid: 'C', startFrame: 0, endFrame: 100, points: [{ t: 0, x: 8, y: 8 }, { t: 100, x: 9, y: 8 }] },
]

function harness(over: Partial<ReplayDocument> = {}) {
  const doc = testReplayDoc({ frameIntervalMs: 100, tracks, ...over })
  const player = (xuid: string): ReplayPlayer => ({ xuid, lives: doc.tracks.filter((t) => t.xuid === xuid) })
  // Le roster à deux joueurs de la plupart des cas : « Seul X » masque l'autre, « Masquer X » masque X.
  const duo = ['A', 'B']
  const hiddenFor = (mode: 'solo' | 'hide', xuid: string, roster: readonly string[]) =>
    (mode === 'hide' ? [xuid] : roster.filter((k) => k !== xuid)).map(player)
  const filter = (mode: 'solo' | 'hide', xuid = 'A'): FocusFilter => buildFocusFilter(doc, hiddenFor(mode, xuid, duo))!
  const hiding = (...xuids: string[]): FocusFilter => buildFocusFilter(doc, xuids.map(player))!
  return { doc, filter, hiding }
}

describe('buildFocusFilter', () => {
  it('aucun joueur masqué : aucun filtre', () => {
    const { doc } = harness()
    expect(buildFocusFilter(doc, null)).toBeNull()
  })
})

describe('armes au sol — lâcheur et ramasseur', () => {
  const arme = (over: object) => ({ dropper: -1, picker: -1, t0: 20, t1: 40, ...over })

  it('une arme lâchée par le joueur : Seul la garde, Masquer la retire', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 1 }))).toBe(true)
    expect(filter('hide').groundWeapon(arme({ dropper: 1 }))).toBe(false)
  })

  it('lâchée À LA MORT (une image après la fin de la vie) : elle reste au joueur qui vient de mourir', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 1, t0: 51 }))).toBe(true)
    expect(filter('hide').groundWeapon(arme({ dropper: 1, t0: 51 }))).toBe(false)
  })

  it('lâchée par un autre, reprise par le joueur : elle est aux deux, et se dessine tant que l’un est affiché', () => {
    const { filter } = harness()
    const a = arme({ dropper: 2, picker: 3, t1: 70 })
    expect(filter('solo').groundWeapon(a)).toBe(true)
    expect(filter('hide').groundWeapon(a)).toBe(true)
  })

  it('lâchée par un autre : Seul la retire, Masquer la garde', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 2 }))).toBe(false)
    expect(filter('hide').groundWeapon(arme({ dropper: 2 }))).toBe(true)
  })

  it('sans lâcheur ni ramasseur lu (arme de la carte) : elle n’est à personne, elle reste', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({}))).toBe(true)
    expect(filter('hide').groundWeapon(arme({}))).toBe(true)
  })

  it('lâchée par B (slot 2, toute la partie) : jamais à A, même tard dans le match', () => {
    const { filter } = harness()
    expect(filter('solo').groundWeapon(arme({ dropper: 2, t0: 80 }))).toBe(false)
  })
})

describe('poses d’équipement — le poseur', () => {
  it('une pose du joueur : Seul la garde, Masquer la retire', () => {
    const { filter } = harness()
    expect(filter('solo').placement({ owner: 1, t0: 20 })).toBe(true)
    expect(filter('hide').placement({ owner: 1, t0: 20 })).toBe(false)
  })

  it('un objet lâché à la mort (une image après la fin de la vie) est au joueur qui vient de mourir', () => {
    const { filter } = harness()
    expect(filter('hide').placement({ owner: 1, t0: 51 })).toBe(false)
    expect(filter('solo').placement({ owner: 1, t0: 51 })).toBe(true)
  })

  it('la pose d’un autre : Seul la retire, Masquer la garde', () => {
    const { filter } = harness()
    expect(filter('solo').placement({ owner: 2, t0: 20 })).toBe(false)
    expect(filter('hide').placement({ owner: 2, t0: 20 })).toBe(true)
  })

  it('sans poseur mesuré (owner négatif) : à personne, elle reste', () => {
    const { filter } = harness()
    expect(filter('solo').placement({ owner: -1, t0: 20 })).toBe(true)
    expect(filter('hide').placement({ owner: -1, t0: 20 })).toBe(true)
  })
})

describe('faille et aller-retour du translocateur — le porteur à l’image de l’échange', () => {
  it('le slot du joueur à cette image', () => {
    const { filter } = harness()
    expect(filter('solo').slotAt(3, 70)).toBe(true)
    expect(filter('hide').slotAt(3, 70)).toBe(false)
    expect(filter('solo').slotAt(2, 70)).toBe(false)
    expect(filter('hide').slotAt(2, 70)).toBe(true)
  })
})

describe('marque de révélation d’un capteur — le poseur ET le joueur révélé', () => {
  it('Seul la garde si le joueur est le poseur ou le révélé', () => {
    const { filter } = harness()
    expect(filter('solo').reveal({ owner: 1, ownerFrame: 20, slot: 2 }, 30)).toBe(true)
    expect(filter('solo').reveal({ owner: 2, ownerFrame: 20, slot: 1 }, 30)).toBe(true)
    expect(filter('solo', 'B').reveal({ owner: 1, ownerFrame: 20, slot: 2 }, 30)).toBe(true)
  })

  it('Masquer la retire quand le joueur est seul concerné, la garde tant que l’autre est affiché', () => {
    const { filter } = harness()
    expect(filter('hide').reveal({ owner: 1, ownerFrame: 20, slot: 2 }, 30)).toBe(true) // A et B : B est affiché
    expect(filter('hide').reveal({ owner: 2, ownerFrame: 20, slot: 1 }, 30)).toBe(true)
    expect(filter('hide', 'B').reveal({ owner: 2, ownerFrame: 20, slot: 9 }, 30)).toBe(false) // B seul (slot 9 : personne)
    expect(filter('hide').reveal({ owner: 2, ownerFrame: 20, slot: 2 }, 30)).toBe(true)
  })
})

describe('porteur, VIP, auteur d’une action — le xuid du document', () => {
  it('Seul garde ce que le joueur porte ou accomplit, Masquer le retire', () => {
    const { filter } = harness()
    expect(filter('solo').author('A')).toBe(true)
    expect(filter('hide').author('A')).toBe(false)
    expect(filter('solo').author('B')).toBe(false)
    expect(filter('hide').author('B')).toBe(true)
  })

  it('un bot masqué est reconnu par le xuid du fil de sa ligne de feuille', () => {
    const { doc } = harness()
    const marc = { xuid: 'bot:Marc', lives: [], board: { xuid: 'bid(1.0)' } } as never
    const f = buildFocusFilter(doc, [marc])!
    expect(f.author('bid(1.0)')).toBe(false)
    expect(f.author('A')).toBe(true)
  })
})

describe('véhicules — leurs occupants', () => {
  const rides = (...slots: number[]) =>
    slots.map((slot, i) => ({ t0: 0, t1: 100, slot, seat: i, src: 'event', aim: [] }))
  const vehicule = (...slots: number[]) => ({ slot: 700, rides: rides(...slots) }) as never

  it('un véhicule où le joueur est monté : Seul le garde, Masquer le retire quand il est seul à bord', () => {
    const { filter } = harness()
    expect(filter('solo').vehicleAt(vehicule(1), 20)).toBe(true)
    expect(filter('hide').vehicleAt(vehicule(1), 20)).toBe(false)
  })

  it('un véhicule d’un autre : Seul le retire, Masquer le garde', () => {
    const { filter } = harness()
    expect(filter('solo').vehicleAt(vehicule(2), 20)).toBe(false)
    expect(filter('hide').vehicleAt(vehicule(2), 20)).toBe(true)
  })

  it('PARTAGÉ : Seul le garde dès que le joueur est à bord ; Masquer ne l’emporte pas avec son co-occupant', () => {
    const { filter } = harness()
    expect(filter('solo').vehicleAt(vehicule(2, 1), 20)).toBe(true)
    expect(filter('hide').vehicleAt(vehicule(2, 1), 20)).toBe(true)
  })

  it('un véhicule VIDE n’est à personne : il reste', () => {
    const { filter } = harness()
    expect(filter('solo').vehicleAt(vehicule(), 20)).toBe(true)
    expect(filter('hide').vehicleAt(vehicule(), 20)).toBe(true)
  })

  it('l’occupant nommé : à lui seul, y compris dans un véhicule partagé', () => {
    const { filter } = harness()
    expect(filter('hide').occupant({ slot: 1 }, 20)).toBe(false)
    expect(filter('hide').occupant({ slot: 2 }, 20)).toBe(true)
    expect(filter('solo').occupant({ slot: 2 }, 20)).toBe(false)
  })

  it('l’occupant se reconnaît aussi par le xuid que le document lui donne', () => {
    const { filter } = harness()
    expect(filter('hide').occupant({ slot: 99, xuid: 'A' }, 20)).toBe(false)
  })

  it('l’explosion est à ceux qui étaient à bord ET à celui qui a détruit', () => {
    const { filter } = harness()
    const detruit = (by?: string) =>
      ({ slot: 700, rides: rides(2), destruction: by ? { by: { i: 0, xuid: by }, killed: [] } : { killed: [] } }) as never
    expect(filter('hide').vehicleDestruction(detruit(), 20)).toBe(true)
    expect(filter('hide').vehicleDestruction(detruit('A'), 20)).toBe(true) // A masqué, B à bord affiché
    expect(filter('solo').vehicleDestruction(detruit('A'), 20)).toBe(true)
    expect(filter('solo').vehicleDestruction(detruit(), 20)).toBe(false)
    expect(filter('hide').vehicleDestruction({ slot: 700, rides: rides(1) } as never, 20)).toBe(false)
  })
})

describe('poussée du répulseur — la personne poussée ET son auteur', () => {
  const poussee = (slot: number, by: number | null) => ({ slot, by, frame: 30 })

  it('Seul la garde si le joueur est l’auteur, et si le joueur est le poussé', () => {
    const { filter } = harness()
    expect(filter('solo').knockback(poussee(2, 1))).toBe(true) // A pousse B
    expect(filter('solo').knockback(poussee(1, 2))).toBe(true) // B pousse A
    expect(filter('solo').knockback(poussee(2, 9))).toBe(false) // ni l'un ni l'autre
  })

  it('Masquer l’un des deux la laisse visible, et garde celle qui ne le concerne pas', () => {
    const { filter } = harness()
    expect(filter('hide').knockback(poussee(2, 1))).toBe(true) // A pousse B : B est affiché
    expect(filter('hide').knockback(poussee(1, 2))).toBe(true) // B pousse A
    expect(filter('hide').knockback(poussee(2, 9))).toBe(true)
  })

  it('sans auteur publié : le poussé décide', () => {
    const { filter } = harness()
    expect(filter('solo').knockback(poussee(1, null))).toBe(true)
    expect(filter('solo').knockback(poussee(2, null))).toBe(false)
    expect(filter('hide').knockback(poussee(1, null))).toBe(false)
    expect(filter('hide').knockback(poussee(2, null))).toBe(true)
  })
})

describe('DU-3 — un élément se dessine dès qu’un de ses joueurs est affiché (A, B et C au roster)', () => {
  const poussee = (slot: number, by: number | null) => ({ slot, by, frame: 30 })
  const rides = (...slots: number[]) => slots.map((slot, i) => ({ t0: 0, t1: 100, slot, seat: i, src: 'event', aim: [] }))
  const vehicule = (...slots: number[]) => ({ slot: 700, rides: rides(...slots) }) as never

  it('un élément MIXTE (un joueur masqué, un joueur affiché) se dessine', () => {
    const { hiding } = harness()
    expect(hiding('A').knockback(poussee(2, 1))).toBe(true)
    expect(hiding('A').groundWeapon({ dropper: 1, picker: 2, t0: 20, t1: 40 })).toBe(true)
    expect(hiding('A').vehicleAt(vehicule(1, 2), 20)).toBe(true)
    expect(hiding('A').reveal({ owner: 1, ownerFrame: 20, slot: 2 }, 30)).toBe(true)
  })

  it('un élément dont TOUS les joueurs sont masqués est retiré', () => {
    const { hiding } = harness()
    expect(hiding('A', 'B').knockback(poussee(2, 1))).toBe(false)
    expect(hiding('A', 'B').groundWeapon({ dropper: 1, picker: 2, t0: 20, t1: 40 })).toBe(false)
    expect(hiding('A', 'B').vehicleAt(vehicule(1, 2), 20)).toBe(false)
    expect(hiding('A', 'B').reveal({ owner: 1, ownerFrame: 20, slot: 2 }, 30)).toBe(false)
  })

  it('un élément où un joueur AFFICHÉ figure encore se dessine, même avec deux joueurs masqués', () => {
    const { hiding } = harness()
    expect(hiding('A', 'B').knockback(poussee(4, 1))).toBe(true) // C est poussé par A
    expect(hiding('A', 'B').vehicleAt(vehicule(1, 2, 4), 20)).toBe(true)
  })

  it('Seul C (A et B masqués) : seul ce qui concerne C se dessine', () => {
    const { hiding } = harness()
    const seulC = hiding('A', 'B')
    expect(seulC.slotAt(4, 20)).toBe(true)
    expect(seulC.slotAt(1, 20)).toBe(false)
    expect(seulC.slotAt(2, 20)).toBe(false)
    expect(seulC.author('C')).toBe(true)
    expect(seulC.author('A')).toBe(false)
  })

  it('le décor (aucun joueur) reste affiché, quel que soit l’ensemble masqué', () => {
    const { hiding } = harness()
    const tous = hiding('A', 'B', 'C')
    expect(tous.groundWeapon({ dropper: -1, picker: -1, t0: 20, t1: 40 })).toBe(true)
    expect(tous.placement({ owner: -1, t0: 20 })).toBe(true)
    expect(tous.vehicleAt(vehicule(), 20)).toBe(true)
    expect(tous.knockback(poussee(9, null))).toBe(true)
  })
})
