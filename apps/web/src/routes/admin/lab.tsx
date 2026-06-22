/**
 * Route /admin/lab — Lab : explorateur de ressources Waypoint + explorateur
 * d'API, outil opérateur de maintenance et de préparation multi-titre. Gardé par
 * RequireAuth+RequireAdmin (AdminLayout + middleware serveur).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminLabPage } from '@/features/admin/lab/AdminLabPage'

export const Route = createFileRoute('/admin/lab')({
  component: AdminLabPage,
})
