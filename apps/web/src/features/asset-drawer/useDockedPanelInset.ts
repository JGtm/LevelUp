/**
 * Largeur occupée par le volet des référentiels ancré (application de bureau), en pixels : 0 tant
 * qu'il est fermé. Les éléments posés par rapport à la fenêtre (toasts) s'en servent pour rester dans
 * la colonne du contenu au lieu de couvrir le volet.
 */
import { useEffect, useState } from 'react'
import { useSidebarWidthStore } from '@/lib/layout/sidebarWidth'
import { useAssetDrawerStore } from './assetDrawer.store'
import { clampPanelWidth } from './panelWidth'

/** Largeur de la fenêtre, suivie au redimensionnement. */
export function useViewportWidth(): number {
  const [width, setWidth] = useState(() => window.innerWidth)
  useEffect(() => {
    const onResize = () => setWidth(window.innerWidth)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  return width
}

/** Largeur effective du volet (même bornes que `AssetDockedPanel`), 0 s'il est fermé. */
export function useDockedPanelInset(): number {
  const { isOpen, panelWidth } = useAssetDrawerStore()
  const viewport = useViewportWidth()
  const sidebarWidth = useSidebarWidthStore((s) => s.width)
  return isOpen ? clampPanelWidth(panelWidth, viewport, sidebarWidth) : 0
}
