/**
 * authRequiredGuard.ts — décision de la coquille sur un 401 `auth_required` reçu hors
 * `/bootstrap` (événement `levelup:auth-required`, cf. lib/api/client.ts).
 *
 * Deux gardes, chacun contre une boucle de rechargement :
 *  - `/bootstrap` fait foi : un 401 d'une route secondaire n'éjecte pas tant que
 *    `/bootstrap` dit la session connectée (route qui répond 401 à tort) ;
 *  - un plafond de rechargements par fenêtre glissante, tenu dans `sessionStorage` pour
 *    survivre au rechargement lui-même (une `ref` React y est remise à zéro). Au-delà, la
 *    coquille quitte vers la connexion SANS recharger.
 *
 * Stockage indisponible (navigation privée stricte, accès refusé) : le plafond ne tient
 * plus d'un chargement à l'autre, le premier garde reste.
 */
import type { BootstrapResponse } from '@/lib/api/types'

/** Clé `sessionStorage` des instants (ms epoch) des rechargements pour 401. */
export const AUTH_RELOAD_STORAGE_KEY = 'levelup:auth-reloads'
/** Fenêtre glissante du plafond de rechargements. */
export const AUTH_RELOAD_WINDOW_MS = 60_000
/** Rechargements permis dans la fenêtre ; le suivant est refusé. */
export const AUTH_RELOAD_MAX_IN_WINDOW = 2

/**
 * Verdicts :
 *  - `still_authenticated` : `/bootstrap` dit connecté, rien à faire ;
 *  - `reload` : session expirée, rechargement plein permis (et compté) ;
 *  - `reload_blocked` : session expirée mais plafond atteint — quitter sans recharger ;
 *  - `unknown` : `/bootstrap` injoignable, ne rien décider.
 */
export type AuthRequiredVerdict =
  | { kind: 'still_authenticated' }
  | { kind: 'reload' }
  | { kind: 'reload_blocked'; bootstrap: BootstrapResponse }
  | { kind: 'unknown'; error: unknown }

export interface AuthRequiredDeps {
  fetchBootstrap: () => Promise<BootstrapResponse>
  storage: Storage | null
  now: () => number
}

function readReloads(storage: Storage | null): number[] {
  if (!storage) return []
  try {
    const parsed: unknown = JSON.parse(storage.getItem(AUTH_RELOAD_STORAGE_KEY) ?? '[]')
    return Array.isArray(parsed) ? parsed.filter((v): v is number => typeof v === 'number') : []
  } catch {
    return []
  }
}

/**
 * Réserve un rechargement : `true` (instant enregistré) si moins de
 * AUTH_RELOAD_MAX_IN_WINDOW rechargements ont eu lieu dans la fenêtre, sinon `false`.
 */
export function claimAuthReload(storage: Storage | null, now: number): boolean {
  const recent = readReloads(storage).filter((t) => t <= now && now - t < AUTH_RELOAD_WINDOW_MS)
  if (recent.length >= AUTH_RELOAD_MAX_IN_WINDOW) return false
  recent.push(now)
  if (storage) {
    try {
      storage.setItem(AUTH_RELOAD_STORAGE_KEY, JSON.stringify(recent))
    } catch {
      // Écriture refusée (quota, accès) : le plafond ne survivra pas au rechargement ;
      // le garde `/bootstrap` reste. Rien d'autre à faire.
    }
  }
  return true
}

/** Décide de la suite d'un 401 `auth_required` d'une route secondaire. */
export async function decideOnAuthRequired(deps: AuthRequiredDeps): Promise<AuthRequiredVerdict> {
  let bootstrap: BootstrapResponse
  try {
    bootstrap = await deps.fetchBootstrap()
  } catch (error) {
    return { kind: 'unknown', error }
  }
  if (bootstrap.current_username) return { kind: 'still_authenticated' }
  if (claimAuthReload(deps.storage, deps.now())) return { kind: 'reload' }
  return { kind: 'reload_blocked', bootstrap }
}

/** `window.sessionStorage`, ou `null` si son accès lève (navigateur restrictif). */
export function sessionStorageOrNull(): Storage | null {
  try {
    return window.sessionStorage
  } catch {
    return null
  }
}
