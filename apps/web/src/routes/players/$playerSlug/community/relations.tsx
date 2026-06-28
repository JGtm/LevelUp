import { createFileRoute } from '@tanstack/react-router'

import { PalmaresRelationsPage } from '@/features/palmares/PalmaresRelationsPage'

export const Route = createFileRoute('/players/$playerSlug/community/relations')({
  component: PalmaresRelationsPage,
})
