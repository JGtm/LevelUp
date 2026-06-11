/**
 * Route /admin/access — Accès : comptes utilisateurs + codes d'invitation.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminAccessPage } from '@/features/admin/access/AdminAccessPage'

export const Route = createFileRoute('/admin/access')({
  component: AdminAccessPage,
})
