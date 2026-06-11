/**
 * Route /admin/sync — Sync & Jobs : scheduler auto-sync, historique des
 * cycles, jobs asynchrones récents.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminSyncPage } from '@/features/admin/sync/AdminSyncPage'

export const Route = createFileRoute('/admin/sync')({
  component: AdminSyncPage,
})
