/**
 * eventLoopYield.test.ts — la primitive rend bien la main, et par le canal non bridé.
 *
 * CE QUI NE PEUT PAS ÊTRE TESTÉ ICI : le bridage lui-même. Il n'existe que dans un onglet
 * caché d'un vrai navigateur — jsdom n'a pas de notion de visibilité d'onglet. La mesure qui
 * justifie ce module est consignée dans son en-tête (673 ms contre 0,06 ms) ; ce test-ci
 * verrouille ce qui est vérifiable : la promesse se résout, et elle passe par `MessageChannel`
 * quand il existe.
 */
import { describe, expect, it, vi } from 'vitest'

import { yieldToEvents } from './eventLoopYield'

describe('yieldToEvents', () => {
  it('rend la main puis reprend', async () => {
    let repris = false
    const attente = yieldToEvents().then(() => {
      repris = true
    })
    // La reprise est ASYNCHRONE : elle n'a pas eu lieu au retour de l'appel.
    expect(repris).toBe(false)
    await attente
    expect(repris).toBe(true)
  })

  it('passe par MessageChannel, pas par un minuteur', async () => {
    const timer = vi.spyOn(globalThis, 'setTimeout')
    await yieldToEvents()
    // LE POINT DU TEST : un `setTimeout` ici serait bridé à une seconde en onglet caché, et
    // c'est exactement le bug que ce module corrige.
    expect(timer).not.toHaveBeenCalled()
    timer.mockRestore()
  })

  it('se replie sur un minuteur là où MessageChannel n’existe pas', async () => {
    const vrai = globalThis.MessageChannel
    // @ts-expect-error — on simule un environnement qui n'a pas la primitive.
    delete globalThis.MessageChannel
    try {
      await expect(yieldToEvents()).resolves.toBeUndefined()
    } finally {
      globalThis.MessageChannel = vrai
    }
  })
})
