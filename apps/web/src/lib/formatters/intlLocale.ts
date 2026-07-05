/**
 * intlLocale — pont canonique locale applicative (`'fr' | 'en'`, ManifestLocale)
 * → locale BCP-47 (`'fr-FR' | 'en-US'`) pour `toLocaleString` / `Intl.*`.
 *
 * SOURCE UNIQUE (CLAUDE.md n°6) : le ternaire `locale === 'en' ? 'en-US' : 'fr-FR'`
 * était dupliqué dans des dizaines de composants, et de nombreux sites figeaient
 * carrément `'fr-FR'` (nombres FR affichés à un utilisateur EN — casse la règle
 * n°1). Router tout formatage nombre/date locale-sensitive via ce pont
 * (ou `formatNumber`/`formatDate` qui prennent déjà une `Locale`).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import type { Locale } from './date'

export function intlLocale(locale: ManifestLocale): Locale {
  return locale === 'en' ? 'en-US' : 'fr-FR'
}
