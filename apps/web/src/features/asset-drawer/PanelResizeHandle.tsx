/**
 * Poignée de redimensionnement du volet des référentiels (bord gauche du volet) : séparateur vertical
 * accessible, déplaçable à la souris (la largeur suit le bord droit du volet) et au clavier (flèches,
 * Maj pour le grand pas, Début et Fin pour les bornes).
 */
import type { KeyboardEvent, PointerEvent, RefObject } from 'react'
import { useAssetDrawerStore } from './assetDrawer.store'
import { PANEL_KEY_STEP, PANEL_KEY_STEP_LARGE, PANEL_MIN_WIDTH } from './panelWidth'

interface PanelResizeHandleProps {
  /** Le volet : sa position donne le bord droit qui sert de repère au glissement. */
  panelRef: RefObject<HTMLElement | null>
  /** Largeur courante et borne haute effective, en pixels. */
  width: number
  max: number
  label: string
  dragging: boolean
  onDraggingChange: (dragging: boolean) => void
}

export function PanelResizeHandle({ panelRef, width, max, label, dragging, onDraggingChange }: PanelResizeHandleProps) {
  const setPanelWidth = useAssetDrawerStore((s) => s.setPanelWidth)

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
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
    onDraggingChange(true)
  }

  function onPointerMove(e: PointerEvent<HTMLDivElement>) {
    if (!dragging || !panelRef.current) return
    setPanelWidth(panelRef.current.getBoundingClientRect().right - e.clientX)
  }

  function onPointerUp(e: PointerEvent<HTMLDivElement>) {
    if (e.currentTarget.hasPointerCapture(e.pointerId)) e.currentTarget.releasePointerCapture(e.pointerId)
    onDraggingChange(false)
  }

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuemin={PANEL_MIN_WIDTH}
      aria-valuemax={max}
      aria-valuenow={width}
      tabIndex={0}
      onKeyDown={onKeyDown}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      className={`absolute inset-y-0 left-0 z-10 w-1.5 -translate-x-1/2 cursor-col-resize touch-none transition-colors hover:bg-primary/40 focus-visible:bg-primary/60 focus-visible:outline-none ${dragging ? 'bg-primary/60' : ''}`}
    />
  )
}
