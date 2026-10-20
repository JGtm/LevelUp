/**
 * tokenHealthDisplay — logique pure d'affichage de la section Santé des tokens
 * (séparée de AdminPage.tsx pour testabilité sans rendu).
 */
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

/** Clés i18n par classe du dernier échec OAuth (resolver, plan anti-bruit 2026-06-11). */
export const TOKEN_ERROR_KEY: Record<string, CommonManifestKey> = {
  config: 'common.admin.token_error_config',
  revoked: 'common.admin.token_error_revoked',
  transient: 'common.admin.token_error_transient',
}

/**
 * credentialSourceParts — réduit le label du scan en familles courtes
 * dédupliquées. Depuis ADR 0023 Phase 5 (2026-08-25) le scan ne peut produire
 * que "watcher_oauth" → "store" ; tout autre label est rendu tel quel et
 * signalé comme dette (cf. hasLegacyCredentialSource) — c'est le garde-rail
 * visuel si une source legacy réapparaissait côté back.
 */
export function credentialSourceParts(source: string): string[] {
  const mapped = source.split('+').map((part) => (part.startsWith('watcher_') ? 'store' : part))
  return [...new Set(mapped)]
}

/** true si la source contient autre chose que le store canonique ADR-0023 (dette à signaler en warning). */
export function hasLegacyCredentialSource(parts: string[]): boolean {
  return parts.some((p) => p !== 'store')
}
