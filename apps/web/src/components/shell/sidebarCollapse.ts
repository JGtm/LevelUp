/**
 * Repliement de la barre latérale de l'application de bureau (`SidebarNav`) en barre d'icônes.
 *
 * Trois états de préférence : `auto` (le défaut : replié sous `SIDEBAR_AUTO_COLLAPSE_QUERY`, déplié
 * au-delà, la barre suit la largeur de la fenêtre), `collapsed` et `expanded` (un choix MANUEL : bouton
 * au pied de la barre ou `Ctrl+B`). Un choix manuel n'est jamais écrasé par la largeur de la fenêtre ;
 * rien ne replie la barre au défilement. Le choix survit au redémarrage (`localStorage`, `try/catch` :
 * un stockage refusé laisse la barre en `auto`).
 */
import { useCallback, useEffect, useState } from 'react'
import { useMediaQuery } from '@/lib/hooks/useMediaQuery'

/** Largeur de fenêtre sous laquelle la barre se replie d'elle-même (préférence `auto`). */
export const SIDEBAR_AUTO_COLLAPSE_QUERY = '(max-width: 1099px)'
export const SIDEBAR_PREFERENCE_STORAGE_KEY = 'levelup-sidebar-collapse'

export type SidebarPreference = 'auto' | 'collapsed' | 'expanded'

/** La barre est-elle repliée ? `narrow` : la fenêtre est sous le seuil. */
export function resolveSidebarCollapsed(preference: SidebarPreference, narrow: boolean): boolean {
  return preference === 'auto' ? narrow : preference === 'collapsed'
}

/** Préférence après un basculement : le contraire de l'état affiché, en choix manuel. */
export function toggledPreference(preference: SidebarPreference, narrow: boolean): SidebarPreference {
  return resolveSidebarCollapsed(preference, narrow) ? 'expanded' : 'collapsed'
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

  const toggle = useCallback(() => {
    const next = toggledPreference(preference, narrow)
    setPreference(next)
    saveSidebarPreference(next)
  }, [preference, narrow])

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.defaultPrevented || e.isComposing || !isSidebarShortcut(e)) return
      e.preventDefault()
      toggle()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [toggle])

  return { collapsed, toggle }
}
