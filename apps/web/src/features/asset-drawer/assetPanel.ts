/**
 * Ce que le volet ancré des référentiels partage avec son bouton d'ouverture (barre latérale) : les
 * identifiants, la fermeture qui rend le focus, le raccourci clavier.
 */
import { useEffect } from 'react'
import { useAssetDrawerStore } from './assetDrawer.store'

/** Identifiant du bouton qui ouvre le volet (barre latérale) : le focus y revient à la fermeture. */
export const ASSET_PANEL_TRIGGER_ID = 'asset-panel-trigger'
export const ASSET_PANEL_ID = 'asset-docked-panel'

/** Ferme le volet et rend le focus à son bouton d'ouverture, s'il est à l'écran. */
export function closeAssetPanel(): void {
  useAssetDrawerStore.getState().close()
  document.getElementById(ASSET_PANEL_TRIGGER_ID)?.focus()
}

/**
 * Raccourci d'ouverture et de fermeture. `Ctrl+Maj+R` est écarté : c'est le rechargement sans cache du
 * moteur web de la fenêtre. `Ctrl+Maj+L` n'a pas d'accélérateur connu du moteur et la page n'en utilise
 * aucun (les touches du rejeu ignorent Ctrl). Le défaut est neutralisé par précaution.
 */
export function useAssetPanelShortcut(): void {
  useEffect(() => {
    function onKey(e: globalThis.KeyboardEvent) {
      if (!e.ctrlKey || !e.shiftKey || e.altKey || e.metaKey || e.key.toLowerCase() !== 'l') return
      e.preventDefault()
      if (useAssetDrawerStore.getState().isOpen) closeAssetPanel()
      else useAssetDrawerStore.getState().open()
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])
}
