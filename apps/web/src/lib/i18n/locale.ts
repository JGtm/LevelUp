/**
 * Locale — type central de langue de l'application (source canonique UNIQUE).
 *
 * La liste des langues vient de `config/locales.toml` (ADR 0040) par le module généré
 * `generated/locales.ts` : ce module n'importe que lui, jamais une feature — tout ce qui a besoin
 * du type de locale ou de la liste runtime des locales dépend d'ici, jamais l'inverse, ce qui
 * garantit l'absence de cycle d'imports. Les consommateurs historiques s'y branchent : l'alias
 * `ManifestLocale` (lib/i18n/format) ré-exporte ce type, le parsing du segment de langue de
 * l'URL (title-routing) et les headers API (lib/api/client, stores/appShellStore) l'utilisent
 * directement.
 *
 * `KNOWN_LOCALES` est la liste RUNTIME (le type `Locale` en est dérivé) dont le parsing de
 * segment a besoin ; `isKnownLocale` en est le type guard ; `normalizeLocale` ramène toute
 * valeur (étiquette de navigateur, valeur de session, chaîne libre) sur une langue de la liste.
 */
import { DEFAULT_LOCALE, FALLBACK_LOCALE, KNOWN_LOCALES, type KnownLocale } from './generated/locales'

export { DEFAULT_LOCALE, FALLBACK_LOCALE, KNOWN_LOCALES }

export type Locale = KnownLocale

/** Type guard : le segment est-il une locale de `config/locales.toml` ? */
export function isKnownLocale(segment: string): segment is Locale {
  return (KNOWN_LOCALES as readonly string[]).includes(segment)
}

/**
 * Ramène une valeur quelconque sur une locale connue : étiquette BCP-47 (`en-US`, `fr_FR`),
 * casse et espaces tolérés, sous-étiquette primaire retenue. Vide, absente ou inconnue →
 * `DEFAULT_LOCALE` : jamais de repli binaire `en`-ou-`fr`, qui servirait une 3e langue dans
 * la mauvaise.
 */
export function normalizeLocale(value: string | null | undefined): Locale {
  const primary = (value ?? '').trim().toLowerCase().split(/[-_,;\s]/)[0]
  return isKnownLocale(primary) ? primary : DEFAULT_LOCALE
}
