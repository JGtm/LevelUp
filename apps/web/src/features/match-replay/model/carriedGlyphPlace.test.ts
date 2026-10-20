import { describe, expect, it, vi } from 'vitest'

import { carriedGlyphPlaceAt, type CarrierPosAt } from './carriedGlyphPlace'

/** Une lecture de position qui ne répond QUE sur les images listées. */
const posOfFrames = (byFrame: Record<number, { x: number; y: number }>): CarrierPosAt =>
  (_xuid, frame) => byFrame[frame] ?? null

describe('carriedGlyphPlaceAt', () => {
  it('rend `carried` sur la position du porteur quand il est localisable', () => {
    const place = carriedGlyphPlaceAt(posOfFrames({ 10: { x: 5, y: 6 } }), { xuid: 'a', t0: 0 }, 10, null)
    expect(place).toEqual({ state: 'carried', at: { x: 5, y: 6 } })
  })

  it('rend `free` à la DERNIÈRE position connue du porteur quand l’image est muette', () => {
    // Le porteur est localisable jusqu'à l'image 7, muet ensuite : l'objet reste où il a été vu.
    const place = carriedGlyphPlaceAt(posOfFrames({ 7: { x: 1, y: 2 } }), { xuid: 'a', t0: 0 }, 10, null)
    expect(place).toEqual({ state: 'free', at: { x: 1, y: 2 } })
  })

  it('ne remonte JAMAIS avant le début du portage', () => {
    // La seule position connue est AVANT t0 : elle appartient à la vie d'avant la prise.
    const posOf = posOfFrames({ 2: { x: 9, y: 9 } })
    expect(carriedGlyphPlaceAt(posOf, { xuid: 'a', t0: 5 }, 10, null)).toEqual({ state: 'absent' })
    // Le repli de l'appelant, lui, sert : c'est LUI qui connaît le dernier repos de l'objet.
    expect(carriedGlyphPlaceAt(posOf, { xuid: 'a', t0: 5 }, 10, { x: 3, y: 4 }))
      .toEqual({ state: 'free', at: { x: 3, y: 4 } })
  })

  it('sert le repli de l’appelant quand le porteur n’a JAMAIS de position', () => {
    const place = carriedGlyphPlaceAt(() => null, { xuid: 'a', t0: 0 }, 10, { x: 3, y: 4 })
    expect(place).toEqual({ state: 'free', at: { x: 3, y: 4 } })
  })

  it('rend `absent` sans porteur nommé et sans repli — on n’invente aucune position', () => {
    const posOf = vi.fn(() => ({ x: 0, y: 0 }))
    expect(carriedGlyphPlaceAt(posOf, { xuid: null, t0: 0 }, 10, null)).toEqual({ state: 'absent' })
    // Un portage sans xuid ne déclenche AUCUNE relecture : il n'y a pas de trajectoire à lire.
    expect(posOf).not.toHaveBeenCalled()
  })

  it('MUTATION — le balayage arrière s’arrête à la PREMIÈRE position trouvée, la plus récente', () => {
    // Deux positions connues : si le balayage partait de t0 vers l'avant (ou n'interrompait pas),
    // il servirait la plus ANCIENNE. L'objet doit rester là où il a été vu EN DERNIER.
    const place = carriedGlyphPlaceAt(
      posOfFrames({ 1: { x: 100, y: 100 }, 8: { x: 1, y: 2 } }),
      { xuid: 'a', t0: 0 },
      10,
      null,
    )
    expect(place).toEqual({ state: 'free', at: { x: 1, y: 2 } })
  })

  it('MUTATION — les trois états sont EXCLUSIFS : jamais deux glyphes pour le même objet', () => {
    // La même image, la même période : une seule réponse, donc un seul glyphe possible.
    const posOf = posOfFrames({ 10: { x: 5, y: 6 }, 7: { x: 1, y: 2 } })
    const span = { xuid: 'a', t0: 0 }
    expect(carriedGlyphPlaceAt(posOf, span, 10, { x: 3, y: 4 }).state).toBe('carried')
    expect(carriedGlyphPlaceAt(posOf, span, 9, { x: 3, y: 4 }).state).toBe('free')
  })
})
