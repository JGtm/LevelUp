/**
 * Volet des référentiels ancré à droite du contenu (application de bureau, capacité
 * `can_use_compact_chrome`). Il POUSSE le contenu : l'`AppShell` le range dans la même rangée que la
 * barre latérale et la page, rien ne se superpose. Il reste ouvert de page en page tant qu'on ne le
 * ferme pas (état persisté par `assetDrawer.store`), se redimensionne par sa poignée (souris, flèches
 * du clavier) et se rouvre au raccourci `Ctrl+Maj+L`.
 */
import { useEffect, useRef, useState, type KeyboardEvent, type PointerEvent } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { useAssetDrawerStore } from './assetDrawer.store'
import { ASSET_PANEL_ID, closeAssetPanel, useAssetPanelShortcut } from './assetPanel'
import { AssetPanelContent } from './AssetPanelContent'
import {
  PANEL_KEY_STEP,
  PANEL_KEY_STEP_LARGE,
  PANEL_MIN_WIDTH,
  clampPanelWidth,
  maxPanelWidth,
} from './panelWidth'

function useViewportWidth(): number {
  const [width, setWidth] = useState(() => window.innerWidth)
  useEffect(() => {
    const onResize = () => setWidth(window.innerWidth)
    window.addEventListener('resize', onResize)
    return () => window.removeEventListener('resize', onResize)
  }, [])
  return width
}

export function AssetDockedPanel() {
  const locale = useAppShellStore((s) => s.locale)
  const { isOpen, panelWidth, setPanelWidth, focusToken } = useAssetDrawerStore()
  const viewport = useViewportWidth()
  const panelRef = useRef<HTMLElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const t = (key: keyof typeof assetDrawerManifest) => formatMessage(assetDrawerManifest, key, locale)

  useAssetPanelShortcut()

  // Ouverture demandée (bouton, raccourci) : le focus va au champ de recherche.
  useEffect(() => {
    if (isOpen && focusToken > 0) searchRef.current?.focus()
  }, [isOpen, focusToken])

  if (!isOpen) return null

  const width = clampPanelWidth(panelWidth, viewport)
  const max = maxPanelWidth(viewport)

  function onPanelKeyDown(e: KeyboardEvent<HTMLElement>) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      closeAssetPanel()
    }
  }

  function onHandleKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const step = e.shiftKey ? PANEL_KEY_STEP_LARGE : PANEL_KEY_STEP
    // Le volet est à droite : la flèche gauche l'élargit, la flèche droite le rétrécit.
    if (e.key === 'ArrowLeft') setPanelWidth(width + step)
    else if (e.key === 'ArrowRight') setPanelWidth(width - step)
    else if (e.key === 'Home') setPanelWidth(PANEL_MIN_WIDTH)
    else if (e.key === 'End') setPanelWidth(max)
    else return
    e.preventDefault()
  }

  function onPointerDown(e: PointerEvent<HTMLDivElement>) {
    e.currentTarget.setPointerCapture(e.pointerId)
    setDragging(true)
  }

  function onPointerMove(e: PointerEvent<HTMLDivElement>) {
    if (!dragging || !panelRef.current) return
    setPanelWidth(panelRef.current.getBoundingClientRect().right - e.clientX)
  }

  function onPointerUp(e: PointerEvent<HTMLDivElement>) {
    if (e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId)
    setDragging(false)
  }

  return (
    <aside
      ref={panelRef}
      id={ASSET_PANEL_ID}
      role="complementary"
      aria-label={t('asset_drawer.docked.aria')}
      onKeyDown={onPanelKeyDown}
      style={{ width }}
      className={`relative flex h-full shrink-0 flex-col border-l border-border bg-popover text-popover-foreground ${dragging ? 'select-none' : ''}`}
    >
      <div
        role="separator"
        aria-orientation="vertical"
        aria-label={t('asset_drawer.docked.resize_aria')}
        aria-valuemin={PANEL_MIN_WIDTH}
        aria-valuemax={max}
        aria-valuenow={width}
        tabIndex={0}
        onKeyDown={onHandleKeyDown}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onPointerCancel={onPointerUp}
        className={`absolute inset-y-0 left-0 z-10 w-1.5 -translate-x-1/2 cursor-col-resize touch-none transition-colors hover:bg-primary/40 focus-visible:bg-primary/60 focus-visible:outline-none ${dragging ? 'bg-primary/60' : ''}`}
      />
      <AssetPanelContent onClose={closeAssetPanel} searchFirst searchRef={searchRef} />
    </aside>
  )
}
