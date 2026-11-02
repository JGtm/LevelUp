/**
 * Tests — logique pure du repliement de la barre latérale et libellés réduits de la barre d'icônes.
 */
import { describe, expect, it, vi } from 'vitest'
import {
  SIDEBAR_PREFERENCE_STORAGE_KEY,
  isSidebarShortcut,
  loadSidebarPreference,
  resolveSidebarCollapsed,
  toggledPreference,
  type SidebarPreference,
} from './sidebarCollapse'
import { playerInitial, titleInitials } from './railLabels'

describe('resolveSidebarCollapsed', () => {
  it.each<[SidebarPreference, boolean, boolean]>([
    ['auto', true, true],
    ['auto', false, false],
    ['collapsed', false, true],
    ['collapsed', true, true],
    ['expanded', true, false],
    ['expanded', false, false],
  ])('préférence %s, fenêtre étroite %s : replié = %s', (preference, narrow, expected) => {
    expect(resolveSidebarCollapsed(preference, narrow)).toBe(expected)
  })
})

describe('toggledPreference', () => {
  it('rend le contraire de l’état affiché, toujours en choix manuel', () => {
    expect(toggledPreference('auto', false)).toBe('collapsed')
    expect(toggledPreference('auto', true)).toBe('expanded')
    expect(toggledPreference('collapsed', false)).toBe('expanded')
    expect(toggledPreference('expanded', true)).toBe('collapsed')
    expect(toggledPreference('expanded', false)).toBe('collapsed')
  })
})

describe('loadSidebarPreference', () => {
  it('rend auto sans rien de mémorisé ou avec une valeur inconnue', () => {
    expect(loadSidebarPreference()).toBe('auto')
    localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, 'tout-replie')
    expect(loadSidebarPreference()).toBe('auto')
  })

  it('relit un choix manuel', () => {
    localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, 'collapsed')
    expect(loadSidebarPreference()).toBe('collapsed')
    localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, 'expanded')
    expect(loadSidebarPreference()).toBe('expanded')
  })

  it('un stockage refusé laisse la barre en auto, sans lever', () => {
    const warn = vi.spyOn(console, 'warn').mockImplementation(() => {})
    const get = vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('accès refusé')
    })
    expect(loadSidebarPreference()).toBe('auto')
    expect(warn).toHaveBeenCalled()
    get.mockRestore()
    warn.mockRestore()
  })
})

describe('isSidebarShortcut', () => {
  const key = (init: Partial<KeyboardEvent>) =>
    ({ key: 'b', ctrlKey: true, shiftKey: false, altKey: false, metaKey: false, ...init }) as KeyboardEvent
  it('est Ctrl+B et rien d’autre', () => {
    expect(isSidebarShortcut(key({}))).toBe(true)
    expect(isSidebarShortcut(key({ key: 'B' }))).toBe(true)
    expect(isSidebarShortcut(key({ ctrlKey: false }))).toBe(false)
    expect(isSidebarShortcut(key({ shiftKey: true }))).toBe(false)
    expect(isSidebarShortcut(key({ altKey: true }))).toBe(false)
    expect(isSidebarShortcut(key({ metaKey: true }))).toBe(false)
    expect(isSidebarShortcut(key({ key: 'l' }))).toBe(false)
  })
})

describe('libellés réduits', () => {
  it('playerInitial : première lettre en majuscule', () => {
    expect(playerInitial('jgtm')).toBe('J')
    expect(playerInitial('  élodie')).toBe('É')
    expect(playerInitial('')).toBe('?')
    expect(playerInitial(null)).toBe('?')
  })

  it('titleInitials : sigle de deux lettres', () => {
    expect(titleInitials('Halo Infinite')).toBe('HI')
    expect(titleInitials('Halo 5: Guardians')).toBe('H5')
    expect(titleInitials('Reach')).toBe('R')
    expect(titleInitials('')).toBe('?')
  })
})
