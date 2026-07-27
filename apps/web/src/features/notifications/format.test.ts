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

// Localisation des noms de palier (2026-07-27) — le backend les envoie TOUJOURS
// en anglais : next_tier_name vient de TierState.Label (nom EN + sous-palier) et
// tier/previous_tier du composant CSR de la signature « rating_type|tier|sub_tier ».
// Sans enrichissement, une UI française annonçait « Gold I ».
describe('enrichParams — noms de palier localisés', () => {
  it('traduit next_tier_name en FR en preservant le sous-palier', () => {
    expect(enrichParams({ next_tier_name: 'Gold I' }, 'fr')?.next_tier_name).toBe('Or I')
    expect(enrichParams({ next_tier_name: 'Gold' }, 'fr')?.next_tier_name).toBe('Or')
  })

  it('laisse next_tier_name en anglais sous locale EN', () => {
    expect(enrichParams({ next_tier_name: 'Gold I' }, 'en')?.next_tier_name).toBe('Gold I')
  })

  it('traduit tier et previous_tier de notif.skill_tier', () => {
    const out = enrichParams({ tier: 'Platinum', previous_tier: 'Gold' }, 'fr')
    expect(out?.tier).toBe('Platine')
    expect(out?.previous_tier).toBe('Or')
  })

  it('laisse inchange un nom de palier inconnu et les valeurs non-string', () => {
    const out = enrichParams({ tier: 'Placement', previous_tier: 42 }, 'fr')
    expect(out?.tier).toBe('Placement')
    expect(out?.previous_tier).toBe(42)
  })

  it('resout le titre lusr_tier_approach avec le palier traduit', () => {
    const title = resolveTitle(
      {
        title_key: 'notif.lusr_tier_approach.title',
        params: { gap: 12.8, next_tier_name: 'Gold III' },
      },
      'fr',
    )
    expect(title).toContain('Or III')
    expect(title).not.toContain('Gold')
  })
})
