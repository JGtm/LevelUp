/**
 * Volet des référentiels ancré à droite du contenu (application de bureau, capacité
 * `can_use_compact_chrome`). Il POUSSE le contenu : l'`AppShell` le range dans la même rangée que la
 * barre latérale et la page, rien ne se superpose. Il reste ouvert de page en page tant qu'on ne le
 * ferme pas (état persisté par `assetDrawer.store`), se redimensionne par sa poignée (souris, flèches
 * du clavier) et se rouvre au raccourci `Ctrl+Maj+L`.
 */
import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { useAssetDrawerStore } from './assetDrawer.store'
import { ASSET_PANEL_ID, closeAssetPanel, useAssetPanelShortcut } from './assetPanel'
import { AssetPanelContent } from './AssetPanelContent'
import { PanelResizeHandle } from './PanelResizeHandle'
import { clampPanelWidth, maxPanelWidth } from './panelWidth'

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
  const { isOpen, panelWidth, focusToken } = useAssetDrawerStore()
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
      <PanelResizeHandle
        panelRef={panelRef}
        width={width}
        max={max}
        label={t('asset_drawer.docked.resize_aria')}
        onDraggingChange={setDragging}
        dragging={dragging}
      />
      <AssetPanelContent onClose={closeAssetPanel} searchFirst searchRef={searchRef} />
    </aside>
  )
}
