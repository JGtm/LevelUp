/**
 * AssetDrawer — fermeture : Échap ferme le tiroir, un clic sur le voile aussi, un clic dans le
 * panneau non.
 */
import { describe, it, expect, beforeEach } from 'vitest'
import { fireEvent } from '@testing-library/react'
import { http, HttpResponse } from 'msw'
import { renderWithProviders } from '@/test/render-utils'
import { server } from '@/test/setup'
import { useAppShellStore } from '@/stores/appShellStore'
import { useAssetDrawerStore } from './assetDrawer.store'
import { AssetDrawer } from './AssetDrawer'

beforeEach(() => {
  useAppShellStore.setState({ locale: 'fr', currentTitleSlug: 'halo_infinite' })
  useAssetDrawerStore.setState({ isOpen: true, activeTab: 'maps', search: '' })
  server.use(http.get('/api/v1/assets/:titleSlug/:kind', () => HttpResponse.json([])))
})

function ouvre() {
  const { container } = renderWithProviders(<AssetDrawer />)
  const voile = container.querySelector('div.fixed.inset-0') as HTMLElement
  const panneau = container.querySelector('[role="complementary"]') as HTMLElement
  expect(voile).not.toBeNull()
  return { voile, panneau }
}

describe('AssetDrawer — fermeture', () => {
  it('Échap ferme le tiroir', () => {
    ouvre()
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
  })

  it('un clic sur le voile ferme le tiroir', () => {
    const { voile } = ouvre()
    fireEvent.click(voile)
    expect(useAssetDrawerStore.getState().isOpen).toBe(false)
  })

  it('un clic dans le panneau ne ferme pas le tiroir', () => {
    const { panneau } = ouvre()
    fireEvent.mouseDown(panneau)
    fireEvent.click(panneau)
    expect(useAssetDrawerStore.getState().isOpen).toBe(true)
  })
})
