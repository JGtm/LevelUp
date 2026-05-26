/**
 * useProfileI18n — accès typé au manifest profile + locale courante.
 *
 * Sucre syntaxique : `t('profile.section.identity.title')` ou
 * `t('profile.insufficient.cta', { missing: 5 })` pour les ICU placeholders.
 */
import { useCallback } from 'react'
import { formatMessage, type ManifestLocale } from '@/lib/i18n/format'
import { profileManifest, type ProfileManifestKey } from '@/lib/i18n/generated/profile'
import { useAppShellStore } from '@/stores/appShellStore'

export function useProfileI18n() {
  const locale = useAppShellStore((s) => s.locale) as ManifestLocale

  const t = useCallback(
    (key: ProfileManifestKey, vars?: Record<string, unknown>) =>
      formatMessage(profileManifest, key, locale, vars),
    [locale],
  )

  return { t, locale }
}
