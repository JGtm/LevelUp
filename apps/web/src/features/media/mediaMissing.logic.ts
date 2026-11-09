/**
 * Logique pure de la fenêtre « Fichiers introuvables » et de son bandeau : formats de date,
 * écart de capture, retrouvailles de la sélection, nature du bandeau du dossier.
 */
import type { Locale } from '@/lib/i18n/locale'
import { intlLocale } from '@/lib/formatters'
import type { MediaMissingItem } from '@/lib/api/types'
import type { MissingMediaText } from './i18n-modals'

const EMPTY_DATE = '—'

/** Date et heure locales d'une capture ; tiret quand l'instant est absent ou illisible. */
export function formatCaptureTime(iso: string | null | undefined, locale: Locale): string {
  if (!iso) return EMPTY_DATE
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return EMPTY_DATE
  return date.toLocaleString(intlLocale(locale), {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Écart de capture avec la fiche introuvable ; vide quand le serveur ne l'a pas calculé. */
export function formatCaptureDelta(
  deltaSeconds: number | null | undefined,
  text: Pick<MissingMediaText, 'deltaFormat'>,
): string {
  if (deltaSeconds == null) return ''
  return text.deltaFormat(Math.round(Math.abs(deltaSeconds) / 60))
}

/** Média introuvable désigné par `filePath`, ou `null` quand il a quitté la liste (fusion, oubli). */
export function findMissingItem(
  items: readonly MediaMissingItem[],
  filePath: string | null,
): MediaMissingItem | null {
  if (filePath === null) return null
  return items.find((item) => item.file_path === filePath) ?? null
}

/** Nature du bandeau au-dessus de la grille : un seul à la fois, le dossier inaccessible d'abord. */
export type FolderNoticeKind = 'root-unavailable' | 'bulk-pending'

export function folderNoticeKind(state: {
  rootUnavailable: boolean
  bulkPending: number
}): FolderNoticeKind | null {
  if (state.rootUnavailable) return 'root-unavailable'
  if (state.bulkPending > 0) return 'bulk-pending'
  return null
}
