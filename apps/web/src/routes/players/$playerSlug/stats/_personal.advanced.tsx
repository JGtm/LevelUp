import { createFileRoute } from '@tanstack/react-router'
import { AdvancedTab } from '@/features/personal-stats/tabs/AdvancedTab'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal/advanced')({
  component: AdvancedTab,
})
