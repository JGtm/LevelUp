/**
 * identitiesDisplay.test.ts — la mise en forme de l'annuaire, sans rendu.
 */
import { describe, expect, it } from 'vitest'

import type { IdentityProfileRef, IdentityRecord } from '@/lib/api/types'
import {
  anomalyLabelKey,
  anomalySeverityToken,
  identityRowKey,
  profileStateKeys,
  tokenState,
  tokenStateToken,
  TOKEN_STATE_KEY,
  warningCount,
} from './identitiesDisplay'

function profile(over: Partial<IdentityProfileRef> = {}): IdentityProfileRef {
  return {
    title_slug: 'halo_infinite',
    key: 'Spartan',
    sync_enabled: true,
    auth_only: false,
    dir_exists: true,
    db_exists: true,
    ...over,
  }
}

function record(over: Partial<IdentityRecord> = {}): IdentityRecord {
  return { profiles: [], watched: [], anomalies: [], ...over }
}

describe('anomalySeverityToken', () => {
  it('warning et info ont chacun leur token, jamais une couleur en dur', () => {
    expect(anomalySeverityToken('warning')).toBe('warning')
    expect(anomalySeverityToken('info')).toBe('info')
  })

  it('sévérité inconnue retombe sur info (ne crie pas au loup)', () => {
    expect(anomalySeverityToken('inattendu')).toBe('info')
  })
})

describe('anomalyLabelKey', () => {
  it('traduit les six codes du serveur', () => {
    for (const code of [
      'account_without_profile',
      'token_orphan',
      'player_dir_orphan',
      'watched_without_profile',
      'profile_without_account',
      'profile_without_token',
    ]) {
      expect(anomalyLabelKey(code)).toBe(`admin.identities.anomaly_${code}`)
    }
  })

  it('rend null pour un code inconnu — le composant affiche alors le code brut', () => {
    expect(anomalyLabelKey('code_du_futur')).toBeNull()
  })
})

describe('warningCount', () => {
  it('ne compte que les warning — c est la valeur de tri de la colonne', () => {
    const rec = record({
      anomalies: [
        { code: 'account_without_profile', severity: 'warning' },
        { code: 'player_dir_orphan', severity: 'warning' },
        { code: 'profile_without_token', severity: 'info' },
      ],
    })
    expect(warningCount(rec)).toBe(2)
    expect(warningCount(record())).toBe(0)
  })
})

describe('identityRowKey', () => {
  it('préfère le xuid, retombe sur le gamertag, puis sur l index', () => {
    expect(identityRowKey(record({ xuid: '111', gamertag: 'Spartan' }), 0)).toBe('xuid:111')
    expect(identityRowKey(record({ gamertag: 'Fantome' }), 0)).toBe('gt:fantome')
    expect(identityRowKey(record(), 3)).toBe('row:3')
  })
})

describe('profileStateKeys', () => {
  it('un profil normal ne porte aucun état', () => {
    expect(profileStateKeys(profile())).toEqual([])
  })

  it('pause, auth seule et dossier absent sont signalés', () => {
    expect(profileStateKeys(profile({ sync_enabled: false }))).toEqual([
      'admin.identities.profile_paused',
    ])
    expect(profileStateKeys(profile({ auth_only: true, sync_enabled: false }))).toEqual([
      'admin.identities.profile_auth_only',
    ])
    expect(profileStateKeys(profile({ dir_exists: false }))).toEqual([
      'admin.identities.profile_no_dir',
    ])
  })
})

describe('tokenState', () => {
  it('résume l état des identifiants par ordre de gravité', () => {
    expect(tokenState(undefined)).toBe('absent')
    expect(tokenState({ has_refresh_token: true, reauth_required: false })).toBe('present')
    expect(tokenState({ has_refresh_token: true, reauth_required: true })).toBe('reauth')
    expect(
      tokenState({ has_refresh_token: true, reauth_required: false, last_auth_error: 'invalid_grant' }),
    ).toBe('error')
    expect(tokenState({ has_refresh_token: false, reauth_required: false })).toBe('absent')
  })

  it('chaque état a sa clé i18n ; « absent » n a PAS de couleur (cas normal)', () => {
    expect(Object.keys(TOKEN_STATE_KEY).sort()).toEqual(['absent', 'error', 'present', 'reauth'])
    expect(tokenStateToken('absent')).toBeNull()
    expect(tokenStateToken('present')).toBe('success')
    expect(tokenStateToken('reauth')).toBe('warning')
    expect(tokenStateToken('error')).toBe('destructive')
  })
})
