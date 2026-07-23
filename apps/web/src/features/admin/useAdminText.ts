/**
 * useAdminText — helpers i18n partagés par les pages/sections admin.
 *
 * Deux manifests coexistent :
 * - common.toml (`common.admin.*`) : clés historiques des sections Users/
 *   Invites/Invariants/Contention/Tokens — inchangées (zéro migration).
 * - admin.toml (`admin.*`) : clés du dashboard monitoring (nav, overview,
 *   sync, statuts, jobs…).
 */
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { intlLocale } from '@/lib/formatters'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'
import { adminManifest, type AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { Locale } from '@/lib/i18n/locale'

export type T = (key: CommonManifestKey) => string
export type TAdmin = (key: AdminManifestKey) => string

/** Traducteur des clés historiques `common.admin.*`. */
export function useT(): T {
  const locale = useAppShellStore((s) => s.locale)
  return (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)
}

/** Traducteur des clés du dashboard monitoring `admin.*`. */
export function useAdminT(): TAdmin {
  const locale = useAppShellStore((s) => s.locale)
  return (key: AdminManifestKey) => formatMessage(adminManifest, key, locale)
}

/** Locale BCP-47 pour toLocaleString/toLocaleDateString. */
export function useDateLocale(): string {
  const locale = useAppShellStore((s) => s.locale)
  return intlLocale(locale)
}

/** Locale courte ('fr' | 'en') pour les helpers purs (format.ts). */
export function useAdminLocale(): Locale {
  const locale = useAppShellStore((s) => s.locale)
  return locale === 'fr' ? 'fr' : 'en'
}
