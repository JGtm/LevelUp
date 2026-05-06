import { type ReactNode, useLayoutEffect } from 'react'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { UiTheme } from '@/stores/settingsDraftStore'
import { applyPalette } from '@/lib/accessibility'
import { pickPalette } from './palette-picker'

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
    applyPalette(pickPalette(colorPalette), colorPalette)
  }, [colorPalette])

  return <>{children}</>
}
