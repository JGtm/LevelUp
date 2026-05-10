import { createFileRoute } from '@tanstack/react-router'

import { SeasonPassPage } from '@/features/palmares/SeasonPassPage'

export const Route = createFileRoute('/players/$playerSlug/career/season-pass')({
  component: SeasonPassPage,
})
