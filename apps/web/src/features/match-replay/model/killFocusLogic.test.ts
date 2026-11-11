/**
 * Tests — killFocusLogic : la mise en valeur d'une élimination.
 *
 * Ce qu'ils protègent : (1) l'état — remplacé par un autre clic, rejoué à CHAQUE clic (jeton),
 * effacé par la Lecture ou un saut manuel, jamais mémorisé ; (2) l'instant — la ligne reste
 * visible au fil une fois la lecture posée dessus ; (3) la résolution — jamais d'équerre à une
 * position inventée : seul ce qui est positionné est mis en valeur, tueur et victime séparément.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayFeedEntry, ReplayKill } from './killFeedLogic'
import {
  NO_KILL_FOCUS,
  focusTargetOf,
  frameAtOrAfter,
  hiddenNameSlots,
  reduceKillFocus,
  resolveKillFocus,
  type KillFocusResolvers,
  type KillFocusTarget,
} from './killFocusLogic'

const TARGET: KillFocusTarget = {
  entryKey: 'k1',
  frame: 120,
  killerXuid: 'killer',
  victimXuid: 'victim',
  weapon: { imageUrl: '/w.png', tinted: true },
}

describe('reduceKillFocus — l’état', () => {
  it('part sans mise en valeur', () => {
    expect(NO_KILL_FOCUS.current).toBeNull()
  })

  it('un clic pose la mise en valeur, avec un jeton', () => {
    const s = reduceKillFocus(NO_KILL_FOCUS, { type: 'select', target: TARGET })
    expect(s.current).toMatchObject({ ...TARGET, token: 1 })
  })

  it('un autre clic REMPLACE la mise en valeur', () => {
    const a = reduceKillFocus(NO_KILL_FOCUS, { type: 'select', target: TARGET })
    const b = reduceKillFocus(a, { type: 'select', target: { ...TARGET, entryKey: 'k2', frame: 300 } })
    expect(b.current?.entryKey).toBe('k2')
    expect(b.current?.frame).toBe(300)
  })

  it('le MÊME clic répété change le jeton : l’animation se rejoue à chaque clic', () => {
    const a = reduceKillFocus(NO_KILL_FOCUS, { type: 'select', target: TARGET })
    const b = reduceKillFocus(a, { type: 'select', target: TARGET })
    expect(b.current?.token).toBe((a.current?.token ?? 0) + 1)
  })

  it('l’effacement (Lecture, saut manuel) ne laisse rien', () => {
    const a = reduceKillFocus(NO_KILL_FOCUS, { type: 'select', target: TARGET })
    expect(reduceKillFocus(a, { type: 'clear' }).current).toBeNull()
  })

  it('le jeton ne retombe jamais : un clic après un effacement a un jeton neuf', () => {
    const a = reduceKillFocus(NO_KILL_FOCUS, { type: 'select', target: TARGET })
    const vide = reduceKillFocus(a, { type: 'clear' })
    const c = reduceKillFocus(vide, { type: 'select', target: TARGET })
    expect(c.current?.token).toBeGreaterThan(a.current?.token ?? 0)
  })

  it('effacer un état déjà vide rend le MÊME objet (aucun réveil inutile)', () => {
    expect(reduceKillFocus(NO_KILL_FOCUS, { type: 'clear' })).toBe(NO_KILL_FOCUS)
  })
})

describe('frameAtOrAfter — l’image où poser la lecture', () => {
  const doc = { frameIntervalMs: 50 }

  it('tombe sur l’image exacte quand l’instant en est une', () => {
    expect(frameAtOrAfter(1_000, doc)).toBe(20)
  })

  it('prend l’image SUIVANTE quand l’instant tombe entre deux : la ligne reste visible au fil', () => {
    const f = frameAtOrAfter(1_010, doc)
    expect(f).toBe(21)
    expect(f * 50).toBeGreaterThanOrEqual(1_010)
  })

  it('ne rend jamais une image dont l’instant précède celui du frag', () => {
    for (const ms of [0, 49.9, 50, 123_456.7, 7_777.7777]) {
      expect(frameAtOrAfter(ms, doc) * 50).toBeGreaterThanOrEqual(ms)
    }
  })
})

function killEntry(over: Partial<ReplayKill> = {}, key = 'k1'): ReplayFeedEntry {
  const kill = {
    tMs: 0,
    xuid: 'killer',
    teamID: 0,
    weaponKey: 'br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '/br75.png',
    weaponTinted: true,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: 'victim',
    victimGamertag: 'Victim',
    replayMs: 6_000,
    medals: [],
    ...over,
  } as ReplayKill
  return { key, replayMs: kill.replayMs, kill, medal: null, death: null }
}

describe('focusTargetOf — ce qu’une ligne demande', () => {
  const doc = { frameIntervalMs: 50 }

  it('rend la cible d’une ligne de frag : image, tueur, victime, arme', () => {
    expect(focusTargetOf(killEntry(), doc)).toEqual({
      entryKey: 'k1',
      frame: 120,
      killerXuid: 'killer',
      victimXuid: 'victim',
      weapon: { imageUrl: '/br75.png', tinted: true },
    })
  })

  it('une ligne qui n’est pas un frag n’a pas de cible', () => {
    const medal: ReplayFeedEntry = { key: 'm', replayMs: 0, kill: null, medal: null, death: null }
    expect(focusTargetOf(medal, doc)).toBeNull()
  })

  it('une victime non nommée reste vide : le tueur seul sera mis en valeur', () => {
    const t = focusTargetOf(killEntry({ victimXuid: '', victimGamertag: '' }), doc)
    expect(t?.victimXuid).toBe('')
  })

  it('une arme sans icône reste sans icône : jamais celle d’une autre', () => {
    const t = focusTargetOf(killEntry({ weaponImageUrl: '', weaponTinted: false }), doc)
    expect(t?.weapon).toEqual({ imageUrl: '', tinted: false })
  })
})

describe('resolveKillFocus — jamais de position inventée', () => {
  const request = { ...TARGET, token: 4 }
  const base: KillFocusResolvers = {
    posOf: (xuid) => (xuid === 'killer' ? { x: 1, y: 2 } : xuid === 'victim' ? { x: 3, y: 4 } : null),
    seated: () => true,
    colorOf: (xuid) => (xuid === 'killer' ? '#k' : '#v'),
    nameOf: (xuid) => (xuid === 'killer' ? 'Killer' : 'Victim'),
    slotOf: (xuid) => (xuid === 'killer' ? 7 : 8),
  }

  it('résout le tueur ET la victime quand les deux sont positionnés', () => {
    const r = resolveKillFocus(request, base)
    expect(r?.token).toBe(4)
    expect(r?.frame).toBe(120)
    expect(r?.killer).toEqual({ xuid: 'killer', pos: { x: 1, y: 2 }, color: '#k', name: 'Killer', slot: 7 })
    expect(r?.victim).toEqual({ xuid: 'victim', pos: { x: 3, y: 4 }, color: '#v', name: 'Victim', slot: 8 })
    expect(r?.weapon).toEqual(TARGET.weapon)
  })

  it('une victime sans position : seul le tueur est mis en valeur', () => {
    const r = resolveKillFocus(request, { ...base, posOf: (x) => (x === 'killer' ? { x: 1, y: 2 } : null) })
    expect(r?.killer).not.toBeNull()
    expect(r?.victim).toBeNull()
  })

  it('un tueur sans position (bot, mort hors carte) : seule la victime est mise en valeur', () => {
    const r = resolveKillFocus(request, { ...base, posOf: (x) => (x === 'victim' ? { x: 3, y: 4 } : null) })
    expect(r?.killer).toBeNull()
    expect(r?.victim).not.toBeNull()
  })

  it('personne de positionné : rien à mettre en valeur', () => {
    expect(resolveKillFocus(request, { ...base, posOf: () => null })).toBeNull()
  })

  it('un joueur qui n’occupe plus de fiche n’a pas de pion : il n’est pas mis en valeur', () => {
    const r = resolveKillFocus(request, { ...base, seated: (x) => x !== 'killer' })
    expect(r?.killer).toBeNull()
    expect(r?.victim).not.toBeNull()
  })

  it('un joueur sans couleur (absent du roster) n’est pas dessiné, donc pas mis en valeur', () => {
    const r = resolveKillFocus(request, { ...base, colorOf: (x) => (x === 'killer' ? null : '#v') })
    expect(r?.killer).toBeNull()
  })

  it('une victime non nommée n’est jamais cherchée', () => {
    const vue: string[] = []
    const posOf: KillFocusResolvers['posOf'] = (x, f) => {
      vue.push(x)
      return base.posOf(x, f)
    }
    resolveKillFocus({ ...request, victimXuid: '' }, { ...base, posOf })
    expect(vue).toEqual(['killer'])
  })

  it('un nom inconnu reste null : l’étiquette ne s’invente pas un nom', () => {
    const r = resolveKillFocus(request, { ...base, nameOf: () => null })
    expect(r?.killer?.name).toBeNull()
  })

  it('les positions se lisent à l’image du frag', () => {
    const images: number[] = []
    const posOf: KillFocusResolvers['posOf'] = (x, f) => {
      images.push(f)
      return base.posOf(x, f)
    }
    resolveKillFocus(request, { ...base, posOf })
    expect(new Set(images)).toEqual(new Set([120]))
  })
})

describe('hiddenNameSlots — le nom sous le pion cède la place à l’étiquette', () => {
  it('rend le slot du tueur et celui de la victime', () => {
    const r = resolveKillFocus({ ...TARGET, token: 1 }, {
      posOf: () => ({ x: 0, y: 0 }),
      seated: () => true,
      colorOf: () => '#c',
      nameOf: () => 'n',
      slotOf: (x) => (x === 'killer' ? 7 : 8),
    })
    expect(hiddenNameSlots(r)).toEqual(new Set([7, 8]))
  })

  it('rien de mis en valeur : aucun nom masqué', () => {
    expect(hiddenNameSlots(null).size).toBe(0)
  })

  it('un acteur non résolu ne masque rien', () => {
    const r = resolveKillFocus({ ...TARGET, token: 1 }, {
      posOf: (x) => (x === 'killer' ? { x: 0, y: 0 } : null),
      seated: () => true,
      colorOf: () => '#c',
      nameOf: () => 'n',
      slotOf: () => 7,
    })
    expect(hiddenNameSlots(r)).toEqual(new Set([7]))
  })
})
