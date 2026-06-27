/**
 * instances.test.ts — Snapshot des seuils métier.
 *
 * Tout changement de seuil fait échouer ce test.
 * Cela force une revue consciente et une mise à jour du snapshot.
 */
import { describe, it, expect } from 'vitest'
import {
  perfScale, accuracyScale, kdScale, progressScale,
  mmrDeltaScale, skillDeltaScale, kdaDivergentScale, outcomeScale, narrativeScale,
} from '../instances'

describe('instances — snapshot des seuils', () => {
  it('perfScale — table de vérité complète', () => {
    const cases: [number, string][] = [
      [100, 'perf-tier-1'], [80, 'perf-tier-1'], [79, 'perf-tier-2'],
      [65, 'perf-tier-2'],  [64, 'perf-tier-3'], [50, 'perf-tier-3'],
      [49, 'perf-tier-4'],  [35, 'perf-tier-4'], [34, 'perf-tier-5'],
      [0,  'perf-tier-5'],
    ]
    for (const [v, expected] of cases) {
      expect(perfScale(v), `perfScale(${v})`).toBe(expected)
    }
  })

  it('kdScale — table de vérité §9.7', () => {
    const cases: [number, string][] = [
      [2.0,  'perf-tier-1'], [1.0, 'perf-tier-1'], // ≥1 = bon
      [0.99, 'perf-tier-3'], [0.5, 'perf-tier-3'], [0.0, 'perf-tier-3'], // [0,1[ = moyen
      [-0.1, 'perf-tier-5'],                        // <0 = mauvais
    ]
    for (const [v, expected] of cases) {
      expect(kdScale(v), `kdScale(${v})`).toBe(expected)
    }
  })

  it('mmrDeltaScale — bande ±10', () => {
    expect(mmrDeltaScale(11)).toBe('divergent-pos')
    expect(mmrDeltaScale(10)).toBe('divergent-neutral')  // borne inclusive
    expect(mmrDeltaScale(0)).toBe('divergent-neutral')
    expect(mmrDeltaScale(-10)).toBe('divergent-neutral') // borne inclusive
    expect(mmrDeltaScale(-11)).toBe('divergent-neg')
  })

  it('skillDeltaScale — strict zéro', () => {
    expect(skillDeltaScale(0.001)).toBe('divergent-pos')
    expect(skillDeltaScale(0)).toBe('divergent-neutral')
    expect(skillDeltaScale(-0.001)).toBe('divergent-neg')
  })

  it('kdaDivergentScale — FDA signée native (Halo 5), strict zéro', () => {
    expect(kdaDivergentScale(2.33)).toBe('divergent-pos')  // FDA positive = bon
    expect(kdaDivergentScale(0)).toBe('divergent-neutral') // pile neutre
    expect(kdaDivergentScale(-1.5)).toBe('divergent-neg')  // FDA négative = mauvais (légitime)
  })

  it('cohérence croisée — kdScale(1.0) et perfScale(80) retournent le même tier', () => {
    expect(kdScale(1.0)).toBe(perfScale(80)) // les deux = 'perf-tier-1'
  })

  it('outcomeScale — toutes clés', () => {
    expect(outcomeScale('win')).toBe('outcome-win')
    expect(outcomeScale('loss')).toBe('outcome-loss')
    expect(outcomeScale('draw')).toBe('outcome-draw')
    expect(outcomeScale('dnf')).toBe('outcome-dnf')
    expect(outcomeScale(null)).toBeNull()
    expect(outcomeScale('unknown')).toBeNull()
  })

  it('narrativeScale — toutes clés', () => {
    expect(narrativeScale('dominant')).toBe('narrative-dominant')
    expect(narrativeScale('humiliation')).toBe('narrative-humiliation')
    expect(narrativeScale('remontada')).toBe('narrative-remontada')
    expect(narrativeScale('debacle')).toBe('narrative-debacle')
    expect(narrativeScale('contre_remontada')).toBe('narrative-contre-remontada')
    expect(narrativeScale(null)).toBeNull()
  })

  it('snapshot complet des instances (drift guard)', () => {
    // Valeurs représentatives — snapshot bloquant pour tout drift accidentel
    expect({
      perf: [perfScale(85), perfScale(70), perfScale(55), perfScale(40), perfScale(20)],
      kd: [kdScale(1.5), kdScale(0.5), kdScale(-0.5)],
      accuracy: [accuracyScale(60), accuracyScale(47), accuracyScale(30)],
      progress: [progressScale(80), progressScale(60), progressScale(30), progressScale(10)],
      mmrDelta: [mmrDeltaScale(50), mmrDeltaScale(5), mmrDeltaScale(-50)],
      skillDelta: [skillDeltaScale(1), skillDeltaScale(0), skillDeltaScale(-1)],
    }).toMatchSnapshot()
  })
})
