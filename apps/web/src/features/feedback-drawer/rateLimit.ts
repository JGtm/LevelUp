/**
 * Anti-spam soft : limite à 5 submits / heure / browser via localStorage.
 *
 * Pas une vraie protection (le user peut clear localStorage), mais bloque
 * les accidents de double/triple clic et les bots naïfs. Fail-open
 * silencieux si localStorage est indisponible (mode privé strict, quota
 * exceeded, sandbox sans storage).
 */

const STORAGE_KEY = 'levelup-feedback-submits'
const WINDOW_MS = 60 * 60 * 1000 // 1h
export const MAX_SUBMITS_PER_HOUR = 5

function readTimestamps(): number[] {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter((x): x is number => typeof x === 'number' && Number.isFinite(x))
  } catch {
    return []
  }
}

function writeTimestamps(ts: number[]): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(ts))
  } catch {
    // fail-open : pas de storage dispo, on n'empêche pas l'envoi
  }
}

function pruneExpired(ts: number[], now: number): number[] {
  return ts.filter((t) => now - t < WINDOW_MS)
}

/** Renvoie le nombre de submits encore disponibles dans la fenêtre 1h. */
export function getRemainingSubmits(now: number = Date.now()): number {
  const fresh = pruneExpired(readTimestamps(), now)
  return Math.max(0, MAX_SUBMITS_PER_HOUR - fresh.length)
}

/** Enregistre un submit. Renvoie `true` si autorisé, `false` si rate-limited. */
export function recordSubmit(now: number = Date.now()): boolean {
  const fresh = pruneExpired(readTimestamps(), now)
  if (fresh.length >= MAX_SUBMITS_PER_HOUR) return false
  fresh.push(now)
  writeTimestamps(fresh)
  return true
}

/** Vide les timestamps (tests + future option "reset" UI). */
export function _resetSubmitsForTests(): void {
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}
