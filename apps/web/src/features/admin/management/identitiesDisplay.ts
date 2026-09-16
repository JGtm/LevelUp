/**
 * identitiesDisplay — mise en forme de l'annuaire des joueurs (ADR 0035 D7).
 *
 * Le serveur rend des CODES machine (`account_without_profile`, `warning`…) et
 * un contexte machine (slug de titre, nom de dossier) : tout ce qui se lit à
 * l'écran se décide ici, hors composant, et se teste sans rendu
 * (`identitiesDisplay.test.ts`).
 */
import type { SemanticToken } from '@/lib/accessibility/semantic-tokens'
import type { AdminManifestKey } from '@/lib/i18n/generated/admin'
import type { IdentityProfileRef, IdentityRecord, IdentityTokenRef } from '@/lib/api/types'

/** Sévérité d'anomalie → token sémantique (jamais une couleur en dur). */
export function anomalySeverityToken(severity: string): SemanticToken {
  return severity === 'warning' ? 'warning' : 'info'
}

/**
 * Code d'anomalie → clé i18n. `null` pour un code inconnu : le serveur peut en
 * livrer un nouveau avant que le web le connaisse, et une ligne muette serait
 * pire que le code brut (que le composant affiche alors tel quel).
 */
const ANOMALY_LABEL_KEY: Record<string, AdminManifestKey> = {
  account_without_profile: 'admin.identities.anomaly_account_without_profile',
  token_orphan: 'admin.identities.anomaly_token_orphan',
  player_dir_orphan: 'admin.identities.anomaly_player_dir_orphan',
  watched_without_profile: 'admin.identities.anomaly_watched_without_profile',
  profile_without_account: 'admin.identities.anomaly_profile_without_account',
  profile_without_token: 'admin.identities.anomaly_profile_without_token',
  account_duplicate: 'admin.identities.anomaly_account_duplicate',
}

export function anomalyLabelKey(code: string): AdminManifestKey | null {
  return ANOMALY_LABEL_KEY[code] ?? null
}

/**
 * Nombre d'anomalies `warning` d'une identité — la valeur de tri de la colonne
 * Anomalies, et donc l'ordre par défaut du tableau : ce qu'un administrateur
 * ouvre cette page pour voir passe devant.
 */
export function warningCount(rec: IdentityRecord): number {
  return (rec.anomalies ?? []).filter((a) => a.severity === 'warning').length
}

/** Clé React stable d'une ligne (le xuid peut être vide, cf. types.ts). */
export function identityRowKey(rec: IdentityRecord, index: number): string {
  if (rec.xuid) return `xuid:${rec.xuid}`
  if (rec.gamertag) return `gt:${rec.gamertag.toLowerCase()}`
  return `row:${index}`
}

/**
 * États d'un profil à afficher à côté de son titre. Un profil « normal » n'en a
 * aucun — on ne décore que ce qui s'écarte du cas courant.
 */
export function profileStateKeys(profile: IdentityProfileRef): AdminManifestKey[] {
  const keys: AdminManifestKey[] = []
  if (profile.auth_only) keys.push('admin.identities.profile_auth_only')
  else if (!profile.sync_enabled) keys.push('admin.identities.profile_paused')
  if (!profile.dir_exists) keys.push('admin.identities.profile_no_dir')
  return keys
}

/** État des identifiants d'une identité, résumé en un badge. */
export type TokenState = 'absent' | 'error' | 'reauth' | 'present'

export function tokenState(token?: IdentityTokenRef): TokenState {
  if (!token) return 'absent'
  if (token.reauth_required) return 'reauth'
  if (token.last_auth_error) return 'error'
  if (token.has_refresh_token) return 'present'
  return 'absent'
}

export const TOKEN_STATE_KEY: Record<TokenState, AdminManifestKey> = {
  absent: 'admin.identities.token_absent',
  error: 'admin.identities.token_error',
  reauth: 'admin.identities.token_reauth',
  present: 'admin.identities.token_present',
}

/**
 * Token de couleur de l'état des identifiants. `absent` n'en a AUCUN : c'est le
 * cas normal d'un ami dont le pool d'auth prête les identifiants — le peindre
 * en rouge apprendrait à ignorer la couleur.
 */
export function tokenStateToken(state: TokenState): SemanticToken | null {
  switch (state) {
    case 'present':
      return 'success'
    case 'reauth':
      return 'warning'
    case 'error':
      return 'destructive'
    default:
      return null
  }
}
