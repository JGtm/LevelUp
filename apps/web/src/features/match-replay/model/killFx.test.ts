/**
 * killFx.test.ts — les règles de l'effet de mort, celles qui ne se voient pas à l'œil :
 * l'orientation n'existe QUE sur couple complet (règle POC 89/93), la position d'une
 * victime se relit dans la fenêtre DEATH après sa vie, et la famille vient de la table
 * `killEffects` par weapon_key — jamais d'une arme voisine.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import type { ReplayKill } from './killFeedLogic'

import { buildKillFx, MELEE_LINK_MAX_M } from './killFx'
import { testReplayDoc } from '../test/testDoc'

/**
 * Un kill DU FIL, donc DÉJÀ RECALÉ : `replayMs` est l'instant sur l'axe du rejeu, le seul
 * que ce module lise. `tMs` reste porté pour mémoire (l'instant servi par la Match View).
 */
function kill(over: Partial<ReplayKill> = {}): ReplayKill {
  return {
    replayMs: 2_000,
    medals: [],
    tMs: 1_000,
    xuid: 'K',
    ally: true,
    teamID: 0,
    weaponKey: 'hinf_br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: 'V',
    victimGamertag: 'Victime',
    victimTeamID: 1,
    ...over,
  }
}

/** Un document 10 Hz avec la vie du tueur (frames 0-100) et celle de la victime (0-20). */
function docWithCouple(over: Partial<ReplayDocument> = {}) {
  return testReplayDoc({
    frameIntervalMs: 100,
    killEffects: { hinf_br75: 'ballistic', hinf_energy_sword: 'melee' },
    tracks: [
      {
        slot: 1,
        team: -1,
        xuid: 'K',
        points: [
          { t: 0, x: 0, y: 0 },
          { t: 100, x: 10, y: 0 },
        ],
        startFrame: 0,
        endFrame: 100,
      },
      {
        // La victime meurt à la frame 20 : sa vie est CLOSE à l'instant du kill (frame 20),
        // sa position se relit dans la fenêtre DEATH.
        slot: 2,
        team: -1,
        xuid: 'V',
        points: [
          { t: 0, x: 5, y: 5 },
          { t: 20, x: 5, y: 4 },
        ],
        startFrame: 0,
        endFrame: 20,
      },
    ],
    ...over,
  })
}

/**
 * LE VÉHICULE DU TUEUR : immobile en (0,0) puis montant en Y, pendant que la trace de bipède
 * de K file en X. Aucune image ne peut confondre les deux — c'est ce qui rend le test
 * discriminant (`posOfPlayerAt` seule rendrait la ligne droite).
 */
function docTueurEmbarque(rideT1 = 50) {
  return docWithCouple({
    vehicles: [
      {
        slot: 700, gen: 1, t0: 0, t1: 100, t1max: 100, end: 'unknown', family: 'warthog',
        samples: [{ t: 0, x: 0, y: 0 }, { t: 100, x: 0, y: 10 }],
        rides: [{ t0: 10, t1: rideT1, slot: 1, seat: 0, src: 'event', xuid: 'K' }],
      },
    ] as never,
  })
}

describe('buildKillFx — le tueur EMBARQUÉ explose sur son véhicule (2026-09-05)', () => {
  it("à une image de l'épisode, l'effet part du VÉHICULE et non de la ligne droite du bipède", () => {
    // Kill à 2 000 ms -> frame 20. Bipède interpolé en (2, 0) ; véhicule en (0, 2).
    const fx = buildKillFx(docTueurEmbarque(), [kill({ tMs: 2_000 })])
    expect(fx).toHaveLength(1)
    expect({ x: fx[0].x, y: fx[0].y }).toEqual({ x: 0, y: 2 })
  })

  it('APRÈS LA DESCENTE, il repart du bipède, à l’unité près', () => {
    // L'épisode se ferme à la frame 15 ; le kill est à la frame 20, à pied.
    const fx = buildKillFx(docTueurEmbarque(15), [kill({ tMs: 2_000 })])
    expect({ x: fx[0].x, y: fx[0].y }).toEqual({ x: 2, y: 0 })
  })

  it('document ANTÉRIEUR AUX VÉHICULES (schéma <= 38) : rigoureusement inchangé', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000 })])
    expect({ x: fx[0].x, y: fx[0].y }).toEqual({ x: 2, y: 0 })
  })

  it("le SLOT du tueur reste celui de sa vie de bipède — aucun véhicule ne le déplace", () => {
    const fx = buildKillFx(docTueurEmbarque(), [kill({ tMs: 2_000 })])
    expect(fx[0].slot).toBe(1)
  })
})

