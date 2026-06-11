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
 * credentialSourceParts — réduit le label composite du scan (ex.
 * "watcher_msal+watcher_oauth", "duckdb_msal+env_oauth") en familles courtes
 * dédupliquées : store / sync_meta / env / legacy.
 */
export function credentialSourceParts(source: string): string[] {
  const mapped = source.split('+').map((part) => {
    if (part === 'watcher_legacy') return 'legacy'
    if (part.startsWith('watcher_')) return 'store'
    if (part.startsWith('duckdb_')) return 'sync_meta'
    if (part === 'env_oauth') return 'env'
    return part
  })
  return [...new Set(mapped)]
}

/** true si la source contient autre chose que le store canonique ADR-0023 (dette à signaler en warning). */
export function hasLegacyCredentialSource(parts: string[]): boolean {
  return parts.some((p) => p !== 'store')
}
