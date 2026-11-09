/**
 * i18n de la feature match-replay (rejeu 2D vue du dessus). Strings UI FR + EN,
 * parité par typage Record<ReplayLocale, ReplayText>.
 *
 * LE CONTRAT (un champ par string, et sa justification) VIT DANS `i18nContract.ts` ; chaque
 * langue vit dans son fichier (`fr.ts`, `en.ts`) et le satisfait ; ce fichier-ci ne porte que
 * la table qui les assemble. Ajouter une langue : un fichier, une ligne ici.
 */
import type { Locale } from '@/lib/i18n/locale'

import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'
import type { ReplayText } from './i18nContract'

/** Alias local du type central : la feature ne redéclare pas l'union des langues. */
export type ReplayLocale = Locale

export const REPLAY_TEXT: Record<ReplayLocale, ReplayText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}
