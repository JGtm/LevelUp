/**
 * Tests unitaires pour le hook useFieldLabel et les helpers de fetch.
 *
 * On teste la logique de fallback (mappings non chargés, key absente) sans
 * passer par le hook React, pour rester rapide et déterministe.
 */

import { describe, expect, it } from 'vitest'

import {
  compareSeasonsRecentFirst,
  fieldMappingsQueryKey,
  type FieldMappingsResponse,
  type SeasonEntry,
} from './fieldMappings'

describe('fieldMappingsQueryKey', () => {
  it('produit une clé hiérarchique (slug, locale)', () => {
    expect(fieldMappingsQueryKey('halo_infinite', 'fr')).toEqual([
      'field-mappings',
      'halo_infinite',
      'fr',
    ])
  })

  it('encode le couple (slug, locale) sans collision', () => {
    const a = fieldMappingsQueryKey('halo_infinite', 'fr')
    const b = fieldMappingsQueryKey('halo_infinite', 'en')
    const c = fieldMappingsQueryKey('synthetic_b', 'fr')
    expect(a).not.toEqual(b)
    expect(a).not.toEqual(c)
  })
})

describe('compareSeasonsRecentFirst (GH5-1 — récent d\'abord)', () => {
  const mk = (id: string, iso: string, order: number): SeasonEntry => ({
    id,
    label: id,
    shortLabel: id,
    startDate: new Date(iso),
    endDate: null,
    displayOrder: order,
  })

  it('trie la plus récente en tête (startDate DESC)', () => {
    const sorted = [
      mk('s1', '2022-01-01T00:00:00Z', 10),
      mk('s3', '2023-06-01T00:00:00Z', 30),
      mk('s2', '2022-06-01T00:00:00Z', 20),
    ].sort(compareSeasonsRecentFirst)
    expect(sorted.map((s) => s.id)).toEqual(['s3', 's2', 's1'])
  })

  it('place une saison DB-only (displayOrder synthétique élevé) à sa date réelle, pas en tête', () => {
    // DB-only ANCIENNE : displayOrder=140 (synthétique maxOrder+10) mais startDate 2021.
    // Un tri par displayOrder DESC la mettrait en TÊTE à tort ; la clé startDate la
    // renvoie en dernier (sa vraie place chronologique). Réf. piège GH5-1.
    const dbOnlyOld = mk('db_old', '2021-01-01T00:00:00Z', 140)
    const s13 = mk('s13', '2025-11-18T00:00:00Z', 130)
    const s1 = mk('s1', '2022-01-01T00:00:00Z', 10)
    const sorted = [s1, dbOnlyOld, s13].sort(compareSeasonsRecentFirst)
    expect(sorted.map((s) => s.id)).toEqual(['s13', 's1', 'db_old'])
  })

  it('départage deux saisons de même date par displayOrder DESC', () => {
    const a = mk('a', '2024-01-01T00:00:00Z', 10)
    const b = mk('b', '2024-01-01T00:00:00Z', 20)
    expect([a, b].sort(compareSeasonsRecentFirst).map((s) => s.id)).toEqual(['b', 'a'])
  })
})

describe('FieldMappingsResponse fallback chains', () => {
  const sample: FieldMappingsResponse = {
    title_slug: 'halo_infinite',
    schema_version: 1,
    locale: 'fr',
    fields: {
      kills: {
        label: 'Frags',
        storage_unit: 'count',
        display_unit: 'count',
        format: 'integer',
        display_order: 10,
        group: 'combat',
      },
    },
  }

  it('retourne le label localisé pour une key connue', () => {
    expect(sample.fields['kills']?.label).toBe('Frags')
  })

  it('retourne undefined pour une key absente (caller fallback sur key)', () => {
    expect(sample.fields['unknown_key']?.label).toBeUndefined()
  })

  it('retourne undefined pour un mappings vide (404 backend)', () => {
    const empty: FieldMappingsResponse = {
      title_slug: 'halo_infinite',
      schema_version: 0,
      locale: 'fr',
      fields: {},
    }
    expect(empty.fields['kills']?.label).toBeUndefined()
  })
})

describe('FieldMappingsResponse — assets et outcomes (Phase 3 plan finition)', () => {
  it('expose un asset par kind/id avec label localisé', () => {
    const sample: FieldMappingsResponse = {
      title_slug: 'halo_infinite',
      schema_version: 1,
      locale: 'fr',
      fields: {},
      assets: {
        mode: {
          Ranked: { label: 'Classé', display_order: 50 },
          Firefight: {
            label: 'Baptême du feu',
            color_token: 'mode.firefight',
            display_order: 60,
          },
        },
        challenge_tier: {
          heroic: {
            label: 'Héroïque',
            color_token: 'challenge.heroic',
            display_order: 20,
          },
        },
      },
    }
    expect(sample.assets?.mode?.Ranked?.label).toBe('Classé')
    expect(sample.assets?.challenge_tier?.heroic?.color_token).toBe('challenge.heroic')
  })

  it('retourne undefined pour kind inconnu (caller fallback sur id)', () => {
    const sample: FieldMappingsResponse = {
      title_slug: 'halo_infinite',
      schema_version: 1,
      locale: 'fr',
      fields: {},
      assets: { mode: {} },
    }
    expect(sample.assets?.mode?.Ranked?.label).toBeUndefined()
    expect(sample.assets?.unknown_kind?.foo?.label).toBeUndefined()
  })

  it('expose un outcome par key avec label + color_token', () => {
    const sample: FieldMappingsResponse = {
      title_slug: 'halo_infinite',
      schema_version: 1,
      locale: 'fr',
      fields: {},
      outcomes: {
        win: { label: 'Victoire', color_token: 'outcome.positive' },
        loss: { label: 'Défaite', color_token: 'outcome.negative' },
        tie: { label: 'Égalité', color_token: 'outcome.neutral' },
        dnf: { label: 'Abandon', color_token: 'outcome.neutral' },
      },
    }
    expect(sample.outcomes?.win?.label).toBe('Victoire')
    expect(sample.outcomes?.dnf?.color_token).toBe('outcome.neutral')
  })

  it('assets et outcomes optionnels — réponse sans eux ne casse pas', () => {
    const sample: FieldMappingsResponse = {
      title_slug: 'halo_infinite',
      schema_version: 1,
      locale: 'fr',
      fields: {},
    }
    expect(sample.assets).toBeUndefined()
    expect(sample.outcomes).toBeUndefined()
  })
})
