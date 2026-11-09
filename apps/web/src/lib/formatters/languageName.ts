/**
 * languageName — nom d'une langue de `config/locales.toml` écrit dans la langue de l'interface
 * (`fr` vu depuis `en` donne « French », depuis `fr` « français »).
 *
 * Le nom vient de `Intl.DisplayNames` (étiquette de formatage de `INTL_TAG`, via `intlLocale`) :
 * aucune table de noms de langues à tenir à jour quand une langue s'ajoute. Si l'environnement
 * ne sait pas nommer la langue, le code lui-même est rendu (jamais une chaîne vide).
 */
import type { ManifestLocale } from '@/lib/i18n/format'
import { intlLocale } from './intlLocale'

export function languageName(code: ManifestLocale, uiLocale: ManifestLocale): string {
  try {
    return new Intl.DisplayNames([intlLocale(uiLocale)], { type: 'language' }).of(code) ?? code
  } catch {
    return code
  }
}
