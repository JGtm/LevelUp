// Auto-genere par scripts/build_i18n_manifests.mjs - NE PAS EDITER A LA MAIN.
// Source : config/locales.toml (ADR 0040)

export const KNOWN_LOCALES = ['fr', 'en'] as const

export type KnownLocale = (typeof KNOWN_LOCALES)[number]

export const DEFAULT_LOCALE: KnownLocale = 'fr'

export const FALLBACK_LOCALE: KnownLocale = 'en'

export const INTL_TAG: Record<KnownLocale, string> = {
  fr: 'fr-FR',
  en: 'en-GB',
}
