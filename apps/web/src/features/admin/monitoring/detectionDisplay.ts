/**
 * detectionDisplay — mapping pur (statut de détection ↔ libellé/token,
 * niveau de log ↔ token). Testable sans React (A2.6). Aucune couleur en dur :
 * uniquement des tokens sémantiques (résolus en CSS var par le composant).
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'

export interface DetectionStatusMeta {
  labelKey: AdminManifestKey
  /** Token de couleur du badge ; absent = neutre (sourdine). */
  token?: SemanticToken
}

const STATUS_META: Record<string, DetectionStatusMeta> = {
  open: { labelKey: 'admin.detections.status_open', token: 'warning' },
  acked: { labelKey: 'admin.detections.status_acked', token: 'info' },
  muted: { labelKey: 'admin.detections.status_muted' },
  resolved: { labelKey: 'admin.detections.status_resolved', token: 'success' },
}

/** Métadonnées d'affichage d'un statut ; défaut open pour toute valeur inconnue. */
export function detectionStatusMeta(status: string): DetectionStatusMeta {
  return STATUS_META[status] ?? STATUS_META.open
}

/** Token de badge d'un niveau de log (ERROR/WARN/…). */
export function detectionLevelToken(level: string): SemanticToken {
  switch (level.toUpperCase()) {
    case 'ERROR':
      return 'destructive'
    case 'WARN':
    case 'WARNING':
      return 'warning'
    default:
      return 'info'
  }
}

/** Statut minimal d'une détection pour le filtrage client-side de la table. */
interface HasStatus {
  status: string
}

/** Filtre la liste par statut ('all' = pas de filtre). Fonction pure (testable). */
export function filterDetectionsByStatus<T extends HasStatus>(rows: T[], filter: string): T[] {
  if (filter === 'all') return rows
  return rows.filter((r) => r.status === filter)
}
