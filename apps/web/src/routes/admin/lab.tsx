/**
 * Route /admin/lab — REDIRECTION (A3, DC-9) : le Lab est retiré de l'app (son
 * workflow « préparer un nouveau titre » est servi par les CLI + le runbook
 * docs/RUNBOOK_ADD_TITLE.md). Sa valeur opérationnelle résiduelle (diagnostic
 * par titre) vit dans Gestion → Titres.
 */
import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/lab')({
  beforeLoad: () => {
    throw redirect({ to: '/admin/management', replace: true })
  },
})
