/**
 * Tests des libellés de variante : un groupe de file classée du CSR, une chaîne LUSR et un
 * gamertag, dans les deux langues, et leur écriture dans une ligne d'indicateur.
 */
import { renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { getTendancesText } from './i18n'
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

describe('useIndicatorLabeler — groupe « Rythme »', () => {
  const labeler = (locale: 'fr' | 'en') =>
    renderHook(() => useIndicatorLabeler(locale)).result.current

  it('les cinq stats par minute sont libellées dans les deux langues', () => {
    expect(labeler('fr')('kills_per_minute')).toBe('Frags par minute')
    expect(labeler('fr')('deaths_per_minute')).toBe('Morts par minute')
    expect(labeler('fr')('assists_per_minute')).toBe('Assistances par minute')
    expect(labeler('fr')('damage_dealt_per_minute')).toBe('Dégâts infligés par minute')
    expect(labeler('fr')('damage_taken_per_minute')).toBe('Dégâts subis par minute')
    expect(labeler('en')('kills_per_minute')).toBe('Kills per minute')
    expect(labeler('en')('damage_taken_per_minute')).toBe('Damage taken per minute')
  })

  it('le nom du groupe : « Rythme » / « Pace »', () => {
    expect(getTendancesText('fr').groupLabel('pace')).toBe('Rythme')
    expect(getTendancesText('en').groupLabel('pace')).toBe('Pace')
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
