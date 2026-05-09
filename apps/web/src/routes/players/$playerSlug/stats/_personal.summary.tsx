import { createFileRoute } from '@tanstack/react-router'
import { SummaryTab } from '@/features/personal-stats/tabs/SummaryTab'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal/summary')({
  component: SummaryTab,
})
