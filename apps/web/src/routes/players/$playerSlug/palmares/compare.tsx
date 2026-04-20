import { createFileRoute } from '@tanstack/react-router'

import { PalmaresComparePage } from '@/features/palmares/PalmaresComparePage'

export const Route = createFileRoute('/players/$playerSlug/palmares/compare')({
  component: PalmaresComparePage,
})
