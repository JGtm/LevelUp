import { describe, expect, it } from 'vitest'

import { buildFlagReturnDrops, flagReturnAt, harmonic, type FlagReturnRule } from './flagReturnZone'

import type { ReplayFlagCarryReady } from '../../../lib/replay/replayNormalize'

/**
 * flagReturnZone.test.ts — LES RÈGLES DE LA ZONE DE RETOUR, sans canvas.
 *
 * Chaque test fige UNE règle et la fait tomber si elle disparaît : la loi harmonique que le jeu
 * applique, la fusion des lâchers contigus, l'atterrissage de la jauge sur le retour OBSERVÉ, et
 * le silence quand le titre ne déclare pas de zone.
 */

/** La règle mesurée du CTF d'Halo Infinite, telle que le manifeste la publie. */
const RULE: FlagReturnRule = { radiusM: 1.3, resetSeconds: 30, soloSeconds: 3.1 }

/** Un drapeau lâché en (0,0) de l'image `t0` à `t1`, suivi de ce que l'on veut. */
function carry(
  spans: { state: string; t0: number; t1: number; x?: number; y?: number }[],
): ReplayFlagCarryReady {
  return {
    team: 0,
    spans: spans.map((s) => ({
      state: s.state,
      t0: s.t0,
      t1: s.t1,
      xuid: null,
      x: s.x ?? 0,
      y: s.y ?? 0,
    })),
  } as ReplayFlagCarryReady
}

/** Personne sur la carte : aucun défenseur n'a de position. */
const PERSONNE = {
  posOf: () => null,
  defendersOf: () => [] as string[],
}

/** Un défenseur immobile SUR le drapeau — il est donc dans la zone à toutes les images. */
const SUR_LE_DRAPEAU = {
  posOf: () => ({ x: 0, y: 0 }),
  defendersOf: () => ['1'],
}

describe('harmonic', () => {
  it('rend la série harmonique, et zéro pour une zone vide', () => {
    expect(harmonic(0)).toBe(0)
    expect(harmonic(1)).toBe(1)
    expect(harmonic(2)).toBe(1.5)
    expect(harmonic(3)).toBeCloseTo(1 + 1 / 2 + 1 / 3, 6)
  })
})

