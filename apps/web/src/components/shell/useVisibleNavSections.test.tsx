/**
 * Tests — sections de navigation visibles : l'onglet « Classements » de la Communauté dépend de la
 * capacité de déploiement `can_compare_accounts` (ADR 0044), en plus de la capacité de titre.
 */
import { afterEach, describe, expect, it, vi } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useAppShellStore } from '@/stores/appShellStore'
import { desktopCapabilities, testCapabilities } from '@/test/capabilities'
import { useVisibleNavSections } from './useVisibleNavSections'

vi.mock('@/features/settings/queries', () => ({
  useSettings: () => ({ data: { show_progression: true } }),
}))

const community = () => renderHook(() => useVisibleNavSections()).result.current.find((s) => s.key === 'community')

describe('useVisibleNavSections — Communauté', () => {
  afterEach(() => useAppShellStore.setState({ capabilities: null }))

  it('bureau : pas de Classements, et la section mène à Relations', () => {
    useAppShellStore.setState({ capabilities: testCapabilities(desktopCapabilities()) })
    const section = community()
    expect(section?.tabs?.map((t) => t.key)).toEqual(['relations', 'compare'])
    expect(section?.defaultPath).toBe('/{-$lang}/t/$titleSlug/players/$playerSlug/community/relations')
  })

  it('site : Classements reste le premier onglet et la destination de la section', () => {
    useAppShellStore.setState({ capabilities: testCapabilities({}) })
    const section = community()
    expect(section?.tabs?.map((t) => t.key)).toEqual(['leaderboard', 'relations', 'compare'])
    expect(section?.defaultPath).toBe('/{-$lang}/t/$titleSlug/players/$playerSlug/community')
  })

  it('capacités pas encore connues : le défaut est celui du site', () => {
    useAppShellStore.setState({ capabilities: null })
    expect(community()?.tabs?.map((t) => t.key)).toContain('leaderboard')
  })
})
