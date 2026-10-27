/**
 * i18n — Dictionnaire des strings UI de la feature Escouade, un fichier par langue.
 *
 * Frontière stricte avec les mappings TOML multi-titres :
 *  - Les libellés métier (FieldKey, assets, outcomes) restent dans les TOML
 *    et passent par useFieldLabel / useAssetLabel / useOutcomeLabel.
 *  - Ce module ne contient que des strings UI non-titre-bound : titres de
 *    cartes, navigation, empty states, boutons, descriptions, unités.
 *
 * Le type est déclaré une fois (`contract.ts`) ; chaque langue vit dans son fichier
 * (`fr.ts`, `en.ts`) et satisfait ce type. Ajouter une langue : un fichier, une ligne ici.
 *
 * Pattern aligné avec features/compare/i18n.ts et features/home/*.i18n.ts.
 */

import type { Locale } from '@/lib/i18n/locale'
import type { SquadText } from './contract'
import { EN_TEXT } from './en'
import { FR_TEXT } from './fr'

export type { SquadText } from './contract'

const DICTS: Record<Locale, SquadText> = {
  fr: FR_TEXT,
  en: EN_TEXT,
}

/** Retourne le dictionnaire pour la locale demandée. */
export function getSquadText(locale: Locale): SquadText {
  return DICTS[locale]
}

// Exports nommés pour permettre des tests de parité FR/EN sans réimporter le helper.
export { FR_TEXT, EN_TEXT }
