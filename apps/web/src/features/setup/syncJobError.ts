/**
 * Code d'échec d'un job de synchronisation (`AsyncJobStatus.error.code`) -> clé du message du
 * manifeste commun. Table unique : l'assistant de première synchronisation et l'accueil d'un
 * joueur sans match disent la même cause avec les mêmes mots.
 */
import type { CommonManifestKey } from '@/lib/i18n/generated/common'

const SYNC_JOB_ERROR_KEYS: Readonly<Record<string, CommonManifestKey>> = {
  sync_auth_expired: 'common.initial_sync.err_auth_expired',
  halo_tokens_missing: 'common.initial_sync.err_halo_tokens_missing',
  sync_halo_api_error: 'common.initial_sync.err_halo_api',
  sync_db_error: 'common.initial_sync.err_db',
  sync_aborted: 'common.initial_sync.err_aborted',
  internal_error: 'common.initial_sync.err_internal',
  db_busy: 'common.initial_sync.err_busy',
  sync_failed: 'common.initial_sync.err_internal',
}

/** Message d'un code d'échec de job ; `undefined` pour un code absent ou inconnu (l'appelant choisit son repli). */
export function syncJobErrorKey(code: string | null | undefined): CommonManifestKey | undefined {
  if (code && Object.hasOwn(SYNC_JOB_ERROR_KEYS, code)) return SYNC_JOB_ERROR_KEYS[code]
  return undefined
}
