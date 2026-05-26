/**
 * Route /players/$playerSlug/objectifs — redirect vers la nouvelle URL
 * /players/$playerSlug/ascension (refonte 2026-05-26, 2 onglets).
 *
 * Conservée comme redirection pour préserver les liens externes
 * (notifications, deep links, anciens bookmarks).
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/objectifs/')({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: '/players/$playerSlug/ascension',
      params,
      replace: true,
    })
  },
})
