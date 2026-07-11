/**
 * Route /admin/data — Données : intégrité du warehouse (DC-8). Fusion des
 * ex-onglets Qualité données + Convergence + section Invariants.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminDataPage } from '@/features/admin/data/AdminDataPage'

export const Route = createFileRoute('/admin/data')({
  component: AdminDataPage,
})
