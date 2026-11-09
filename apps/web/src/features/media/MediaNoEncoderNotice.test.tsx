/**
 * Tests — message de la page Médias quand le serveur n'a aucun encodeur H.264 (`media_h264_missing`) :
 * il explique la cause (édition Windows N sans Media Foundation) en français et en anglais, pointe le
 * « Media Feature Pack » dans un nouvel onglet, et n'apparaît que sous ce signal. Le signal vient du
 * bootstrap par le store.
 */
import { afterEach, describe, expect, it } from 'vitest'
import { act, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { MEDIA_FEATURE_PACK_URL } from '@/lib/appLinks'
import type { BootstrapResponse } from '@/lib/api/types'
import { MediaNoEncoderNotice } from './MediaNoEncoderNotice'

afterEach(() => {
  act(() => useAppShellStore.setState({ locale: 'fr', mediaH264Missing: false }))
})

describe('MediaNoEncoderNotice', () => {
  it('ne rend rien quand un encodeur existe', () => {
    const { container } = renderWithProviders(<MediaNoEncoderNotice />)
    expect(container).toBeEmptyDOMElement()
  })

  it('explique et pointe le Media Feature Pack (fr)', () => {
    act(() => useAppShellStore.setState({ mediaH264Missing: true }))
    renderWithProviders(<MediaNoEncoderNotice />)
    const notice = screen.getByTestId('media-no-h264')
    expect(notice).toHaveTextContent('aucun encodeur vidéo H.264')
    expect(notice).toHaveTextContent('Windows N')
    const link = screen.getByRole('link', { name: 'Page du Media Feature Pack' })
    expect(link).toHaveAttribute('href', MEDIA_FEATURE_PACK_URL)
    expect(link).toHaveAttribute('target', '_blank')
  })

  it('en anglais', () => {
    act(() => useAppShellStore.setState({ mediaH264Missing: true, locale: 'en' }))
    renderWithProviders(<MediaNoEncoderNotice />)
    expect(screen.getByTestId('media-no-h264')).toHaveTextContent('no H.264 video encoder')
    expect(screen.getByRole('link', { name: 'Media Feature Pack page' })).toBeInTheDocument()
  })

  it('le bootstrap pose le signal, et son absence le retire', () => {
    const base = { setup_required: false } as unknown as BootstrapResponse
    act(() => useAppShellStore.getState().hydrateFromBootstrap({ ...base, media_h264_missing: true }))
    expect(useAppShellStore.getState().mediaH264Missing).toBe(true)
    act(() => useAppShellStore.getState().hydrateFromBootstrap(base))
    expect(useAppShellStore.getState().mediaH264Missing).toBe(false)
  })
})
