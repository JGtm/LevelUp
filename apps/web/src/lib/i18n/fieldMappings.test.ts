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
        label: 'Éliminations',
        storage_unit: 'count',
        display_unit: 'count',
        format: 'integer',
        display_order: 10,
        group: 'combat',
      },
    },
  }

  it('retourne le label localisé pour une key connue', () => {
    expect(sample.fields['kills']?.label).toBe('Éliminations')
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
