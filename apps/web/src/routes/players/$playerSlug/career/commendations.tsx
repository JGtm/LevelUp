/**
 * Route /players/$playerSlug/career/commendations — Totaux des commendations natives
 * (Halo 5, section Carrière). État vide pour les titres sans commendations natives
 * → dégradation gracieuse, pas de gating par slug.
 */
import { createFileRoute } from '@tanstack/react-router'
import { UnifiedCitationsPage } from '@/features/citations/UnifiedCitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/career/commendations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <UnifiedCitationsPage source="native" />
    </RouteCapabilityGate>
  ),
})
