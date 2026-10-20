import { describe, expect, it } from 'vitest'

import { CHART_REVIEW, chartReview } from './chart-review'
import { REVIEW_TEXT } from './i18n'

describe('chartReview', () => {
  it('retourne undefined pour une clé absente ou vide (badge inerte)', () => {
    expect(chartReview('cle.inexistante')).toBeUndefined()
    expect(chartReview(undefined)).toBeUndefined()
    expect(chartReview('')).toBeUndefined()
  })

  it('retourne l’entrée du manifeste pour chaque clé inscrite', () => {
    // Tournée close le 2026-09-14 : le manifeste est vide, la boucle ne tourne pas et
    // aucune pastille ne s’affiche. Elle reprend tout son sens dès qu’une tournée
    // réinscrit des graphes.
    for (const key of Object.keys(CHART_REVIEW)) {
      expect(chartReview(key)).toBe(CHART_REVIEW[key])
    }
  })
})

describe('manifeste de revue — intégrité', () => {
  it('chaque entrée porte un statut connu et une note FR ET EN non vides', () => {
    for (const [key, entry] of Object.entries(CHART_REVIEW)) {
      expect(['verify', 'new', 'removal'], `statut inconnu pour ${key}`).toContain(entry.status)
      expect(entry.note.fr, `note FR manquante pour ${key}`).toBeTruthy()
      expect(entry.note.en, `note EN manquante pour ${key}`).toBeTruthy()
    }
  })

  it('les libellés de badge existent en FR et en EN pour les 3 statuts', () => {
    for (const locale of ['fr', 'en'] as const) {
      for (const status of ['verify', 'new', 'removal'] as const) {
        expect(REVIEW_TEXT[locale][status].label).toBeTruthy()
        expect(REVIEW_TEXT[locale][status].aria).toBeTruthy()
      }
    }
  })
})
