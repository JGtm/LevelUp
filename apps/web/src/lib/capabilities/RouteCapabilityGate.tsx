import type { ReactNode } from 'react'

import { useCapability, type TitleCapability } from './capabilities'
import { FeatureUnavailable } from './FeatureUnavailable'

interface RouteCapabilityGateProps {
  /** Capability produit requise pour monter la page (cf. TITLE_CAPABILITIES). */
  capability: TitleCapability
  children: ReactNode
}

/**
 * RouteCapabilityGate — garde au niveau d'une route : si le titre courant ne
 * déclare pas `capability`, rend {@link FeatureUnavailable} À LA PLACE des
 * `children` (la page n'est donc PAS montée → ses hooks de fetch ne tournent pas,
 * pas de requête inutile). NO-OP pour `halo_infinite` (fail-open + toutes les
 * capabilities) → zéro régression mono-titre.
 *
 * Usage dans une route TanStack :
 *   component: () => (
 *     <RouteCapabilityGate capability="media"><MediaPage /></RouteCapabilityGate>
 *   )
 */
export function RouteCapabilityGate({ capability, children }: RouteCapabilityGateProps) {
  const available = useCapability(capability)
  if (!available) return <FeatureUnavailable capability={capability} />
  return <>{children}</>
}
