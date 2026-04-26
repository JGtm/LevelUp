/**
 * Route /players/$playerSlug/notifications — page dédiée des notifications.
 */
import { createFileRoute } from '@tanstack/react-router'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'

export const Route = createFileRoute('/players/$playerSlug/notifications')({
  component: NotificationsPage,
})
