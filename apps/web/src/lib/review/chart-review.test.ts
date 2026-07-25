import { describe, expect, it } from 'vitest'

import { CHART_REVIEW, chartReview } from './chart-review'
import { REVIEW_TEXT } from './i18n'

describe('chartReview', () => {
  it('retourne undefined pour une clé absente ou vide (badge inerte)', () => {
    expect(chartReview('cle.inexistante')).toBeUndefined()
    expect(chartReview(undefined)).toBeUndefined()
    expect(chartReview('')).toBeUndefined()
  })

  it('retourne l’entrée du manifeste pour une clé connue', () => {
    const keys = Object.keys(CHART_REVIEW)
    if (keys.length === 0) return // manifeste vidé en fin de tournée : rien à vérifier
    const entry = chartReview(keys[0])
    expect(entry).toBeDefined()
    expect(entry).toBe(CHART_REVIEW[keys[0]])
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
