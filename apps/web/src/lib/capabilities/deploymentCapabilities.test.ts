import { describe, it, expect } from 'vitest'
import {
  DEPLOYMENT_CAPABILITY_DEFAULTS,
  deploymentCapabilityOf,
  passwordOnlyAccounts,
} from './deploymentCapabilities'
import {
  accountsCapabilities,
  demoCapabilities,
  desktopCapabilities,
  testCapabilities,
} from '@/test/capabilities'

describe('deploymentCapabilityOf', () => {
  it('avant hydratation (null) ou sans capacités : les anciens défauts — aucun compte, rien de figé', () => {
    for (const caps of [null, undefined, {}]) {
      expect(deploymentCapabilityOf(caps, 'can_use_accounts')).toBe(false)
      expect(deploymentCapabilityOf(caps, 'can_sign_in_with_xbox')).toBe(false)
      expect(deploymentCapabilityOf(caps, 'can_edit_settings')).toBe(true)
      expect(deploymentCapabilityOf(caps, 'can_upload_media')).toBe(true)
      expect(deploymentCapabilityOf(caps, 'can_pool_tokens')).toBe(true)
    }
  })

  it('la valeur publiée par le serveur l’emporte sur le défaut', () => {
    const demo = testCapabilities(demoCapabilities())
    expect(deploymentCapabilityOf(demo, 'can_edit_settings')).toBe(false)
    expect(deploymentCapabilityOf(demo, 'can_upload_media')).toBe(false)
    expect(deploymentCapabilityOf(demo, 'can_pool_tokens')).toBe(false)
  })

  it('chaque capacité par défaut existe dans la réponse du serveur (type généré)', () => {
    const published = testCapabilities()
    for (const key of Object.keys(DEPLOYMENT_CAPABILITY_DEFAULTS)) {
      expect(key in published).toBe(true)
    }
  })
})

describe('capacités médias du bureau', () => {
  const keys = ['can_keep_media_sources', 'can_reveal_files_in_explorer', 'can_trash_media_sources'] as const

  it('avant hydratation et sans capacités : fausses (rien ne change hors du bureau)', () => {
    for (const caps of [null, undefined, {}]) {
      for (const key of keys) expect(deploymentCapabilityOf(caps, key)).toBe(false)
    }
  })

  it('le profil bureau les publie vraies', () => {
    const desktop = testCapabilities(desktopCapabilities())
    for (const key of keys) expect(deploymentCapabilityOf(desktop, key)).toBe(true)
  })
})

describe('capacité de choix de dossier', () => {
  it('fausse avant hydratation et sans capacités', () => {
    for (const caps of [null, undefined, {}]) {
      expect(deploymentCapabilityOf(caps, 'can_pick_folder')).toBe(false)
    }
  })

  it('vraie pour le profil bureau seulement', () => {
    expect(deploymentCapabilityOf(testCapabilities(desktopCapabilities()), 'can_pick_folder')).toBe(true)
    expect(deploymentCapabilityOf(testCapabilities(demoCapabilities()), 'can_pick_folder')).toBe(false)
    expect(deploymentCapabilityOf(testCapabilities(), 'can_pick_folder')).toBe(false)
  })
})

describe('passwordOnlyAccounts', () => {
  it.each([
    ['none', false],
    ['password', true],
    ['xbox', false],
  ] as const)('mode %s → %s', (mode, want) => {
    expect(passwordOnlyAccounts(testCapabilities(accountsCapabilities(mode)))).toBe(want)
  })

  it('sans capacités : faux (pas de redirection avant hydratation)', () => {
    expect(passwordOnlyAccounts(null)).toBe(false)
  })
})
