/**
 * Tests — bombCountdown (le compte à rebours de la bombe d'Assaut, schéma 29).
 *
 * CE QU'ILS PROTÈGENT :
 *  - la FENÊTRE : le compte à rebours ne vit que sur [t, t + fuseMs], dérivé de la position de
 *    lecture — avant, après, rien ;
 *  - la MÈCHE lue dans l'ARTEFACT (`fuseMs`), jamais codée ici : une variante future à mèche
 *    différente s'affiche juste ;
 *  - la GARDE D'HORLOGE : sans origine résolue, `t` ne veut rien dire — bandeau ET son se
 *    taisent (la même règle que la déflagration) ;
 *  - le SON de `bomb_armed` : un événement par armement, sur le stem de la NOUVELLE COLLINE
 *    emprunté PAR RÉFÉRENCE (`ZONE_SOUND_STEMS.newZone`) — pas une copie du nom de fichier.
 */
import { describe, expect, it } from 'vitest'

import { activeBombCountdown, bombArmingSoundEvents } from './bombCountdown'
import { testReplayDoc } from './test/testDoc'
import { ZONE_SOUND_STEMS } from './zoneSound'

/** Un armement : hold de 50 frames, armé à la frame 100, mèche 4 930 ms (49,3 frames à 100 ms). */
const ARMING = { t: 100, timeMs: 10_000, startT: 50, startMs: 5_000, fuseMs: 4_930 }

function docWith(bombArmings: unknown[], over: Record<string, unknown> = {}) {
  return testReplayDoc({ frameIntervalMs: 100, bombArmings: bombArmings as never, ...over })
}

describe('activeBombCountdown', () => {
  it("à l'instant armé : toute la mèche reste, la course est à zéro", () => {
    const s = activeBombCountdown(docWith([ARMING]), 100)
    expect(s).toEqual({ remainingMs: 4_930, progress: 0 })
  })

  it('à mi-mèche : le restant et la course suivent la position de lecture', () => {
    const s = activeBombCountdown(docWith([ARMING]), 125)
    expect(s?.remainingMs).toBe(4_930 - 2_500)
    expect(s?.progress).toBeCloseTo(2_500 / 4_930, 5)
  })

  it('AVANT l’armement et APRÈS la mèche : rien — le bandeau est dérivé, pas un état', () => {
    expect(activeBombCountdown(docWith([ARMING]), 99)).toBeNull()
    // 49,3 frames de mèche -> fenêtre close après la frame 100 + 49.
    expect(activeBombCountdown(docWith([ARMING]), 150)).toBeNull()
  })

  it("la MÈCHE vient de l'artefact : une mèche différente change la fenêtre et le restant", () => {
    const husky = { ...ARMING, fuseMs: 2_000 }
    expect(activeBombCountdown(docWith([husky]), 110)?.remainingMs).toBe(1_000)
    expect(activeBombCountdown(docWith([husky]), 121)).toBeNull()
  })

  it('HORLOGE NON FIABLE : rien du tout — un bandeau muet vaut mieux qu’un bandeau faux', () => {
    const doc = docWith([ARMING], { coverage: { originResolved: false } as never })
    expect(activeBombCountdown(doc, 100)).toBeNull()
  })

  it('sans armement : rien (mode non couvert, calque retenu — le silence est la donnée)', () => {
    expect(activeBombCountdown(docWith([]), 100)).toBeNull()
  })
})

describe('bombArmingSoundEvents', () => {
  it('un événement PAR armement, à la frame armée, sur le stem de la nouvelle colline', () => {
    const evs = bombArmingSoundEvents(docWith([ARMING, { ...ARMING, t: 300, timeMs: 30_000 }]))
    expect(evs.map((e) => e.ms)).toEqual([10_000, 30_000])
    expect(new Set(evs.map((e) => e.stem))).toEqual(new Set([ZONE_SOUND_STEMS.newZone]))
  })

  it('HORLOGE NON FIABLE : aucun son — même garde que le bandeau', () => {
    const doc = docWith([ARMING], { coverage: { originResolved: false } as never })
    expect(bombArmingSoundEvents(doc)).toEqual([])
  })
})
