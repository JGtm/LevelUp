/**
 * i18n du système de notifications, un fichier par langue (`fr.ts`, `en.ts`) pour un type
 * déclaré une fois (`contract.ts`).
 *
 * Pattern : dictionnaire typé par locale, accessible via getNotificationsText(locale).
 * Aligné avec features/{settings,help,...}/i18n.
 */
import type { Locale } from '@/lib/i18n/locale'
import type { NotificationsText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { NotificationsText } from './contract'

const DICTS: Record<Locale, NotificationsText> = { fr: FR_TEXT, en: EN_TEXT }

export function getNotificationsText(locale: Locale): NotificationsText {
  return DICTS[locale] ?? FR_TEXT
}
