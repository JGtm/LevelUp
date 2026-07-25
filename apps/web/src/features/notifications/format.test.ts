import { describe, it, expect } from 'vitest'
import { resolveTitle, resolveBody, enrichParams } from './format'
import { getNotificationsText } from './i18n'

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

// current_mu / next_tier_mu voyagent dans le MÊME Params que `gap`
// (buildLUSRTierApproachAlert, internal/progression/coach/generator.go) et sont
// des float64 bruts — même classe de défaut. Ils sont arrondis en DÉFENSE :
// aucun template ne les interpole aujourd'hui, donc l'effet n'est observable
// que sur les params enrichis (d'où enrichParams exporté pour test).
describe('enrichParams — arrondi défensif des μ LUSR (V72 lot découvertes)', () => {
  it('arrondit current_mu et next_tier_mu à 2 décimales', () => {
    const out = enrichParams(
      { current_mu: 1487.123456789, next_tier_mu: 1499.987654321, next_tier_name: 'Or III' },
      'fr',
    )
    expect(out?.current_mu).toBe(1487.12)
    expect(out?.next_tier_mu).toBe(1499.99)
  })

  it('préserve les entiers (1500 → 1500, pas "1500.00")', () => {
    const out = enrichParams({ current_mu: 1490, next_tier_mu: 1500 }, 'fr')
    expect(out?.current_mu).toBe(1490)
    expect(out?.next_tier_mu).toBe(1500)
  })

  it('ignore les valeurs non numériques (aucune coercition)', () => {
    const out = enrichParams({ current_mu: 'n/a', next_tier_mu: null }, 'fr')
    expect(out?.current_mu).toBe('n/a')
    expect(out?.next_tier_mu).toBeNull()
  })

  // Documente l'état constaté le 2026-07-25 : l'arrondi est purement défensif.
  // Ce test n'interdit PAS d'exposer le μ un jour — il rappelle simplement que
  // les templates actuels n'affichent que {gap} / {next_tier_name}.
  it('constat : aucun template lusr_tier_approach n interpole le μ brut', () => {
    for (const locale of ['fr', 'en'] as const) {
      const { templates } = getNotificationsText(locale)
      expect(templates['notif.lusr_tier_approach.title']).not.toContain('{current_mu}')
      expect(templates['notif.lusr_tier_approach.body']).not.toContain('{next_tier_mu}')
    }
  })
})
