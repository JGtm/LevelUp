import { createFileRoute } from '@tanstack/react-router'

import { SeasonPassPage } from '@/features/palmares/SeasonPassPage'
import { RouteCapabilityGate } from '@/lib/capabilities/RouteCapabilityGate'

export const Route = createFileRoute('/players/$playerSlug/career/season-pass')({
  component: () => (
    <RouteCapabilityGate capability="career">
      <SeasonPassPage />
    </RouteCapabilityGate>
  ),
})
