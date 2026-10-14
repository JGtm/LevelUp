import { useEffect, useState, type RefObject } from 'react'

/** useTailleDuCanvas — la taille affichée du canvas, suivie : le glisser et la molette convertissent des pixels en mètres. */
export function useTailleDuCanvas(ref: RefObject<HTMLCanvasElement | null>) {
  const [taille, setTaille] = useState({ width: 0, height: 0 })
  useEffect(() => {
    const canvas = ref.current
    if (!canvas) return
    const lire = () => setTaille({ width: canvas.clientWidth, height: canvas.clientHeight })
    lire()
    // Le patron du dépôt : l'observateur est optionnel (jsdom ne le fournit pas).
    if (typeof ResizeObserver === 'undefined') return
    const ro = new ResizeObserver(lire)
    ro.observe(canvas)
    return () => ro.disconnect()
  }, [ref])
  return taille
}
