import { useEffect, useRef, useState } from 'react'

interface GifHoverThumbnailProps {
  src: string
  isActive: boolean
  alt?: string
  className?: string
}

/**
 * Miniature GIF figée au repos, animée au survol.
 *
 * Au repos : un <canvas> peint avec la première frame du GIF (statique).
 * En actif (survol/focus) : le <img> réel remonté (key) pour relancer l'animation
 * depuis la frame 1 à chaque entrée.
 *
 * Rendu purement frontend, aucune regénération côté serveur requise.
 */
export function GifHoverThumbnail({ src, isActive, alt = '', className }: GifHoverThumbnailProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [posterReady, setPosterReady] = useState(false)

  useEffect(() => {
    setPosterReady(false)
    const canvas = canvasRef.current
    if (!canvas) return

    const img = new Image()
    img.crossOrigin = 'anonymous'
    let cancelled = false

    img.onload = () => {
      if (cancelled) return
      const ctx = canvas.getContext('2d')
      if (!ctx) return
      canvas.width = img.naturalWidth || 1
      canvas.height = img.naturalHeight || 1
      ctx.drawImage(img, 0, 0)
      setPosterReady(true)
    }
    img.onerror = () => {
      if (cancelled) return
      setPosterReady(false)
    }
    img.src = src

    return () => {
      cancelled = true
      img.onload = null
      img.onerror = null
    }
  }, [src])

  return (
    <div className={className ?? 'relative h-full w-full'}>
      <canvas
        ref={canvasRef}
        aria-hidden="true"
        className={`absolute inset-0 h-full w-full object-cover transition-opacity duration-150 ${
          isActive && posterReady ? 'opacity-0' : 'opacity-100'
        }`}
      />
      {isActive && (
        <img
          key="gif-active"
          src={src}
          alt={alt}
          aria-hidden={alt === '' ? true : undefined}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
    </div>
  )
}
