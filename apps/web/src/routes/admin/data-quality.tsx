/**
 * Route /admin/data-quality — compteurs et listes d'inconnus data + actions
 * de résolution (traductions, backfill registry names).
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminDataQualityPage } from '@/features/admin/data-quality/AdminDataQualityPage'

export const Route = createFileRoute('/admin/data-quality')({
  component: AdminDataQualityPage,
})
