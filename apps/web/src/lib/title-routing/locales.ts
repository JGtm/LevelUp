/**
 * Locales connues de l'application, pour le parsing du segment de langue de l'URL.
 *
 * NOTE (2026-07-22) : il n'existe pas (encore) de type `Locale` central dans
 * `apps/web`. La valeur `'fr' | 'en'` est aujourd'hui redéclarée localement à
 * plusieurs endroits (ManifestLocale, FieldMappingLocale, MetricLocale, et inline
 * dans client.ts / appShellStore.ts). Ce module fournit la liste RUNTIME
 * (`KNOWN_LOCALES`) dont le parsing de segment a besoin ; le type dérivé reste
 * structurellement `'fr' | 'en'`, donc assignable à `setApiLocale` / `setLocale`
 * sans conversion. (Centralisation d'un `Locale` unique = hors périmètre D7.)
 */
export const KNOWN_LOCALES = ['fr', 'en'] as const

export type Locale = (typeof KNOWN_LOCALES)[number]

/** Type guard : le segment est-il une locale connue (fr | en) ? */
export function isKnownLocale(segment: string): segment is Locale {
  return (KNOWN_LOCALES as readonly string[]).includes(segment)
}
