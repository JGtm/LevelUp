import type { ReactNode } from 'react'

import { useCapability, type TitleCapability } from './capabilities'

interface FeatureGateProps {
  /** Capability produit requise pour afficher `children` (cf. TITLE_CAPABILITIES). */
  capability: TitleCapability
  children: ReactNode
  /** Rendu alternatif si la capability est absente du titre courant (défaut : rien). */
  fallback?: ReactNode
}

/**
 * FeatureGate masque ses enfants si le titre courant ne déclare pas
 * `capability`. NO-OP pour `halo_infinite` (déclare toutes les capabilities) ;
 * prépare le multi-titre — un titre sans `firefight` ne montre pas la page PvE,
 * un titre sans `ranked` masque les sections classées, etc. Évite les
 * graphes / onglets / pages morts ou vides.
 *
 * Cf. `lib/capabilities/capabilities.ts` (fail-open pendant le bootstrap).
 */
export function FeatureGate({ capability, children, fallback = null }: FeatureGateProps) {
  const available = useCapability(capability)
  return <>{available ? children : fallback}</>
}
