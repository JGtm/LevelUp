/**
 * Tests unitaires pour le hook useFieldLabel et les helpers de fetch.
 *
 * On teste la logique de fallback (mappings non chargés, key absente) sans
 * passer par le hook React, pour rester rapide et déterministe.
 */

import { describe, expect, it } from 'vitest'

import {
  fieldMappingsQueryKey,
  type FieldMappingsResponse,
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
