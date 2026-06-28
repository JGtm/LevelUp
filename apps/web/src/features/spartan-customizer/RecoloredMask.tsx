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
 * Affiche un masque Spartan recolorisé en direct via canvas (modèle additif,
 * cf. recolor.ts). L'image est chargée quand `src` change ; le canvas est
 * (re)dessiné quand l'image est prête OU quand une couleur change. Le canvas est
 * rendu à la taille native du masque ; la taille d'affichage est pilotée par
 * `className` (CSS).
 */
export function RecoloredMask({ src, colors, alt, className }: RecoloredMaskProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [img, setImg] = useState<HTMLImageElement | null>(null)
  const { primary, secondary, tertiary } = colors

  // Chargement de l'image (déclenché par src)
  useEffect(() => {
    let cancelled = false
    const image = new Image()
    image.onload = () => {
      if (!cancelled) setImg(image)
    }
    image.src = src
    return () => {
      cancelled = true
    }
  }, [src])

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

  return <canvas ref={canvasRef} role="img" aria-label={alt} className={className} />
}
