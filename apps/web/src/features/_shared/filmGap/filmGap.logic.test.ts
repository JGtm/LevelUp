import { describe, expect, it } from 'vitest'

import { filmExpected, filmGapOfBlock } from './filmGap.logic'
import { FILM_GAP_TEXT } from './filmGapStrings'

describe('filmGapOfBlock', () => {
  it('une analyse en cours prime : « pending »', () => {
    expect(filmGapOfBlock({ matches_film_pending: 1, matches_film_missing: 4 })).toBe('pending')
  })
  it('aucune en cours mais des films non lus : « missing »', () => {
    expect(filmGapOfBlock({ matches_film_missing: 2 })).toBe('missing')
  })
  it('tous les films lus : null (une section sans sujet reste absente)', () => {
    expect(filmGapOfBlock({})).toBeNull()
    expect(filmGapOfBlock({ matches_film_pending: 0, matches_film_missing: 0 })).toBeNull()
  })
  it('titre qui ne lit pas le film, ou bloc absent : null (capability absente, jamais un faux état)', () => {
    expect(filmGapOfBlock({ film_unavailable: 'film_unsupported', matches_film_missing: 5 })).toBeNull()
    expect(filmGapOfBlock(null)).toBeNull()
    expect(filmGapOfBlock(undefined)).toBeNull()
  })
  it('filmExpected : le bloc existe et le titre lit le film', () => {
    expect(filmExpected({})).toBe(true)
    expect(filmExpected({ film_unavailable: 'film_load_failed' })).toBe(true)
    expect(filmExpected({ film_unavailable: 'film_unsupported' })).toBe(false)
    expect(filmExpected(null)).toBe(false)
  })
})

describe('textes d’une section sans film lu', () => {
  it('constats factuels FR et EN : aucune action, aucun mot interne', () => {
    const interdits = /\b(réanalys|relanc|cuisson|artefact|reconstruct|vous|tu |ton |ta |mon |ma |your|please|retry)/i
    for (const loc of ['fr', 'en'] as const) {
      const t = FILM_GAP_TEXT[loc]
      for (const s of [t.pending, t.missingTitle, t.missingDescription]) {
        expect(s, `${loc}: ${s}`).not.toMatch(interdits)
        expect(s.length).toBeGreaterThan(0)
      }
    }
  })
})
