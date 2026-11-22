/**
 * TacticalPlanSurface — L'IMAGE QUE LE RELIEF POSE À LA PLACE DU FOND `<img>` : le relief teinté, ou le plan / le
 * rendu modulé par la coupe. Le relief produit un canvas (pas une URL) : cette toile le recopie, au calage du fond,
 * dans un `<canvas>` qui prend EXACTEMENT la place de l'image (`cadrageDuFond`, rotation comprise) — le calque et
 * le fond ne peuvent donc pas se désaligner.
 *
 * Elle ne dessine que lorsque la surface change (une nouvelle frame du relief : un curseur qui glisse repeint la toile, jamais le plan).
 */
import { useLayoutEffect, useRef } from 'react'

import type { ReliefSurface } from '@/lib/replay/useReliefView'

import type { CadrageDuFond } from './plan.logic'

export function TacticalPlanSurface({ surface, cadrage }: { surface: ReliefSurface; cadrage: CadrageDuFond }) {
  const canvasRef = useRef<HTMLCanvasElement>(null)
  // Avant la peinture : une toile vide d'une image ferait clignoter le fond.
  useLayoutEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const { width, height } = surface.image
    if (!(width > 0) || !(height > 0)) return
    canvas.width = width
    canvas.height = height
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    ctx.clearRect(0, 0, width, height)
    ctx.drawImage(surface.image, 0, 0, width, height)
  }, [surface])
  return (
    <canvas
      ref={canvasRef}
      aria-hidden
      className="absolute max-w-none"
      style={cadrage}
      data-testid="tactical-plan-surface"
    />
  )
}
