/**
 * Tests — la logique pure du panneau de joueur (lot 4 du plan « maquettes du rejeu »).
 *
 * Ce qu'ils protègent :
 *  1. LE JOUEUR CHOISI : un clic sur une fiche ouvre le panneau, un autre joueur REMPLACE le
 *     contenu (Seul et Masquer remis à zéro), recliquer la fiche ouverte referme.
 *  2. SUIVRE EST LA CAMÉRA, INDÉPENDANTE DE LA FICHE : fermer ou changer de fiche ne l'éteint pas, la
 *     demande est acceptée fiche fermée ; le joueur suivi est le joueur REGARDÉ (`trackedPlayerOf`).
 *  3. LES EXCLUSIONS : Seul coupe Masquer ; Suivre coupe Masquer ; Masquer coupe Seul et
 *     le Suivre de ce joueur. « Retour au fil » remet Seul et Masquer à zéro.
 *  4. CE QUE LA CARTE MONTRE : sans Seul ni Masquer, aucun filtre (`null`, rien à brancher) ;
 *     sinon le prédicat désigne les vies du joueur choisi, bot compris (par ses vies, jamais par
 *     un xuid qu'il n'a pas).
 *  5. LES ÉLÉMENTS À PARTIES (effets de mort…) : une règle, `ownershipShown`, pour tous les calques.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  followedPlayerOf,
  killFxShown,
  NO_PLAYER_PANEL,
  ownershipShown,
  panelFocusOf,
  reducePlayerPanel,
  shownSlotPredicate,
  trackedPlayerOf,
  type PlayerPanelState,
} from './playerPanelLogic'
import { killFxEntry } from '../test/killFxEntry'
import { scoreboardRow } from '../test/scoreboardRow'

const OPEN_A: PlayerPanelState = { open: true, xuid: 'A', followXuid: null, solo: false, hide: false }

describe('reducePlayerPanel — le joueur choisi', () => {
  it('un clic sur une fiche ouvre le panneau sur ce joueur, actions éteintes', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'card', xuid: 'A' })).toEqual(OPEN_A)
  })

  it('une AUTRE fiche remplace le joueur et remet Seul et Masquer à zéro, la caméra garde son joueur', () => {
    const busy: PlayerPanelState = { ...OPEN_A, followXuid: 'A', solo: true }
    expect(reducePlayerPanel(busy, { type: 'card', xuid: 'B' })).toEqual({ ...OPEN_A, xuid: 'B', followXuid: 'A' })
    const hidden: PlayerPanelState = { ...OPEN_A, hide: true }
    expect(reducePlayerPanel(hidden, { type: 'card', xuid: 'B' })).toEqual({ ...OPEN_A, xuid: 'B' })
  })

  it('recliquer la fiche OUVERTE referme et remet Seul et Masquer à zéro', () => {
    const busy: PlayerPanelState = { ...OPEN_A, hide: true }
    expect(reducePlayerPanel(busy, { type: 'card', xuid: 'A' })).toEqual(NO_PLAYER_PANEL)
  })

  it('recliquer la fiche ouverte referme sans éteindre la caméra', () => {
    const busy: PlayerPanelState = { ...OPEN_A, followXuid: 'A', solo: true }
    expect(reducePlayerPanel(busy, { type: 'card', xuid: 'A' })).toEqual({ ...NO_PLAYER_PANEL, followXuid: 'A' })
  })

  it('« Retour au fil » referme, arrête Seul et Masquer, et laisse la caméra allumée', () => {
    const busy: PlayerPanelState = { ...OPEN_A, followXuid: 'A', solo: true }
    expect(reducePlayerPanel(busy, { type: 'close' })).toEqual({ ...NO_PLAYER_PANEL, followXuid: 'A' })
  })

  it('fermer un panneau déjà fermé rend le même objet (personne n’est réveillé)', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'close' })).toBe(NO_PLAYER_PANEL)
    const following: PlayerPanelState = { ...NO_PLAYER_PANEL, followXuid: 'A' }
    expect(reducePlayerPanel(following, { type: 'close' })).toBe(following)
  })
})

describe('reducePlayerPanel — Suivre, la caméra, indépendante de la fiche', () => {
  it('« Suivre tel joueur » est accepté fiche fermée', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'follow', xuid: 'B' })).toEqual({
      ...NO_PLAYER_PANEL, followXuid: 'B',
    })
  })

  it('suivre un joueur dont la fiche n’est pas ouverte ne touche pas à la fiche ouverte', () => {
    const s = reducePlayerPanel({ ...OPEN_A, solo: true }, { type: 'follow', xuid: 'B' })
    expect(s).toEqual({ open: true, xuid: 'A', followXuid: 'B', solo: true, hide: false })
  })

  it('ouvrir une fiche sur le joueur suivi, puis une autre, laisse la caméra sur lui', () => {
    let s = reducePlayerPanel(NO_PLAYER_PANEL, { type: 'follow', xuid: 'A' })
    s = reducePlayerPanel(s, { type: 'card', xuid: 'A' })
    s = reducePlayerPanel(s, { type: 'card', xuid: 'B' })
    expect(s).toMatchObject({ open: true, xuid: 'B', followXuid: 'A' })
  })

  it('« Suivre » de la fiche allume la caméra sur son joueur, et rebasculé l’éteint', () => {
    const on = reducePlayerPanel(OPEN_A, { type: 'toggle', action: 'follow' })
    expect(on).toMatchObject({ followXuid: 'A', open: true })
    expect(reducePlayerPanel(on, { type: 'toggle', action: 'follow' })).toMatchObject({ followXuid: null })
  })

  it('« Suivre » de la fiche B reprend la caméra à A', () => {
    const s = reducePlayerPanel({ ...OPEN_A, xuid: 'B', followXuid: 'A' }, { type: 'toggle', action: 'follow' })
    expect(s).toMatchObject({ followXuid: 'B' })
  })

  it('« Suivre personne » éteint la caméra, fiche ouverte ou fermée', () => {
    expect(reducePlayerPanel({ ...OPEN_A, followXuid: 'A' }, { type: 'follow', xuid: null }))
      .toMatchObject({ followXuid: null, open: true })
    expect(reducePlayerPanel({ ...NO_PLAYER_PANEL, followXuid: 'A' }, { type: 'follow', xuid: null }))
      .toEqual(NO_PLAYER_PANEL)
  })

  it('suivre le joueur déjà suivi rend le même objet', () => {
    const s: PlayerPanelState = { ...NO_PLAYER_PANEL, followXuid: 'A' }
    expect(reducePlayerPanel(s, { type: 'follow', xuid: 'A' })).toBe(s)
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'follow', xuid: null })).toBe(NO_PLAYER_PANEL)
  })

  it('un geste manuel sur la carte arrête Suivre seulement, fiche ouverte ou fermée', () => {
    const s = reducePlayerPanel({ ...OPEN_A, followXuid: 'A', solo: true }, { type: 'stopFollow' })
    expect(s).toMatchObject({ followXuid: null, solo: true, open: true })
    expect(reducePlayerPanel({ ...NO_PLAYER_PANEL, followXuid: 'A' }, { type: 'stopFollow' })).toEqual(NO_PLAYER_PANEL)
    expect(reducePlayerPanel(OPEN_A, { type: 'stopFollow' })).toBe(OPEN_A)
  })
})

describe('reducePlayerPanel — les exclusions de D-7', () => {
  it('Seul coupe Masquer', () => {
    const s = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'toggle', action: 'solo' })
    expect(s).toMatchObject({ solo: true, hide: false })
  })

  it('Suivre coupe Masquer et laisse Seul', () => {
    const s = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'toggle', action: 'follow' })
    expect(s).toMatchObject({ followXuid: 'A', hide: false, solo: false })
    const keep = reducePlayerPanel({ ...OPEN_A, solo: true }, { type: 'toggle', action: 'follow' })
    expect(keep).toMatchObject({ followXuid: 'A', solo: true })
  })

  it('suivre, par la demande directe, le joueur masqué lève son masquage', () => {
    const s = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'follow', xuid: 'A' })
    expect(s).toMatchObject({ followXuid: 'A', hide: false })
    const other = reducePlayerPanel({ ...OPEN_A, hide: true }, { type: 'follow', xuid: 'B' })
    expect(other).toMatchObject({ followXuid: 'B', hide: true })
  })

  it('Masquer coupe Seul ET Suivre du joueur masqué', () => {
    const s = reducePlayerPanel({ ...OPEN_A, solo: true, followXuid: 'A' }, { type: 'toggle', action: 'hide' })
    expect(s).toMatchObject({ hide: true, solo: false, followXuid: null })
  })

  it('Masquer ne coupe pas la caméra posée sur un AUTRE joueur', () => {
    const s = reducePlayerPanel({ ...OPEN_A, followXuid: 'B' }, { type: 'toggle', action: 'hide' })
    expect(s).toMatchObject({ hide: true, followXuid: 'B' })
  })

  it('rebasculer une action allumée l’éteint, sans toucher aux autres', () => {
    const s = reducePlayerPanel({ ...OPEN_A, solo: true, followXuid: 'A' }, { type: 'toggle', action: 'solo' })
    expect(s).toMatchObject({ solo: false, followXuid: 'A' })
  })

  it('Seul et Masquer sans panneau ouvert sont sans effet', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'toggle', action: 'solo' })).toBe(NO_PLAYER_PANEL)
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'toggle', action: 'hide' })).toBe(NO_PLAYER_PANEL)
  })
})

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
const BOT = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], { bot: true })

describe('panelFocusOf — le joueur dont le panneau est ouvert', () => {
  it('rend le joueur et ses actions, ou null panneau fermé / joueur parti du roster', () => {
    expect(panelFocusOf(OPEN_A, [A, BOT])).toEqual({ player: A, follow: false, solo: false, hide: false })
    expect(panelFocusOf(NO_PLAYER_PANEL, [A])).toBeNull()
    expect(panelFocusOf({ ...OPEN_A, xuid: 'Z' }, [A])).toBeNull()
  })

  it('un bot est un joueur comme les autres : sa clé de roster le retrouve', () => {
    const s: PlayerPanelState = { ...OPEN_A, xuid: 'bot:Marc', followXuid: 'bot:Marc' }
    expect(panelFocusOf(s, [A, BOT])?.player).toBe(BOT)
  })

  it('Suivre n’est vrai que pour le joueur de la fiche ouverte', () => {
    expect(panelFocusOf({ ...OPEN_A, followXuid: 'A' }, [A, BOT])?.follow).toBe(true)
    expect(panelFocusOf({ ...OPEN_A, followXuid: 'bot:Marc' }, [A, BOT])?.follow).toBe(false)
  })

  it('fiche fermée, la caméra garde son joueur dans l’état et la fiche ne rend rien', () => {
    const closed = reducePlayerPanel({ ...OPEN_A, followXuid: 'A' }, { type: 'close' })
    expect(closed.followXuid).toBe('A')
    expect(panelFocusOf(closed, [A, BOT])).toBeNull()
    expect(followedPlayerOf(closed, [A, BOT])).toBe(A)
  })
})

describe('followedPlayerOf — le joueur que la caméra suit, fiche ouverte ou non', () => {
  it('rend le joueur de followXuid, par sa clé de roster, bot compris', () => {
    expect(followedPlayerOf({ ...NO_PLAYER_PANEL, followXuid: 'A' }, [A, BOT])).toBe(A)
    expect(followedPlayerOf({ ...NO_PLAYER_PANEL, followXuid: 'bot:Marc' }, [A, BOT])).toBe(BOT)
  })

  it('ne dépend pas de la fiche : un autre joueur peut être suivi pendant que la fiche en montre un autre', () => {
    expect(followedPlayerOf({ ...OPEN_A, followXuid: 'bot:Marc' }, [A, BOT])).toBe(BOT)
    expect(panelFocusOf({ ...OPEN_A, followXuid: 'bot:Marc' }, [A, BOT])?.player).toBe(A)
  })

  it('personne n’est suivi, ou le joueur suivi n’est plus au roster : null, jamais un joueur deviné', () => {
    expect(followedPlayerOf(NO_PLAYER_PANEL, [A])).toBeNull()
    expect(followedPlayerOf({ ...NO_PLAYER_PANEL, followXuid: 'Z' }, [A])).toBeNull()
  })
})

describe('trackedPlayerOf — le joueur suivi est le joueur regardé', () => {
  const B_BOARD = player('B', [], { board: scoreboardRow('B', 'Beta', null) })
  const BOT_BOARD = player('bot:Marc', [], { bot: true, board: scoreboardRow('bid(7.0)', 'Marc', null) })

  it('rend le joueur dont la page prend le point de vue, par sa clé de roster', () => {
    expect(trackedPlayerOf('A', [A, B_BOARD])).toBe(A)
  })

  it('un bot se retrouve par la clé de sa ligne de feuille, celle que le menu de point de vue rend', () => {
    expect(trackedPlayerOf('bid(7.0)', [A, BOT_BOARD])).toBe(BOT_BOARD)
    expect(trackedPlayerOf('bot:Marc', [A, BOT_BOARD])).toBe(BOT_BOARD)
  })

  it('personne n’est regardé, ou le joueur regardé n’est pas au roster : null, jamais un joueur deviné', () => {
    expect(trackedPlayerOf(null, [A])).toBeNull()
    expect(trackedPlayerOf('', [A])).toBeNull()
    expect(trackedPlayerOf('Z', [A])).toBeNull()
  })

  it('changer de joueur regardé change le joueur suivi', () => {
    expect(trackedPlayerOf('A', [A, B_BOARD])).toBe(A)
    expect(trackedPlayerOf('B', [A, B_BOARD])).toBe(B_BOARD)
  })
})

describe('shownSlotPredicate — ce que la carte montre', () => {
  it('ni Seul ni Masquer : aucun filtre à brancher', () => {
    expect(shownSlotPredicate({ player: A, follow: true, solo: false, hide: false })).toBeNull()
    expect(shownSlotPredicate(null)).toBeNull()
  })

  it('Masquer retire les vies du joueur, et seulement elles', () => {
    const shown = shownSlotPredicate({ player: A, follow: false, solo: false, hide: true })!
    expect(shown(1, 10)).toBe(false) // sa première vie
    expect(shown(3, 70)).toBe(false) // sa seconde vie, autre slot
    expect(shown(2, 10)).toBe(true) // un autre joueur
  })

  it('le slot réattribué : la vie d’un AUTRE joueur sur le même slot reste affichée', () => {
    // Le slot 1 appartient à A jusqu'à l'image 50 ; à l'image 80 il est tenu par un autre.
    const shown = shownSlotPredicate({ player: A, follow: false, solo: false, hide: true })!
    expect(shown(1, 80)).toBe(true)
  })

  it('Seul n’affiche que les vies du joueur', () => {
    const shown = shownSlotPredicate({ player: A, follow: false, solo: true, hide: false })!
    expect(shown(1, 10)).toBe(true)
    expect(shown(3, 70)).toBe(true)
    expect(shown(2, 10)).toBe(false)
  })

  it('un bot se filtre par ses vies : il n’a pas de xuid', () => {
    const shown = shownSlotPredicate({ player: BOT, follow: false, solo: false, hide: true })!
    expect(shown(7, 20)).toBe(false)
    expect(shown(1, 20)).toBe(true)
  })
})

const solo = (p: ReplayPlayer) => ({ player: p, follow: false, solo: true, hide: false })
const hide = (p: ReplayPlayer) => ({ player: p, follow: false, solo: false, hide: true })

describe('killFxShown — un effet de mort appartient au tueur ET à la victime', () => {
  // A (slot 1 puis 3), un autre joueur au slot 2 ; le tueur et la victime se lisent dans leur vie.
  const aTue = killFxEntry({ slot: 1, slotFrame: 10, victimSlot: 2, victimFrame: 10 })
  const aMeurt = killFxEntry({ slot: 2, slotFrame: 10, victimSlot: 1, victimFrame: 10 })
  const sansA = killFxEntry({ slot: 2, slotFrame: 10, victimSlot: 5, victimFrame: 10 })

  it('ni Seul ni Masquer : aucun filtre', () => {
    expect(killFxShown(null)).toBeNull()
    expect(killFxShown({ player: A, follow: true, solo: false, hide: false })).toBeNull()
  })

  it('Seul garde l’effet où le joueur TUE et celui où il MEURT, et rien d’autre', () => {
    const shown = killFxShown(solo(A))!
    expect(shown(aTue)).toBe(true)
    expect(shown(aMeurt)).toBe(true)
    expect(shown(sansA)).toBe(false)
  })

  it('Masquer retire l’effet où le joueur tue et celui où il meurt, et garde les autres', () => {
    const shown = killFxShown(hide(A))!
    expect(shown(aTue)).toBe(false)
    expect(shown(aMeurt)).toBe(false)
    expect(shown(sansA)).toBe(true)
  })

  it('le slot réattribué : l’effet d’un AUTRE joueur sur le slot 1 après la vie de A reste à lui', () => {
    const autre = killFxEntry({ slot: 1, slotFrame: 80, victimSlot: 5, victimFrame: 80 })
    expect(killFxShown(solo(A))!(autre)).toBe(false)
    expect(killFxShown(hide(A))!(autre)).toBe(true)
  })

  it('tueur inconnu : la victime décide (Seul comme Masquer)', () => {
    const tueurInconnu = (victimSlot: number) =>
      killFxEntry({ slot: null, killerXuid: '', victimSlot, victimFrame: 10 })
    expect(killFxShown(solo(A))!(tueurInconnu(1))).toBe(true)
    expect(killFxShown(solo(A))!(tueurInconnu(2))).toBe(false)
    expect(killFxShown(hide(A))!(tueurInconnu(1))).toBe(false)
    expect(killFxShown(hide(A))!(tueurInconnu(2))).toBe(true)
  })

  it('victime inconnue : le tueur décide', () => {
    const victimeInconnue = (slot: number) =>
      killFxEntry({ slot, slotFrame: 10, victimSlot: null, victimXuid: null })
    expect(killFxShown(solo(A))!(victimeInconnue(1))).toBe(true)
    expect(killFxShown(solo(A))!(victimeInconnue(2))).toBe(false)
    expect(killFxShown(hide(A))!(victimeInconnue(1))).toBe(false)
    expect(killFxShown(hide(A))!(victimeInconnue(2))).toBe(true)
  })

  it('aucune partie lue : l’effet reste affiché, rien n’est deviné', () => {
    const rien = killFxEntry({ slot: null, killerXuid: '', victimSlot: null, victimXuid: null })
    expect(killFxShown(solo(A))!(rien)).toBe(true)
    expect(killFxShown(hide(A))!(rien)).toBe(true)
  })

  it('un bot se reconnaît par la clé du fil de sa ligne de feuille quand sa vie n’est pas retrouvée', () => {
    const marc = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], {
      bot: true,
      board: { xuid: 'bid(1.0)' } as never,
    })
    const parCle = killFxEntry({ slot: null, killerXuid: 'bid(1.0)', victimSlot: 2, victimFrame: 10 })
    expect(killFxShown(solo(marc))!(parCle)).toBe(true)
    expect(killFxShown(hide(marc))!(parCle)).toBe(false)
  })
})

describe('ownershipShown — la règle unique des éléments à parties', () => {
  it('un élément sans partie n’appartient à personne : il reste, Seul comme Masquer', () => {
    expect(ownershipShown(solo(A))!([])).toBe(true)
    expect(ownershipShown(hide(A))!([])).toBe(true)
  })

  it('une partie par xuid du document (porteur, auteur)', () => {
    expect(ownershipShown(solo(A))!([{ xuid: 'A' }])).toBe(true)
    expect(ownershipShown(solo(A))!([{ xuid: 'B' }])).toBe(false)
    expect(ownershipShown(hide(A))!([{ xuid: 'A' }])).toBe(false)
    expect(ownershipShown(hide(A))!([{ xuid: 'B' }])).toBe(true)
  })

  it('partagé (occupants d’un véhicule) : Masquer ne retire que si TOUS les occupants sont le joueur', () => {
    const seul = [{ xuid: 'A' }]
    const duo = [{ xuid: 'A' }, { xuid: 'B' }]
    const hideA = ownershipShown(hide(A))!
    expect(hideA(seul, true)).toBe(false)
    expect(hideA(duo, true)).toBe(true)
    expect(hideA(duo)).toBe(false)
    // Seul garde un véhicule où le joueur est l'un des occupants.
    expect(ownershipShown(solo(A))!(duo, true)).toBe(true)
    expect(ownershipShown(solo(A))!([{ xuid: 'B' }], true)).toBe(false)
  })

  it('une partie prouvée par sa vie OU par son xuid', () => {
    const shown = ownershipShown(solo(A))!
    expect(shown([{ life: { slot: 3, frame: 70 } }])).toBe(true)
    expect(shown([{ life: { slot: 9, frame: 70 }, xuid: 'A' }])).toBe(true)
    expect(shown([{ life: { slot: 9, frame: 70 }, xuid: 'B' }])).toBe(false)
  })
})
