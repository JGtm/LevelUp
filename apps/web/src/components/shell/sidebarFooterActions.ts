/** Ce que partagent le pied de la barre latérale dépliée (`SidebarFooter`) et celui de la barre repliée (`SidebarRail`). */
import { useAssetDrawerStore } from '@/features/asset-drawer'
import { useFeedbackDrawerStore } from '@/features/feedback-drawer'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'

/** Raccourci annoncé dans le titre et `aria-keyshortcuts` du bouton de repliement. */
export const SIDEBAR_SHORTCUT_LABEL = 'Ctrl+B'

/**
 * Ce que le pied de barre déclenche, commun à la barre dépliée et à la barre repliée : le volet des
 * référentiels (ouvert ou non, bascule), le signalement d'un problème et la bascule de thème (`isDark`
 * donne l'état de l'interrupteur). L'ouverture de « À propos » est tenue plus haut, par `SidebarNav` : la
 * barre est remontée au basculement replié / déplié et l'état d'un pied de barre y serait perdu (la
 * fenêtre se fermerait).
 */
export function useSidebarFooterActions() {
  const assetsOpen = useAssetDrawerStore((s) => s.isOpen)
  const toggleAssets = useAssetDrawerStore((s) => s.toggle)
  const openFeedback = useFeedbackDrawerStore((s) => s.open)
  const isDark = useSettingsDraftStore((s) => s.localUiPrefs.theme === 'dark')
  const toggleTheme = useSettingsDraftStore((s) => s.toggleTheme)
  return { assetsOpen, toggleAssets, openFeedback, isDark, toggleTheme }
}
