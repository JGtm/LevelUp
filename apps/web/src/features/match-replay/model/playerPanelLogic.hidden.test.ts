/**
 * Tests — ce que les joueurs masqués font à la carte (suite de `playerPanelLogic.test.ts`) : le prédicat des pions
 * (`shownSlotPredicate`), celui des effets de mort (`killFxShown`) et LA règle des éléments à parties
 * (`ownershipShown`, DU-3 : un élément se dessine dès qu'un de ses joueurs est affiché ; sans joueur, il reste).
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { killFxShown, ownershipShown, shownSlotPredicate } from './playerPanelLogic'
import { killFxEntry } from '../test/killFxEntry'

function player(xuid: string, lives: Array<{ slot: number; start: number; end: number }>, extra: Partial<ReplayPlayer> = {}): ReplayPlayer {
  return {
    xuid,
    lives: lives.map((l) => ({
      slot: l.slot, team: 0, xuid, startFrame: l.start, endFrame: l.end,
      points: [{ t: l.start, x: 0, y: 0 }],
    })) as never,
    ...extra,
  }
}

const A = player('A', [{ slot: 1, start: 0, end: 50 }, { slot: 3, start: 60, end: 100 }])
const B = player('B', [{ slot: 2, start: 0, end: 100 }])
const BOT = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], { bot: true })

describe('shownSlotPredicate — un pion se dessine si son joueur n’est pas masqué', () => {
  it('aucun joueur masqué : aucun filtre à brancher', () => {
    expect(shownSlotPredicate(null)).toBeNull()
  })

  it('un joueur masqué : ses vies sont retirées, et seulement elles', () => {
    const shown = shownSlotPredicate([A])!
    expect(shown(1, 10)).toBe(false) // sa première vie
    expect(shown(3, 70)).toBe(false) // sa seconde vie, autre slot
    expect(shown(2, 10)).toBe(true) // un autre joueur
  })

  it('plusieurs joueurs masqués : les vies de chacun sont retirées', () => {
    const shown = shownSlotPredicate([A, B])!
    expect(shown(1, 10)).toBe(false)
    expect(shown(2, 10)).toBe(false)
    expect(shown(7, 10)).toBe(true)
  })

  it('le slot réattribué : la vie d’un AUTRE joueur sur le même slot reste affichée', () => {
    // Le slot 1 appartient à A jusqu'à l'image 50 ; à l'image 80 il est tenu par un autre.
    expect(shownSlotPredicate([A])!(1, 80)).toBe(true)
  })

  it('Seul (tous les autres masqués) : seules les vies du joueur seul restent', () => {
    const shown = shownSlotPredicate([B, BOT])!
    expect(shown(1, 10)).toBe(true)
    expect(shown(3, 70)).toBe(true)
    expect(shown(2, 10)).toBe(false)
    expect(shown(7, 10)).toBe(false)
  })

  it('un bot se filtre par ses vies : il n’a pas de xuid', () => {
    const shown = shownSlotPredicate([BOT])!
    expect(shown(7, 20)).toBe(false)
    expect(shown(1, 20)).toBe(true)
  })
})

describe('killFxShown — un effet de mort appartient au tueur ET à la victime (DU-3)', () => {
  // A (slot 1 puis 3), B au slot 2, le bot au slot 7 ; le tueur et la victime se lisent dans leur vie.
  const aTueB = killFxEntry({ slot: 1, slotFrame: 10, victimSlot: 2, victimFrame: 10 })
  const bTueA = killFxEntry({ slot: 2, slotFrame: 10, victimSlot: 1, victimFrame: 10 })
  const bTueBot = killFxEntry({ slot: 2, slotFrame: 10, victimSlot: 7, victimFrame: 10 })

  it('aucun joueur masqué : aucun filtre', () => {
    expect(killFxShown(null)).toBeNull()
  })

  it('un effet dont UN SEUL joueur est masqué se dessine : l’autre joueur est affiché', () => {
    const shown = killFxShown([A])!
    expect(shown(aTueB)).toBe(true)
    expect(shown(bTueA)).toBe(true)
  })

  it('un effet dont TOUS les joueurs sont masqués est retiré, un effet sans eux se dessine', () => {
    const shown = killFxShown([A, B])!
    expect(shown(aTueB)).toBe(false)
    expect(shown(bTueA)).toBe(false)
    expect(shown(bTueBot)).toBe(true) // le bot est affiché
  })

  it('Seul A (B et le bot masqués) : l’effet où A tue ou meurt reste, celui qui ne le concerne pas est retiré', () => {
    const shown = killFxShown([B, BOT])!
    expect(shown(aTueB)).toBe(true)
    expect(shown(bTueA)).toBe(true)
    expect(shown(bTueBot)).toBe(false)
  })

  it('le slot réattribué : l’effet d’un AUTRE joueur sur le slot 1 après la vie de A reste à lui', () => {
    const autre = killFxEntry({ slot: 1, slotFrame: 80, victimSlot: 2, victimFrame: 80 })
    // À l'image 80, le slot 1 n'est plus à A : cette partie n'est pas masquée, l'effet se dessine.
    expect(killFxShown([A])!(autre)).toBe(true)
    expect(killFxShown([A, B])!(autre)).toBe(true)
  })

  it('tueur inconnu : la victime décide', () => {
    const tueurInconnu = (victimSlot: number) =>
      killFxEntry({ slot: null, killerXuid: '', victimSlot, victimFrame: 10 })
    expect(killFxShown([A])!(tueurInconnu(1))).toBe(false)
    expect(killFxShown([A])!(tueurInconnu(2))).toBe(true)
  })

  it('victime inconnue : le tueur décide', () => {
    const victimeInconnue = (slot: number) =>
      killFxEntry({ slot, slotFrame: 10, victimSlot: null, victimXuid: null })
    expect(killFxShown([A])!(victimeInconnue(1))).toBe(false)
    expect(killFxShown([A])!(victimeInconnue(2))).toBe(true)
  })

  it('aucune partie lue : l’effet reste affiché, rien n’est deviné', () => {
    const rien = killFxEntry({ slot: null, killerXuid: '', victimSlot: null, victimXuid: null })
    expect(killFxShown([A])!(rien)).toBe(true)
    expect(killFxShown([A, B, BOT])!(rien)).toBe(true)
  })

  it('un bot se reconnaît par la clé du fil de sa ligne de feuille quand sa vie n’est pas retrouvée', () => {
    const marc = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], {
      bot: true,
      board: { xuid: 'bid(1.0)' } as never,
    })
    const parCle = killFxEntry({ slot: null, killerXuid: 'bid(1.0)', victimSlot: null, victimXuid: null })
    expect(killFxShown([marc])!(parCle)).toBe(false)
    expect(killFxShown([A])!(parCle)).toBe(true)
  })
})

describe('ownershipShown — la règle unique des éléments à parties (DU-3)', () => {
  it('aucun joueur masqué : aucun filtre', () => {
    expect(ownershipShown(null)).toBeNull()
  })

  it('un élément sans partie n’appartient à personne : il reste', () => {
    expect(ownershipShown([A])!([])).toBe(true)
    expect(ownershipShown([A, B])!([])).toBe(true)
  })

  it('une partie par xuid du document (porteur, auteur) : masquée, elle retire l’élément', () => {
    expect(ownershipShown([A])!([{ xuid: 'A' }])).toBe(false)
    expect(ownershipShown([A])!([{ xuid: 'B' }])).toBe(true)
  })

  it('un élément MIXTE (un joueur masqué, un joueur affiché) se dessine', () => {
    expect(ownershipShown([A])!([{ xuid: 'A' }, { xuid: 'B' }])).toBe(true)
    expect(ownershipShown([A])!([{ xuid: 'B' }, { xuid: 'A' }])).toBe(true)
  })

  it('un élément dont TOUS les joueurs sont masqués est retiré', () => {
    expect(ownershipShown([A, B])!([{ xuid: 'A' }, { xuid: 'B' }])).toBe(false)
    expect(ownershipShown([A, B])!([{ xuid: 'A' }, { xuid: 'B' }, { xuid: 'C' }])).toBe(true)
  })

  it('Seul A : l’élément de A reste, celui de B retiré, le mixte A + B reste', () => {
    const seulA = ownershipShown([B, BOT])!
    expect(seulA([{ xuid: 'A' }])).toBe(true)
    expect(seulA([{ xuid: 'B' }])).toBe(false)
    expect(seulA([{ xuid: 'B' }, { xuid: 'bot:Marc' }])).toBe(false)
    expect(seulA([{ xuid: 'A' }, { xuid: 'B' }])).toBe(true)
  })

  it('une partie prouvée par sa vie OU par son xuid', () => {
    const shown = ownershipShown([A])!
    expect(shown([{ life: { slot: 3, frame: 70 } }])).toBe(false)
    expect(shown([{ life: { slot: 9, frame: 70 }, xuid: 'A' }])).toBe(false)
    expect(shown([{ life: { slot: 9, frame: 70 }, xuid: 'B' }])).toBe(true)
  })
})
