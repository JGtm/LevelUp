import { useState, useRef, useEffect, useLayoutEffect, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { useAppShellStore } from '@/stores/appShellStore'
import { formatMessage } from '@/lib/i18n/format'
import { commonManifest, type CommonManifestKey } from '@/lib/i18n/generated/common'

interface InfoTooltipProps {
  /** Texte ou contenu affiché dans le tooltip */
  content: ReactNode
  /** Taille de l'icône ⓘ en classes Tailwind (défaut: "w-4 h-4") */
  iconClass?: string
}

const PANEL_WIDTH = 256 // px — équivalent de l'ancien w-64
const GAP = 8 // marge bouton/panneau et marge de viewport

interface PanelPos {
  top: number
  left: number
  /** Décalage horizontal de la flèche depuis le bord gauche du panneau (pointe vers le bouton). */
  arrowLeft: number
  /** true = panneau replié SOUS le bouton (flèche en haut) ; false = au-dessus (flèche en bas). */
  below: boolean
}

/**
 * Icône ⓘ qui affiche un tooltip explicatif au hover/focus.
 *
 * Le panneau est rendu dans `document.body` via `createPortal` + `position: fixed`
 * pour échapper aux parents `overflow:hidden` (ex. `KpiCard`, cellules de tableau) qui
 * clippaient l'ancien rendu inline. Position calculée depuis le rect du bouton : centré
 * au-dessus, repli en dessous s'il n'y a pas la place. Fermeture : clic extérieur, scroll,
 * resize, blur/mouseleave. `role="tooltip"` + aria conservés.
 */
export function InfoTooltip({ content, iconClass = 'w-4 h-4' }: InfoTooltipProps) {
  const [open, setOpen] = useState(false)
  const buttonRef = useRef<HTMLButtonElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [pos, setPos] = useState<PanelPos | null>(null)
  const locale = useAppShellStore((s) => s.locale)
  const t = (key: CommonManifestKey) => formatMessage(commonManifest, key, locale)

  // Position calculée depuis le rect du bouton (centré au-dessus, repli en dessous),
  // clampée dans la viewport ; la flèche pointe vers le centre du bouton.
  useLayoutEffect(() => {
    if (!open || !buttonRef.current) return
    const b = buttonRef.current.getBoundingClientRect()
    const panelH = panelRef.current?.getBoundingClientRect().height ?? 0
    const buttonCenterX = b.left + b.width / 2
    let left = buttonCenterX - PANEL_WIDTH / 2
    left = Math.max(GAP, Math.min(left, window.innerWidth - PANEL_WIDTH - GAP))
    const above = b.top - panelH - GAP
    const below = above < GAP // pas la place au-dessus → repli sous le bouton
    const top = below ? b.bottom + GAP : above
    const arrowLeft = Math.max(12, Math.min(buttonCenterX - left, PANEL_WIDTH - 12))
    setPos({ top, left, arrowLeft, below })
  }, [open, content])

  // Fermeture : clic extérieur (bouton + panneau exclus), scroll (capture), resize.
  useEffect(() => {
    if (!open) return
    function handleClick(e: MouseEvent) {
      const target = e.target as Node
      if (buttonRef.current?.contains(target)) return
      if (panelRef.current?.contains(target)) return
      setOpen(false)
    }
    function close() {
      setOpen(false)
    }
    document.addEventListener('mousedown', handleClick)
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
    return () => {
      document.removeEventListener('mousedown', handleClick)
      window.removeEventListener('scroll', close, true)
      window.removeEventListener('resize', close)
    }
  }, [open])

  return (
    <span className="inline-flex items-center">
      <button
        ref={buttonRef}
        type="button"
        className={`${iconClass} inline-flex items-center justify-center rounded-full border border-input text-muted-foreground hover:text-foreground hover:border-border text-2xs font-bold leading-none cursor-help transition-colors`}
        onMouseEnter={() => setOpen(true)}
        onMouseLeave={() => setOpen(false)}
        onFocus={() => setOpen(true)}
        onBlur={() => setOpen(false)}
        onClick={() => setOpen((v) => !v)}
        aria-label={t('common.tooltip.more_info_aria')}
      >
        i
      </button>
      {open &&
        createPortal(
          <div
            ref={panelRef}
            role="tooltip"
            style={{
              position: 'fixed',
              top: pos?.top ?? -9999,
              left: pos?.left ?? -9999,
              width: PANEL_WIDTH,
              visibility: pos ? 'visible' : 'hidden',
            }}
            className="z-[9999] rounded-lg border border-border bg-background p-3 text-xs font-normal normal-case tracking-normal text-foreground shadow-lg"
          >
            {content}
            {pos?.below ? (
              <div
                className="absolute bottom-full -translate-x-1/2 -mb-px border-4 border-transparent border-b-background"
                style={{ left: pos.arrowLeft }}
              />
            ) : (
              <div
                className="absolute top-full -translate-x-1/2 -mt-px border-4 border-transparent border-t-background"
                style={{ left: pos?.arrowLeft ?? PANEL_WIDTH / 2 }}
              />
            )}
          </div>,
          document.body,
        )}
    </span>
  )
}
