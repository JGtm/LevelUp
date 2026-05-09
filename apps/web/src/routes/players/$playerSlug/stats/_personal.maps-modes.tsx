import { createFileRoute } from '@tanstack/react-router'
import { MapsModesTab } from '@/features/personal-stats/tabs/MapsModesTab'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal/maps-modes')({
  component: MapsModesTab,
})
