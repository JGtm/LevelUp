import type { CapabilityMap } from '@/lib/api/types'
import { DEPLOYMENT_CAPABILITY_DEFAULTS } from '@/lib/capabilities/deploymentCapabilities'

/** Capacités d'un bootstrap de test : tout offert, profil dev, aucun compte. */
export function testCapabilities(overrides: Partial<CapabilityMap> = {}): CapabilityMap {
  return {
    can_read_local_data: true,
    can_run_sync: true,
    can_use_live_halo: true,
    can_manage_settings: true,
    can_reset_media_index: true,
    can_view_media: true,
    can_self_provision: true,
    can_start_initial_sync: true,
    can_manage_instance: true,
    ...DEPLOYMENT_CAPABILITY_DEFAULTS,
    deployment: 'dev',
    ...overrides,
  }
}

/**
 * Les capacités de comptes qu'un mode d'authentification du serveur publie : `none` = aucun
 * compte, `password` = comptes locaux, `xbox` = comptes avec le SSO Xbox (ADR 0044).
 */
export function accountsCapabilities(mode: 'none' | 'password' | 'xbox'): Partial<CapabilityMap> {
  return {
    can_use_accounts: mode !== 'none',
    can_use_invitations: mode !== 'none',
    can_sign_in_with_xbox: mode === 'xbox',
  }
}

/** Les capacités de la démo : réglages et dépôt figés, mise en commun des jetons non proposée. */
export function demoCapabilities(): Partial<CapabilityMap> {
  return { can_edit_settings: false, can_upload_media: false, can_pool_tokens: false, deployment: 'demo' }
}

/** Les capacités du profil desktop : chrome d'application de bureau, aucun compte, mise à jour par la coquille. */
export function desktopCapabilities(): Partial<CapabilityMap> {
  return {
    can_use_compact_chrome: true,
    can_login_via_browser: true,
    can_compare_accounts: false,
    can_watch_game_process: true,
    can_integrate_shell: true,
    can_schedule_refresh: true,
    can_backup_to_folder: true,
    can_import_from_site: true,
    can_report_diagnostic: true,
    can_limit_memory: true,
    can_regenerate_replays: true,
    can_update_app: true,
    can_offer_windows_download: false,
    can_use_accounts: false,
    can_use_admin: false,
    can_use_groups: false,
    can_pool_tokens: false,
    can_keep_media_sources: true,
    can_reveal_files_in_explorer: true,
    can_trash_media_sources: true,
    deployment: 'desktop',
  }
}
