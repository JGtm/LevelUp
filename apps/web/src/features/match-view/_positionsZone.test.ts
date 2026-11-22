/**
 * _positionsZone.test.ts — la boîte d'« Occupation du terrain » : 72 % de la largeur utile sur 21,6 rem de haut.
 */
import { describe, expect, it } from 'vitest'

import { boiteDeLOccupation, dimensionsDuCadre, PLAN_MAX_HEIGHT_REM, PLAN_MAX_WIDTH_PCT, PX_PAR_REM } from './_positionsZone'

describe('boiteDeLOccupation', () => {
  it('72 % de la largeur utile, sur 21,6 rem (345,6 px)', () => {
    expect(PLAN_MAX_WIDTH_PCT).toBe(72)
    expect(PLAN_MAX_HEIGHT_REM * PX_PAR_REM).toBeCloseTo(345.6, 10)
    const boite = boiteDeLOccupation(1000)
    expect(boite.largeur).toBeCloseTo(720, 10)
    expect(boite.hauteur).toBeCloseTo(345.6, 10)
  })

  it('une largeur utile pas mesurée rend une boîte nulle', () => {
    expect(boiteDeLOccupation(0)).toEqual({ largeur: 0, hauteur: 0 })
    expect(boiteDeLOccupation(Number.NaN)).toEqual({ largeur: 0, hauteur: 0 })
  })
})

describe('dimensionsDuCadre', () => {
  it('la largeur et la hauteur du cadre du fond, en mètres', () => {
    expect(dimensionsDuCadre({ originX: 0, originY: 10, widthM: 20, heightM: 10 })).toEqual({ largeur: 20, hauteur: 10 })
  })

  it('sans cadre, ou un cadre vide : null', () => {
    expect(dimensionsDuCadre(null)).toBeNull()
    expect(dimensionsDuCadre({ originX: 0, originY: 0, widthM: 0, heightM: 10 })).toBeNull()
  })
})
