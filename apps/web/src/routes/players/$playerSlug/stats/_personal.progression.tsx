import { createFileRoute } from '@tanstack/react-router'
import { ProgressionTab } from '@/features/personal-stats/tabs/ProgressionTab'

export const Route = createFileRoute('/players/$playerSlug/stats/_personal/progression')({
  component: ProgressionTab,
})
