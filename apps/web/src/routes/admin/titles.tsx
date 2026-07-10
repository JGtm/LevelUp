/**
 * Route /admin/titles — REDIRECTION (A3, DC-8) : le registre des titres (et
 * son diagnostic) est une section de l'onglet Gestion.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/titles')({
  beforeLoad: () => {
    throw redirect({ to: '/admin/management', replace: true })
  },
})
