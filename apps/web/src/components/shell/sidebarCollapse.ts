/**
 * Repliement de la barre latérale de l'application de bureau (`SidebarNav`) en barre d'icônes.
 *
 * Trois états de préférence : `auto` (le défaut : replié sous `SIDEBAR_AUTO_COLLAPSE_QUERY`, déplié
 * au-delà, la barre suit la largeur de la fenêtre), `collapsed` et `expanded` (un choix MANUEL : bouton
 * au pied de la barre ou `Ctrl+B`). Seul un choix CONTRAIRE à ce que produirait `auto` à cet instant est
 * mémorisé : replier à la main une fenêtre large, déplier à la main une fenêtre étroite. Un choix qui
 * coïncide avec `auto` (redéplier une barre repliée à la main sur une fenêtre large, replier sur une
 * fenêtre étroite une barre dépliée à la main) remet la préférence à `auto`, la barre suit de nouveau la
 * fenêtre. Un choix manuel mémorisé n'est jamais écrasé par la largeur de la fenêtre ; rien ne replie la
 * barre au défilement ; `Ctrl+B` ne se déclenche pas pendant une saisie. Le choix survit au redémarrage
 * (`localStorage`, `try/catch` : un stockage refusé laisse la barre en `auto`).
 *
 * Au basculement, la barre est remontée (dépliée et repliée sont deux composants) : si le focus était dans
 * la barre, il est rendu au bouton de bascule de la nouvelle barre. La largeur réelle est publiée dans
 * `lib/layout/sidebarWidth`.
 */
import { useCallback, useEffect, useRef, useState } from 'react'
import { isEditableTarget } from '@/lib/dom/editableTarget'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'
import { useSidebarWidthStore } from '@/lib/layout/sidebarWidth'

/** Largeur de fenêtre sous laquelle la barre se replie d'elle-même (préférence `auto`). */
export const SIDEBAR_AUTO_COLLAPSE_QUERY = '(max-width: 1099px)'
export const SIDEBAR_PREFERENCE_STORAGE_KEY = 'levelup-sidebar-collapse'

export type SidebarPreference = 'auto' | 'collapsed' | 'expanded'

/** La barre est-elle repliée ? `narrow` : la fenêtre est sous le seuil. */
export function resolveSidebarCollapsed(preference: SidebarPreference, narrow: boolean): boolean {
  return preference === 'auto' ? narrow : preference === 'collapsed'
}

/** Attribut des deux boutons de bascule (pied de la barre dépliée, rail replié) : cible du focus rendu. */
export const SIDEBAR_TOGGLE_ATTR = 'data-sidebar-toggle'

/**
 * Préférence après un basculement : le contraire de l'état affiché ; `auto` quand ce contraire est
 * exactement ce que produirait `auto` (fenêtre étroite : replié), un choix manuel sinon.
 */
export function toggledPreference(preference: SidebarPreference, narrow: boolean): SidebarPreference {
  const nextCollapsed = !resolveSidebarCollapsed(preference, narrow)
  if (nextCollapsed === narrow) return 'auto'
  return nextCollapsed ? 'collapsed' : 'expanded'
}

/** Le focus est-il dans une barre latérale (dépliée ou repliée) ? */
function focusIsInSidebar(): boolean {
  return document.activeElement?.closest('[data-sidebar]') != null
}

/** Préférence mémorisée ; `auto` quand rien n'est mémorisé, la valeur est inconnue ou le stockage refusé. */
export function loadSidebarPreference(): SidebarPreference {
  try {
    const raw = window.localStorage.getItem(SIDEBAR_PREFERENCE_STORAGE_KEY)
    return raw === 'collapsed' || raw === 'expanded' ? raw : 'auto'
  } catch (err) {
    console.warn('[shell] préférence de la barre latérale : lecture impossible', err)
    return 'auto'
  }
}

function saveSidebarPreference(preference: SidebarPreference): void {
  try {
    window.localStorage.setItem(SIDEBAR_PREFERENCE_STORAGE_KEY, preference)
  } catch (err) {
    console.warn('[shell] préférence de la barre latérale : écriture impossible', err)
  }
}

/** `Ctrl+B`, sans autre modificateur. */
export function isSidebarShortcut(e: Pick<KeyboardEvent, 'key' | 'ctrlKey' | 'shiftKey' | 'altKey' | 'metaKey'>): boolean {
  return e.ctrlKey && !e.shiftKey && !e.altKey && !e.metaKey && e.key.toLowerCase() === 'b'
}

export interface SidebarCollapse {
  collapsed: boolean
  toggle: () => void
}

/** L'état replié de la barre, son basculement et le raccourci `Ctrl+B` (à poser une fois, dans `SidebarNav`). */
export function useSidebarCollapse(): SidebarCollapse {
  const narrow = useMediaQuery(SIDEBAR_AUTO_COLLAPSE_QUERY)
  const [preference, setPreference] = useState<SidebarPreference>(loadSidebarPreference)
  const collapsed = resolveSidebarCollapsed(preference, narrow)
  const restoreFocusRef = useRef(false)
  const setSidebarCollapsed = useSidebarWidthStore((s) => s.setCollapsed)

  useEffect(() => setSidebarCollapsed(collapsed), [collapsed, setSidebarCollapsed])

  // La barre basculée est un autre composant : le bouton qui avait le focus a disparu, le focus est
  // rendu au bouton de bascule de la nouvelle barre (sinon il retombe sur le corps de la page).
  useEffect(() => {
    if (!restoreFocusRef.current) return
    restoreFocusRef.current = false
    document.querySelector<HTMLElement>(`[data-sidebar] [${SIDEBAR_TOGGLE_ATTR}]`)?.focus()
  }, [collapsed])

  const toggle = useCallback(() => {
    restoreFocusRef.current = focusIsInSidebar()
    const next = toggledPreference(preference, narrow)
    setPreference(next)
    saveSidebarPreference(next)
  }, [preference, narrow])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.defaultPrevented || e.isComposing || isEditableTarget(e.target) || !isSidebarShortcut(e)) return
      e.preventDefault()
      toggle()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [toggle])

  return { collapsed, toggle }
}
