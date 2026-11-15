/**
 * Volet des référentiels ancré à droite du contenu (application de bureau, capacité
 * `can_use_compact_chrome`). Toujours monté : il glisse depuis le bord droit à l'ouverture et à la
 * fermeture, jamais de saut.
 *
 * Deux modes, mémorisés par `assetDrawer.store` :
 * - épinglé : il POUSSE le contenu (la fente qui le porte prend sa largeur dans la rangée de
 *   l'`AppShell`), reste ouvert de page en page jusqu'à sa fermeture ;
 * - non épinglé : il se SUPERPOSE au contenu (la fente reste à largeur nulle) sous un voile
 *   transparent dont le clic le referme. Il ne s'ouvre alors que pour la consultation.
 *
 * Fermé, le panneau est hors de l'écran, `inert` et `aria-hidden` : ni focus ni lecteur d'écran. Il
 * se redimensionne par sa poignée (souris, flèches du clavier) et se rouvre au raccourci `Ctrl+Maj+L`.
 */
import { useEffect, useLayoutEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useAppShellStore } from '@/stores/appShellStore'
import { useSidebarWidthStore } from '@/lib/layout/sidebarWidth'
import { DOCK_TRANSITION_MS, beginLayoutTransition, endLayoutTransition } from '@/lib/layout/layoutSettling'
import { formatMessage } from '@/lib/i18n/format'
import { assetDrawerManifest } from '@/lib/i18n/generated/asset_drawer'
import { useAssetDrawerStore } from './assetDrawer.store'
import { ASSET_PANEL_ID, closeAssetPanel, useAssetPanelShortcut } from './assetPanel'
import { AssetPanelContent } from './AssetPanelContent'
import { PanelResizeHandle } from './PanelResizeHandle'
import { clampPanelWidth, maxPanelWidth } from './panelWidth'
import { useViewportWidth } from './useDockedPanelInset'

export function AssetDockedPanel() {
  const locale = useAppShellStore((s) => s.locale)
  const { isOpen, pinned, panelWidth, focusToken, close, setPinned } = useAssetDrawerStore()
  const viewport = useViewportWidth()
  const sidebarWidth = useSidebarWidthStore((s) => s.width)
  const panelRef = useRef<HTMLElement>(null)
  const searchRef = useRef<HTMLInputElement>(null)
  const [dragging, setDragging] = useState(false)
  const t = (key: keyof typeof assetDrawerManifest) => formatMessage(assetDrawerManifest, key, locale)

  const width = clampPanelWidth(panelWidth, viewport, sidebarWidth)
  const max = maxPanelWidth(viewport, sidebarWidth)
  // Largeur que le volet prend au contenu : la sienne s'il est ouvert ET épinglé, rien sinon.
  const slotWidth = isOpen && pinned ? width : 0

  useAssetPanelShortcut()

  // Ouverture demandée (bouton, raccourci) : le focus va au champ de recherche. `preventScroll` : le
  // panneau commence sa course hors de l'écran, un défilement du conteneur le ferait sauter.
  useEffect(() => {
    if (isOpen && focusToken > 0) searchRef.current?.focus({ preventScroll: true })
  }, [isOpen, focusToken])

  // La largeur de la fente s'anime : les graphes attendent la fin plutôt que de se redessiner à chaque
  // image. Pas de transition (donc pas de déclaration) pendant le glissement de la poignée.
  const previousSlotWidth = useRef(slotWidth)
  useLayoutEffect(() => {
    if (previousSlotWidth.current === slotWidth) return
    previousSlotWidth.current = slotWidth
    if (!dragging) beginLayoutTransition()
  }, [slotWidth, dragging])

  function onPanelKeyDown(e: KeyboardEvent<HTMLElement>) {
    if (e.key === 'Escape') {
      e.stopPropagation()
      closeAssetPanel()
    }
  }

  const animation = { transitionDuration: `${DOCK_TRANSITION_MS}ms` }

  return (
    <div
      data-testid="asset-panel-slot"
      style={{ width: slotWidth, ...animation }}
      onTransitionEnd={(e) => {
        if (e.target === e.currentTarget) endLayoutTransition()
      }}
      className={`relative h-full shrink-0 ease-out ${dragging ? '' : 'transition-[width] motion-reduce:transition-none'}`}
    >
      {isOpen && !pinned && (
        <div data-testid="asset-panel-scrim" className="fixed inset-0 z-[49]" onClick={close} aria-hidden="true" />
      )}
      <aside
        ref={panelRef}
        id={ASSET_PANEL_ID}
        role="complementary"
        aria-label={t('asset_drawer.docked.aria')}
        aria-hidden={!isOpen}
        inert={!isOpen}
        onKeyDown={onPanelKeyDown}
        style={{ width, ...animation }}
        className={`absolute inset-y-0 right-0 flex flex-col border-l border-border bg-popover text-popover-foreground transition-transform ease-out motion-reduce:transition-none ${
          isOpen ? 'translate-x-0' : 'translate-x-full'
        } ${pinned ? '' : 'z-50 shadow-xl'} ${dragging ? 'select-none' : ''}`}
      >
        <PanelResizeHandle
          panelRef={panelRef}
          width={width}
          max={max}
          label={t('asset_drawer.docked.resize_aria')}
          onDraggingChange={setDragging}
          dragging={dragging}
        />
        <AssetPanelContent
          onClose={closeAssetPanel}
          searchFirst
          searchRef={searchRef}
          pinned={pinned}
          onTogglePin={() => setPinned(!pinned)}
        />
      </aside>
    </div>
  )
}
