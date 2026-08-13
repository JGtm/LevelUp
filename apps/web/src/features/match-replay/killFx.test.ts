/**
 * killFx.test.ts — les règles de l'effet de mort, celles qui ne se voient pas à l'œil :
 * l'orientation n'existe QUE sur couple complet (règle POC 89/93), la position d'une
 * victime se relit dans la fenêtre DEATH après sa vie, et la famille vient de la table
 * `killEffects` par weapon_key — jamais d'une arme voisine.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { ReplayDocument } from '@/lib/api/types'

import { buildKillFx, MELEE_LINK_MAX_M, posOfPlayerAt } from './killFx'
import { testReplayDoc } from './test/testDoc'

/** Un kill minimal ; tMs est sur l'horloge gameplay (le recalage ajoute t0Ms). */
function kill(over: Partial<KillEvent> = {}): KillEvent {
  return {
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

describe('posOfPlayerAt', () => {
  const doc = docWithCouple()
  const lives = doc.tracks.filter((t) => t.xuid === 'V')

  it('rend la position exacte pendant la vie', () => {
    expect(posOfPlayerAt(lives, 10, 15)).toEqual({ x: 5, y: 4.5 })
  })

  it('rend la DERNIÈRE position dans la fenêtre DEATH après la fin de vie', () => {
    expect(posOfPlayerAt(lives, 30, 15)).toEqual({ x: 5, y: 4 })
  })

  it("ne rend RIEN au-delà de la fenêtre — aucune position inventée", () => {
    expect(posOfPlayerAt(lives, 40, 15)).toBeNull()
  })
})

describe('buildKillFx', () => {
  it('oriente tueur -> victime quand le couple est complet, et porte la distance monde', () => {
    // Kill à 2 000 ms gameplay + t0 0 -> frame 20 : tueur en (2, 0), victime en (5, 4).
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000 })], 0)
    expect(fx).toHaveLength(1)
    expect(fx[0].frame).toBe(20)
    expect(fx[0].vx).toBe(5)
    expect(fx[0].vy).toBe(4)
    expect(fx[0].dist).toBeCloseTo(5, 5)
    expect(fx[0].fam).toBe('ballistic')
    expect(fx[0].slot).toBe(1)
  })

  it('recale sur l horloge du film : t0Ms s ajoute (même règle que le fil)', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 1_000 })], 1_000)
    expect(fx[0]?.frame).toBe(20)
  })

  it('règle 89/93 : victime introuvable = marqueur NON orienté, jamais un axe inventé', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, victimXuid: 'inconnu' })], 0)
    expect(fx).toHaveLength(1)
    expect(fx[0].vx).toBeNull()
    expect(fx[0].dist).toBeNull()
  })

  it('tueur introuvable : l origine est la VICTIME, sans orientation', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, xuid: 'fantome' })], 0)
    expect(fx).toHaveLength(1)
    expect(fx[0].x).toBe(5)
    expect(fx[0].vx).toBeNull()
    expect(fx[0].slot).toBeNull()
  })

  it('ni tueur ni victime localisable : rien — on ne dessine pas', () => {
    const fx = buildKillFx(
      docWithCouple(),
      [kill({ tMs: 2_000, xuid: 'fantome', victimXuid: 'inconnu' })],
      0,
    )
    expect(fx).toHaveLength(0)
  })

  it('sans weapon_key ou hors table : famille neutre, jamais celle d une voisine', () => {
    const fx = buildKillFx(
      docWithCouple(),
      [kill({ tMs: 2_000, weaponKey: '' }), kill({ tMs: 2_000, weaponKey: 'hinf_inconnu' })],
      0,
    )
    expect(fx.map((e) => e.fam)).toEqual(['plain', 'plain'])
  })

  it('le seuil mêlée est en mètres MONDE : 5 m < 8 m, l arc de liaison est permis', () => {
    const fx = buildKillFx(docWithCouple(), [kill({ tMs: 2_000, weaponKey: 'hinf_energy_sword' })], 0)
    expect(fx[0].fam).toBe('melee')
    expect(fx[0].dist).not.toBeNull()
    expect((fx[0].dist ?? Infinity) < MELEE_LINK_MAX_M).toBe(true)
  })
})
