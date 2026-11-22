/**
 * Tests — la logique pure du panneau de joueur.
 *
 * Ce qu'ils protègent :
 *  1. LE JOUEUR CHOISI : un clic sur une fiche ouvre le panneau, un autre joueur REMPLACE le contenu,
 *     recliquer la fiche ouverte referme. Ouvrir, changer ou fermer la fiche ne touche NI aux joueurs
 *     masqués NI à la caméra.
 *  2. SUIVRE EST LA CAMÉRA, INDÉPENDANTE DE LA FICHE ; le joueur suivi est le joueur REGARDÉ.
 *  3. L'ENSEMBLE DES JOUEURS MASQUÉS (D-7) : masquer et afficher joueur par joueur, « Tout afficher »,
 *     « Seul » (masque tous les autres ; refait, réaffiche tout) ; « Seul » se lit sur l'ensemble.
 *  4. LES RÈGLES DE LA CAMÉRA AVEC LE MASQUAGE (D-15) : suivre un masqué le réaffiche, masquer le suivi
 *     arrête le suivi, Seul arrête le suivi d'un joueur qu'il masque.
 *  5. Ce que la carte montre des joueurs masqués (pions, effets de mort, éléments à parties) est dans
 *     `playerPanelLogic.hidden.test.ts`.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import {
  followedPlayerOf,
  hiddenPlayersOf,
  isSolo,
  NO_PLAYER_PANEL,
  panelFocusOf,
  reducePlayerPanel,
  trackedPlayerOf,
  type PlayerPanelState,
} from './playerPanelLogic'
import { scoreboardRow } from '../test/scoreboardRow'

const ROSTER = ['A', 'B', 'C'] as const
const keys = (...k: string[]) => new Set(k)
const OPEN_A: PlayerPanelState = { open: true, xuid: 'A', followXuid: null, hidden: keys() }
const withHidden = (state: PlayerPanelState, ...k: string[]): PlayerPanelState => ({ ...state, hidden: keys(...k) })
const solo = (state: PlayerPanelState, xuid: string) => reducePlayerPanel(state, { type: 'solo', xuid, roster: ROSTER })
const toggleHidden = (state: PlayerPanelState, xuid: string) => reducePlayerPanel(state, { type: 'toggleHidden', xuid })

describe('reducePlayerPanel — le joueur choisi', () => {
  it('un clic sur une fiche ouvre le panneau sur ce joueur', () => {
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'card', xuid: 'A' })).toEqual(OPEN_A)
  })

  it('une AUTRE fiche remplace le joueur ; la caméra et les joueurs masqués ne bougent pas', () => {
    const busy: PlayerPanelState = { ...withHidden(OPEN_A, 'C'), followXuid: 'A' }
    const next = reducePlayerPanel(busy, { type: 'card', xuid: 'B' })
    expect(next).toMatchObject({ open: true, xuid: 'B', followXuid: 'A' })
    expect(next.hidden).toBe(busy.hidden)
  })

  it('recliquer la fiche OUVERTE referme, sans éteindre la caméra ni réafficher personne', () => {
    const busy: PlayerPanelState = { ...withHidden(OPEN_A, 'B', 'C'), followXuid: 'A' }
    const next = reducePlayerPanel(busy, { type: 'card', xuid: 'A' })
    expect(next).toMatchObject({ open: false, xuid: null, followXuid: 'A' })
    expect(next.hidden).toBe(busy.hidden)
  })

  it('fermer la fiche (« Retour au fil », Échap) laisse la caméra ET les joueurs masqués', () => {
    const busy: PlayerPanelState = { ...withHidden(OPEN_A, 'B'), followXuid: 'A' }
    const next = reducePlayerPanel(busy, { type: 'close' })
    expect(next).toMatchObject({ open: false, xuid: null, followXuid: 'A' })
    expect(next.hidden).toBe(busy.hidden)
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
    const s = reducePlayerPanel(withHidden(OPEN_A, 'C'), { type: 'follow', xuid: 'B' })
    expect(s).toMatchObject({ open: true, xuid: 'A', followXuid: 'B' })
    expect([...s.hidden]).toEqual(['C'])
  })

  it('ouvrir une fiche sur le joueur suivi, puis une autre, laisse la caméra sur lui', () => {
    let s = reducePlayerPanel(NO_PLAYER_PANEL, { type: 'follow', xuid: 'A' })
    s = reducePlayerPanel(s, { type: 'card', xuid: 'A' })
    s = reducePlayerPanel(s, { type: 'card', xuid: 'B' })
    expect(s).toMatchObject({ open: true, xuid: 'B', followXuid: 'A' })
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
    const s = reducePlayerPanel({ ...withHidden(OPEN_A, 'B'), followXuid: 'A' }, { type: 'stopFollow' })
    expect(s).toMatchObject({ followXuid: null, open: true })
    expect([...s.hidden]).toEqual(['B'])
    expect(reducePlayerPanel({ ...NO_PLAYER_PANEL, followXuid: 'A' }, { type: 'stopFollow' })).toEqual(NO_PLAYER_PANEL)
    expect(reducePlayerPanel(OPEN_A, { type: 'stopFollow' })).toBe(OPEN_A)
  })
})

describe('reducePlayerPanel — l’ensemble des joueurs masqués (D-7)', () => {
  it('masquer deux joueurs, l’un après l’autre : les deux sont masqués', () => {
    const s = toggleHidden(toggleHidden(NO_PLAYER_PANEL, 'B'), 'C')
    expect([...s.hidden].sort()).toEqual(['B', 'C'])
  })

  it('l’œil d’un joueur masqué le réaffiche, les autres restent masqués', () => {
    const s = toggleHidden(withHidden(NO_PLAYER_PANEL, 'B', 'C'), 'B')
    expect([...s.hidden]).toEqual(['C'])
  })

  it('masquer ne demande aucune fiche ouverte, et ne l’ouvre pas', () => {
    const s = toggleHidden(NO_PLAYER_PANEL, 'B')
    expect(s).toMatchObject({ open: false, xuid: null })
    expect([...s.hidden]).toEqual(['B'])
  })

  it('l’ensemble ne dépend pas de la fiche : ouvrir, changer, fermer ne le touche pas', () => {
    let s = toggleHidden(toggleHidden(NO_PLAYER_PANEL, 'B'), 'C')
    const ensemble = s.hidden
    s = reducePlayerPanel(s, { type: 'card', xuid: 'A' })
    expect(s.hidden).toBe(ensemble)
    s = reducePlayerPanel(s, { type: 'card', xuid: 'B' }) // la fiche d'un joueur masqué s'ouvre aussi
    expect(s).toMatchObject({ open: true, xuid: 'B' })
    expect(s.hidden).toBe(ensemble)
    s = reducePlayerPanel(s, { type: 'close' })
    expect(s.hidden).toBe(ensemble)
  })

  it('« Tout afficher » vide l’ensemble ; sans joueur masqué il rend le même objet', () => {
    const busy = withHidden({ ...OPEN_A, followXuid: 'A' }, 'B', 'C')
    const s = reducePlayerPanel(busy, { type: 'showAll' })
    expect(s.hidden.size).toBe(0)
    expect(s).toMatchObject({ open: true, xuid: 'A', followXuid: 'A' })
    expect(reducePlayerPanel(NO_PLAYER_PANEL, { type: 'showAll' })).toBe(NO_PLAYER_PANEL)
  })

  it('le réducteur ne modifie jamais l’ensemble reçu', () => {
    const avant = keys('B')
    const state: PlayerPanelState = { ...NO_PLAYER_PANEL, hidden: avant }
    toggleHidden(state, 'C')
    solo(state, 'A')
    reducePlayerPanel(state, { type: 'showAll' })
    expect([...avant]).toEqual(['B'])
  })
})

describe('reducePlayerPanel — Seul (D-8)', () => {
  it('Seul masque tous les autres joueurs et laisse le joueur affiché', () => {
    const s = solo(NO_PLAYER_PANEL, 'A')
    expect([...s.hidden].sort()).toEqual(['B', 'C'])
    expect(isSolo(s.hidden, 'A', ROSTER)).toBe(true)
  })

  it('Seul ne dépend pas de la fiche ouverte', () => {
    const ouverte = solo({ ...OPEN_A, xuid: 'C' }, 'A')
    expect([...ouverte.hidden].sort()).toEqual(['B', 'C'])
    expect(ouverte).toMatchObject({ open: true, xuid: 'C' })
  })

  it('Seul puis l’œil d’un joueur : ce joueur est réaffiché, les autres restent masqués', () => {
    const s = toggleHidden(solo(NO_PLAYER_PANEL, 'A'), 'B')
    expect([...s.hidden]).toEqual(['C'])
    expect(isSolo(s.hidden, 'A', ROSTER)).toBe(false)
  })

  it('Seul deux fois réaffiche tout', () => {
    const s = solo(solo(NO_PLAYER_PANEL, 'A'), 'A')
    expect(s.hidden.size).toBe(0)
  })

  it('Seul sur un autre joueur déplace le Seul', () => {
    const s = solo(solo(NO_PLAYER_PANEL, 'A'), 'B')
    expect([...s.hidden].sort()).toEqual(['A', 'C'])
    expect(isSolo(s.hidden, 'B', ROSTER)).toBe(true)
  })

  it('Seul sur un joueur lui-même masqué le réaffiche', () => {
    const s = solo(withHidden(NO_PLAYER_PANEL, 'A'), 'A')
    expect([...s.hidden].sort()).toEqual(['B', 'C'])
  })

  it('Seul ne masque jamais le joueur seul, même masqué à la main avant', () => {
    expect(solo(withHidden(NO_PLAYER_PANEL, 'A', 'B'), 'A').hidden.has('A')).toBe(false)
  })
})

describe('isSolo — Seul se lit sur l’ensemble', () => {
  it('vrai quand tous les autres sont masqués et lui non', () => {
    expect(isSolo(keys('B', 'C'), 'A', ROSTER)).toBe(true)
  })

  it('faux quand un autre est affiché, ou quand lui-même est masqué, ou sans joueur masqué', () => {
    expect(isSolo(keys('B'), 'A', ROSTER)).toBe(false)
    expect(isSolo(keys('A', 'B', 'C'), 'A', ROSTER)).toBe(false)
    expect(isSolo(keys(), 'A', ROSTER)).toBe(false)
  })

  it('un roster d’un seul joueur n’a pas de Seul', () => {
    expect(isSolo(keys(), 'A', ['A'])).toBe(false)
  })
})

describe('reducePlayerPanel — Suivre et le masquage (D-15)', () => {
  it('suivre un joueur masqué le réaffiche ; les autres masqués le restent', () => {
    const s = reducePlayerPanel(withHidden(NO_PLAYER_PANEL, 'B', 'C'), { type: 'follow', xuid: 'B' })
    expect(s.followXuid).toBe('B')
    expect([...s.hidden]).toEqual(['C'])
  })

  it('suivre un joueur affiché ne réaffiche personne', () => {
    const base = withHidden(NO_PLAYER_PANEL, 'C')
    const s = reducePlayerPanel(base, { type: 'follow', xuid: 'B' })
    expect(s.hidden).toBe(base.hidden)
  })

  it('masquer le joueur suivi arrête le suivi', () => {
    const s = toggleHidden({ ...NO_PLAYER_PANEL, followXuid: 'B' }, 'B')
    expect(s.followXuid).toBeNull()
    expect([...s.hidden]).toEqual(['B'])
  })

  it('masquer un AUTRE joueur que le suivi ne coupe pas la caméra', () => {
    expect(toggleHidden({ ...NO_PLAYER_PANEL, followXuid: 'B' }, 'C').followXuid).toBe('B')
  })

  it('réafficher un joueur ne remet pas le suivi', () => {
    const s = toggleHidden(toggleHidden({ ...NO_PLAYER_PANEL, followXuid: 'B' }, 'B'), 'B')
    expect(s.followXuid).toBeNull()
  })

  it('Seul qui masque le joueur suivi arrête le suivi', () => {
    expect(solo({ ...NO_PLAYER_PANEL, followXuid: 'B' }, 'A').followXuid).toBeNull()
  })

  it('Seul sur le joueur suivi garde le suivi', () => {
    expect(solo({ ...NO_PLAYER_PANEL, followXuid: 'A' }, 'A').followXuid).toBe('A')
  })

  it('« Seul » répété (qui réaffiche tout) ne touche pas à la caméra', () => {
    const s = solo(solo({ ...NO_PLAYER_PANEL, followXuid: 'A' }, 'A'), 'A')
    expect(s.followXuid).toBe('A')
    expect(s.hidden.size).toBe(0)
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
const B = player('B', [{ slot: 2, start: 0, end: 100 }])
const BOT = player('bot:Marc', [{ slot: 7, start: 0, end: 100 }], { bot: true })

describe('hiddenPlayersOf — les joueurs masqués, retrouvés au roster', () => {
  it('rend les joueurs masqués dans l’ordre du roster, ou null quand personne n’est masqué', () => {
    expect(hiddenPlayersOf(keys('B', 'A'), [A, B, BOT])).toEqual([A, B])
    expect(hiddenPlayersOf(keys(), [A, B])).toBeNull()
  })

  it('une clé qui n’est plus au roster (autre document) est ignorée, jamais un joueur deviné', () => {
    expect(hiddenPlayersOf(keys('Z'), [A, B])).toBeNull()
    expect(hiddenPlayersOf(keys('Z', 'B'), [A, B])).toEqual([B])
  })

  it('un bot se retrouve par sa clé de roster', () => {
    expect(hiddenPlayersOf(keys('bot:Marc'), [A, BOT])).toEqual([BOT])
  })
})

describe('panelFocusOf — le joueur dont le panneau est ouvert', () => {
  it('rend le joueur, son Seul et son masquage ; null panneau fermé ou joueur parti du roster', () => {
    expect(panelFocusOf(OPEN_A, [A, B, BOT])).toEqual({ player: A, solo: false, hidden: false })
    expect(panelFocusOf(NO_PLAYER_PANEL, [A])).toBeNull()
    expect(panelFocusOf({ ...OPEN_A, xuid: 'Z' }, [A])).toBeNull()
  })

  it('un bot est un joueur comme les autres : sa clé de roster le retrouve', () => {
    const s: PlayerPanelState = { ...OPEN_A, xuid: 'bot:Marc' }
    expect(panelFocusOf(s, [A, BOT])?.player).toBe(BOT)
  })

  it('Seul est vrai pour le joueur de la fiche quand tous les autres sont masqués', () => {
    expect(panelFocusOf(withHidden(OPEN_A, 'B', 'bot:Marc'), [A, B, BOT])?.solo).toBe(true)
    expect(panelFocusOf(withHidden(OPEN_A, 'B'), [A, B, BOT])?.solo).toBe(false)
    expect(panelFocusOf(withHidden({ ...OPEN_A, xuid: 'B' }, 'A', 'bot:Marc'), [A, B, BOT])?.solo).toBe(true)
  })

  it('« masqué » dit si le joueur de la fiche est dans l’ensemble', () => {
    expect(panelFocusOf(withHidden(OPEN_A, 'A'), [A, B])?.hidden).toBe(true)
    expect(panelFocusOf(withHidden(OPEN_A, 'B'), [A, B])?.hidden).toBe(false)
  })

  it('fiche fermée, la caméra garde son joueur dans l’état et la fiche ne rend rien', () => {
    const closed = reducePlayerPanel({ ...OPEN_A, followXuid: 'A' }, { type: 'close' })
    expect(closed.followXuid).toBe('A')
    expect(panelFocusOf(closed, [A, B])).toBeNull()
    expect(followedPlayerOf(closed, [A, B])).toBe(A)
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
