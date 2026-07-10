/**
 * Route /admin/detections — Détections : triage des anomalies persistées avec
 * cycle de vie (open/acked/muted/resolved). Remplace l'onglet Logs pour le
 * triage (DC-8) ; le viewer de logs bruts vit dans Système.
 */
import { createFileRoute } from '@tanstack/react-router'
import { AdminDetectionsPage } from '@/features/admin/detections/AdminDetectionsPage'

export const Route = createFileRoute('/admin/detections')({
  component: AdminDetectionsPage,
})
