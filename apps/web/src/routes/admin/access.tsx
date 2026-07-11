/**
 * Route /admin/access — REDIRECTION (A3, DC-8) : les comptes utilisateurs
 * sont une section de l'onglet Gestion.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/access')({
  beforeLoad: () => {
    throw redirect({ to: '/admin/management', replace: true })
  },
})
