/**
 * Tests — killSourceIcons : le référentiel des pictogrammes d'arme, rangé par clé.
 *
 * Ce qu'ils protègent : une entrée complète donne son dessin, sa teinte et son nom ; une entrée sans
 * clé ou sans image n'invente rien ; l'absence de réponse (chargement, erreur, titre sans
 * pictogrammes) donne le référentiel vide.
 */
import { describe, expect, it } from 'vitest'

import { NO_WEAPON_ICONS, weaponIconsOf } from './killSourceIcons'

describe('weaponIconsOf', () => {
  it('range chaque entrée par sa clé, avec son dessin, sa teinte et son nom', () => {
    const icons = weaponIconsOf([
      { key: 'hinf_br75', label: 'BR75', image_url: '/br75.png', tinted: true },
      { key: 'hinf_hydra', label: 'Hydra', image_url: '/hydra.png', tinted: false },
    ])
    expect(icons.size).toBe(2)
    expect(icons.get('hinf_br75')).toEqual({ imageUrl: '/br75.png', tinted: true, label: 'BR75' })
    expect(icons.get('hinf_hydra')).toEqual({ imageUrl: '/hydra.png', tinted: false, label: 'Hydra' })
  })

  it('écarte une entrée sans clé ou sans image', () => {
    const icons = weaponIconsOf([
      { key: '', label: 'Sans clé', image_url: '/x.png', tinted: true },
      { key: 'hinf_sans_image', label: 'Sans image', image_url: '', tinted: true },
    ])
    expect(icons.size).toBe(0)
  })

  it('une réponse absente ou vide donne un référentiel vide', () => {
    expect(weaponIconsOf(null).size).toBe(0)
    expect(weaponIconsOf(undefined).size).toBe(0)
    expect(weaponIconsOf([]).size).toBe(0)
    expect(NO_WEAPON_ICONS.size).toBe(0)
  })
})
