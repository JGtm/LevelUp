/** Ce que partagent le pied de la barre latérale dépliée (`SidebarFooter`) et celui de la barre repliée (`SidebarRail`). */
import { useState } from 'react'
import { useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'

/** Raccourci annoncé dans le titre et `aria-keyshortcuts` du bouton de repliement. */
export const SIDEBAR_SHORTCUT_LABEL = 'Ctrl+B'

/**
 * Ce que le pied de barre déclenche, commun à la barre dépliée et à la barre repliée : le volet des
 * référentiels (ouvert ou non, bascule), le signalement d'un problème, la fenêtre « À propos ».
 */
export function useSidebarFooterActions() {
  const [aboutOpen, setAboutOpen] = useState(false)
  const assetsOpen = useAssetDrawerStore((s) => s.isOpen)
  const toggleAssets = useAssetDrawerStore((s) => s.toggle)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)
  return { aboutOpen, setAboutOpen, assetsOpen, toggleAssets, openFeedback }
}
