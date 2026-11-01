/**
 * Capacités du profil de déploiement (ADR 0044) lues dans `capabilities` de /bootstrap.
 *
 * Le serveur décide une fois, au chargement de sa configuration, ce que l'instance OFFRE
 * (comptes, mise en commun des jetons, réglages modifiables, dépôt de captures…). Le client
 * branche sur ces capacités, jamais sur le nom du profil (`capabilities.deployment`, réservé à
 * l'affichage et au diagnostic) ni sur `auth_mode` / `demo_mode` : même règle que pour les
 * capacités des titres (`useCapability`).
 *
 * Avant l'hydratation du store, et pour une réponse sans `capabilities`, les capacités valent ce
 * qu'étaient les anciens défauts du store : authentification absente, instance qui n'est pas la
 * démo (les écrans ne se verrouillent pas avant que le serveur l'ait dit).
 */

import type { CapabilityMap } from '@/lib/api/types'

/** Valeurs d'avant l'hydratation : aucun compte, rien de figé. */
export const DEPLOYMENT_CAPABILITY_DEFAULTS = {
  can_use_accounts: false,
  can_sign_in_with_xbox: false,
  can_use_invitations: false,
  can_use_groups: true,
  can_pool_tokens: true,
  can_use_admin: true,
  can_use_restic: true,
  can_use_discord: true,
  can_use_outbound_network: true,
  can_update_app: false,
  can_offer_windows_download: true,
  can_edit_settings: true,
  can_upload_media: true,
} as const satisfies Partial<CapabilityMap>

/** Clé d'une capacité de profil de déploiement (booléen de `capabilities`). */
export type DeploymentCapabilityKey = keyof typeof DEPLOYMENT_CAPABILITY_DEFAULTS

/**
 * Lit une capacité de profil dans les `capabilities` du bootstrap. Pure : sert aux
 * composants (via `useDeploymentCapability`) comme aux gardes de route et aux effets qui
 * lisent le store ou la réponse brute.
 */
export function deploymentCapabilityOf(
  capabilities: Partial<CapabilityMap> | null | undefined,
  key: DeploymentCapabilityKey,
): boolean {
  return capabilities?.[key] ?? DEPLOYMENT_CAPABILITY_DEFAULTS[key]
}

/**
 * Comptes locaux avec mot de passe SEULEMENT (ni SSO Xbox, ni instance sans compte) : le cas
 * où une page joueur exige une session avant tout rendu. Avec le SSO Xbox, l'écran de connexion
 * et les gardes du serveur s'en chargent.
 */
export function passwordOnlyAccounts(capabilities: Partial<CapabilityMap> | null | undefined): boolean {
  return (
    deploymentCapabilityOf(capabilities, 'can_use_accounts') &&
    !deploymentCapabilityOf(capabilities, 'can_sign_in_with_xbox')
  )
}
