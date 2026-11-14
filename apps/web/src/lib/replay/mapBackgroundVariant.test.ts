import { describe, expect, it } from 'vitest'

import {
  backgroundImagePath,
  isWallPlanDrawn,
  loadedBackground,
  normalizeBackgroundStyle,
  resolveBackgroundVariant,
} from './mapBackgroundVariant'

describe('normalizeBackgroundStyle', () => {
  it('ne reconnait que plan ; tout le reste est le rendu', () => {
    expect(normalizeBackgroundStyle('plan')).toBe('plan')
    expect(normalizeBackgroundStyle('rendu')).toBe('rendu')
    expect(normalizeBackgroundStyle('PLAN')).toBe('rendu')
    expect(normalizeBackgroundStyle('autre')).toBe('rendu')
    expect(normalizeBackgroundStyle(null)).toBe('rendu')
    expect(normalizeBackgroundStyle(undefined)).toBe('rendu')
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

describe('backgroundImagePath', () => {
  it('laisse l adresse du rendu nue et marque celle du plan', () => {
    expect(backgroundImagePath('/a/background.png', 'rendu')).toBe('/a/background.png')
    expect(backgroundImagePath('/a/background.png', 'plan')).toBe('/a/background.png?variante=plan')
  })
})

describe('isWallPlanDrawn', () => {
  it('trace les murs quand l’interrupteur est allumé sur le rendu', () => {
    expect(isWallPlanDrawn(true, 'rendu')).toBe(true)
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
