/**
 * Route /admin — layout parent de la section admin (guard isAdmin + onglets +
 * Outlet). Les pages enfants (overview, sync, access, system…) sont montées
 * dans l'Outlet via routes/admin/.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminLayout } from '@/features/admin/AdminLayout'

export const Route = createFileRoute('/admin')({
  component: AdminLayout,
})
