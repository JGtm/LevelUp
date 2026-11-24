/**
 * Tests — la frontière du document rattache à leur joueur les éliminations et les actions de score d'un BOT : un bot
 * n'a pas d'identifiant, le film le désigne par sa place (`roster[].filmIndex`) ; la frontière la résout par la table
 * du roster, une seule, et ne devine jamais.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayDocument } from '@/lib/api/types'

import { normalizeReplayDocument } from './replayNormalize'

const roster = [
  { xuid: '2533274800000001', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 },
  { xuid: '', filmIndex: 2, seat: 1, name: 'Marc', team: 1, bot: true },
]

function normalize(over: Partial<ReplayDocument>) {
  return normalizeReplayDocument({ roster, tracks: [], ...over } as unknown as ReplayDocument)
}

describe('éliminations d’un bot, désignées par place', () => {
  const human = '2533274800000001'

  it('un tueur bot reçoit sa clé de roster `bot:<nom>`, un humain garde son xuid', () => {
    const doc = normalize({
      eliminations: [
        { t: 10, xuid: human, victim: undefined },
        { t: 20, place: 2, victim: human },
      ],
    })
    expect(doc.eliminations.map((e) => [e.t, e.xuid, e.victim])).toEqual([
      [10, human, undefined],
      [20, 'bot:Marc', human],
    ])
  })

  it('une victime bot (`victimPlace`) reçoit sa clé de roster dans `victim`', () => {
    const doc = normalize({ eliminations: [{ t: 10, xuid: human, victimPlace: 2 }] })
    expect(doc.eliminations).toMatchObject([{ t: 10, xuid: human, victim: 'bot:Marc' }])
  })

  it('une place que le roster ne désigne pas écarte la ligne, jamais un joueur deviné', () => {
    const doc = normalize({ eliminations: [{ t: 10, place: 9 }, { t: 11, xuid: human, victimPlace: 9 }] })
    expect(doc.eliminations).toMatchObject([{ t: 11, xuid: human }])
    expect(doc.eliminations[0].victim).toBeUndefined()
  })

  it('une ligne sans xuid ni place n’a pas d’auteur : écartée', () => {
    expect(normalize({ eliminations: [{ t: 10 }] }).eliminations).toEqual([])
  })

  it('les marques de tir à la tête et de frag parfait voyagent', () => {
    const doc = normalize({ eliminations: [{ t: 10, place: 2, headshot: true, perfect: true }] })
    expect(doc.eliminations).toMatchObject([{ xuid: 'bot:Marc', headshot: true, perfect: true }])
  })
})

describe('actions de score d’un bot, désignées par place', () => {
  const human = '2533274800000001'

  it('un destinataire bot reçoit sa clé de roster, une cible bot aussi', () => {
    const doc = normalize({
      scoreActions: [
        { t: 5, place: 2, name: 'killPersonalScore', points: 100, target: human },
        { t: 6, xuid: human, name: 'killPersonalScore', points: 100, targetPlace: 2 },
      ],
    })
    expect(doc.scoreActions.map((a) => [a.xuid, a.target])).toEqual([
      ['bot:Marc', human],
      [human, 'bot:Marc'],
    ])
  })

  it('une place inconnue écarte l’action ; une cible inconnue la garde sans cible', () => {
    const doc = normalize({
      scoreActions: [
        { t: 5, place: 9, name: 'killPersonalScore', points: 100 },
        { t: 6, xuid: human, name: 'killPersonalScore', points: 100, targetPlace: 9 },
      ],
    })
    expect(doc.scoreActions).toHaveLength(1)
    expect(doc.scoreActions[0]).toMatchObject({ xuid: human, points: 100 })
    expect(doc.scoreActions[0].target).toBeUndefined()
  })
})
