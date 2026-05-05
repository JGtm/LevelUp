/**
 * i18n.test.ts — buildContextLabel : produit un label localisé depuis un
 * MatchFilterSpec pour la barre de navigation contextuelle (Phase 2b).
 */
import { describe, it, expect } from 'vitest'

import { buildContextLabel } from './i18n'
import type { MatchFilterSpec } from '@/lib/match-nav/navContext'

describe('buildContextLabel', () => {
  it('spec null : chaîne vide', () => {
    expect(buildContextLabel(null, 'fr')).toBe('')
    expect(buildContextLabel(undefined, 'fr')).toBe('')
  })

  it('playlist seule', () => {
    expect(
      buildContextLabel({ playlist_name: 'Classée Arena' }, 'fr'),
    ).toBe('Classée Arena')
  })

  it('playlist + mode', () => {
    expect(
      buildContextLabel(
        { playlist_name: 'Classée Arena', mode_category: 'Ranked' },
        'fr',
      ),
    ).toBe('Classée Arena · Ranked')
  })

  it('outcome FR vs EN', () => {
    const spec: MatchFilterSpec = { outcome: 'win' }
    expect(buildContextLabel(spec, 'fr')).toBe('Victoires')
    expect(buildContextLabel(spec, 'en')).toBe('Wins')
  })

  it('range complet de dates', () => {
    const spec: MatchFilterSpec = {
      date_from: '2026-04-01T00:00:00Z',
      date_to: '2026-05-01T00:00:00Z',
    }
    const got = buildContextLabel(spec, 'fr')
    expect(got).toContain('→')
    expect(got).toContain('04')
    expect(got).toContain('05')
  })

  it('seulement date_from : "Depuis JJ/MM/YYYY"', () => {
    const spec: MatchFilterSpec = { date_from: '2026-04-01T00:00:00Z' }
    expect(buildContextLabel(spec, 'fr')).toMatch(/^Depuis /)
    expect(buildContextLabel(spec, 'en')).toMatch(/^From /)
  })

  it('seulement date_to : "Jusqu\'au JJ/MM/YYYY"', () => {
    const spec: MatchFilterSpec = { date_to: '2026-05-01T00:00:00Z' }
    expect(buildContextLabel(spec, 'fr')).toMatch(/^Jusqu'au /)
    expect(buildContextLabel(spec, 'en')).toMatch(/^To /)
  })

  it('combinaison complète FR', () => {
    const spec: MatchFilterSpec = {
      playlist_name: 'Classée',
      mode_category: 'BTB',
      outcome: 'loss',
      date_from: '2026-04-01T00:00:00Z',
    }
    const got = buildContextLabel(spec, 'fr')
    expect(got).toContain('Classée')
    expect(got).toContain('BTB')
    expect(got).toContain('Défaites')
    expect(got).toContain('Depuis')
    expect(got.split(' · ')).toHaveLength(4)
  })

  it('session_id : préfixé par #', () => {
    expect(
      buildContextLabel({ session_id: 'sess-2026-04-30' }, 'fr'),
    ).toBe('#sess-2026-04-30')
  })
})
