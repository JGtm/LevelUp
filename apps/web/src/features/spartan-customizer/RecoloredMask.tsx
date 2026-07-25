import { useEffect, useRef, useState } from 'react'

import { recolorMaskData, type RecolorColors } from './recolor'

interface RecoloredMaskProps {
  /** URL du masque (same-origin → canvas non tainté, getImageData OK). */
  src: string
  colors: RecolorColors
  alt: string
  className?: string
}

/**
 * Affiche un masque Spartan recolorisé en direct via canvas (modèle additif, cf.
 * recolor.ts). LAZY : l'image n'est chargée/dessinée que lorsque le canvas approche le
 * viewport (IntersectionObserver) — indispensable pour la grille de la modale (300+
 * masques recolorisés en direct). Fallback eager si IntersectionObserver est indisponible
 * (jsdom/tests). Le canvas est rendu à la taille native du masque ; la taille d'affichage
 * vient de `className` (CSS). (Re)dessin à la disponibilité de l'image OU au changement
 * de couleur.
 */
export function RecoloredMask({ src, colors, alt, className }: RecoloredMaskProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [img, setImg] = useState<HTMLImageElement | null>(null)
  // Lazy : visible d'emblée si IntersectionObserver est indisponible (jsdom/SSR),
  // sinon on attend l'entrée dans le viewport (init paresseux → pas de setState en effet).
  const [visible, setVisible] = useState(() => typeof IntersectionObserver === 'undefined')
  const { primary, secondary, tertiary } = colors

  // Observe l'entrée dans le viewport tant que pas encore visible.
  useEffect(() => {
    if (visible) return
    const canvas = canvasRef.current
    if (!canvas) return
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) setVisible(true)
      },
      { rootMargin: '200px' },
    )
    io.observe(canvas)
    return () => io.disconnect()
  }, [visible])

  // Chargement de l'image (une fois visible, ou quand src change)
  useEffect(() => {
    if (!visible) return
    let cancelled = false
    const image = new Image()
    image.onload = () => {
      if (!cancelled) setImg(image)
    }
    image.src = src
    return () => {
      cancelled = true
    }
  }, [src, visible])

  // (Re)dessin : à la disponibilité de l'image et à chaque changement de couleur
  useEffect(() => {
    const canvas = canvasRef.current
    if (!img || !canvas) return
    canvas.width = img.naturalWidth
    canvas.height = img.naturalHeight
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, canvas.width, canvas.height)
    ctx.drawImage(img, 0, 0)
    const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height)
    imageData.data.set(recolorMaskData(imageData.data, { primary, secondary, tertiary }))
    ctx.putImageData(imageData, 0, 0)
  }, [img, primary, secondary, tertiary])

  // `data-mask-src` expose l'URL de l'asset title-scopé dans le DOM (le rendu passe par
  // canvas + `new Image()`, donc la src n'y apparaîtrait pas sinon) : indispensable pour
  // que les garde-rails anti-fuite cross-titre (V72-29) assertent qu'aucun asset d'un
  // autre titre n'est rendu quand le titre courant ne le déclare pas.
  return (
    <canvas ref={canvasRef} role="img" aria-label={alt} className={className} data-mask-src={src} />
  )
}
