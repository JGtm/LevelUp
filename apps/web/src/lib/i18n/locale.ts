/**
 * Locale — type central de langue de l'application (source canonique UNIQUE).
 *
 * Module FEUILLE : il n'importe RIEN. Tout ce qui a besoin du type de locale ou de
 * la liste runtime des locales connues dépend d'ici, jamais l'inverse — cette
 * asymétrie garantit l'absence de cycle d'imports. Les consommateurs historiques
 * s'y branchent : l'alias `ManifestLocale` (lib/i18n/format) ré-exporte ce type, le
 * parsing du segment de langue de l'URL (title-routing) et les headers API
 * (lib/api/client, stores/appShellStore) l'utilisent directement.
 *
 * `KNOWN_LOCALES` est la liste RUNTIME (le type `Locale` en est dérivé) dont le
 * parsing de segment a besoin ; `isKnownLocale` en est le type guard.
 */
export const KNOWN_LOCALES = ['fr', 'en'] as const

export type Locale = (typeof KNOWN_LOCALES)[number]

/** Type guard : le segment est-il une locale connue (fr | en) ? */
export function isKnownLocale(segment: string): segment is Locale {
  return (KNOWN_LOCALES as readonly string[]).includes(segment)
}
