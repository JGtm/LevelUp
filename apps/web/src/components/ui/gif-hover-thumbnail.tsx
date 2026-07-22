import { useEffect, useRef, useState } from 'react'

interface GifHoverThumbnailProps {
  src: string
  isActive: boolean
  alt?: string
  className?: string
}

/**
 * Miniature animée (GIF ou WebP) figée au repos, animée au survol.
 *
 * Au repos : un <canvas> peint avec la première frame via fetch+createImageBitmap
 *   (statique, n'amorce pas le decoder d'animation du browser).
 * Au survol : un <img> monté avec une URL fragmentée distincte (`#h=N`), ce
 *   qui force le browser à instancier un decoder séparé et relancer l'animation
 *   depuis la frame 0. Sans ça, Chrome partage le decoder entre instances
 *   pointant vers la même URL et l'animation reste figée (notable sur AWebP).
 */
export function GifHoverThumbnail({ src, isActive, alt = '', className }: GifHoverThumbnailProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [posterReady, setPosterReady] = useState(false)
  const [posterFailed, setPosterFailed] = useState(false)
  const hoverCountRef = useRef(0)
  const [hoverKey, setHoverKey] = useState(0)

  // Reset de l'état poster quand src change — ajustement pendant le rendu (pattern
  // React « prop précédente ») au lieu d'un setState en tête d'effet, avant le
  // fetch asynchrone du poster ci-dessous.
  const [prevSrc, setPrevSrc] = useState(src)
  if (prevSrc !== src) {
    setPrevSrc(src)
    setPosterReady(false)
    setPosterFailed(false)
  }

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return

    let cancelled = false
    const controller = new AbortController()

    void (async () => {
      try {
        const resp = await fetch(src, { signal: controller.signal })
        if (!resp.ok) throw new Error(`http ${resp.status}`)
        const blob = await resp.blob()
        if (cancelled) return
        const bitmap = await createImageBitmap(blob)
        if (cancelled) {
          bitmap.close()
          return
        }
        const ctx = canvas.getContext('2d')
        if (!ctx) {
          bitmap.close()
          setPosterFailed(true)
          return
        }
        canvas.width = bitmap.width || 1
        canvas.height = bitmap.height || 1
        ctx.drawImage(bitmap, 0, 0)
        bitmap.close()
        setPosterReady(true)
      } catch (err) {
        if (cancelled || (err instanceof Error && err.name === 'AbortError')) return
        setPosterFailed(true)
      }
    })()

    return () => {
      cancelled = true
      controller.abort()
    }
  }, [src])

  useEffect(() => {
    if (isActive) {
      hoverCountRef.current += 1
      setHoverKey(hoverCountRef.current)
    }
  }, [isActive])

  return (
    <div className={className ?? 'relative h-full w-full'}>
      <canvas
        ref={canvasRef}
        aria-hidden="true"
        className={`absolute inset-0 h-full w-full object-cover transition-opacity duration-150 ${
          posterReady && !isActive ? 'opacity-100' : 'opacity-0'
        }`}
      />
      {posterFailed && !isActive && (
        <img
          src={src}
          alt={alt}
          aria-hidden={alt === '' ? true : undefined}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
      {isActive && (
        <img
          key={`gif-active-${hoverKey}`}
          src={`${src}#h=${hoverKey}`}
          alt={alt}
          aria-hidden={alt === '' ? true : undefined}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
    </div>
  )
}
