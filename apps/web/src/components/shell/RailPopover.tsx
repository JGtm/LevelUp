/**
 * RailPopover — infobulle et menu volant de la barre latérale repliée (`SidebarRail`).
 *
 * Les deux se placent en `position: fixed` à droite de leur ancre, mesurée à l'ouverture : la zone des
 * sections défile (`overflow-y-auto`), un panneau absolu y serait rogné. Le menu volant touche son
 * ancre (marge intérieure transparente, pas d'espace mort) : passer de l'icône au menu ne le ferme pas.
 *
 * - `RailTip` : une infobulle au survol et au focus, pour une entrée sans onglets.
 * - `RailMenu` : une entrée à onglets. Le survol ouvre le menu ; un clic (ou Entrée, Espace, flèche
 *   droite ou basse au clavier) l'épingle. Flèches haut et bas parcourent les entrées, Échap ferme et rend
 *   le focus au déclencheur, sortir du focus ferme. Au focus clavier sans ouverture, l'infobulle dit le
 *   libellé.
 */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type FocusEvent,
  type KeyboardEvent,
  type ReactNode,
} from 'react'

const TIP_GAP_PX = 8
const VIEWPORT_MARGIN_PX = 8

interface Anchor {
  /** Bord droit de l'ancre. */
  right: number
  /** Milieu vertical de l'ancre. */
  centerY: number
  top: number
}

function useAnchor() {
  const ref = useRef<HTMLDivElement>(null)
  const [anchor, setAnchor] = useState<Anchor | null>(null)
  const measure = useCallback(() => {
    const rect = ref.current?.getBoundingClientRect()
    if (rect) setAnchor({ right: rect.right, centerY: rect.top + rect.height / 2, top: rect.top })
  }, [])
  return { ref, anchor, measure }
}

const POPOVER_CLASS =
  'rounded-md border border-border bg-popover px-2 py-1 text-xs font-medium text-popover-foreground shadow-lg whitespace-nowrap'

function Tooltip({ label, anchor }: { label: string; anchor: Anchor }) {
  return (
    <span
      role="tooltip"
      style={{ position: 'fixed', left: anchor.right + TIP_GAP_PX, top: anchor.centerY, transform: 'translateY(-50%)' }}
      className={`pointer-events-none z-50 ${POPOVER_CLASS}`}
    >
      {label}
    </span>
  )
}

/** Une entrée sans menu : le libellé en infobulle au survol et au focus. */
export function RailTip({ label, children }: { label: string; children: ReactNode }) {
  const { ref, anchor, measure } = useAnchor()
  const [shown, setShown] = useState(false)
  const show = () => {
    measure()
    setShown(true)
  }
  return (
    <div
      ref={ref}
      onMouseEnter={show}
      onMouseLeave={() => setShown(false)}
      onFocusCapture={show}
      onBlurCapture={() => setShown(false)}
    >
      {children}
      {shown && anchor && <Tooltip label={label} anchor={anchor} />}
    </div>
  )
}

interface RailMenuProps {
  /** Libellé de l'entrée : titre du menu, nom accessible du déclencheur, infobulle au focus. */
  label: string
  /** Contenu du déclencheur (l'icône ou l'initiale). */
  trigger: ReactNode
  /** Classes du déclencheur (état actif compris). */
  triggerClassName: string
  /** L'entrée désigne la page courante (`aria-current`). */
  current?: boolean
  /** Entrées du menu : des éléments `role="menuitem"` (ou `menuitemradio`). Reçoit la fermeture. */
  children: (close: () => void) => ReactNode
}

export function RailMenu({ label, trigger, triggerClassName, current = false, children }: RailMenuProps) {
  const { ref, anchor, measure } = useAnchor()
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [hovered, setHovered] = useState(false)
  const [pinned, setPinned] = useState(false)
  const [focused, setFocused] = useState(false)
  // Incrémenté à chaque ouverture au clavier : l'effet qui donne le focus à la première entrée en dépend.
  const [focusToken, setFocusToken] = useState(0)
  const open = hovered || pinned

  const items = () => Array.from(menuRef.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]') ?? [])
  const close = useCallback(() => {
    setHovered(false)
    setPinned(false)
  }, [])

  // Un menu ouvert au clavier prend le focus sur sa première entrée ; ouvert au survol, il ne le vole pas.
  useEffect(() => {
    if (focusToken > 0) menuRef.current?.querySelector<HTMLElement>('[role^="menuitem"]')?.focus()
  }, [focusToken])

  // Un clic hors de l'entrée ferme le menu épinglé.
  useEffect(() => {
    if (!pinned) return
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) close()
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [pinned, close, ref])

  function openFromKeyboard() {
    measure()
    setPinned(true)
    setFocusToken((n) => n + 1)
  }

  function onTriggerKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      openFromKeyboard()
    }
  }

  function onTriggerClick(e: { detail: number }) {
    measure()
    if (e.detail === 0) {
      // Entrée ou Espace : le clic vient du clavier.
      if (pinned) close()
      else openFromKeyboard()
      return
    }
    setPinned((p) => !p)
  }

  function onMenuKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    if (e.key === 'Escape') {
      close()
      triggerRef.current?.focus()
      return
    }
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
    e.preventDefault()
    const list = items()
    if (list.length === 0) return
    const at = list.indexOf(document.activeElement as HTMLElement)
    list[(at + (e.key === 'ArrowDown' ? 1 : -1) + list.length) % list.length]?.focus()
  }

  function onBlur(e: FocusEvent<HTMLDivElement>) {
    setFocused(false)
    if (!e.currentTarget.contains(e.relatedTarget as Node | null)) close()
  }

  return (
    <div
      ref={ref}
      onMouseEnter={() => {
        measure()
        setHovered(true)
      }}
      onMouseLeave={() => setHovered(false)}
      onFocus={() => {
        measure()
        setFocused(true)
      }}
      onBlur={onBlur}
    >
      <button
        ref={triggerRef}
        type="button"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-current={current ? 'true' : undefined}
        onClick={onTriggerClick}
        onKeyDown={onTriggerKeyDown}
        className={triggerClassName}
      >
        {trigger}
      </button>
      {!open && focused && anchor && <Tooltip label={label} anchor={anchor} />}
      {open && anchor && (
        <div
          style={{
            position: 'fixed',
            left: anchor.right,
            top: Math.max(VIEWPORT_MARGIN_PX, anchor.top),
            maxHeight: `calc(100vh - ${2 * VIEWPORT_MARGIN_PX}px)`,
          }}
          className="z-50 overflow-y-auto pl-2"
        >
          <div
            ref={menuRef}
            role="menu"
            aria-label={label}
            onKeyDown={onMenuKeyDown}
            className="min-w-[10rem] rounded-md border border-border bg-popover py-1 text-popover-foreground shadow-lg"
          >
            <p className="px-3 py-1 text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</p>
            {children(close)}
          </div>
        </div>
      )}
    </div>
  )
}
