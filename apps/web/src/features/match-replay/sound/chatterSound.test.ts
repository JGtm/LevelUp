import { describe, expect, it } from 'vitest'

import type { ChatterLine } from '../model/chatterLines'
import { chatterVoiceEvents } from './chatterSound'
import { withChatterVoices } from './exportSoundFamilies'

function line(frame: number, voice: string | null): ChatterLine {
  return { key: `k${frame}`, frame, xuid: 'A', sound: 's', family: 'enemy', text: 't', voice, count: 1 }
}

describe('chatterVoiceEvents', () => {
  it("fait un événement par réplique qui a une voix, à l'instant de sa parole, marqué voix de réplique", () => {
    const events = chatterVoiceEvents([line(10, 'repliques/fr/0000000a'), line(20, null), line(30, 'repliques/fr/0000000b')], { frameIntervalMs: 100 })
    expect(events).toEqual([
      { ms: 1000, stem: 'repliques/fr/0000000a', chatter: true },
      { ms: 3000, stem: 'repliques/fr/0000000b', chatter: true },
    ])
  })
})

describe('withChatterVoices', () => {
  it("range les stems de voix de réplique dans la famille « voix » de l'export, sans doublon", () => {
    const f = withChatterVoices(
      { voice: ['annonceur'], music: ['intro'] },
      [
        { ms: 0, stem: 'tir' },
        { ms: 1, stem: 'repliques/fr/a', chatter: true },
        { ms: 2, stem: 'repliques/fr/a', chatter: true },
      ],
    )
    expect(f).toEqual({ voice: ['annonceur', 'repliques/fr/a'], music: ['intro'] })
  })

  it("rend les familles telles quelles quand la piste n'a aucune voix de réplique", () => {
    const base = { voice: ['x'], music: ['y'] }
    expect(withChatterVoices(base, [{ ms: 0, stem: 'tir' }])).toBe(base)
  })
})
