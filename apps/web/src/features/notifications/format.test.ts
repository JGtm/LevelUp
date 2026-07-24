import { describe, it, expect } from 'vitest'
import { resolveTitle, resolveBody } from './format'

describe('resolveTitle — arrondi des params numériques (enrichParams)', () => {
  // V72-27b : le backend envoie `gap` en float64 brut (internal/progression/
  // coach/generator.go — buildLUSRTierApproachAlert). Sans arrondi, le titre
  // notif.lusr_tier_approach interpolait la mantisse complète (illisible).
  it('arrondit gap à 2 décimales dans notif.lusr_tier_approach.title (fr)', () => {
    const title = resolveTitle(
      {
        title_key: 'notif.lusr_tier_approach.title',
        params: { gap: 12.847213698, next_tier_name: 'Or III' },
      },
      'fr',
    )
    expect(title).toBe('À 12.85 pts de Or III')
  })

  it('arrondit gap à 2 décimales dans notif.lusr_tier_approach.title (en)', () => {
    const title = resolveTitle(
      {
        title_key: 'notif.lusr_tier_approach.title',
        params: { gap: 12.847213698, next_tier_name: 'Gold III' },
      },
      'en',
    )
    expect(title).toBe('12.85 pts from Gold III')
  })

  it('préserve un gap entier sans mantisse artificielle (5 → "5", pas "5.00")', () => {
    const title = resolveTitle(
      { title_key: 'notif.lusr_tier_approach.title', params: { gap: 5, next_tier_name: 'Platine I' } },
      'fr',
    )
    expect(title).toBe('À 5 pts de Platine I')
  })

  it('arrondit toujours value/target/previous_value (non-régression)', () => {
    const body = resolveBody(
      {
        body_key: 'notif.threshold_crossed.body',
        params: { metric_label: 'K/D ratio', value: 1.66666667 },
      },
      'fr',
    )
    expect(body).not.toContain('1.66666667')
  })
})
