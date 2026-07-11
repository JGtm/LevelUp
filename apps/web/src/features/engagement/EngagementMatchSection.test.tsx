import { describe, expect, it } from 'vitest'

import { buildSubtitle } from '@/features/engagement/engagementSubtitle'
import type { EngagementScoreResultAPI } from '@/lib/api/types'

// Base minimale d'un résultat engagement exploitable (bin standard, score normal).
function baseResult(overrides: Partial<EngagementScoreResultAPI> = {}): EngagementScoreResultAPI {
  return {
    engagement_score: 50,
    residual_brut: 0,
    engagement_curve: [],
    match_intensity: 2,
    confidence: 'full',
    n_history_matches: 40,
    expected_basis: 'bin',
    intensity_bin: 'standard',
    ...overrides,
  }
}

describe('EngagementMatchSection buildSubtitle — calibration provisoire (F7 E5)', () => {
  it('appose la mention « calibration provisoire » quand calibration=provisional', () => {
    const sub = buildSubtitle(baseResult({ calibration: 'provisional' }), 'fr')
    expect(sub).toContain('calibration provisoire')
  })

  it('EN : provisional calibration', () => {
    const sub = buildSubtitle(baseResult({ calibration: 'provisional' }), 'en')
    expect(sub).toContain('provisional calibration')
  })

  it('ne montre AUCUNE mention quand calibration=validated', () => {
    const sub = buildSubtitle(baseResult({ calibration: 'validated' }), 'fr')
    expect(sub).not.toContain('provisoire')
  })

  it('ne montre AUCUNE mention quand calibration absente (défaut validé)', () => {
    const sub = buildSubtitle(baseResult(), 'fr')
    expect(sub).not.toContain('provisoire')
  })

  it('mention affichée même en cold_start (calibration = axe distinct de l’attendu)', () => {
    const sub = buildSubtitle(
      baseResult({ expected_basis: 'cold_start', calibration: 'provisional' }),
      'fr',
    )
    expect(sub).toContain('calibration provisoire')
  })
})
