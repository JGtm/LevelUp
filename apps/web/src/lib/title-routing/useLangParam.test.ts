/**
 * Tests — useLangParam (I10). Jumeau de useTitleSlug : lit la locale active du store
 * appShell, pour ÉMETTRE le param `lang` des liens title-scoped.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'

import { useLangParam } from './useLangParam'
import { useAppShellStore } from '@/stores/appShellStore'

describe('useLangParam', () => {
  beforeEach(() => {
    act(() => useAppShellStore.setState({ locale: 'fr' }))
  })

  it('retourne la locale active du store', () => {
    const { result } = renderHook(() => useLangParam())
    expect(result.current).toBe('fr')
  })

  it('suit la bascule de locale (reactif)', () => {
    const { result } = renderHook(() => useLangParam())
    act(() => useAppShellStore.setState({ locale: 'en' }))
    expect(result.current).toBe('en')
  })
})
