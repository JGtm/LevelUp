/**
 * Helpers canoniques pour le formatage de durées (revue 2026-04-29 P2.6bis).
 *
 * Centralise le pattern dispersé dans :
 *   - features/squad/v2/components/HistoryTable.tsx::formatDuration (M:SS)
 *
 * Convention : entrée en secondes (le DTO API `time_played_seconds` est
 * une convention interne stable).
 */

/**
 * Format durée en MM:SS (utile pour la durée d'un match Halo).
 *
 * @example
 *   formatDurationMMSS(125)        // "2:05"
 *   formatDurationMMSS(3661)       // "61:01" (pas d'heure)
 *   formatDurationMMSS(undefined)  // "-"
 *   formatDurationMMSS(0)          // "-" (pas de durée)
 */
export function formatDurationMMSS(seconds?: number | null, fallback = '-'): string {
  if (seconds == null || seconds <= 0 || !Number.isFinite(seconds)) {
    return fallback
  }
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

/**
 * Format durée en HH:MM:SS (utile pour les durées longues — total de jeu
 * sur la saison, par exemple).
 *
 * @example
 *   formatDurationHMS(3661)    // "1:01:01"
 *   formatDurationHMS(125)     // "0:02:05"
 */
export function formatDurationHMS(seconds?: number | null, fallback = '-'): string {
  if (seconds == null || seconds <= 0 || !Number.isFinite(seconds)) {
    return fallback
  }
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = Math.floor(seconds % 60)
  return `${h}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`
}
