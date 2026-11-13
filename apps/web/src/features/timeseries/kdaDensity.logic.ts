/**
 * kdaDensity.logic — le MODÈLE du graphique « Densité du FDA » (timeseries.03), sans couleur ni
 * option ECharts : la densité tirée des buckets, le tapis des valeurs individuelles, la moyenne et la
 * médiane de TOUS les matchs (population, pas échantillon), et les positions de leurs repères.
 *
 * Pur : le composant `TimeseriesKdaDensity` n'en fait que de l'option de graphe.
 */
import { meanOfValid, medianOfValid } from '@/lib/charts/stats'
import type { DistributionBucket, TimeseriesMatchRow } from '@/lib/api/types'

export interface KdaDensityModel {
  /** Points (milieu du bucket, part des matchs en %, 1 décimale). */
  densityPoints: number[][]
  /** Borne haute de l'axe : le plus grand `bucket_upper`. */
  xMax: number
  /** Moyenne et médiane de tous les FDA finis, à 2 décimales (`null` sans FDA). */
  meanRounded: number | null
  medianRounded: number | null
  /** Position des repères, bornée à l'axe visible. */
  meanX: number | null
  medianX: number | null
  /** Tapis : un tick par match dont le FDA tient dans l'axe, à la hauteur du pied du tapis. */
  rugScatter: number[][]
  /** Hauteurs du haut et du bas du tapis (négatives, sous l'axe des densités). */
  rugYTop: number
  rugYBot: number
}

const round = (v: number, decimals: number): number => {
  const f = 10 ** decimals
  return Math.round(v * f) / f
}

/** `null` sans bucket ou sans match : le graphique n'a rien à dessiner. */
export function buildKdaDensityModel(
  buckets: readonly DistributionBucket[],
  rows: readonly TimeseriesMatchRow[],
): KdaDensityModel | null {
  if (buckets.length === 0) return null

  // Total des matchs pour normaliser en pourcentage.
  const total = buckets.reduce((s, b) => s + b.count, 0)
  if (total === 0) return null

  // Densité : (milieu, %) → série lisse avec aire.
  const densityPoints = buckets.map((b) => [
    (b.bucket_lower + b.bucket_upper) / 2,
    Math.round((b.count / total) * 1000) / 10, // % avec 1 décimale
  ])
  const xMax = buckets.reduce((m, b) => (b.bucket_upper > m ? b.bucket_upper : m), 0)

  // Tapis : valeurs FDA individuelles (r.kda — colonne BDD, sync ADR 0006). On garde aussi les valeurs
  // pleine étendue (sans borne xMax) pour la moyenne et la médiane, qui doivent refléter toutes les données,
  // queue comprise.
  const rugValues: number[] = []
  const allKds: number[] = []
  for (const r of rows) {
    if (r.kda == null || !Number.isFinite(r.kda)) continue
    allKds.push(r.kda)
    if (r.kda <= xMax) rugValues.push(Math.round(r.kda * 1000) / 1000)
  }

  const mean = meanOfValid(allKds)
  const median = medianOfValid(allKds)
  const meanRounded = mean != null ? round(mean, 2) : null
  const medianRounded = median != null ? round(median, 2) : null

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
