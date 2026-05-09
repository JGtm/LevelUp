import { createFileRoute } from '@tanstack/react-router'
import { DistributionsTab } from '@/features/personal-stats/tabs/DistributionsTab'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal/distributions')({
  component: DistributionsTab,
})
