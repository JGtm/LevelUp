import { type ReactNode, useLayoutEffect } from 'react'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import type { UiTheme } from '@/stores/settingsDraftStore'

function applyDocumentTheme(theme: UiTheme) {
  document.documentElement.setAttribute('data-theme', theme)
}

export function ThemeProvider({ children }: { children: ReactNode }) {
  const theme = useSettingsDraftStore((state) => state.localUiPrefs.theme)

  useLayoutEffect(() => {
    applyDocumentTheme(theme)
  }, [theme])

  return <>{children}</>
}