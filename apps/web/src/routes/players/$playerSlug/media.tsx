/**
 * Route /players/$playerSlug/media — page Médias.
 */
import { createFileRoute } from '@tanstack/react-router'
import { MediaPage } from '@/features/media/MediaPage'

export const Route = createFileRoute('/players/$playerSlug/media')({
  component: MediaPage,
})
