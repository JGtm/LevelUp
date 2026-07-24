import { describe, it, expect } from 'vitest'

import { hijacksLabelKey } from './hijacksLabel'
import { synthesisManifest } from '@/lib/i18n/generated/synthesis'

describe('hijacksLabelKey (libellé « vol à la tire » par titre)', () => {
  it('Halo 5 → clé h5, FR « Vol à la tire », EN « Hijacks »', () => {
    const key = hijacksLabelKey('halo_5')
    expect(key).toBe('synthesis.combat_profile.hijacks_h5')
    expect(synthesisManifest[key].fr).toBe('Vol à la tire')
    expect(synthesisManifest[key].en).toBe('Hijacks')
  })

  it('Halo Infinite → clé infinite, FR « Dépositaire » (pas d\'anglicisme), EN « Hijacks »', () => {
    const key = hijacksLabelKey('halo_infinite')
    expect(key).toBe('synthesis.combat_profile.hijacks_infinite')
    expect(synthesisManifest[key].fr).toBe('Dépositaire')
    expect(synthesisManifest[key].en).toBe('Hijacks')
  })

  it('titre inconnu / bootstrap non chargé → défaut Infinite', () => {
    expect(hijacksLabelKey('')).toBe('synthesis.combat_profile.hijacks_infinite')
    expect(hijacksLabelKey('unknown_title')).toBe('synthesis.combat_profile.hijacks_infinite')
  })
})
