/**
 * Helpers purs de la section « Coordination d'équipe » (lot F, maquette 034b1915).
 *
 * `positionCategorie` porte le seul calcul de la carte, et il est GÉOMÉTRIQUE : où poser le
 * trait de seuil sur un axe de CATÉGORIES. Se tromper d'un demi-intervalle déplace la règle
 * du jeu sur l'écran sans rien casser ailleurs — d'où ces cas.
 */
import { describe, expect, it } from 'vitest'

import { getTacticalText } from './i18n'
import { libelleRayons, positionCategorie } from './tacticalView.logic'

const BINS = [
  { min_m: 0, max_m: 10 },
  { min_m: 10, max_m: 20 },
  { min_m: 20, max_m: 30 },
  { min_m: 30, max_m: 40 },
  { min_m: 40, max_m: 50 },
  { min_m: 50, max_m: null },
]

describe('positionCategorie — le seuil tombe à sa VRAIE place sur l’axe', () => {
  it('18 m sur des intervalles de 10 m : entre la deuxième et la troisième barre', () => {
    // Barre « 10-20 » d'indice 1 ; 18 m est à 80 % de l'intervalle ; le centre de la barre
    // est à l'indice 1, son bord gauche à 0,5 => 0,5 + 0,8 = 1,3.
    expect(positionCategorie(18, BINS)).toBeCloseTo(1.3, 5)
  })

  it('24 m : dans l’intervalle 20-30, à 40 % de celui-ci', () => {
    expect(positionCategorie(24, BINS)).toBeCloseTo(2.4 - 0.5, 5)
  })

  it('une distance hors des intervalles servis ne pose aucun seuil', () => {
    expect(positionCategorie(-1, BINS)).toBeNull()
    expect(positionCategorie(10, [])).toBeNull()
  })

  it('le dernier intervalle est OUVERT : tout ce qui dépasse y tombe', () => {
    expect(positionCategorie(120, BINS)).toBe(5)
  })
})

describe('libelleRayons — les portées, jamais leur moyenne', () => {
  const t = getTacticalText('fr')

  it('une seule portée', () => {
    expect(libelleRayons(t, [18])).toBe('18 m')
  })

  it('deux formats dans le filtre : les deux valeurs, jointes', () => {
    expect(libelleRayons(t, [18, 24])).toBe('18 m ou 24 m')
  })
})
