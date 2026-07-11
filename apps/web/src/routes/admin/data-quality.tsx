/**
 * Route /admin/data-quality — REDIRECTION (A3, DC-8) : la qualité des données
 * est une section de l'onglet Données.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/data-quality')({
  beforeLoad: () => {
    throw redirect({ to: '/admin/data', replace: true })
  },
})
