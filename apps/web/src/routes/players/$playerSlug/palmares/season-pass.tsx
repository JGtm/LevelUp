/**
 * Route /players/$playerSlug/palmares/season-pass — redirect legacy.
 * Pass saisonnier appartient désormais à la section Carrière.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/palmares/season-pass')({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: '/players/$playerSlug/career/season-pass',
      params: { playerSlug: params.playerSlug },
      replace: true,
    })
  },
  component: () => null,
})
