import { describe, expect, it } from 'vitest'

import { EN_RELIEF, FR_RELIEF } from './relief.i18n'

describe('infobulle du fond Relief du panneau « Vue »', () => {
  it('dit la lumière de l’ÉCRAN (haut à gauche, D-15), jamais un point cardinal de la carte', () => {
    expect(FR_RELIEF.backgroundHint).toContain('haut à gauche')
    expect(EN_RELIEF.backgroundHint).toContain('upper left')
    for (const hint of [FR_RELIEF.backgroundHint, EN_RELIEF.backgroundHint]) expect(hint).not.toMatch(/nord|north/i)
  })
})
