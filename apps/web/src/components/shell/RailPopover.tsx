/**
 * RailPopover — infobulle et menu volant de la barre latérale repliée (`SidebarRail`).
 *
 * Les deux se placent en `position: fixed` à droite de leur ancre, mesurée à l'ouverture : la zone des
 * sections défile (`overflow-y-auto`), un panneau absolu y serait rogné. Le menu volant touche son
 * ancre (marge intérieure transparente, pas d'espace mort) : passer de l'icône au menu ne le ferme pas.
 *
 * - `RailTip` : une infobulle au survol et au focus clavier, pour une entrée sans onglets.
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
  type CSSProperties,
  type FocusEvent,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
} from 'react'
import { isKeyboardFocus } from '@/lib/dom/focusVisible'

const TIP_GAP_PX = 8
const VIEWPORT_MARGIN_PX = 8

interface Anchor {
  /** Bord droit de l'ancre. */
  right: number
  /** Milieu vertical de l'ancre. */
  centerY: number
  top: number
  /** Bord bas de l'ancre. */
  bottom: number
}

function useAnchor() {
  const ref = useRef<HTMLDivElement>(null)
  const [anchor, setAnchor] = useState<Anchor | null>(null)
  const measure = useCallback(() => {
    const rect = ref.current?.getBoundingClientRect()
    if (rect) setAnchor({ right: rect.right, centerY: rect.top + rect.height / 2, top: rect.top, bottom: rect.bottom })
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

/** Une entrée sans menu : le libellé en infobulle au survol et au focus clavier (pas au focus rendu après un clic). */
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
      onFocusCapture={(e) => {
        if (isKeyboardFocus(e.target)) show()
      }}
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
  /**
   * Bord de l'ancre auquel le menu s'aligne : `top` (défaut) le fait descendre depuis le haut de l'ancre,
   * `bottom` le fait monter depuis son bas. Une entrée du pied de barre prend `bottom` : un menu qui
   * descendrait sortirait de la fenêtre.
   */
  placement?: 'top' | 'bottom'
  /** Entrées du menu : des éléments `role="menuitem"` (ou `menuitemradio`, `menuitemcheckbox`). Reçoit la fermeture. */
  children: (close: () => void) => ReactNode
}

/** Position fixe du conteneur du menu, calée sur le bord haut ou bas de l'ancre et bornée par la fenêtre. */
function menuPlacementStyle(anchor: Anchor, placement: 'top' | 'bottom'): CSSProperties {
  const maxHeight = `calc(100vh - ${2 * VIEWPORT_MARGIN_PX}px)`
  if (placement === 'bottom') {
    return { position: 'fixed', left: anchor.right, bottom: Math.max(VIEWPORT_MARGIN_PX, window.innerHeight - anchor.bottom), maxHeight }
  }
  return { position: 'fixed', left: anchor.right, top: Math.max(VIEWPORT_MARGIN_PX, anchor.top), maxHeight }
}

/** Un menu ouvert au clavier prend le focus sur sa première entrée ; ouvert au survol, il ne le vole pas. */
function useFocusFirstItem(menuRef: RefObject<HTMLDivElement | null>, token: number) {
  useEffect(() => {
    if (token > 0) menuRef.current?.querySelector<HTMLElement>('[role^="menuitem"]')?.focus()
  }, [menuRef, token])
}

/** Un clic hors de l'entrée ferme le menu épinglé. */
function useDismissOnOutsidePress(ref: RefObject<HTMLDivElement | null>, active: boolean, close: () => void) {
  useEffect(() => {
    if (!active) return
    function onDown(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) close()
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [active, close, ref])
}

/**
 * L'état d'un menu volant : ouvert par le survol (`hovered`) ou épinglé (`pinned`, clic ou clavier), focus
 * de l'entrée, jeton qui donne le focus à la première entrée à une ouverture au clavier, et les gestes
 * (clic, flèches, Échap, sortie du focus). Rend aussi les références du conteneur, du déclencheur et du menu.
 */
function useRailMenu() {
  const { ref, anchor, measure } = useAnchor()
  const triggerRef = useRef<HTMLButtonElement>(null)
  const menuRef = useRef<HTMLDivElement>(null)
  const [hovered, setHovered] = useState(false)
  const [pinned, setPinned] = useState(false)
  const [focused, setFocused] = useState(false)
  // Incrémenté à chaque ouverture au clavier : l'effet qui donne le focus à la première entrée en dépend.
  const [focusToken, setFocusToken] = useState(0)
  const open = hovered || pinned

  const close = useCallback(() => {
    setHovered(false)
    setPinned(false)
  }, [])

  useFocusFirstItem(menuRef, focusToken)
  useDismissOnOutsidePress(ref, pinned, close)

  function openFromKeyboard() {
    measure()
    setPinned(true)
    setFocusToken((n) => n + 1)
  }

  const onTriggerKeyDown = (e: KeyboardEvent<HTMLButtonElement>) => {
    // Menu épinglé à la souris, focus resté sur le bouton : Échap le ferme comme au clavier dans le menu.
    if (e.key === 'Escape' && open) {
      close()
      return
    }
    if (e.key === 'ArrowRight' || e.key === 'ArrowDown') {
      e.preventDefault()
      openFromKeyboard()
    }
  }

  const onTriggerClick = (e: { detail: number }) => {
    measure()
    if (e.detail === 0) {
      // Entrée ou Espace : le clic vient du clavier.
      if (pinned) close()
      else openFromKeyboard()
      return
    }
    setPinned((p) => !p)
  }

  const onMenuKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Escape') {
      close()
      triggerRef.current?.focus()
      return
    }
    if (e.key !== 'ArrowDown' && e.key !== 'ArrowUp') return
    e.preventDefault()
    const list = Array.from(menuRef.current?.querySelectorAll<HTMLElement>('[role^="menuitem"]') ?? [])
    if (list.length === 0) return
    const at = list.indexOf(document.activeElement as HTMLElement)
    list[(at + (e.key === 'ArrowDown' ? 1 : -1) + list.length) % list.length]?.focus()
  }

  const wrapperProps = {
    onMouseEnter: () => {
      measure()
      setHovered(true)
    },
    onMouseLeave: () => setHovered(false),
    onFocus: (e: FocusEvent<HTMLDivElement>) => {
      measure()
      setFocused(isKeyboardFocus(e.target))
    },
    onBlur: (e: FocusEvent<HTMLDivElement>) => {
      setFocused(false)
      if (!e.currentTarget.contains(e.relatedTarget as Node | null)) close()
    },
  }

  return { ref, anchor, triggerRef, menuRef, open, focused, close, wrapperProps, onTriggerClick, onTriggerKeyDown, onMenuKeyDown }
}

export function RailMenu({ label, trigger, triggerClassName, current = false, placement = 'top', children }: RailMenuProps) {
  const { ref, triggerRef, menuRef, anchor, open, focused, close, wrapperProps, onTriggerClick, onTriggerKeyDown, onMenuKeyDown } =
    useRailMenu()
  return (
    <div ref={ref} {...wrapperProps}>
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
        <div style={menuPlacementStyle(anchor, placement)} className="z-50 overflow-y-auto pl-2">
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
