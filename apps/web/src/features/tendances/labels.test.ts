/**
 * Tests des libellés de variante : un groupe de file classée du CSR, une chaîne LUSR et un
 * gamertag, dans les deux langues, et leur écriture dans une ligne d'indicateur.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useIndicatorLabeler, variantLabel } from './labels'

vi.mock('@/lib/i18n/fieldMappings', () => ({ useFieldMappings: () => ({ data: undefined }) }))

describe('variantLabel', () => {
  it('CSR : le groupe de file classée est traduit', () => {
    expect(variantLabel('csr_value', 'ranked', 'fr')).toBe('Classé')
    expect(variantLabel('csr_value', 'ranked', 'en')).toBe('Ranked')
  })

  it('CSR : un groupe inconnu reste tel quel', () => {
    expect(variantLabel('csr_value', 'groupe_inconnu', 'fr')).toBe('groupe_inconnu')
  })

  it('LUSR : le nom connu de la chaîne', () => {
    expect(variantLabel('lusr_value', 'arena_slayer', 'fr')).toBe('Social · Assassin')
  })

  it('un gamertag reste tel quel', () => {
    expect(variantLabel('kda', 'Alice', 'fr')).toBe('Alice')
  })
})

describe('useIndicatorLabeler — variante', () => {
  const labeler = (locale: 'fr' | 'en') =>
    renderHook(() => useIndicatorLabeler(locale)).result.current

  it('« CSR · Classé » en français, « CSR · Ranked » en anglais', () => {
    expect(labeler('fr')('csr_value', 'ranked')).toMatch(/ · Classé$/)
    expect(labeler('en')('csr_value', 'ranked')).toMatch(/ · Ranked$/)
  })

  it('un gamertag est écrit tel quel après le libellé', () => {
    expect(labeler('fr')('kda', 'Alice')).toMatch(/ · Alice$/)
  })
})
