/**
 * tacticalLecture.logic — l'état de la lecture du plan (retours rejeu L2, 2026-09-23).
 */
import { describe, expect, it } from 'vitest'

import type { BornesMonde } from '@/lib/api/types'

import { aspectDuPlan, dimensionsDuPlan, etatLecture, questionServie } from './tacticalLecture.logic'
import { PLAN_ASPECT_DEFAUT } from './tacticalView.logic'

const BASE = { aDesDonnees: true, enEchec: false, surPlaceholder: false, perimetreEnRelecture: false }

describe('etatLecture', () => {
  it('aucune donnée : attente (premier chargement)', () => {
    expect(etatLecture({ ...BASE, aDesDonnees: false })).toBe('attente')
  })

  it('réponse précédente servie en placeholder : relecture', () => {
    expect(etatLecture({ ...BASE, surPlaceholder: true })).toBe('relecture')
  })

  it('périmètre en cours de relecture : relecture, même si le raster est à jour', () => {
    expect(etatLecture({ ...BASE, perimetreEnRelecture: true })).toBe('relecture')
  })

  it('l’échec prime, données gardées ou non', () => {
    expect(etatLecture({ ...BASE, enEchec: true })).toBe('echec')
    expect(etatLecture({ ...BASE, aDesDonnees: false, enEchec: true })).toBe('echec')
  })

  it('réponse courante : pret', () => {
    expect(etatLecture(BASE)).toBe('pret')
  })
})

describe('questionServie', () => {
  it('rend la question à laquelle la réponse répond, pas la demandée', () => {
    expect(questionServie('morts', 'kills')).toBe('morts')
  })

  it('une valeur hors vocabulaire retombe sur la question demandée', () => {
    expect(questionServie('inconnue', 'kills')).toBe('kills')
    expect(questionServie('', 'temps')).toBe('temps')
  })
})

describe('aspectDuPlan', () => {
  const FOND = { originX: -50, originY: 40, widthM: 100, heightM: 50 }

  it('le repère de la lecture quand il existe', () => {
    expect(aspectDuPlan(FOND, { minX: 0, maxX: 30, minY: 0, maxY: 10, pasM: 1 })).toBe(3)
  })

  it('sans lecture, le calage du fond seul — le cadre a déjà sa taille finale', () => {
    expect(aspectDuPlan(FOND, null)).toBe(2)
  })

  it('ni repère ni calage exploitable : le rapport par défaut', () => {
    expect(aspectDuPlan(null, null)).toBe(PLAN_ASPECT_DEFAUT)
    expect(aspectDuPlan({ ...FOND, heightM: 0 }, null)).toBe(PLAN_ASPECT_DEFAUT)
  })
})

describe('dimensionsDuPlan', () => {
  const FOND = { originX: -50, originY: 40, widthM: 100, heightM: 50 }
  const BORNES: BornesMonde = { min_x: -10, max_x: 20, min_y: 0, max_y: 60, valide: true }

  it('le calage du fond quand il est connu, posé droit', () => {
    expect(dimensionsDuPlan(FOND, BORNES)).toEqual({ largeur: 100, hauteur: 50 })
    expect(dimensionsDuPlan(FOND, undefined)).toEqual({ largeur: 100, hauteur: 50 })
  })

  it('sans fond, la boîte englobante des cellules', () => {
    expect(dimensionsDuPlan(null, BORNES)).toEqual({ largeur: 30, hauteur: 60 })
    expect(dimensionsDuPlan({ ...FOND, heightM: 0 }, BORNES)).toEqual({ largeur: 30, hauteur: 60 })
  })

  it('ni fond ni bornes exploitables : null (le plan reste droit)', () => {
    expect(dimensionsDuPlan(null, undefined)).toBeNull()
    expect(dimensionsDuPlan(null, { ...BORNES, valide: false })).toBeNull()
    expect(dimensionsDuPlan(null, { ...BORNES, max_x: BORNES.min_x })).toBeNull()
  })
})
