/**
 * Tests — setupRouting : un compte connecté sans profil à lui va au wizard, et
 * n'en sort pas tant qu'il n'a pas déclaré de profil (ADR 0035 D3).
 */
import { describe, expect, it } from 'vitest'

import {
  needsOwnProfile,
  resolveSetupStep,
  SETUP_PATH,
  setupRedirectPath,
  shouldLeaveSetup,
  type SetupAudience,
} from './setupRouting'

/** Compte SSO connecté, non administrateur, sans aucun profil accessible. */
const compteSansProfil: SetupAudience = {
  authMode: 'xbox',
  currentUsername: 'inconnuaubataillon',
  isAdmin: false,
  availablePlayerCount: 0,
}

const neJamaisRediriger = () => false

describe('needsOwnProfile', () => {
  it('vrai pour un compte connecté, non admin, sans profil accessible', () => {
    expect(needsOwnProfile(compteSansProfil)).toBe(true)
  })

  it('faux dès que le compte a un profil', () => {
    expect(needsOwnProfile({ ...compteSansProfil, availablePlayerCount: 1 })).toBe(false)
  })

  it("faux pour un administrateur (il voit tous les profils ; l'instance vide est couverte par setup_required)", () => {
    expect(needsOwnProfile({ ...compteSansProfil, isAdmin: true })).toBe(false)
  })

  it("faux quand l'authentification n'est pas activée (mono-utilisateur, démo)", () => {
    expect(needsOwnProfile({ ...compteSansProfil, authMode: 'none' })).toBe(false)
  })

  it('faux quand personne n’est connecté (la garde de login passe avant)', () => {
    expect(needsOwnProfile({ ...compteSansProfil, currentUsername: null })).toBe(false)
  })

  it('vrai aussi en mode mot de passe', () => {
    expect(needsOwnProfile({ ...compteSansProfil, authMode: 'password' })).toBe(true)
  })
})

describe('setupRedirectPath', () => {
  it('conduit au wizard depuis une page applicative', () => {
    expect(setupRedirectPath(compteSansProfil, '/', neJamaisRediriger)).toBe(SETUP_PATH)
  })

  it('ne rebondit pas sur le wizard lui-même (anti-boucle)', () => {
    expect(setupRedirectPath(compteSansProfil, SETUP_PATH, neJamaisRediriger)).toBeNull()
  })

  it('laisse les pages d’authentification tranquilles', () => {
    expect(setupRedirectPath(compteSansProfil, '/login', neJamaisRediriger)).toBeNull()
    expect(setupRedirectPath(compteSansProfil, '/register', neJamaisRediriger)).toBeNull()
  })

  it('laisse les pages consultables sans compte', () => {
    const anonyme = (p: string) => p === '/privacy'
    expect(setupRedirectPath(compteSansProfil, '/privacy', anonyme)).toBeNull()
  })

  it('ne redirige pas un compte qui a déjà un profil', () => {
    const etabli = { ...compteSansProfil, availablePlayerCount: 2 }
    expect(setupRedirectPath(etabli, '/', neJamaisRediriger)).toBeNull()
  })
})

describe('resolveSetupStep', () => {
  it("montre l'étape profil à un compte sans profil MAIS déjà lié à Xbox, même si l'instance est « prête »", () => {
    expect(resolveSetupStep('ready', true, true)).toBe('player')
  })

  it('demande d’abord la liaison Xbox si elle manque', () => {
    expect(resolveSetupStep('ready', true, false)).toBe('device_code')
  })

  it('suit l’état d’instance quand le compte a déjà un profil', () => {
    expect(resolveSetupStep('no_halo_link', false, false)).toBe('device_code')
    expect(resolveSetupStep('halo_linked_no_profile', false, true)).toBe('player')
    expect(resolveSetupStep('profile_ready_no_sync', false, true)).toBe('initial_sync')
    expect(resolveSetupStep('ready', false, true)).toBe('done')
    expect(resolveSetupStep(null, false, true)).toBe('done')
  })
})

describe('shouldLeaveSetup', () => {
  it('retient un compte sans profil dans le wizard (sinon : boucle de redirection)', () => {
    expect(shouldLeaveSetup('ready', false, true)).toBe(false)
    expect(shouldLeaveSetup('profile_ready_no_sync', true, true)).toBe(false)
  })

  it('rend la main dès que le compte a un profil et que l’instance est prête', () => {
    expect(shouldLeaveSetup('ready', true, false)).toBe(true)
    expect(shouldLeaveSetup('halo_linked_no_profile', false, false)).toBe(true)
  })

  it('garde dans le wizard une instance encore à configurer', () => {
    expect(shouldLeaveSetup('halo_linked_no_profile', true, false)).toBe(false)
  })
})
