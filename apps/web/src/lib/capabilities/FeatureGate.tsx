import type { ReactNode } from 'react'

import { hasCapabilityIn, useTitleCapabilities, type TitleCapability } from './capabilities'
import {
  hasDataCapabilityIn,
  useTitleDataCapabilities,
  type DataCapabilityKey,
} from './dataCapabilities'

interface FeatureGateProps {
  /**
   * Capability produit TITLE-LEVEL requise (cf. TITLE_CAPABILITIES) : « cette page /
   * cette section existe-t-elle pour ce titre ? ». Servie par le bootstrap.
   */
  capability?: TitleCapability
  /**
   * Capability DATA-LEVEL requise (cf. DATA_CAPABILITIES) : « ce titre PRODUIT-il cette
   * donnée ? ». Servie par `GET /titles/{slug}/capabilities`.
   */
  dataCapability?: DataCapabilityKey
  children: ReactNode
  /** Rendu alternatif si une porte est fermée (défaut : rien). */
  fallback?: ReactNode
}

/**
 * FeatureGate masque ses enfants si le titre courant ne franchit pas la (ou les) porte(s)
 * demandée(s) — évite les graphes / onglets / pages morts ou vides.
 *
 * UN SEUL composant pour les DEUX systèmes de capabilities (title-level et data-level),
 * cumulables : passer les deux exige les deux. Deux gates séparés auraient divergé sur le
 * fail-open, et l'appelant aurait dû savoir de quel système relève sa clé — alors que la
 * question posée à l'écran est toujours la même : « affiche-t-on ce bloc ? ».
 *
 * Sans aucune des deux props, le composant rend ses enfants : un gate qui ne garde rien
 * ne masque rien (une prop oubliée ne doit pas faire disparaître une section en silence).
 *
 * POURQUOI DEUX SOUS-COMPOSANTS. La porte data-level ouvre une requête
 * (`GET /titles/{slug}/capabilities`), donc un hook TanStack Query, donc l'exigence d'un
 * QueryClientProvider. Les dizaines de `FeatureGate` title-level de l'application n'ont
 * aucune raison de la payer — ni de la faire exiger par leurs tests. La branche est prise
 * sur la PRÉSENCE de la prop, jamais sur sa valeur : `dataCapability` est une constante
 * d'appel à chaque site, il n'y a donc pas de bascule de branche au fil des rendus.
 *
 * Fail-open pendant le chargement dans les deux systèmes — cf. `capabilities.ts` et
 * `dataCapabilities.ts`.
 */
export function FeatureGate({
  capability,
  dataCapability,
  children,
  fallback = null,
}: FeatureGateProps) {
  if (dataCapability == null) {
    return (
      <PorteDeTitre capability={capability} fallback={fallback}>
        {children}
      </PorteDeTitre>
    )
  }
  return (
    <PorteDeTitreEtDeDonnee
      capability={capability}
      dataCapability={dataCapability}
      fallback={fallback}
    >
      {children}
    </PorteDeTitreEtDeDonnee>
  )
}

/** Porte title-level seule : aucune requête, aucun provider requis. */
function PorteDeTitre({
  capability,
  children,
  fallback,
}: {
  capability?: TitleCapability
  children: ReactNode
  fallback: ReactNode
}) {
  const titleCaps = useTitleCapabilities()
  const available = capability == null || hasCapabilityIn(titleCaps, capability)
  return <>{available ? children : fallback}</>
}

/** Les deux portes : il faut les DEUX pour rendre les enfants. */
function PorteDeTitreEtDeDonnee({
  capability,
  dataCapability,
  children,
  fallback,
}: {
  capability?: TitleCapability
  dataCapability: DataCapabilityKey
  children: ReactNode
  fallback: ReactNode
}) {
  const titleCaps = useTitleCapabilities()
  const dataCaps = useTitleDataCapabilities()
  const available =
    (capability == null || hasCapabilityIn(titleCaps, capability)) &&
    hasDataCapabilityIn(dataCaps, dataCapability)
  return <>{available ? children : fallback}</>
}
