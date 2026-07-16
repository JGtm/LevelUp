/**
 * postLoginDestination — source unique de la destination après un login réussi.
 *
 * Règle produit (bug post-login) : un joueur DÉJÀ établi (profil prêt, données
 * synchronisées) doit atterrir directement sur le dashboard. Seul un joueur
 * réellement nouveau (setup pas encore « ready ») voit la page d'onboarding
 * « Bienvenue sur LevelUp / on synchronise tes derniers matchs ».
 *
 * Utilisé par les DEUX chemins de login SSO Xbox (device code + guard de la page
 * onboarding) pour qu'ils traitent l'utilisateur existant de façon identique.
 *
 * Signal d'autorité = `setup_state === 'ready'` (le domaine Go ne renvoie 'ready'
 * qu'une fois le profil provisionné avec un joueur courant). Le fallback
 * `current_player` couvre un éventuel payload sans setup_state.
 */
import type { BootstrapResponse } from '@/lib/api/types'

export const ONBOARDING_PATH = '/onboarding/openspartan'
export const DASHBOARD_PATH = '/'

/** true si le joueur a déjà un profil prêt (données existantes) → dashboard direct. */
export function isEstablishedPlayer(boot: BootstrapResponse | null | undefined): boolean {
  if (!boot) return false
  return boot.setup_state === 'ready' && !!boot.current_player
}

/** Destination post-login : dashboard pour un joueur établi, onboarding sinon. */
export function postLoginDestination(boot: BootstrapResponse | null | undefined): string {
  return isEstablishedPlayer(boot) ? DASHBOARD_PATH : ONBOARDING_PATH
}
