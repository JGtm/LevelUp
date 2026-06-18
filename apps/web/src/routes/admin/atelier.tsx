/**
 * Route /admin/atelier — Atelier : explorateur de ressources Waypoint (ex-Lab),
 * outil opérateur de maintenance et de préparation multi-titre. Gardé par
 * RequireAuth+RequireAdmin (AdminLayout + middleware serveur).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminAtelierPage } from '@/features/admin/atelier/AdminAtelierPage'

export const Route = createFileRoute('/admin/atelier')({
  component: AdminAtelierPage,
})
