/**
 * viewPresetHints.test.ts — LA DESCRIPTION DE CHAQUE PRÉRÉGLAGE DIT CE QUE LA TABLE ALLUME. Les
 * textes sont écrits à la main dans chaque langue : sans ce test, un calque ajouté à la table
 * (les trois des vies) laissait « Tous les calques » ou une liste fausse à l'écran.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import { KNOWN_LOCALES } from '@/lib/i18n/generated/locales'
import { PRESET_LAYERS, VIEW_PRESET_IDS, type LayerKey } from './viewPresetLogic'

const LIVES: ReadonlyArray<[LayerKey, 'layerSpawns' | 'layerDeaths' | 'layerKillers']> = [
  ['spawns', 'layerSpawns'],
  ['deaths', 'layerDeaths'],
  ['killers', 'layerKillers'],
]

describe.each([...KNOWN_LOCALES])('descriptions des préréglages (%s)', (locale) => {
  const t = REPLAY_TEXT[locale]

  it('chaque préréglage nomme les calques des vies qu’il allume, et seulement ceux-là', () => {
    for (const id of VIEW_PRESET_IDS) {
      if (id === 'complet') continue
      for (const [key, textKey] of LIVES) {
        const label = t.lives[textKey].toLowerCase()
        const named = t.viewPresetHint[id].toLowerCase().includes(label)
        expect(named, `${locale} ${id} / ${key}`).toBe(PRESET_LAYERS[id].includes(key))
      }
    }
  })

  it('Complet n’annonce pas « tous les calques » : il dit que le groupe des vies en est exclu', () => {
    expect(PRESET_LAYERS.complet).not.toContain('spawns')
    expect(t.viewPresetHint.complet).toContain(t.lives.title)
  })
})
