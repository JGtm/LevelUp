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
 * PORTÉE : capabilities TITLE-LEVEL uniquement (« cette page existe-t-elle pour ce
 * titre ? »). Les capabilities DATA-LEVEL (« ce titre produit-il cette donnée ? »,
 * clés pointées `film.*`) se lisent avec {@link useDataCapability} au point d'usage.
 * Une prop `dataCapability` a existé ici du 2026-09-05 au 2026-09-06 : aucune surface
 * ne l'a jamais utilisée — les trois du lot appellent le hook en direct, passent par
 * `usageAvailability` ou par un rendu conditionnel — et la revue adversariale l'a
 * relevée comme code mort (règle CLAUDE.md n°7). Elle est retirée plutôt que dotée
 * d'un appelant de circonstance : un composant qui a déjà des retours conditionnels
 * lit plus clair avec `if (!useDataCapability(k)) return null` qu'enveloppé dans un
 * gate. Si une surface a un jour besoin des DEUX portes, elle les cumule sur place.
 *
 * Cf. `lib/capabilities/capabilities.ts` (fail-open pendant le bootstrap).
 */
export function FeatureGate({ capability, children, fallback = null }: FeatureGateProps) {
  const available = useCapability(capability)
  return <>{available ? children : fallback}</>
}
