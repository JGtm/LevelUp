/**
 * killFeedLogic.test.ts — le recalage des deux horloges, le fil PERMANENT, et le
 * rattachement des médailles.
 *
 * Les chiffres viennent du match 000d5950 : T0 = 18 465 ms (real_start_time −
 * start_time_utc), premier kill à 35 306 ms sur l'horloge BRUTE, donc 16 841 ms sur
 * l'horloge gameplay que sert la Match View. Rattachement médaille→kill mesuré sur ce
 * même match : 42/44 à ≤ 500 ms, 2 médailles seules.
 */
import { describe, expect, it } from 'vitest'

import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchHighlightEvent } from '@/lib/api/types'

import {
  buildFeedEntries,
  collectMedalEvents,
  feedAt,
  toReplayKills,
  type MedalEvent,
} from './killFeedLogic'

const T0 = 18_465

function kill(tMs: number, xuid = 'x1'): KillEvent {
  return {
    tMs,
    xuid,
    ally: true,
    teamID: 0,
    weaponKey: 'hinf_br75',
    weaponLabel: 'BR75',
    weaponImageUrl: '/static/weapons/br75.png',
    weaponTinted: true,
    assistState: '',
    assistGamertag: '',
    assistTeamID: null,
    killerDamagePct: null,
    assistDamagePct: null,
    victimXuid: '',
    victimGamertag: '',
    victimTeamID: null,
  }
}

function medal(tMs: number, xuid = 'x1', name = 'No Scope'): MedalEvent {
  return {
    tMs,
    xuid,
    gamertag: 'GT',
    teamID: 0,
    name,
    label: 'Sans lunette',
    description: 'Desc',
    imageUrl: '/static/medals/1.png',
  }
}

describe('toReplayKills', () => {
  it('remet le countdown que la Match View avait retranché', () => {
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

  it('trie chronologiquement — feedAt en dépend', () => {
    const out = toReplayKills([kill(30_000), kill(10_000), kill(20_000)], T0)
    expect(out.map((k) => k.tMs)).toEqual([10_000, 20_000, 30_000])
  })
})

describe('collectMedalEvents', () => {
  const ev = (over: Partial<MatchHighlightEvent>): MatchHighlightEvent => ({
    event_type: 'medal',
    event_time_ms: 1_000,
    actor_xuid: 'x1',
    target_xuid: null,
    weapon_id: null,
    medal_name: 'No Scope',
    ...over,
  })

  it("lit les events medal avec leur identité résolue", () => {
    const [m] = collectMedalEvents([
      ev({
        actor_gamertag: 'JGtm',
        actor_team_id: 1,
        medal_label: 'Sans lunette',
        medal_description: 'Desc',
        medal_image_url: '/static/medals/1.png',
      }),
    ])
    expect(m).toMatchObject({
      tMs: 1_000,
      xuid: 'x1',
      gamertag: 'JGtm',
      teamID: 1,
      name: 'No Scope',
      label: 'Sans lunette',
      imageUrl: '/static/medals/1.png',
    })
  })

  it('écarte ce qui ne se montre pas : mauvais type, sans acteur, sans instant, sans nom', () => {
    expect(
      collectMedalEvents([
        ev({ event_type: 'kill' }),
        ev({ actor_xuid: null }),
        ev({ event_time_ms: null }),
        ev({ medal_name: null }),
      ]),
    ).toEqual([])
  })
})

describe('buildFeedEntries — rattachement des médailles', () => {
  it('rattache une médaille au kill du MÊME acteur à ≤ 500 ms', () => {
    const entries = buildFeedEntries([kill(10_000)], [medal(10_240)], 0)
    expect(entries).toHaveLength(1)
    expect(entries[0].kill?.medals.map((m) => m.name)).toEqual(['No Scope'])
  })

  it("une médaille d'un AUTRE acteur au même instant fait sa propre ligne", () => {
    const entries = buildFeedEntries([kill(10_000, 'x1')], [medal(10_000, 'x2')], 0)
    expect(entries).toHaveLength(2)
    expect(entries.find((e) => e.medal)?.medal?.xuid).toBe('x2')
  })

  it('au-delà de la tolérance, la médaille reste seule — jamais forcée', () => {
    const entries = buildFeedEntries([kill(10_000)], [medal(11_000)], 0)
    expect(entries).toHaveLength(2)
    expect(entries[0].kill?.medals).toEqual([])
    expect(entries[1].medal?.name).toBe('No Scope')
  })

  it('choisit le kill le PLUS PROCHE quand deux se disputent la médaille', () => {
    const entries = buildFeedEntries([kill(10_000), kill(10_400)], [medal(10_390)], 0)
    const porteur = entries.find((e) => (e.kill?.medals.length ?? 0) > 0)
    expect(porteur?.kill?.tMs).toBe(10_400)
  })

  it('la médaille seule est recalée sur la même horloge que les kills', () => {
    const entries = buildFeedEntries([], [medal(16_841)], T0)
    expect(entries[0].replayMs).toBe(35_306)
  })
})

describe('feedAt — le fil est PERMANENT', () => {
  const entries = buildFeedEntries(
    [kill(10_000), kill(12_000), kill(14_000), kill(60_000)],
    [],
    0,
  )

  it("n'affiche RIEN de ce qui n'est pas encore arrivé", () => {
    const vus = feedAt(entries, 11_000)
    expect(vus.map((e) => e.replayMs)).toEqual([10_000])
  })

  it('GARDE TOUT ce qui est survenu, le plus récent en tête — aucune fenêtre', () => {
    // Le point du verdict user 2026-08-13 : à 59 s de rejeu, les kills de 10/12/14 s
    // sont TOUJOURS là. L'ancien feed les faisait disparaître après 8 s.
    const vus = feedAt(entries, 59_000)
    expect(vus.map((e) => e.replayMs)).toEqual([14_000, 12_000, 10_000])
  })

  it('fil complet en fin de match', () => {
    expect(feedAt(entries, 120_000)).toHaveLength(4)
  })

  it('vide avant le premier événement', () => {
    expect(feedAt(entries, 0)).toEqual([])
  })
})
