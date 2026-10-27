/**
 * i18n de la feature Ascension (V2 progression), un fichier par langue (`fr.ts`, `en.ts`) pour un
 * type déclaré une fois (`contract.ts`).
 *
 * Pattern : dictionnaire typé par locale, accessible via getAscensionText(locale).
 * Aligné avec features/{notifications,settings,help,...}/i18n.
 */
import type { Locale } from '@/lib/i18n/locale'
import type { AscensionText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { AscensionText } from './contract'

const TEXT: Record<Locale, AscensionText> = { fr: FR_TEXT, en: EN_TEXT }

export function getAscensionText(locale: Locale): AscensionText {
  return TEXT[locale]
}
