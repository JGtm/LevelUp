/**
 * useFeedFilterMenuKeys — LE CLAVIER du menu de filtre du fil (`ReplayFeedFilterMenu`) : entrée du
 * focus à l'ouverture, navigation aux flèches, touches retenues. Le composant garde l'état
 * d'ouverture, la fermeture (`useDismissable`) et le rendu.
 *
 * Flèche bas ouvre depuis le déclencheur ; à l'ouverture le focus va au filtre actif ; flèches haut
 * et bas, Début et Fin déplacent le focus (en boucle) ; Tab ferme. Menu ouvert, AUCUNE touche ne
 * traverse vers le clavier du lecteur (`useReplayShortcuts`), sauf Échap, qui remonte jusqu'à la
 * fermeture de la couche.
 */
import { useEffect, type KeyboardEvent, type RefObject } from 'react'

import { isEscapeKey } from '@/lib/hooks/useDismissable'

export type MenuFocusMove = 'next' | 'prev' | 'first' | 'last'

const ITEM_SELECTOR = '[role="menuitemradio"]'
const ACTIVE_ITEM_SELECTOR = `${ITEM_SELECTOR}[aria-checked="true"]`

const MOVE_OF_KEY: Readonly<Record<string, MenuFocusMove>> = {
  ArrowDown: 'next',
  ArrowUp: 'prev',
  Home: 'first',
  End: 'last',
}

/** L'indice d'entrée qui reçoit le focus : `at` est l'entrée focalisée (-1 : aucune), en boucle. */
export function nextMenuIndex(move: MenuFocusMove, at: number, count: number): number {
  if (move === 'first') return 0
  if (move === 'last') return count - 1
  return move === 'next' ? (at + 1) % count : (at - 1 + count) % count
}

function menuItems(root: HTMLElement | null): HTMLElement[] {
  return Array.from(root?.querySelectorAll<HTMLElement>(ITEM_SELECTOR) ?? [])
}

function moveFocus(root: HTMLElement | null, move: MenuFocusMove): void {
  const items = menuItems(root)
  if (items.length === 0) return
  const at = items.indexOf(document.activeElement as HTMLElement)
  items[nextMenuIndex(move, at, items.length)]?.focus()
}

export interface FeedFilterMenuKeysArgs {
  open: boolean
  setOpen: (open: boolean) => void
  /** La racine du menu : le déclencheur et la liste. */
  wrapRef: RefObject<HTMLElement | null>
}

export interface FeedFilterMenuKeys {
  onMenuKeyDown: (e: KeyboardEvent<HTMLElement>) => void
  onTriggerKeyDown: (e: KeyboardEvent<HTMLElement>) => void
}

export function useFeedFilterMenuKeys({ open, setOpen, wrapRef }: FeedFilterMenuKeysArgs): FeedFilterMenuKeys {
  // À l'ouverture, le focus va au filtre actif : le clavier part de là où l'on est.
  useEffect(() => {
    if (!open) return
    const current = wrapRef.current?.querySelector<HTMLElement>(ACTIVE_ITEM_SELECTOR)
    ;(current ?? menuItems(wrapRef.current)[0])?.focus()
  }, [open, wrapRef])

  function onMenuKeyDown(e: KeyboardEvent<HTMLElement>) {
    const move = MOVE_OF_KEY[e.key]
    if (move) {
      e.preventDefault()
      moveFocus(wrapRef.current, move)
    } else if (e.key === 'Tab') {
      setOpen(false)
    } else if (!isEscapeKey(e)) {
      // MENU OUVERT, LE CLAVIER EST AU MENU : Espace choisit l'entrée, et aucune autre touche
      // (flèches gauche et droite, J, L, F, M, R...) ne doit atteindre le clavier du lecteur.
      e.stopPropagation()
    }
  }

  function onTriggerKeyDown(e: KeyboardEvent<HTMLElement>) {
    if (e.key === 'ArrowDown' && !open) {
      e.preventDefault()
      setOpen(true)
    } else if (open || e.key === ' ') {
      // Menu ouvert (focus resté sur le déclencheur) : rien ne traverse vers le clavier du lecteur.
      // Menu fermé, seul Espace est retenu (il active le bouton).
      e.stopPropagation()
    }
  }

  return { onMenuKeyDown, onTriggerKeyDown }
}
