/**
 * Route /admin/convergence — backlog d'enrichissement par joueur + compteurs
 * post-sync du dernier cycle.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminConvergencePage } from '@/features/admin/convergence/AdminConvergencePage'

export const Route = createFileRoute('/admin/convergence')({
  component: AdminConvergencePage,
})
