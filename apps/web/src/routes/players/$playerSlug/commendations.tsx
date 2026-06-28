/**
 * Route /players/$playerSlug/commendations — page Totaux des commendations natives
 * (Halo 5, AXE B). Réponse vide (état vide) pour les titres sans commendations
 * natives → dégradation gracieuse, pas de gating par slug.
 */
import { createFileRoute } from '@tanstack/react-router'
import { UnifiedCitationsPage } from '@/features/citations/UnifiedCitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/commendations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <UnifiedCitationsPage source="native" />
    </RouteCapabilityGate>
  ),
})
