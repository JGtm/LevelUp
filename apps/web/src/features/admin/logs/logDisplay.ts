/**
 * logDisplay — logique pure du viewer de logs (testable sans React) :
 * mapping niveau → statut de badge, aplatissement des attrs pour l'affichage
 * compact, sérialisation du détail.
 */
import type { AdminLogEntry } from '@/lib/api/types'
import type { AdminStatus } from '../statusDisplay'

/** Niveau slog → statut StatusBadge (info/debug neutres, warn/error colorés). */
export function logLevelStatus(level: string): AdminStatus {
  switch (level.toLowerCase()) {
    case 'error':
      return 'error'
    case 'warn':
    case 'warning':
      return 'warning'
    default:
      return 'idle'
  }
}

/**
 * Aplati les attrs restants en chips "clé=valeur" compactes (ordre stable,
 * valeurs objets sérialisées, tronquées à maxLen).
 */
export function flattenLogFields(fields: Record<string, unknown> | undefined, maxLen = 48): string[] {
  if (!fields) return []
  const out: string[] = []
  for (const key of Object.keys(fields).sort()) {
    const value = fields[key]
    let text: string
    if (value === null || value === undefined) {
      text = 'null'
    } else if (typeof value === 'object') {
      try {
        text = JSON.stringify(value)
      } catch {
        text = String(value)
      }
    } else {
      text = String(value)
    }
    if (text.length > maxLen) {
      text = `${text.slice(0, maxLen - 1)}…`
    }
    out.push(`${key}=${text}`)
  }
  return out
}

/** Détail complet d'une entrée pour le panneau d'expansion (<pre>). */
export function logEntryDetail(entry: AdminLogEntry): string {
  if (entry.raw) return entry.raw
  try {
    return JSON.stringify(entry, null, 2)
  } catch {
    return entry.msg ?? ''
  }
}

/** Texte principal d'une ligne (msg, sinon raw, sinon err). */
export function logEntryText(entry: AdminLogEntry): string {
  return entry.msg || entry.raw || entry.err || ''
}
