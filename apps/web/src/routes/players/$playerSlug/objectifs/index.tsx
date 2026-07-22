/**
 * Route /players/$playerSlug/objectifs — redirect vers la nouvelle URL
 * /players/$playerSlug/ascension/objectifs (onglet Objectifs, refonte 4
 * onglets 2026-07, AM-1).
 *
 * Conservée comme redirection pour préserver les liens externes
 * (notifications, deep links, anciens bookmarks).
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/players/$playerSlug/objectifs/')({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: '/players/$playerSlug/ascension/objectifs',
      params,
      replace: true,
    })
  },
})
