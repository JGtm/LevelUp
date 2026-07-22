/**
 * Route /players/$playerSlug/media — page Médias.
 */
import { createFileRoute } from '@tanstack/react-router'
import { MediaPage } from '@/features/media/MediaPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/media')({
  component: () => (
    <RouteCapabilityGate capability="media">
      <MediaPage />
    </RouteCapabilityGate>
  ),
})
