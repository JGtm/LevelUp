/**
 * Route /admin/system — Système : contention DB (B-swap), santé des tokens
 * auth, intégrité des données (invariants).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminSystemPage } from '@/features/admin/system/AdminSystemPage'

export const Route = createFileRoute('/admin/system')({
  component: AdminSystemPage,
})
