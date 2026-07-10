/**
 * Garde-rail — foyer canonique de MetricWithTrend (règle ≤2 copies).
 *
 * MetricWithTrend (flèche de tendance temporelle ▲▼= colorée par token) est
 * défini UNE seule fois dans components/ui/metric-trend.tsx et importé par ses
 * consommateurs (LeaderboardBlock, CareerRankingBlock). Ce test interdit toute
 * RÉ-INLINE de la primitive : re-déclarer `function MetricWithTrend` ailleurs
 * re-diverge (leçon prédicat bot : 8 → 36 copies).
 *
 * NOTE dette : le vocabulaire `above/below/near` (KPIStrip, PlayerScoreCard)
 * compare une valeur à une RÉFÉRENCE, sémantique distincte — hors périmètre de
 * ce garde-rail (voir handoff C1 — Découvertes).
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute — pas de
// dépendance à node:fs ni aux types node dans le tsconfig applicatif.
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CANONICAL = '/src/components/ui/metric-trend.tsx'

describe('garde-rail MetricWithTrend canonique', () => {
  it('MetricWithTrend n’est déclaré que dans components/ui/metric-trend.tsx', () => {
    const decl = /function\s+MetricWithTrend\b|const\s+MetricWithTrend\s*=/
    const offenders = Object.entries(sources)
      .filter(([path]) => path !== CANONICAL)
      .filter(([, code]) => decl.test(code))
      .map(([path]) => path)
    expect(offenders, `MetricWithTrend ré-inliné hors du foyer canonique : ${offenders.join(', ')}`).toEqual([])
  })
})
