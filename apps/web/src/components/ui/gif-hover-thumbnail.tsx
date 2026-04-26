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
  const [posterFailed, setPosterFailed] = useState(false)

  useEffect(() => {
    setPosterReady(false)
    setPosterFailed(false)
    const canvas = canvasRef.current
    if (!canvas) return

    const img = new Image()
    // Pas de crossOrigin : on ne lit pas les pixels (toDataURL/getImageData),
    // donc inutile et casse le canvas si le serveur ne sert pas les headers CORS.
    let cancelled = false

    img.onload = () => {
      if (cancelled) return
      const ctx = canvas.getContext('2d')
      if (!ctx) {
        setPosterFailed(true)
        return
      }
      try {
        canvas.width = img.naturalWidth || 1
        canvas.height = img.naturalHeight || 1
        ctx.drawImage(img, 0, 0)
        setPosterReady(true)
      } catch {
        setPosterFailed(true)
      }
    }
    img.onerror = () => {
      if (cancelled) return
      setPosterFailed(true)
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
          posterReady && !isActive ? 'opacity-100' : 'opacity-0'
        }`}
      />
      {/* Fallback img au repos si le canvas a échoué (CORS, image taintée, etc) */}
      {posterFailed && !isActive && (
        <img
          src={src}
          alt={alt}
          aria-hidden={alt === '' ? true : undefined}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
      {/* Img animée au survol */}
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
