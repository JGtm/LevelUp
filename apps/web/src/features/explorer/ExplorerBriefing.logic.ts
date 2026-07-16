/**
 * ExplorerBriefing.logic — helpers purs du bandeau de briefing (mode Matchs).
 *
 * Aucune dépendance React/DOM : logique testable en isolation (formatage des
 * deltas signés, KDA agrégat, mapping de la frise). Les composants du bandeau
 * consomment ces helpers.
 */
import type { OutcomeValue } from '@/components/charts/OutcomeSequenceTape'

/**
 * Formate un delta signé numérique avec préfixe explicite (+/−/±) et N décimales.
 * ±0 quand nul (le signe ne doit pas dépendre uniquement de la couleur).
 */
export function formatSignedFixed(v: number | null | undefined, decimals: number): string {
  if (v == null || !Number.isFinite(v)) return ''
  const rounded = Number(v.toFixed(decimals))
  if (rounded === 0) return `±${(0).toFixed(decimals)}`
  const abs = Math.abs(rounded).toFixed(decimals)
  return rounded > 0 ? `+${abs}` : `−${abs}`
}

/**
 * Formate un delta de TAUX (ratio 0..1) en points de pourcentage signés
 * (ex. +0.30 → "+30 pts"). Unité « pts » pour distinguer d'un pourcentage absolu.
 */
export function formatSignedPoints(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  const pts = Math.round(v * 100)
  if (pts === 0) return '±0 pts'
  return pts > 0 ? `+${pts} pts` : `−${Math.abs(pts)} pts`
}

/** Signe d'un nombre : -1 / 0 / 1 (0 pour nul, absent ou non fini). */
export function signOf(v: number | null | undefined): -1 | 0 | 1 {
  if (v == null || !Number.isFinite(v) || v === 0) return 0
  return v > 0 ? 1 : -1
}

/** Mappe le code outcome backend (1=égalité,2=victoire,3=défaite,4=abandon) vers OutcomeValue. */
export function outcomeCodeToValue(code: number): OutcomeValue {
  switch (code) {
    case 2:
      return 'win'
    case 3:
      return 'loss'
    case 4:
      return 'dnf'
    default:
      return 'tie' // 1 = égalité
  }
}
