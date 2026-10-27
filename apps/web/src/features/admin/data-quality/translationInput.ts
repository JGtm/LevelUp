/**
 * translationInput — saisie d'une traduction de l'administration : une entrée par langue de
 * `config/locales.toml` (`KNOWN_LOCALES`), envoyée au serveur en `names: { langue: texte }`.
 * Fonctions pures (le formulaire les compose, les tests les couvrent sans rendu).
 */
import { KNOWN_LOCALES, type Locale } from '@/lib/i18n/locale'

/** Un champ du formulaire de résolution (forme de `InlineResolveForm`). */
export interface TranslationField {
  key: Locale
  label: string
  initial: string
  placeholder?: string
}

/** Une saisie par langue de la liste, dans l'ordre de `config/locales.toml`. */
export function translationFields(
  labelOf: (locale: Locale) => string,
  placeholder?: string,
): TranslationField[] {
  return KNOWN_LOCALES.map((locale) => ({
    key: locale,
    label: labelOf(locale),
    initial: '',
    placeholder,
  }))
}

/** Valeurs saisies (par clé de champ) -> `names` du corps de requête : noms rognés, vides écartés. */
export function namesFromValues(values: Record<string, string>): Record<string, string> {
  const names: Record<string, string> = {}
  for (const locale of KNOWN_LOCALES) {
    const name = values[locale]?.trim()
    if (name) names[locale] = name
  }
  return names
}
