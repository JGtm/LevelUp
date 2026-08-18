/**
 * Helpers canoniques pour le formatage de durées (revue 2026-04-29 P2.6bis).
 *
 * Format canonique M:SS des durées de match, consommé par les tableaux de matchs
 * et de sessions (explorer, match-view, session-detail).
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
 * formatClockMMSS — un INSTANT sur l'horloge d'un match, en M:SS, depuis des millisecondes.
 *
 * POURQUOI CE N'EST PAS `formatDurationMMSS`, ET POURQUOI CE N'EN EST PAS UNE COPIE. Une
 * DURÉE nulle n'existe pas : `formatDurationMMSS(0)` rend « - », et c'est juste — un match
 * de zéro seconde n'a pas eu lieu. Un INSTANT nul, lui, est le coup d'envoi : le rendre
 * « - » effacerait le début de toute frise. Les deux fonctions ont donc la même sortie
 * visuelle et deux domaines disjoints ; l'entrée en millisecondes (l'unité des calques du
 * film) achève de les distinguer.
 *
 * @example
 *   formatClockMMSS(0)        // "0:00" (le coup d'envoi, pas une absence)
 *   formatClockMMSS(65_300)   // "1:05"
 *   formatClockMMSS(-5_000)   // "0:00" (jamais un temps négatif)
 */
export function formatClockMMSS(ms: number): string {
  const total = Math.max(0, Math.round((Number.isFinite(ms) ? ms : 0) / 1000))
  const m = Math.floor(total / 60)
  const s = total % 60
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

/**
 * Format durée compact "XminYs" (utile pour des cellules de tableau étroites).
 *
 * @example
 *   formatDurationMinSec(30)   // "0min30s"
 *   formatDurationMinSec(125)  // "2min5s"
 *   formatDurationMinSec(0)    // "-"
 */
export function formatDurationMinSec(seconds?: number | null, fallback = '-'): string {
  if (seconds == null || seconds <= 0 || !Number.isFinite(seconds)) {
    return fallback
  }
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}min${s}s`
}

/**
 * Format durée compact "XmYYs" avec secondes sur 2 chiffres (utile en tooltip
 * pour lever l'ambiguïté d'un MM:SS où "1:05" pourrait se lire 1h05).
 *
 * Diffère de formatDurationMinSec : lettre "m" (pas "min") et secondes zero-paddées.
 *
 * @example
 *   formatDurationMShort(65)    // "1m05s"
 *   formatDurationMShort(37)    // "0m37s"
 *   formatDurationMShort(125)   // "2m05s"
 *   formatDurationMShort(0)     // "-"
 */
export function formatDurationMShort(seconds?: number | null, fallback = '-'): string {
  if (seconds == null || seconds <= 0 || !Number.isFinite(seconds)) {
    return fallback
  }
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}m${s.toString().padStart(2, '0')}s`
}

/**
 * Format durée « h min » pour des TOTAUX de temps de jeu (durée cumulée d'un
 * scope de matchs, ex. tuile « Durée totale » du briefing Explorer). Distinct de
 * formatDurationMMSS (MM:SS d'UN match) et de formatDurationHMS : ici on exprime
 * des heures et des minutes lisibles, jamais de secondes. Unités « h »/« min »
 * universelles (identiques FR/EN). Fallback « — » (tuiles de briefing).
 *
 * @example
 *   formatDurationHM(2530)    // "42 min"  (< 1 h)
 *   formatDurationHM(152400)  // "42 h 20" (heures + minutes zero-paddées)
 *   formatDurationHM(3600)    // "1 h 00"
 *   formatDurationHM(0)       // "—"
 */
export function formatDurationHM(seconds?: number | null, fallback = '—'): string {
  if (seconds == null || seconds <= 0 || !Number.isFinite(seconds)) {
    return fallback
  }
  const totalMin = Math.floor(seconds / 60)
  const h = Math.floor(totalMin / 60)
  const m = totalMin % 60
  if (h === 0) return `${m} min`
  return `${h} h ${m.toString().padStart(2, '0')}`
}
