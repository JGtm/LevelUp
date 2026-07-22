/**
 * Route /players/$playerSlug/notifications — page dédiée des notifications.
 */
import { createFileRoute } from '@tanstack/react-router'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'

export const Route = createFileRoute('/{-$lang}/t/$titleSlug/players/$playerSlug/notifications')({
  component: NotificationsPage,
})
