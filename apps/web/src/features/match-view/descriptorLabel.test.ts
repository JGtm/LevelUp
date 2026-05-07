/**
 * Tests unitaires — buildDescriptorLabel.
 *
 * Couvre toutes les variantes du discriminated union `ContextDescriptor`,
 * la dégradation gracieuse (gamertag/startTimeUtc absents → ''), et le
 * formatage Intl FR/EN des dates pour `session` / `period`.
 *
 * Note : les sorties Intl.DateTimeFormat dépendent de la TZ du runner ;
 * les assertions utilisent des regex pour rester tolérantes au fuseau.
 */
import { describe, it, expect } from 'vitest'

import { buildDescriptorLabel } from './descriptorLabel'
import type { ContextDescriptor } from '@/lib/match-nav/navContext'

describe('buildDescriptorLabel', () => {
  it('null/undefined → chaîne vide', () => {
    expect(buildDescriptorLabel(null, 'fr')).toBe('')
    expect(buildDescriptorLabel(undefined, 'fr')).toBe('')
  })

  describe('kind = recent', () => {
    const d: ContextDescriptor = { kind: 'recent' }
    it('FR : "récents"', () => expect(buildDescriptorLabel(d, 'fr')).toBe('récents'))
    it('EN : "recent"', () => expect(buildDescriptorLabel(d, 'en')).toBe('recent'))
  })

  describe('kind = favorites', () => {
    const d: ContextDescriptor = { kind: 'favorites' }
    it('FR : "favoris"', () => expect(buildDescriptorLabel(d, 'fr')).toBe('favoris'))
    it('EN : "favorites"', () => expect(buildDescriptorLabel(d, 'en')).toBe('favorites'))
  })

  describe('kind = media', () => {
    const d: ContextDescriptor = { kind: 'media' }
    it('FR : "avec média"', () => expect(buildDescriptorLabel(d, 'fr')).toBe('avec média'))
    it('EN : "with media"', () => expect(buildDescriptorLabel(d, 'en')).toBe('with media'))
  })

  describe('kind = top_matches', () => {
    const d: ContextDescriptor = { kind: 'top_matches' }
    it('FR : "top performances"', () =>
      expect(buildDescriptorLabel(d, 'fr')).toBe('top performances'))
    it('EN : "top performances"', () =>
      expect(buildDescriptorLabel(d, 'en')).toBe('top performances'))
  })

  describe('kind = with_player', () => {
    it('FR avec gamertag : "avec X"', () => {
      expect(
        buildDescriptorLabel({ kind: 'with_player', gamertag: 'CoolMate' }, 'fr'),
      ).toBe('avec CoolMate')
    })
    it('EN avec gamertag : "with X"', () => {
      expect(
        buildDescriptorLabel({ kind: 'with_player', gamertag: 'CoolMate' }, 'en'),
      ).toBe('with CoolMate')
    })
    it('gamertag vide → chaîne vide (dégradation)', () => {
      expect(buildDescriptorLabel({ kind: 'with_player', gamertag: '' }, 'fr')).toBe('')
    })
  })

  describe('kind = session', () => {
    it('FR avec startTimeUtc : préfixe "de la session du" + date+heure', () => {
      const got = buildDescriptorLabel(
        { kind: 'session', startTimeUtc: '2026-05-07T21:30:00Z' },
        'fr',
      )
      expect(got).toMatch(/^de la session du \d{2}\/\d{2}\/\d{2} à \d{2}:\d{2}$/)
    })
    it('EN avec startTimeUtc : préfixe "from session of" + date+heure', () => {
      const got = buildDescriptorLabel(
        { kind: 'session', startTimeUtc: '2026-05-07T21:30:00Z' },
        'en',
      )
      expect(got).toMatch(/^from session of \d{2}\/\d{2}\/\d{2} at \d{2}:\d{2}$/)
    })
    it('startTimeUtc absent → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'session', startTimeUtc: '' }, 'fr')).toBe('')
    })
    it('startTimeUtc invalide → fallback ISO brut dans le label', () => {
      const got = buildDescriptorLabel(
        { kind: 'session', startTimeUtc: 'pas-une-date' },
        'fr',
      )
      // fmtShortDateTime renvoie l'ISO brut quand parse échoue
      expect(got).toBe('de la session du pas-une-date')
    })
  })

  describe('kind = period', () => {
    it('FR from + to : "de la période du <from> au <to>"', () => {
      const got = buildDescriptorLabel(
        { kind: 'period', from: '2026-04-01T00:00:00Z', to: '2026-05-01T00:00:00Z' },
        'fr',
      )
      expect(got).toMatch(/^de la période du \d{2}\/\d{2}\/\d{2} au \d{2}\/\d{2}\/\d{2}$/)
    })
    it('FR from seul : "depuis le <from>"', () => {
      const got = buildDescriptorLabel(
        { kind: 'period', from: '2026-04-01T00:00:00Z' },
        'fr',
      )
      expect(got).toMatch(/^depuis le \d{2}\/\d{2}\/\d{2}$/)
    })
    it('FR to seul : "jusqu\'au <to>"', () => {
      const got = buildDescriptorLabel(
        { kind: 'period', to: '2026-05-01T00:00:00Z' },
        'fr',
      )
      expect(got).toMatch(/^jusqu'au \d{2}\/\d{2}\/\d{2}$/)
    })
    it('ni from ni to → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'period' }, 'fr')).toBe('')
    })
    it('EN from + to : "from period <from> to <to>"', () => {
      const got = buildDescriptorLabel(
        { kind: 'period', from: '2026-04-01T00:00:00Z', to: '2026-05-01T00:00:00Z' },
        'en',
      )
      expect(got).toMatch(/^from period \d{2}\/\d{2}\/\d{2} to \d{2}\/\d{2}\/\d{2}$/)
    })
  })

  describe('kind = playlist', () => {
    it('FR : "en <name>"', () => {
      expect(
        buildDescriptorLabel({ kind: 'playlist', name: 'Classée Arena' }, 'fr'),
      ).toBe('en Classée Arena')
    })
    it('EN : "in <name>"', () => {
      expect(
        buildDescriptorLabel({ kind: 'playlist', name: 'Ranked Arena' }, 'en'),
      ).toBe('in Ranked Arena')
    })
    it('name vide → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'playlist', name: '' }, 'fr')).toBe('')
    })
  })

  describe('kind = mode', () => {
    it('FR : "en <category>"', () => {
      expect(buildDescriptorLabel({ kind: 'mode', category: 'BTB' }, 'fr')).toBe('en BTB')
    })
    it('EN : "in <category>"', () => {
      expect(buildDescriptorLabel({ kind: 'mode', category: 'BTB' }, 'en')).toBe('in BTB')
    })
    it('category vide → chaîne vide', () => {
      expect(buildDescriptorLabel({ kind: 'mode', category: '' }, 'fr')).toBe('')
    })
  })
})
