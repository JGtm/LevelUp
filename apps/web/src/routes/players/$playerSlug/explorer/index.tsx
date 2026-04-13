/**
 * Route /players/$playerSlug/explorer — page Explorer.
 */
import { createFileRoute } from '@tanstack/react-router'
import { ExplorerPage } from '@/features/explorer/ExplorerPage'

export const Route = createFileRoute('/players/$playerSlug/explorer/')({
  component: ExplorerPage,
})
