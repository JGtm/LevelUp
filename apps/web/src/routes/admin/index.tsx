/**
 * Route /admin (index) — Vue d'ensemble du dashboard monitoring admin.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminOverviewPage } from '@/features/admin/overview/AdminOverviewPage'

export const Route = createFileRoute('/admin/')({
  component: AdminOverviewPage,
})
