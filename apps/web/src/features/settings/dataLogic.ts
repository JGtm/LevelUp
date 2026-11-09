/**
 * Logique pure de la carte « Données » (lot L22) : bornes des réglages, rattachement d'un code de
 * refus du serveur au champ qui l'a causé, lecture des saisies. Aucune dépendance React.
 */
import { formatBytes } from '@/lib/formatters/bytes'
import { formatDateTime, intlLocale } from '@/lib/formatters'
import type { Locale } from '@/lib/i18n/locale'
import { fillTemplate } from '@/lib/i18n/template'
import type { DataBackupSummary, DataStatus } from './dataQueries'
import type { DataText } from './dataText'

/** Fréquences de sauvegarde offertes ; la première est la valeur par défaut du serveur. */
export const BACKUP_INTERVALS = ['24h', '1h', '6h', '12h', '72h', '168h'] as const
export type BackupInterval = (typeof BACKUP_INTERVALS)[number]
export const DEFAULT_BACKUP_INTERVAL: BackupInterval = '24h'

/** Nombre de jeux de sauvegarde conservés : bornes et défaut du serveur. */
export const BACKUP_KEEP_MIN = 1
export const BACKUP_KEEP_MAX = 60
export const DEFAULT_BACKUP_KEEP = 7

/** Plafond mémoire de chaque base, en Mo : 0 = valeur par défaut du serveur, sinon cette plage. */
export const MEMORY_LIMIT_MIN_MB = 1024
export const MEMORY_LIMIT_MAX_MB = 32768

/** Cadence de relecture de l'état des données, en ms : rapide pendant une sauvegarde, lente sinon. */
export const DATA_STATUS_FAST_MS = 5_000
export const DATA_STATUS_SLOW_MS = 30_000

/** Réglage de la carte auquel un refus se rattache ; `general` = hors champ (action, réseau). */
export type DataField = 'backup_dir' | 'backup_interval' | 'backup_keep' | 'memory_limit_mb' | 'general'

const FIELD_OF_CODE: Record<string, DataField> = {
  backup_dir_not_absolute: 'backup_dir',
  backup_dir_missing: 'backup_dir',
  backup_dir_not_writable: 'backup_dir',
  backup_dir_inside_data: 'backup_dir',
  backup_interval_invalid: 'backup_interval',
  backup_keep_invalid: 'backup_keep',
  memory_limit_invalid: 'memory_limit_mb',
}

/** Un réglage de la carte (un champ), par opposition au refus « général » (action, réseau). */
export type DataSettingKey = Exclude<DataField, 'general'>

/** Le champ visé par un code de refus du serveur, ou `undefined` pour un code qui n'en désigne aucun. */
export function fieldOfCode(code: string | undefined): DataField | undefined {
  return code === undefined ? undefined : FIELD_OF_CODE[code]
}

/** Le texte d'un code de refus du serveur, écrit ici dans chaque langue (jamais le message du serveur). */
export function errorText(t: DataText, code: string | undefined): string | undefined {
  return code === undefined ? undefined : t.errors[code]
}

/** Le texte d'un échec de passage de sauvegarde (`last_attempt.error_code`). */
export function attemptFailureText(t: DataText, code: string | undefined): string {
  return (code !== undefined ? t.attemptErrors[code] : undefined) ?? t.attemptErrors.backup_failed
}

/** Une saisie d'entier ; `null` quand elle est vide ou n'est pas un entier (le serveur arbitre le reste). */
export function parseWholeNumber(raw: string): number | null {
  const trimmed = raw.trim()
  return /^-?\d+$/.test(trimmed) ? Number(trimmed) : null
}

/** Une fréquence valide du serveur : un nombre entier d'heures, « 48h » par exemple. */
const INTERVAL_PATTERN = /^\d+h$/

/**
 * La fréquence réglée telle que le serveur la tient. Seule une valeur vide ou illisible retombe sur le défaut (24 h,
 * comme le serveur) : une valeur valide que la liste n'offre pas (« 48h », posée à la main) reste affichée telle quelle,
 * jamais déguisée en « Tous les jours ».
 */
export function intervalSetting(value: string | undefined): string {
  const v = (value ?? '').trim()
  return INTERVAL_PATTERN.test(v) ? v : DEFAULT_BACKUP_INTERVAL
}

/** Les fréquences de la liste : celles qu'on offre, plus la valeur réglée si elle n'en fait pas partie. */
export function intervalOptions(current: string): string[] {
  return (BACKUP_INTERVALS as readonly string[]).includes(current) ? [...BACKUP_INTERVALS] : [...BACKUP_INTERVALS, current]
}

/** Le libellé d'une fréquence : celui de la liste, sinon « Toutes les N h » avec la valeur réelle. */
export function intervalLabel(t: DataText, value: string): string {
  const known = (t.intervals as Record<string, string>)[value]
  return known ?? fillTemplate(t.intervalEvery, { hours: value.replace(/h$/, '') })
}

/** La ligne de résumé du dernier jeu de sauvegarde : date, taille, nombre de bases, durée. */
export function lastBackupSummary(t: DataText, locale: Locale, last: DataBackupSummary): string {
  const databases = `${last.databases} ${last.databases === 1 ? t.databasesOne : t.databasesMany}`
  return fillTemplate(t.lastSummary, {
    date: formatDateTime(last.created_at, intlLocale(locale)),
    size: formatBytes(last.bytes, locale),
    count: databases,
    duration: `${Math.max(1, Math.round(last.duration_ms / 1000))} s`,
  })
}

/** Cadence de relecture de l'état des données : rapide tant qu'une sauvegarde tourne, lente sinon. */
export function dataStatusRefetchMs(status: DataStatus | undefined): number {
  return status?.backup?.running ? DATA_STATUS_FAST_MS : DATA_STATUS_SLOW_MS
}