describe('buildFlagReturnDrops', () => {
  it('se tait quand le titre ne déclare aucune zone', () => {
    const drops = buildFlagReturnDrops([carry([{ state: 'dropped', t0: 0, t1: 10 }])], {
      rule: null,
      frameIntervalMs: 100,
      ...PERSONNE,
    })
    expect(drops).toEqual([])
  })

  it('fusionne les intervalles `dropped` contigus en UN seul lâcher', () => {
    const drops = buildFlagReturnDrops(
      [
        carry([
          { state: 'dropped', t0: 0, t1: 4, x: 0 },
          { state: 'dropped', t0: 5, t1: 9, x: 2 },
          { state: 'carried', t0: 10, t1: 12 },
        ]),
      ],
      { rule: RULE, frameIntervalMs: 100, ...PERSONNE },
    )
    expect(drops).toHaveLength(1)
    expect(drops[0]).toMatchObject({ t0: 0, t1: 9, returnFrame: null })
    // La position SUIT les poses successives : le drapeau a roulé de 0 à 2.
    expect(drops[0].x[0]).toBe(0)
    expect(drops[0].x[9]).toBe(2)
  })

  it('la jauge atteint exactement 1 à l’image du retour OBSERVÉ', () => {
    const drops = buildFlagReturnDrops(
      [
        carry([
          { state: 'dropped', t0: 0, t1: 299 },
          { state: 'home', t0: 300, t1: 400 },
        ]),
      ],
      { rule: RULE, frameIntervalMs: 100, ...PERSONNE },
    )
    expect(drops).toHaveLength(1)
    expect(drops[0].returnFrame).toBe(300)
    expect(drops[0].progress[299]).toBeCloseTo(1, 5)
    expect(drops[0].progress[0]).toBeLessThan(0.02)
  })

  it('un défenseur dans la zone vide la jauge PLUS VITE qu’une zone déserte', () => {
    const spans = [
      { state: 'dropped', t0: 0, t1: 99 },
      { state: 'carried', t0: 100, t1: 120 },
    ]
    const vide = buildFlagReturnDrops([carry(spans)], {
      rule: RULE,
      frameIntervalMs: 100,
      ...PERSONNE,
    })
    const occupee = buildFlagReturnDrops([carry(spans)], {
      rule: RULE,
      frameIntervalMs: 100,
      ...SUR_LE_DRAPEAU,
    })
    // Sans retour observé, aucune remise à l'échelle : les deux jauges se comparent telles quelles.
    expect(occupee[0].occupants[10]).toBe(1)
    expect(vide[0].occupants[10]).toBe(0)
    expect(occupee[0].progress[10]).toBeGreaterThan(vide[0].progress[10] * 5)
  })

  it('un défenseur HORS du rayon ne compte pas', () => {
    const drops = buildFlagReturnDrops([carry([{ state: 'dropped', t0: 0, t1: 20 }])], {
      rule: RULE,
      frameIntervalMs: 100,
      posOf: () => ({ x: RULE.radiusM + 0.5, y: 0 }),
      defendersOf: () => ['1'],
    })
    expect(drops[0].occupants[5]).toBe(0)
  })

  // LA REPRISE REMET LA JAUGE À ZÉRO, et c'est le SEUL « reset » que le joueur observe en jeu :
  // un intervalle qui n'est pas `dropped` ferme le lâcher, et le suivant en rouvre un neuf.
  // Rien ne l'écrit explicitement dans le code — ce test est ce qui empêche que la fusion des
  // lâchers contigus l'efface un jour par accident.
  it('une REPRISE coupe le lâcher, et le suivant repart d’une jauge NEUVE', () => {
    const drops = buildFlagReturnDrops(
      [
        carry([
          { state: 'dropped', t0: 0, t1: 99 },
          { state: 'carried', t0: 100, t1: 150 },
          { state: 'dropped', t0: 151, t1: 200 },
        ]),
      ],
      { rule: RULE, frameIntervalMs: 100, ...PERSONNE },
    )
    expect(drops).toHaveLength(2)
    expect(drops[1].t0).toBe(151)
    // 1 s après le second lâcher, la jauge en est au même point qu'à 1 s du premier.
    expect(drops[1].progress[10]).toBeCloseTo(drops[0].progress[10], 6)
  })

  it('un DRAPEAU NEUTRE n’a pas de défenseur : la minuterie seule', () => {
    const neutre = { ...carry([{ state: 'dropped', t0: 0, t1: 99 }]), team: -1 }
    const drops = buildFlagReturnDrops([neutre], {
      rule: RULE,
      frameIntervalMs: 100,
      posOf: () => ({ x: 0, y: 0 }),
      // L'appelant ne rend personne pour une équipe négative — c'est SA règle, figée ici.
      defendersOf: (team: number) => (team < 0 ? [] : ['1']),
    })
    expect(drops[0].occupants[50]).toBe(0)
    // 10 s à la minuterie de 30 s : un tiers de la jauge, sans personne pour l'accélérer.
    expect(drops[0].progress[99]).toBeCloseTo(10 / RULE.resetSeconds, 5)
  })
})

describe('flagReturnAt', () => {
  it('ne rend que les lâchers ACTIFS à l’image demandée', () => {
    const drops = buildFlagReturnDrops(
      [
        carry([
          { state: 'dropped', t0: 10, t1: 20 },
          { state: 'carried', t0: 21, t1: 30 },
        ]),
      ],
      { rule: RULE, frameIntervalMs: 100, ...PERSONNE },
    )
    expect(flagReturnAt(drops, 5)).toEqual([])
    expect(flagReturnAt(drops, 25)).toEqual([])
    const now = flagReturnAt(drops, 15)
    expect(now).toHaveLength(1)
    expect(now[0].radiusM).toBe(RULE.radiusM)
  })
})
