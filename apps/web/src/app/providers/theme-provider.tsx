import { type ReactNode, useLayoutEffect } from 'react'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { UiTheme } from '@/stores/settingsDraftStore'
import { applyPalette, paletteForTheme } from '@/lib/accessibility'
import { pickPalette } from './palette-picker'
import { findOutlineColor } from '@/lib/halo/outline-colors'

function applyDocumentTheme(theme: UiTheme) {
  document.documentElement.setAttribute('data-theme', theme)
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  const colorPalette = useSettingsDraftStore((state) => state.localUiPrefs.colorPalette)
  const allyTeamColor = useSettingsDraftStore((state) => state.localUiPrefs.allyTeamColor)
  const enemyTeamColor = useSettingsDraftStore((state) => state.localUiPrefs.enemyTeamColor)

  useLayoutEffect(() => {
    applyDocumentTheme(theme)
  }, [theme])

  // La palette suit aussi le thème : quelques jetons (rampes des rôles d'impact) changent de
  // valeur avec le fond (`paletteForTheme`). Effet posé APRÈS celui du thème : les graphes qui
  // se reconstruisent au changement de `data-theme` lisent déjà les valeurs du nouveau thème.
  useLayoutEffect(() => {
    const palette = paletteForTheme(pickPalette(colorPalette), theme)
    applyPalette(palette, `${colorPalette}:${theme}`)
    // Apply user outline color overrides — always explicit so reset-to-default works.
    const root = document.documentElement
    const ally = findOutlineColor(allyTeamColor)
    const enemy = findOutlineColor(enemyTeamColor)
    root.style.setProperty('--ac-team-ally', ally?.hex ?? palette['team-ally'])
    root.style.setProperty('--ac-team-enemy', enemy?.hex ?? palette['team-enemy'])
  }, [colorPalette, theme, allyTeamColor, enemyTeamColor])

  return <>{children}</>
}
