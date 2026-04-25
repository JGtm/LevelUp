import { type ReactNode, useLayoutEffect } from 'react'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { UiTheme } from '@/stores/settingsDraftStore'
import { applyPalette, defaultPalette, okabePalette } from '@/lib/accessibility'

function applyDocumentTheme(theme: UiTheme) {
  document.documentElement.setAttribute('data-theme', theme)
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)
  const colorPalette = useSettingsDraftStore((state) => state.localUiPrefs.colorPalette)

  useLayoutEffect(() => {
    applyDocumentTheme(theme)
  }, [theme])

  useLayoutEffect(() => {
    const palette = colorPalette === 'okabe-ito' ? okabePalette : defaultPalette
    applyPalette(palette, colorPalette)
  }, [colorPalette])

  return <>{children}</>
}
