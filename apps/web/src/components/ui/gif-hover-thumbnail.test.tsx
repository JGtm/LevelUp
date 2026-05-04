/**
 * Tests GifHoverThumbnail — vérifie l'affichage en cas de succès/échec du canvas.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, act, waitFor } from '@testing-library/react'
import { GifHoverThumbnail } from './gif-hover-thumbnail'

// Mock global Image pour contrôler l'onload/onerror
class MockImage {
  onload: (() => void) | null = null
  onerror: (() => void) | null = null
  naturalWidth = 100
  naturalHeight = 100
  crossOrigin = ''
  set src(_v: string) {
    // Triggered manually via triggerLoad/triggerError below
  }
  triggerLoad() {
    if (this.onload) this.onload()
  }
  triggerError() {
    if (this.onerror) this.onerror()
  }
}

let lastImage: MockImage | null = null
const OriginalImage = globalThis.Image

beforeEach(() => {
  globalThis.Image = function () {
    lastImage = new MockImage()
    return lastImage
  } as unknown as typeof Image
})

afterEach(() => {
  globalThis.Image = OriginalImage
  lastImage = null
})

describe('GifHoverThumbnail', () => {
  it('affiche le canvas si l\'image se charge ET getContext fonctionne', async () => {
    // Mock getContext('2d') pour simuler un canvas fonctionnel
    const drawImageMock = vi.fn()
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      drawImage: drawImageMock,
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext

    try {
      const { container } = render(<GifHoverThumbnail src="/img.gif" isActive={false} />)
      await act(async () => {
        lastImage!.triggerLoad()
      })
      const canvas = container.querySelector('canvas')
      expect(canvas).toBeInTheDocument()
      expect(drawImageMock).toHaveBeenCalled()
      expect(canvas?.className).toContain('opacity-100')
    } finally {
      HTMLCanvasElement.prototype.getContext = originalGetContext
    }
  })

  it('affiche un <img> de fallback si le canvas échoue (CORS, image inaccessible)', async () => {
    const { container } = render(<GifHoverThumbnail src="/broken.gif" isActive={false} alt="test" />)
    expect(lastImage).not.toBeNull()
    await act(async () => {
      lastImage!.triggerError()
    })
    // Le canvas est invisible
    const canvas = container.querySelector('canvas')
    expect(canvas?.className).toContain('opacity-0')
    // Mais un <img> de fallback est rendu
    await waitFor(() => {
      const img = container.querySelector('img[src="/broken.gif"]')
      expect(img).toBeInTheDocument()
    })
  })

  it('affiche l\'<img> animée au survol', () => {
    const { container } = render(<GifHoverThumbnail src="/active.gif" isActive={true} />)
    const img = container.querySelector('img[src="/active.gif"]')
    expect(img).toBeInTheDocument()
  })

  it('au survol, le canvas est masqué (opacity-0) même si prêt', async () => {
    const drawImageMock = vi.fn()
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      drawImage: drawImageMock,
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext

    try {
      const { container, rerender } = render(<GifHoverThumbnail src="/img.gif" isActive={false} />)
      await act(async () => {
        lastImage!.triggerLoad()
      })
      expect(container.querySelector('canvas')?.className).toContain('opacity-100')

      rerender(<GifHoverThumbnail src="/img.gif" isActive={true} />)
      expect(container.querySelector('canvas')?.className).toContain('opacity-0')
      expect(container.querySelector('img[src="/img.gif"]')).toBeInTheDocument()
    } finally {
      HTMLCanvasElement.prototype.getContext = originalGetContext
    }
  })

  it('ne crée pas de crossOrigin (regression : crossOrigin cassait drawImage)', () => {
    render(<GifHoverThumbnail src="/img.gif" isActive={false} />)
    expect(lastImage?.crossOrigin).toBe('')
  })
})
