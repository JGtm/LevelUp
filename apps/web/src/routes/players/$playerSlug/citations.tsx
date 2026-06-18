/**
 * Route /players/$playerSlug/citations — page Citations dédiée.
 */
import { createFileRoute } from '@tanstack/react-router'
import { CitationsPage } from '@/features/citations/CitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/citations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <CitationsPage />
    </RouteCapabilityGate>
  ),
})
