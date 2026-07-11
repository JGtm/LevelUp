/**
 * Route /admin/management — Gestion : administration (DC-8). Regroupe les
 * ex-onglets Accès (comptes utilisateurs) et Titres (registre multi-titres +
 * diagnostic par titre).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminManagementPage } from '@/features/admin/management/AdminManagementPage'

export const Route = createFileRoute('/admin/management')({
  component: AdminManagementPage,
})
