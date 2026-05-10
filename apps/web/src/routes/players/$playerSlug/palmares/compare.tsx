/**
 * Route /players/$playerSlug/palmares/compare — redirect legacy.
 * Compare est désormais canoniquement sous /players/$playerSlug/compare.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/palmares/compare')({
  beforeLoad: ({ params, search }) => {
    throw redirect({
      to: '/players/$playerSlug/compare',
      params: { playerSlug: params.playerSlug },
      search,
      replace: true,
    })
  },
  component: () => null,
})
