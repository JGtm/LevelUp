/**
 * Route /admin/convergence — REDIRECTION (A3, DC-8) : la convergence est une
 * section de l'onglet Données.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/convergence')({
  beforeLoad: () => {
    throw redirect({ to: '/admin/data', replace: true })
  },
})
