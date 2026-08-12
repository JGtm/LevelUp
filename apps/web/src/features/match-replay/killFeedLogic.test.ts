/**
 * killFeedLogic.test.ts — le recalage des deux horloges, et la fenêtre du feed.
 *
 * Les chiffres viennent du match 000d5950 : T0 = 18 465 ms (real_start_time −
 * start_time_utc), premier kill à 35 306 ms sur l'horloge BRUTE, donc 16 841 ms sur
 * l'horloge gameplay que sert la Match View.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'

import { attachVictims, freshnessOf, killsAt, toReplayKills } from './killFeedLogic'

const T0 = 18_465

function kill(tMs: number, xuid = 'x1'): KillEvent {
  return {
    tMs,
    xuid,
    ally: true,
    teamID: 0,
    weaponLabel: 'BR75',
    weaponImageUrl: '/static/weapons/br75.png',
    weaponTinted: true,
  }
}

describe('toReplayKills', () => {
  it("remet le countdown que la Match View avait retranché", () => {
    // 16 841 ms sur l'horloge gameplay = 35 306 ms sur celle du film, la seule que le
    // rejeu connaisse.
    const [k] = toReplayKills([kill(16_841)], T0)
    expect(k.replayMs).toBe(35_306)
  })

  it("n'ajoute rien quand le countdown est inconnu", () => {
    // T0 nul veut dire que la correction n'a pas eu lieu côté events non plus : les deux
    // horloges coïncident déjà, et ajouter quoi que ce soit les décalerait.
    expect(toReplayKills([kill(16_841)], 0)[0].replayMs).toBe(16_841)
    expect(toReplayKills([kill(16_841)], Number.NaN)[0].replayMs).toBe(16_841)
  })

  it('trie chronologiquement — killsAt en dépend', () => {
    const out = toReplayKills([kill(30_000), kill(10_000), kill(20_000)], T0)
    expect(out.map((k) => k.tMs)).toEqual([10_000, 20_000, 30_000])
  })
})

describe('killsAt', () => {
  const kills = toReplayKills([kill(10_000), kill(12_000), kill(14_000), kill(60_000)], 0)

  it("n'affiche RIEN de ce qui n'est pas encore arrivé", () => {
    // LE POINT DE F2 : à 11 s de rejeu, le kill de 12 s ne doit pas être à l'écran.
    const vus = killsAt(kills, 11_000, 8_000, 6)
    expect(vus.map((k) => k.replayMs)).toEqual([10_000])
  })

  it('rend les kills de la fenêtre, du plus récent au plus ancien', () => {
    expect(killsAt(kills, 15_000, 8_000, 6).map((k) => k.replayMs)).toEqual([
      14_000, 12_000, 10_000,
    ])
  })

  it('laisse sortir un kill quand il a dépassé la fenêtre', () => {
    // À 19 s, le kill de 10 s a 9 s d'âge : il est hors fenêtre de 8 s.
    expect(killsAt(kills, 19_000, 8_000, 6).map((k) => k.replayMs)).toEqual([14_000, 12_000])
    // Et à 30 s, plus rien de cette rafale.
    expect(killsAt(kills, 30_000, 8_000, 6)).toEqual([])
  })

  it('borne le nombre de lignes sans changer leur ordre', () => {
    expect(killsAt(kills, 15_000, 8_000, 2).map((k) => k.replayMs)).toEqual([14_000, 12_000])
  })

  it('rend une liste vide avant le premier kill', () => {
    expect(killsAt(kills, 0, 8_000, 6)).toEqual([])
  })
})

describe('freshnessOf', () => {
  const [k] = toReplayKills([kill(10_000)], 0)

  it('est franc à l’instant du kill et atténué à la sortie de fenêtre', () => {
    expect(freshnessOf(k, 10_000, 8_000)).toBeCloseTo(1, 5)
    expect(freshnessOf(k, 18_000, 8_000)).toBeCloseTo(0.4, 5)
  })

  it('ne descend jamais sous le plancher, même très en retard', () => {
    expect(freshnessOf(k, 100_000, 8_000)).toBeCloseTo(0.4, 5)
  })
})

describe('attachVictims — la victime jointe par (tueur, instant)', () => {
  const base = {
    ally: true,
    teamID: 0,
    weaponLabel: '',
    weaponImageUrl: '',
    weaponTinted: false,
    assistState: '' as const,
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
  }

  it('joint la victime sur la clé exacte, et laisse vides les kills sans paire', () => {
    const out = attachVictims(
      [
        { ...base, tMs: 1_000, xuid: 'A' },
        { ...base, tMs: 2_000, xuid: 'A' },
      ],
      [{ killer_xuid: 'A', victim_gamertag: 'V1', time_ms: 1_000 }],
    )
    expect(out[0].victimGamertag).toBe('V1')
    expect(out[1].victimGamertag).toBe('')
  })

  it('DEUX victimes distinctes sur la même clé : personne n’est nommé — jamais au hasard', () => {
    const out = attachVictims(
      [{ ...base, tMs: 1_000, xuid: 'A' }],
      [
        { killer_xuid: 'A', victim_gamertag: 'V1', time_ms: 1_000 },
        { killer_xuid: 'A', victim_gamertag: 'V2', time_ms: 1_000 },
      ],
    )
    expect(out[0].victimGamertag).toBe('')
  })

  it('deux paires IDENTIQUES (double kill unanime) : la victime est nommée', () => {
    const out = attachVictims(
      [{ ...base, tMs: 1_000, xuid: 'A' }],
      [
        { killer_xuid: 'A', victim_gamertag: 'V1', time_ms: 1_000 },
        { killer_xuid: 'A', victim_gamertag: 'V1', time_ms: 1_000 },
      ],
    )
    expect(out[0].victimGamertag).toBe('V1')
  })

  it('paires absentes ou incomplètes : tout reste vide, rien ne casse', () => {
    const kills = [{ ...base, tMs: 1_000, xuid: 'A' }]
    expect(attachVictims(kills, null)[0].victimGamertag).toBe('')
    expect(attachVictims(kills, [{ killer_xuid: 'A', victim_gamertag: '', time_ms: 1_000 }])[0].victimGamertag).toBe('')
  })
})
