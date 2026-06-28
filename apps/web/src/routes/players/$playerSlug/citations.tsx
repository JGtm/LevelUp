/**
 * Route /players/$playerSlug/citations — page Citations dédiée.
 */
import { createFileRoute } from '@tanstack/react-router'
import { UnifiedCitationsPage } from '@/features/citations/UnifiedCitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/citations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <UnifiedCitationsPage source="infinite" />
    </RouteCapabilityGate>
  ),
})
