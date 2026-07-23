/**
 * Route /players/$playerSlug/career/citations — page Citations (section Carrière).
 */
import { createFileRoute } from '@tanstack/react-router'
import { UnifiedCitationsPage } from '@/features/citations/UnifiedCitationsPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/career/citations')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <UnifiedCitationsPage source="infinite" />
    </RouteCapabilityGate>
  ),
})
