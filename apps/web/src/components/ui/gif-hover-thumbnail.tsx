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
 * Trois couches superposées :
 *   1. Un <img> TOUJOURS monté sur `src` : c'est lui qui fait apparaître la
 *      vignette, dès que le navigateur a l'image. Une vignette ne peut donc
 *      jamais rester vide tant que le fichier est servi.
 *   2. Un <canvas> peint avec la première frame de ce même <img> (drawImage au
 *      `load`) : une fois prêt, il recouvre la couche 1 et fige l'image au repos.
 *   3. Au survol, un <img> monté sur une URL fragmentée distincte (`#h=N`), ce
 *      qui force le navigateur à instancier un decoder séparé et à relancer
 *      l'animation depuis la frame 0. Sans ça, Chrome partage le decoder entre
 *      instances pointant vers la même URL et l'animation reste figée (AWebP).
 *
 * POURQUOI PAS `fetch` + `createImageBitmap` (état antérieur, 2026-05-20).
 * Cette variante téléchargeait la miniature HORS du cache d'images du
 * navigateur et n'affichait RIEN tant que le fichier entier (~800 Ko par
 * vignette) n'était pas reçu puis décodé ; une requête interrompue laissait la
 * tuile vide DÉFINITIVEMENT et sans trace (la branche `AbortError` sortait en
 * silence, sans repli ni nouvel essai). Mesuré le 2026-09-13 sur l'accueil :
 * 20 tuiles, 18,7 Mo retéléchargés et 20 requêtes interrompues sur 20. Le
 * `load` du <img> donne le même poster sans aucun de ces défauts.
 */
export function GifHoverThumbnail({ src, isActive, alt = '', className }: GifHoverThumbnailProps) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null)
  const [posterReady, setPosterReady] = useState(false)
  const hoverCountRef = useRef(0)
  const [hoverKey, setHoverKey] = useState(0)

  // Reset du poster quand src change — ajustement pendant le rendu (pattern
  // React « prop précédente ») : le canvas de l'ancienne image ne doit pas
  // recouvrir la nouvelle.
  const [prevSrc, setPrevSrc] = useState(src)
  if (prevSrc !== src) {
    setPrevSrc(src)
    setPosterReady(false)
  }

  useEffect(() => {
    if (isActive) {
      hoverCountRef.current += 1
      setHoverKey(hoverCountRef.current)
    }
  }, [isActive])

  function paintPoster(event: React.SyntheticEvent<HTMLImageElement>) {
    const image = event.currentTarget
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    canvas.width = image.naturalWidth || 1
    canvas.height = image.naturalHeight || 1
    ctx.drawImage(image, 0, 0)
    setPosterReady(true)
  }

  return (
    <div className={className ?? 'relative h-full w-full'}>
      <img
        src={src}
        alt={alt}
        aria-hidden={alt === '' ? true : undefined}
        onLoad={paintPoster}
        className="absolute inset-0 h-full w-full object-cover"
      />
      <canvas
        ref={canvasRef}
        aria-hidden="true"
        className={`absolute inset-0 h-full w-full object-cover transition-opacity duration-150 ${
          posterReady && !isActive ? 'opacity-100' : 'opacity-0'
        }`}
      />
      {isActive && (
        <img
          key={`gif-active-${hoverKey}`}
          src={`${src}#h=${hoverKey}`}
          alt=""
          aria-hidden={true}
          className="absolute inset-0 h-full w-full object-cover"
        />
      )}
    </div>
  )
}
