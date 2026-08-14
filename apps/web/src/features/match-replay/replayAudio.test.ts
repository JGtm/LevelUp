/**
 * replayAudio.test.ts — LE MOTEUR, sur un AudioContext de laboratoire.
 *
 * Ce qui se vérifie ici ne s'entend pas à l'oreille et casse en silence : qu'un son parte
 * bien (déclenchement), qu'il soit COUPÉ à la seconde par l'enveloppe (et non laissé courir
 * jusqu'au bout du fichier), qu'un fichier absent reste un SILENCE mémorisé (jamais une
 * re-tentative par kill, jamais un son de remplacement), et qu'un échange nourri ne fasse
 * pas un mur de voix.
 *
 * jsdom n'a pas de Web Audio : le contexte est un double instrumenté qui note ce qu'on lui
 * demande. Il ne juge pas le rendu sonore — c'est le gate d'écoute qui le fait.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import {
  ReplayAudioPlayer,
  SOUND_CUT_S,
  SOUND_FADE_S,
  SOUND_MAX_VOICES,
  soundEnvelope,
  VOLUME_RAMP_S,
} from './replayAudio'
import {
  type FakeContext,
  flushAudio,
  installFakeAudio,
  okAudioResponse,
} from './test/fakeAudio'

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  const fake = installFakeAudio()
  ctx = fake.ctx
  fetchMock = fake.fetchMock
  vi.spyOn(console, 'warn').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

const flush = flushAudio
const okResponse = okAudioResponse

describe('ReplayAudioPlayer — déclenchement', () => {
  it('joue un son chargé : une source démarrée à l instant courant', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload(['/static/sounds/halo_infinite/hinf_br75.wav'])
    await flush()
    p.play('/static/sounds/halo_infinite/hinf_br75.wav')
    expect(ctx.sources).toHaveLength(1)
    expect(ctx.sources[0].started).toBe(ctx.currentTime)
  })

  it('ne charge qu UNE fois la même URL, même précédée de plusieurs demandes', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload(['/a.wav', '/a.wav'])
    p.preload(['/a.wav'])
    await flush()
    p.preload(['/a.wav'])
    await flush()
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('plusieurs sons SIMULTANÉS partent ensemble (un échange, pas une file)', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload(['/a.wav', '/b.wav', '/c.wav'])
    await flush()
    p.play('/a.wav')
    p.play('/b.wav')
    p.play('/c.wav')
    expect(ctx.sources).toHaveLength(3)
    expect(ctx.sources.every((s) => s.started === ctx.currentTime)).toBe(true)
  })
})

describe('ReplayAudioPlayer — coupure', () => {
  it('coupe un son long à ~1 s : fondu programmé puis arrêt, jamais la fin du fichier', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload(['/long.wav']) // 3 s de source
    await flush()
    p.play('/long.wav')
    const t0 = ctx.currentTime
    // Le gain de voix est le dernier créé (le maître naît avec le lecteur).
    const voice = ctx.gains[ctx.gains.length - 1]
    expect(voice.gain.calls).toEqual([
      ['set', 1, t0],
      ['set', 1, t0 + SOUND_CUT_S - SOUND_FADE_S],
      ['ramp', 0, t0 + SOUND_CUT_S],
    ])
    expect(ctx.sources[0].stopped).toBe(t0 + SOUND_CUT_S)
  })

  it('un son plus court que la coupe joue entier, fondu borné à sa moitié', async () => {
    const p = new ReplayAudioPlayer(1)
    fetchMock.mockResolvedValueOnce(okResponse(0.4))
    p.preload(['/court.wav'])
    await flush()
    p.play('/court.wav')
    expect(ctx.sources[0].stopped).toBeCloseTo(ctx.currentTime + 0.4, 6)
    expect(soundEnvelope(0.4).fadeStartS).toBeCloseTo(0.2, 6)
  })
})

describe('ReplayAudioPlayer — silence propre', () => {
  it('fichier absent (404) : AUCUNE source, et pas de re-tentative à chaque passage', async () => {
    const p = new ReplayAudioPlayer(1)
    fetchMock.mockResolvedValue({ ok: false, status: 404 })
    p.preload(['/absent.wav'])
    await flush()
    p.play('/absent.wav')
    p.play('/absent.wav')
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).toHaveBeenCalledTimes(1)
  })

  it('URL jamais demandée : silence à cet instant, chargement pour les suivants', async () => {
    const p = new ReplayAudioPlayer(1)
    p.play('/tard.wav')
    expect(ctx.sources).toHaveLength(0)
    expect(fetchMock).toHaveBeenCalledTimes(1)
    await flush()
    p.play('/tard.wav')
    expect(ctx.sources).toHaveLength(1)
  })

  it('fichier indécodable : silence mémorisé, jamais une exception qui casse la lecture', async () => {
    const p = new ReplayAudioPlayer(1)
    vi.spyOn(ctx, 'decodeAudioData').mockRejectedValue(new Error('bouillie'))
    p.preload(['/casse.wav'])
    await flush()
    expect(() => p.play('/casse.wav')).not.toThrow()
    expect(ctx.sources).toHaveLength(0)
  })
})

describe('ReplayAudioPlayer — voix et volume', () => {
  it('plafonne les voix simultanées, et en libère une quand un son se termine', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload(['/a.wav'])
    await flush()
    for (let i = 0; i < SOUND_MAX_VOICES + 3; i++) p.play('/a.wav')
    expect(ctx.sources).toHaveLength(SOUND_MAX_VOICES)
    ctx.sources[0].end()
    p.play('/a.wav')
    expect(ctx.sources).toHaveLength(SOUND_MAX_VOICES + 1)
  })

  it('le volume initial est posé à la construction (pas de premier son à plein régime)', () => {
    new ReplayAudioPlayer(0.4)
    expect(ctx.gains[0].gain.value).toBe(0.4)
  })

  it('setVolume rampe (pas de clic) et borne dans 0..1', () => {
    const p = new ReplayAudioPlayer(1)
    const master = ctx.gains[0]
    p.setVolume(0.5)
    p.setVolume(4)
    p.setVolume(-1)
    const t0 = ctx.currentTime
    expect(master.gain.calls.filter((c) => c[0] === 'ramp')).toEqual([
      ['ramp', 0.5, t0 + VOLUME_RAMP_S],
      ['ramp', 1, t0 + VOLUME_RAMP_S],
      ['ramp', 0, t0 + VOLUME_RAMP_S],
    ])
  })

  it('dispose ferme le contexte (rien ne tourne après le démontage)', () => {
    const p = new ReplayAudioPlayer(1)
    p.dispose()
    expect(ctx.closed).toBe(1)
  })
})