describe('buildKillFx', () => {
  it('oriente tueur -> victime quand le couple est complet, et porte la distance monde', () => {
    // Kill à 2 000 ms gameplay + t0 0 -> frame 20 : tueur en (2, 0), victime en (5, 4).
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000 })])
    expect(fx).toHaveLength(1)
    expect(fx[0].frame).toBe(20)
    expect(fx[0].vx).toBe(5)
    expect(fx[0].vy).toBe(4)
    expect(fx[0].dist).toBeCloseTo(5, 5)
    expect(fx[0].fam).toBe('ballistic')
    expect(fx[0].slot).toBe(1)
  })

  it("LIT L'HORLOGE DU FIL telle qu'elle arrive, et ne la rejoue pas (2026-09-05, J2)", () => {
    // Le kill est servi 5 000 ms sur l'horloge gameplay ; le fil l'a déjà recalé à 2 000 ms
    // (fin de vie de sa victime, décalage d'origine mesuré à +3 678 ms sur le témoin
    // 000d5950). L'effet se pose donc frame 20 — victime relue, effet orienté. Avec le
    // recalage brut, la fenêtre de position (1,5 s) le ratait : 1/93 victimes relues avant,
    // 90/93 après. C'est `replayMs` qui décide, jamais `tMs`.
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 5_000, replayMs: 2_000 })])
    expect(fx[0]?.frame).toBe(20)
    expect(fx[0]?.vx).toBe(5)
    expect(fx[0]?.dist).toBeCloseTo(5, 5)
    // Et un instant de fil différent déplace l'effet d'autant : aucune correction cachée.
    const tardif = buildKillFx(docWithCouple(), [kill({ tMs: 5_000, replayMs: 5_000 })])
    expect(tardif[0]?.frame).toBe(50)
  })

  it('règle 89/93 : victime introuvable = marqueur NON orienté, jamais un axe inventé', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, victimXuid: 'inconnu' })])
    expect(fx).toHaveLength(1)
    expect(fx[0].vx).toBeNull()
    expect(fx[0].dist).toBeNull()
  })

  it('tueur introuvable : l origine est la VICTIME, sans orientation', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, xuid: 'fantome' })])
    expect(fx).toHaveLength(1)
    expect(fx[0].x).toBe(5)
    expect(fx[0].vx).toBeNull()
    expect(fx[0].slot).toBeNull()
  })

  it('le LIEU DE LA MORT survit à un tueur introuvable — ce que vx/vy, eux, ne font pas', () => {
    // Couple complet : les deux champs disent la même chose.
    const complet = buildKillFx(docWithCouple(), [kill({ tMs: 2_000 })])
    expect(complet[0].deathX).toBe(5)
    expect(complet[0].deathY).toBe(4)
    // Tueur introuvable : l'effet n'est plus orienté (vx null) MAIS on sait toujours où la
    // victime est morte — c'est cette position que la carte de chaleur compte.
    const sansTueur = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, xuid: 'fantome' })])
    expect(sansTueur[0].vx).toBeNull()
    expect(sansTueur[0].deathX).toBe(5)
    expect(sansTueur[0].deathY).toBe(4)
    // Victime introuvable : aucune position devinée, ni pour l'axe ni pour la chaleur.
    const sansVictime = buildKillFx(
      docWithCouple(),
      [kill({ tMs: 2_000, victimXuid: 'inconnu' })],
    )
    expect(sansVictime[0].deathX).toBeNull()
    expect(sansVictime[0].deathY).toBeNull()
  })

  it('ni tueur ni victime localisable : rien — on ne dessine pas', () => {
    const fx = buildKillFx(
      docWithCouple(),
      [kill({ tMs: 2_000, xuid: 'fantome', victimXuid: 'inconnu' })],
    )
    expect(fx).toHaveLength(0)
  })

  it('sans weapon_key ou hors table : famille neutre, jamais celle d une voisine', () => {
    const fx = buildKillFx(
      docWithCouple(),
      [kill({ tMs: 2_000, weaponKey: '' }), kill({ tMs: 2_000, weaponKey: 'hinf_inconnu' })],
    )
    expect(fx.map((e) => e.fam)).toEqual(['plain', 'plain'])
  })

  it('le seuil mêlée est en mètres MONDE : 5 m < 8 m, l arc de liaison est permis', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, weaponKey: 'hinf_energy_sword' })])
    expect(fx[0].fam).toBe('melee')
    expect(fx[0].dist).not.toBeNull()
    expect((fx[0].dist ?? Infinity) < MELEE_LINK_MAX_M).toBe(true)
  })
})
