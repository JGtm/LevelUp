/**
 * usageMetricKinds.test.ts — le classement et le tri des grandeurs du bloc « usages
 * d'équipement, armes spéciales et objectifs » (S3). Extrait de `usageLogic.test.ts` le
 * 2026-09-09 (étape E5.1bis, scission de taille — CLAUDE.md n°5) au moment du déménagement
 * du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import type { SessionUsageMetric } from '@/lib/api/types'

import { USAGE_TEXT } from './usageI18n'
import { equipmentMetrics, metricKind, metricLabel } from './usageMetricKinds'

const t = USAGE_TEXT.fr

/** Une métrique minimale : les champs requis seuls, le reste ABSENT (nil). */
function metric(overrides: Partial<SessionUsageMetric> & { key: string }): SessionUsageMetric {
  return {
    player_total: 0,
    lobby_total: 0,
    matches_above_lobby_parity: 0,
    ...overrides,
  }
}

describe('classement et tri des grandeurs', () => {
  it('classe les clés du contrat, ensemble ouvert côté deployed_*', () => {
    expect(metricKind('camo_episodes')).toBe('camo')
    expect(metricKind('deployed_wall')).toBe('wall')
    expect(metricKind('deployed_sensor')).toBe('deployed_other')
    expect(metricKind('pad_pickups')).toBe('pads')
    expect(metricKind('mystere')).toBe('other')
  })

  it('exclut pad_pickups ET dropped_objects du bloc équipement (E4.5 : une mort devient un segment, plus une ligne)', () => {
    const sorted = equipmentMetrics([
      metric({ key: 'pad_pickups' }),
      metric({ key: 'dropped_objects' }),
      metric({ key: 'grapple_pulls' }),
      metric({ key: 'camo_episodes' }),
    ])
    expect(sorted.map((m) => m.key)).toEqual(['camo_episodes', 'grapple_pulls'])
  })

  it('classe une clé equipment_<famille>, ensemble ouvert', () => {
    expect(metricKind('equipment_wall')).toBe('equipment')
    expect(metricKind('equipment_translocator_beacon')).toBe('equipment')
  })

  it("equipment_<famille> REMPLACE sa grandeur soeur (deployed_/camo_/overshield_) quand les deux coexistent — une seule ligne par famille (E4.3)", () => {
    const sorted = equipmentMetrics([
      metric({ key: 'deployed_wall' }),
      metric({ key: 'equipment_wall' }),
      metric({ key: 'camo_episodes' }),
      metric({ key: 'equipment_powerup_camo' }),
      metric({ key: 'overshield_episodes' }), // pas d equipment_powerup_overshield : reste seule
      metric({ key: 'grapple_pulls' }), // pas d equipment_grapple : reste seule
    ])
    expect(sorted.map((m) => m.key).sort()).toEqual(
      ['equipment_wall', 'equipment_powerup_camo', 'overshield_episodes', 'grapple_pulls'].sort(),
    )
  })

  it('libellé d une grandeur equipment_<famille> : familles connues nommées, inconnue garde sa clé', () => {
    expect(metricLabel('equipment_wall', t)).toBe(t.metricWall)
    expect(metricLabel('equipment_powerup_camo', t)).toBe(t.metricCamo)
    expect(metricLabel('equipment_powerup_overshield', t)).toBe(t.metricOvershield)
    expect(metricLabel('equipment_sensor', t)).toBe(t.equipSensor)
    expect(metricLabel('equipment_translocator_beacon', t)).toBe(t.equipTranslocator)
    expect(metricLabel('equipment_shroud_screen', t)).toBe(t.equipShroud)
    expect(metricLabel('equipment_threat_seeker', t)).toBe(t.equipSeeker)
    expect(metricLabel('equipment_repair_field', t)).toBe(t.equipField)
    expect(metricLabel('equipment_mystere', t)).toBe(t.metricDeployedFmt('mystere'))
  })
})
