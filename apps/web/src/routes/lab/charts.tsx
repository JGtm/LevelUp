import { createFileRoute } from '@tanstack/react-router'

import { ChartsShowcasePage } from '@/features/lab/ChartsShowcasePage'

export const Route = createFileRoute('/lab/charts')({
  component: ChartsShowcasePage,
})
