/**
 * i18n — Dictionnaire des strings UI de la feature Paramètres, un fichier par langue (`fr.ts`,
 * `en.ts`) pour un type déclaré une fois (`contract.ts`). Ajouter une langue : un fichier, une
 * ligne ici.
 */
import type { Locale } from '@/lib/i18n/locale'
import type { SettingsText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { HintBullets, SettingsText } from './contract'

const TEXT: Record<Locale, SettingsText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

export function getSettingsText(locale: Locale): SettingsText {
  return TEXT[locale]
}
