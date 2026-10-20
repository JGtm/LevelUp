/**
 * Tests GifHoverThumbnail — vérifie que la vignette est TOUJOURS montée (une
 * tuile ne peut pas rester vide), que le poster fige l'image au repos, et le
 * mécanisme de fragment URL `#h=N` qui relance l'animation au survol.
 */
import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, waitFor } from '@testing-library/react'
import { GifHoverThumbnail } from './gif-hover-thumbnail'

/** Remplace getContext par un espion et rend le test responsable du nettoyage. */
function withCanvasContext(run: (drawImage: ReturnType<typeof vi.fn>) => Promise<void> | void) {
  const drawImage = vi.fn()
  const original = HTMLCanvasElement.prototype.getContext
  HTMLCanvasElement.prototype.getContext = vi.fn(() => ({ drawImage })) as unknown as
    typeof HTMLCanvasElement.prototype.getContext
  return Promise.resolve(run(drawImage)).finally(() => {
    HTMLCanvasElement.prototype.getContext = original
  })
}

describe('GifHoverThumbnail', () => {
  it("monte l'image de la vignette immédiatement, avant tout décodage", () => {
    const { container } = render(<GifHoverThumbnail src="/img.webp" isActive={false} />)
    expect(container.querySelector('img[src="/img.webp"]')).toBeInTheDocument()
    // Poster pas encore peint : le canvas ne masque pas encore l'image.
    expect(container.querySelector('canvas')?.className).toContain('opacity-0')
  })

  it('peint le poster depuis cette image au chargement et le montre au repos', async () => {
    await withCanvasContext(async (drawImage) => {
      const { container } = render(<GifHoverThumbnail src="/img.webp" isActive={false} />)
      const img = container.querySelector('img[src="/img.webp"]') as HTMLImageElement
      fireEvent.load(img)
      await waitFor(() => {
        expect(drawImage).toHaveBeenCalledWith(img, 0, 0)
        expect(container.querySelector('canvas')?.className).toContain('opacity-100')
      })
    })
  })

  it("laisse l'image visible si le canvas n'a pas de contexte 2D", async () => {
    const original = HTMLCanvasElement.prototype.getContext
    HTMLCanvasElement.prototype.getContext = vi.fn(
      () => null,
    ) as unknown as typeof HTMLCanvasElement.prototype.getContext
    try {
      const { container } = render(<GifHoverThumbnail src="/broken.webp" isActive={false} alt="test" />)
      fireEvent.load(container.querySelector('img[src="/broken.webp"]') as HTMLImageElement)
      await waitFor(() => {
        expect(container.querySelector('img[src="/broken.webp"]')).toBeInTheDocument()
        expect(container.querySelector('canvas')?.className).toContain('opacity-0')
      })
    } finally {
      HTMLCanvasElement.prototype.getContext = original
    }
  })

  it('affiche le <img> animé au survol avec fragment #h=1', async () => {
    const { container } = render(<GifHoverThumbnail src="/active.webp" isActive={true} />)
    await waitFor(() => {
      expect(container.querySelector('img[src="/active.webp#h=1"]')).toBeInTheDocument()
    })
  })

  it('incrémente le fragment #h=N à chaque entrée de hover pour forcer un decoder neuf', async () => {
    const { container, rerender } = render(<GifHoverThumbnail src="/img.webp" isActive={true} />)
    await waitFor(() => {
      expect(container.querySelector('img[src="/img.webp#h=1"]')).toBeInTheDocument()
    })
    rerender(<GifHoverThumbnail src="/img.webp" isActive={false} />)
    rerender(<GifHoverThumbnail src="/img.webp" isActive={true} />)
    await waitFor(() => {
      expect(container.querySelector('img[src="/img.webp#h=2"]')).toBeInTheDocument()
    })
  })

  it('au survol, le canvas est masqué (opacity-0) même si prêt', async () => {
    await withCanvasContext(async () => {
      const { container, rerender } = render(<GifHoverThumbnail src="/img.webp" isActive={false} />)
      fireEvent.load(container.querySelector('img[src="/img.webp"]') as HTMLImageElement)
      await waitFor(() => {
        expect(container.querySelector('canvas')?.className).toContain('opacity-100')
      })

      rerender(<GifHoverThumbnail src="/img.webp" isActive={true} />)
      expect(container.querySelector('canvas')?.className).toContain('opacity-0')
      await waitFor(() => {
        expect(container.querySelector('img[src="/img.webp#h=1"]')).toBeInTheDocument()
      })
    })
  })

  it('réinitialise le poster quand la source change', async () => {
    await withCanvasContext(async () => {
      const { container, rerender } = render(<GifHoverThumbnail src="/a.webp" isActive={false} />)
      fireEvent.load(container.querySelector('img[src="/a.webp"]') as HTMLImageElement)
      await waitFor(() => {
        expect(container.querySelector('canvas')?.className).toContain('opacity-100')
      })
      rerender(<GifHoverThumbnail src="/b.webp" isActive={false} />)
      expect(container.querySelector('img[src="/b.webp"]')).toBeInTheDocument()
      expect(container.querySelector('canvas')?.className).toContain('opacity-0')
    })
  })
})
