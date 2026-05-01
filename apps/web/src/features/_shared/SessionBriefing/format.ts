/**
 * format.ts — helpers de formatage durées pour le SessionBriefing.
 */

/**
 * Format "HhMMmin" (ex 1h49min) ou "MMmin" si moins d'une heure.
 */
export function formatDurationDhm(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  return h > 0 ? `${h}h ${String(m).padStart(2, '0')}min` : `${m}min`
}

/**
 * Format "M:SS" pour durées courtes (vie moyenne, durée match).
 */
export function formatMmss(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.round(seconds % 60)
  return `${m}:${String(s).padStart(2, '0')}`
}
