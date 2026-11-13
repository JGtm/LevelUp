/**
 * kdaDensity.logic : le modèle du graphique « Densité du FDA ». L'ancien calcul (écrit dans le composant,
 * moyenne et médiane à la main) est gardé ici comme référence : mêmes nombres au bit près (étape 7.8).
 */
import { describe, expect, it } from 'vitest'

import type { DistributionBucket, TimeseriesMatchRow } from '@/lib/api/types'
import { buildKdaDensityModel } from './kdaDensity.logic'

function bucket(lower: number, upper: number, count: number): DistributionBucket {
  return { bucket_lower: lower, bucket_upper: upper, count } as DistributionBucket
}
function row(kda: number | null | undefined): TimeseriesMatchRow {
  return { kda } as TimeseriesMatchRow
}

/** L'ancien calcul, tel qu'il était écrit dans TimeseriesKdaDensity avant 7.8. */
function ancien(buckets: DistributionBucket[], rows: TimeseriesMatchRow[]) {
  if (buckets.length === 0) return null
  const total = buckets.reduce((s, b) => s + b.count, 0)
  if (total === 0) return null
  const densityPoints = buckets.map((b) => [(b.bucket_lower + b.bucket_upper) / 2, Math.round((b.count / total) * 1000) / 10])
  const xMax = buckets.reduce((m, b) => (b.bucket_upper > m ? b.bucket_upper : m), 0)
  const rugValues: number[] = []
  const allKds: number[] = []
  for (const r of rows) {
    if (r.kda == null || !Number.isFinite(r.kda)) continue
    const kda = r.kda
    allKds.push(kda)
    if (kda <= xMax) rugValues.push(Math.round(kda * 1000) / 1000)
  }
  let mean: number | null = null
  let median: number | null = null
  if (allKds.length > 0) {
    mean = allKds.reduce((s, v) => s + v, 0) / allKds.length
    const sorted = [...allKds].sort((a, b) => a - b)
    const mid = Math.floor(sorted.length / 2)
    median = sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid]
  }
  const meanRounded = mean != null ? Math.round(mean * 100) / 100 : null
  const medianRounded = median != null ? Math.round(median * 100) / 100 : null
  const yMaxPct = densityPoints.reduce((m, p) => (p[1] > m ? p[1] : m), 0)
  const rugYTop = -(yMaxPct * 0.05)
  const rugYBot = -(yMaxPct * 0.14)
  return {
    densityPoints,
    xMax,
    meanRounded,
    medianRounded,
    meanX: meanRounded != null ? Math.min(meanRounded, xMax) : null,
    medianX: medianRounded != null ? Math.min(medianRounded, xMax) : null,
    rugScatter: rugValues.map((v) => [v, rugYBot]),
    rugYTop,
    rugYBot,
  }
}

const BUCKETS = [bucket(0, 0.5, 2), bucket(0.5, 1, 7), bucket(1, 1.5, 9), bucket(1.5, 2, 4), bucket(2, 3, 1)]

const CAS: Array<[string, DistributionBucket[], TimeseriesMatchRow[]]> = [
  ['sans bucket', [], [row(1)]],
  ['buckets vides (total nul)', [bucket(0, 1, 0)], [row(1)]],
  ['sans match', BUCKETS, []],
  ['impair', BUCKETS, [row(0.4), row(1.2), row(0.9), row(1.7), row(1.1)]],
  ['pair', BUCKETS, [row(0.4), row(1.2), row(0.9), row(1.7)]],
  ['queue au-delà de l’axe : comptée dans la moyenne, hors du tapis', BUCKETS, [row(0.5), row(1), row(8.25)]],
  ['absents et non finis ignorés', BUCKETS, [row(null), row(undefined), row(NaN), row(Infinity), row(1.0049999), row(1.005)]],
  ['décimales non représentables', BUCKETS, [row(0.1), row(0.2), row(0.3), row(0.7), row(1.0 / 3)]],
]

describe('buildKdaDensityModel : mêmes nombres qu’avant 7.8', () => {
  it.each(CAS)('%s', (_label, buckets, rows) => {
    const attendu = ancien(buckets, rows)
    const obtenu = buildKdaDensityModel(buckets, rows)
    if (attendu == null) {
      expect(obtenu).toBeNull()
      return
    }
    expect(obtenu).not.toBeNull()
    const o = obtenu!
    for (const k of ['xMax', 'meanRounded', 'medianRounded', 'meanX', 'medianX', 'rugYTop', 'rugYBot'] as const) {
      expect(Object.is(o[k], attendu[k]), k).toBe(true)
    }
    expect(o.densityPoints).toEqual(attendu.densityPoints)
    expect(o.rugScatter).toEqual(attendu.rugScatter)
  })

  it('moyenne et médiane arrondies à 2 décimales, repères bornés à l’axe', () => {
    const m = buildKdaDensityModel(BUCKETS, [row(0.5), row(1), row(8.25)])!
    expect(m.meanRounded).toBe(3.25)
    expect(m.medianRounded).toBe(1)
    expect(m.meanX).toBe(3)
    expect(m.rugScatter).toHaveLength(2)
  })
})
