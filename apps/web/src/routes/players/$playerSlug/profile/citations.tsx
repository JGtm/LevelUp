/**
 * Route /players/$playerSlug/profile/citations — redirect legacy.
 * Sprint 55 A4 : redirige vers /players/$playerSlug/career?tab=citations.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/profile/citations')({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: '/players/$playerSlug/career',
      params: { playerSlug: params.playerSlug },
      search: { tab: 'citations' },
      replace: true,
    })
  },
  component: () => null,
})
