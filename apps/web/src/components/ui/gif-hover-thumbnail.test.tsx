/**
 * Tests GifHoverThumbnail — vérifie l'affichage en cas de succès/échec du canvas
 * et le mécanisme de fragment URL `#h=N` pour relancer l'animation au hover.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, act, waitFor } from '@testing-library/react'
import { GifHoverThumbnail } from './gif-hover-thumbnail'

// fetch + createImageBitmap mocks
let fetchMock: ReturnType<typeof vi.fn>
let bitmapMock: ReturnType<typeof vi.fn>
let bitmapClose: ReturnType<typeof vi.fn>
const originalFetch = globalThis.fetch
const originalCreateImageBitmap = globalThis.createImageBitmap

beforeEach(() => {
  bitmapClose = vi.fn()
  fetchMock = vi.fn(async () => ({
    ok: true,
    status: 200,
    blob: async () => new Blob([new Uint8Array([0])], { type: 'image/webp' }),
  }))
  bitmapMock = vi.fn(async () => ({ width: 100, height: 100, close: bitmapClose }))
  globalThis.fetch = fetchMock as unknown as typeof fetch
  globalThis.createImageBitmap = bitmapMock as unknown as typeof createImageBitmap
})

afterEach(() => {
  globalThis.fetch = originalFetch
  globalThis.createImageBitmap = originalCreateImageBitmap
})

describe('GifHoverThumbnail', () => {
  it('affiche le canvas si le fetch+createImageBitmap réussit ET getContext fonctionne', async () => {
    const drawImageMock = vi.fn()
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      drawImage: drawImageMock,
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext

    try {
      const { container } = await act(async () => {
        return render(<GifHoverThumbnail src="/img.webp" isActive={false} />)
      })
      await waitFor(() => {
        expect(drawImageMock).toHaveBeenCalled()
      })
      const canvas = container.querySelector('canvas')
      expect(canvas).toBeInTheDocument()
      expect(canvas?.className).toContain('opacity-100')
      expect(bitmapClose).toHaveBeenCalled()
    } finally {
      HTMLCanvasElement.prototype.getContext = originalGetContext
    }
  })

  it('affiche un <img> de fallback si le fetch échoue', async () => {
    fetchMock.mockImplementationOnce(async () => {
      throw new Error('network error')
    })
    const { container } = await act(async () => {
      return render(<GifHoverThumbnail src="/broken.webp" isActive={false} alt="test" />)
    })
    await waitFor(() => {
      const img = container.querySelector('img[src="/broken.webp"]')
      expect(img).toBeInTheDocument()
    })
    expect(container.querySelector('canvas')?.className).toContain('opacity-0')
  })

  it('affiche le <img> animé au survol avec fragment #h=1', async () => {
    const { container } = render(<GifHoverThumbnail src="/active.webp" isActive={true} />)
    await waitFor(() => {
      expect(container.querySelector('img[src="/active.webp#h=1"]')).toBeInTheDocument()
    })
  })

  it('incrémente le fragment #h=N à chaque entrée de hover pour forcer un decoder neuf', async () => {
    const drawImageMock = vi.fn()
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      drawImage: drawImageMock,
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext

    try {
      const { container, rerender } = render(<GifHoverThumbnail src="/img.webp" isActive={true} />)
      await waitFor(() => {
        expect(container.querySelector('img[src="/img.webp#h=1"]')).toBeInTheDocument()
      })
      rerender(<GifHoverThumbnail src="/img.webp" isActive={false} />)
      rerender(<GifHoverThumbnail src="/img.webp" isActive={true} />)
      await waitFor(() => {
        expect(container.querySelector('img[src="/img.webp#h=2"]')).toBeInTheDocument()
      })
    } finally {
      HTMLCanvasElement.prototype.getContext = originalGetContext
    }
  })

  it('au survol, le canvas est masqué (opacity-0) même si prêt', async () => {
    const drawImageMock = vi.fn()
    const originalGetContext = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(() => ({
      drawImage: drawImageMock,
    })) as unknown as typeof HTMLCanvasElement.prototype.getContext

    try {
      const { container, rerender } = await act(async () => {
        return render(<GifHoverThumbnail src="/img.webp" isActive={false} />)
      })
      await waitFor(() => {
        expect(container.querySelector('canvas')?.className).toContain('opacity-100')
      })

      rerender(<GifHoverThumbnail src="/img.webp" isActive={true} />)
      expect(container.querySelector('canvas')?.className).toContain('opacity-0')
      await waitFor(() => {
        expect(container.querySelector('img[src="/img.webp#h=1"]')).toBeInTheDocument()
      })
    } finally {
      HTMLCanvasElement.prototype.getContext = originalGetContext
    }
  })
})
