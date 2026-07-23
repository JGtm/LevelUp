/**
 * Wrapper runtime des manifests i18n typés (lib/i18n/generated/*).
 *
 * Resout une cle via ICU MessageFormat (pluralisation, select, interpolation
 * typee). Conformement au PLAN_META_FOUNDATIONS_GO § 3.4.2.
 *
 * Usage cote composant :
 *
 *   import { commonManifest } from '@/lib/i18n/generated/common'
 *   import { formatMessage } from '@/lib/i18n/format'
 *
 *   formatMessage(commonManifest, 'common.period.last_1y', 'fr')
 *   // -> "Dernière année"
 *
 *   formatMessage(commonManifest, 'common.kpi.matches_count', 'fr', { n: 5 })
 *   // -> "5 matchs"
 *
 *   formatMessage(commonManifest, 'common.kpi.matches_count', 'fr', { n: 1 })
 *   // -> "1 match"
 */
import { IntlMessageFormat } from 'intl-messageformat'
import type { Locale } from '@/lib/i18n/locale'

/** Locales supportees par les manifests. Alias du type central `Locale` (lib/i18n/locale). */
export type ManifestLocale = Locale

/**
 * Forme generique d'un manifest genere : map<key, {fr, en}>.
 * On utilise `Record<string, ...>` ici car les modules generes sont `as const`
 * avec des cles literales ; les helpers consommateurs doivent re-typer
 * leur manifest pour beneficier de l'autocompletion stricte.
 */
export type ManifestEntries = Readonly<Record<string, Readonly<Record<ManifestLocale, string>>>>

// Cache process des MessageFormat compiles : clef = "<locale>::<message>".
// Compiler une string ICU est non-trivial ; on memoize pour eviter de payer
// le cout a chaque rendu.
const formatterCache = new Map<string, IntlMessageFormat>()

function getFormatter(locale: ManifestLocale, message: string): IntlMessageFormat {
  const cacheKey = `${locale}::${message}`
  let fmt = formatterCache.get(cacheKey)
  if (!fmt) {
    fmt = new IntlMessageFormat(message, locale)
    formatterCache.set(cacheKey, fmt)
  }
  return fmt
}

/**
 * formatMessage resout une cle d'un manifest et applique l'interpolation ICU.
 * Si la cle est absente, retourne la cle elle-meme (degradation visible mais
 * non-cassante en prod).
 *
 * Les `vars` sont passees telles quelles a IntlMessageFormat.format() :
 * pluralisation `{n, plural, ...}`, select `{x, select, ...}`, interpolation
 * de date `{date, date, long}` etc.
 */
export function formatMessage<M extends ManifestEntries>(
  manifest: M,
  key: keyof M & string,
  locale: ManifestLocale,
  vars?: Record<string, unknown>,
): string {
  const entry = manifest[key]
  if (!entry) {
    // Cle inconnue : on retourne la cle (visible en UI -> repere pour fix).
    return key
  }
  const message = entry[locale]
  if (typeof message !== 'string' || message === '') {
    return key
  }
  if (!vars) {
    // Pas d'interpolation : on peut court-circuiter MessageFormat si la
    // string ne contient pas d'accolades (gain perf).
    if (!message.includes('{')) {
      return message
    }
  }
  try {
    const fmt = getFormatter(locale, message)
    const out = fmt.format(vars)
    return Array.isArray(out) ? out.join('') : String(out)
  } catch {
    // Si le format est mal forme cote TOML, ne pas casser la page.
    return message
  }
}

/**
 * resetFormatterCache vide le cache MessageFormat. Reserve aux tests.
 */
export function resetFormatterCache(): void {
  formatterCache.clear()
}
