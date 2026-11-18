import { describe, expect, it } from 'vitest'

import {
  backgroundImagePath,
  baseImageVariant,
  isWallPlanDrawn,
  loadedBackground,
  MAP_BACKGROUND_VARIANTS,
  normalizeBackgroundStyle,
  reliefDataPath,
  resolveBackgroundVariant,
} from './mapBackgroundVariant'

describe('MAP_BACKGROUND_VARIANTS', () => {
  it('propose les fonds dans l’ordre Plan, Rendu, Relief, le défaut en premier', () => {
    expect(MAP_BACKGROUND_VARIANTS).toEqual(['plan', 'rendu', 'relief'])
  })
})

describe('normalizeBackgroundStyle', () => {
  it('reconnait plan, rendu et relief ; tout le reste est le défaut, le plan', () => {
    expect(normalizeBackgroundStyle('plan')).toBe('plan')
    expect(normalizeBackgroundStyle('rendu')).toBe('rendu')
    expect(normalizeBackgroundStyle('relief')).toBe('relief')
    expect(normalizeBackgroundStyle('Relief')).toBe('plan')
    expect(normalizeBackgroundStyle('PLAN')).toBe('plan')
    expect(normalizeBackgroundStyle('autre')).toBe('plan')
    expect(normalizeBackgroundStyle(null)).toBe('plan')
    expect(normalizeBackgroundStyle(undefined)).toBe('plan')
  })
})

describe('resolveBackgroundVariant', () => {
  it('demande le plan quand il est voulu et que le fond en porte un', () => {
    expect(resolveBackgroundVariant('plan', 'plan.png')).toBe('plan')
  })

  it('retombe sur le rendu sans image plan, quelle que soit la preference', () => {
    expect(resolveBackgroundVariant('plan', undefined)).toBe('rendu')
    expect(resolveBackgroundVariant('plan', null)).toBe('rendu')
    expect(resolveBackgroundVariant('plan', '')).toBe('rendu')
  })

  it('garde le rendu quand c est lui qui est voulu, plan disponible ou non', () => {
    expect(resolveBackgroundVariant('rendu', 'plan.png')).toBe('rendu')
    expect(resolveBackgroundVariant('rendu', undefined)).toBe('rendu')
  })
})

describe('resolveBackgroundVariant — le relief et ses replis', () => {
  const offert = { available: true, failed: false }

  it('pose le relief quand il est voulu ET offert', () => {
    expect(resolveBackgroundVariant('relief', 'plan.png', offert)).toBe('relief')
    expect(resolveBackgroundVariant('relief', null, offert)).toBe('relief')
  })

  it('retombe sur le plan quand le fond n’a pas de relief', () => {
    expect(resolveBackgroundVariant('relief', 'plan.png', { available: false, failed: false })).toBe('plan')
  })

  it('retombe sur le plan quand la lecture du relief a échoué (ReliefError)', () => {
    expect(resolveBackgroundVariant('relief', 'plan.png', { available: true, failed: true })).toBe('plan')
  })

  it('retombe sur le rendu quand il n’y a ni relief lisible ni plan', () => {
    expect(resolveBackgroundVariant('relief', undefined, { available: false, failed: false })).toBe('rendu')
    expect(resolveBackgroundVariant('relief', null, { available: true, failed: true })).toBe('rendu')
  })

  it('sans le troisième argument, le relief n’est jamais offert : plan ou rendu, jamais relief', () => {
    expect(resolveBackgroundVariant('relief', 'plan.png')).toBe('plan')
    expect(resolveBackgroundVariant('relief', undefined)).toBe('rendu')
  })

  it('le plan voulu reste le plan, relief offert ou non ; le rendu voulu reste le rendu', () => {
    expect(resolveBackgroundVariant('plan', 'plan.png', offert)).toBe('plan')
    expect(resolveBackgroundVariant('plan', true, offert)).toBe('plan')
    expect(resolveBackgroundVariant('rendu', 'plan.png', offert)).toBe('rendu')
  })
})

describe('baseImageVariant', () => {
  it('le relief voulu charge l’image plan quand elle existe (posée en attendant, ou à sa place), le rendu sinon', () => {
    expect(baseImageVariant('relief', 'plan.png')).toBe('plan')
    expect(baseImageVariant('relief', true)).toBe('plan')
    expect(baseImageVariant('relief', undefined)).toBe('rendu')
  })

  it('le plan et le rendu se chargent tels quels', () => {
    expect(baseImageVariant('plan', undefined)).toBe('plan')
    expect(baseImageVariant('rendu', 'plan.png')).toBe('rendu')
  })
})

describe('backgroundImagePath', () => {
  it('laisse l adresse du rendu nue et marque celle du plan et celle du relief', () => {
    expect(backgroundImagePath('/a/background.png', 'rendu')).toBe('/a/background.png')
    expect(backgroundImagePath('/a/background.png', 'plan')).toBe('/a/background.png?variante=plan')
    expect(backgroundImagePath('/a/background.png', 'relief')).toBe('/a/background.png?variante=relief')
  })
})

describe('reliefDataPath', () => {
  it('donne la route sœur des données du relief, sur les deux préfixes (rejeu et Tactique)', () => {
    expect(reliefDataPath('/players/p/matches/m/replay/background.png')).toBe('/players/p/matches/m/replay/background.relief.json')
    expect(reliefDataPath('/players/p/tactical/streets/background.png')).toBe('/players/p/tactical/streets/background.relief.json')
  })
})

describe('isWallPlanDrawn', () => {
  it('trace les murs quand l’interrupteur est allumé sur le rendu', () => {
    expect(isWallPlanDrawn(true, 'rendu')).toBe(true)
  })

  it('reste disponible sur le relief, qui ne porte pas les murs, indépendamment des traits du relief', () => {
    expect(isWallPlanDrawn(true, 'relief')).toBe(true)
    expect(isWallPlanDrawn(false, 'relief')).toBe(false)
  })

  it('ne trace rien sur le plan, qui porte déjà les murs, interrupteur allumé ou non', () => {
    expect(isWallPlanDrawn(true, 'plan')).toBe(false)
    expect(isWallPlanDrawn(false, 'plan')).toBe(false)
  })

  it('ne trace rien sur le rendu quand l’interrupteur est éteint', () => {
    expect(isWallPlanDrawn(false, 'rendu')).toBe(false)
  })
})

describe('loadedBackground', () => {
  it('plan demandé et chargé : le plan, variante plan', () => {
    expect(loadedBackground('plan', 'P', 'R')).toEqual({ image: 'P', variante: 'plan' })
  })

  it('plan demandé mais échoué : le rendu en repli, variante rendu (les murs reviennent)', () => {
    expect(loadedBackground('plan', undefined, 'R')).toEqual({ image: 'R', variante: 'rendu' })
  })

  it('rendu demandé : le rendu, même si un plan est en cache', () => {
    expect(loadedBackground('rendu', 'P', 'R')).toEqual({ image: 'R', variante: 'rendu' })
  })

  it('rien de chargé : pas d’image, la variante demandée', () => {
    expect(loadedBackground('plan', undefined, undefined)).toEqual({ image: null, variante: 'plan' })
    expect(loadedBackground('rendu', undefined, undefined)).toEqual({ image: null, variante: 'rendu' })
  })
})
