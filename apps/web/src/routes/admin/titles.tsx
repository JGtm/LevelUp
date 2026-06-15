/**
 * Route /admin/titles — Titres : gestion multi-titres (registre, Status
 * lifecycle, capabilities + feature-matrix). PMT-14 volet A.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminTitlesPage } from '@/features/admin/titles/AdminTitlesPage'

export const Route = createFileRoute('/admin/titles')({
  component: AdminTitlesPage,
})
