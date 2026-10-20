/**
 * introAlign.test.ts — L'INTRO CALÉE SUR LE COUP D'ENVOI même quand son tampon arrive en retard
 * (item 7, A4.9, DA-9 du plan backlog du 2026-09-26).
 *
 * LE CAS QUI COMPTE : premier « Lecture » après un rechargement. Le lecteur audio naît dans ce
 * clic, l'extrait n'est pas encore décodé. La voie ordinaire le sautait. Désormais on attend le
 * décodage, puis la source part avec un DÉCALAGE égal au temps écoulé depuis le geste : la
 * résolution de la montée reste sur le coup d'envoi. Si l'attente dépasse le préambule
 * à la vitesse courante (1 s à 1×, 0,5 s à 2×), le coup d'envoi est passé : silence, jamais de
 * son décalé de son image.
 *
 * Le décodage est ASYNCHRONE et piloté à la main (réponse `fetch` tenue), l'horloge murale est
 * injectée : aucun délai réel, aucun aléa.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { type FakeContext, flushAudio, installFakeAudio, okAudioResponse } from '../test/fakeAudio'
import { introLateBoundS, playIntroAligned } from './introSound'
import { ReplayAudioPlayer } from './replayAudio'

const URL = '/static/sounds/halo_infinite/intro_music_01.wav'

let ctx: FakeContext
let fetchMock: ReturnType<typeof vi.fn>
/** La réponse `fetch` tenue : on la libère quand le « décodage » doit finir. */
let libere: () => void
/** L'horloge murale injectée, en ms. */
let maintenant = 0
const horloge = () => maintenant

beforeEach(() => {
  const fake = installFakeAudio()
  ctx = fake.ctx
  fetchMock = fake.fetchMock
  fetchMock.mockImplementation(
    () => new Promise((resolve) => { libere = () => resolve(okAudioResponse(3)) }),
  )
  maintenant = 1_000
})

afterEach(() => {
  vi.unstubAllGlobals()
})

/** Fait « finir » le décodage à l'instant `ms` de l'horloge murale. */
async function decodeA(ms: number) {
  maintenant = ms
  libere()
  await flushAudio()
}

describe('introLateBoundS — le retard admis est le préambule à la vitesse courante', () => {
  it('1 s à 1×, 0,5 s à 2×', () => {
    expect(introLateBoundS(1)).toBeCloseTo(1, 9)
    expect(introLateBoundS(2)).toBeCloseTo(0.5, 9)
  })
})

describe('playIntroAligned — départ calé malgré un décodage en retard', () => {
  it('décodé en 200 ms : la source part avec un décalage de 0,2 s', async () => {
    const p = new ReplayAudioPlayer(1)
    void playIntroAligned(p, URL, 1, horloge)
    expect(ctx.sources).toHaveLength(0) // rien tant que le tampon n'est pas là
    await decodeA(1_200)
    expect(ctx.sources).toHaveLength(1)
    expect(ctx.sources[0].offset).toBeCloseTo(0.2, 9)
    // Le reste de l'enveloppe est décalé d'autant : la source s'arrête 0,2 s plus tôt.
    expect(ctx.sources[0].stopped! - ctx.sources[0].started!).toBeCloseTo(3 - 0.2, 9)
  })

  it('décodé au-delà du préambule : aucune source (le coup d envoi est passé)', async () => {
    const p = new ReplayAudioPlayer(1)
    void playIntroAligned(p, URL, 1, horloge)
    await decodeA(2_050)
    expect(ctx.sources).toHaveLength(0)
  })

  it('à 2×, la borne est de 0,5 s : 0,6 s de retard, c est déjà trop tard', async () => {
    const p = new ReplayAudioPlayer(1)
    void playIntroAligned(p, URL, 2, horloge)
    await decodeA(1_600)
    expect(ctx.sources).toHaveLength(0)
  })

  it('déjà décodé : départ immédiat, dans le geste, sans décalage', async () => {
    const p = new ReplayAudioPlayer(1)
    p.preload([URL])
    await decodeA(1_000)
    maintenant = 5_000
    void playIntroAligned(p, URL, 1, horloge)
    expect(ctx.sources).toHaveLength(1)
    expect(ctx.sources[0].offset).toBe(0)
  })
})
