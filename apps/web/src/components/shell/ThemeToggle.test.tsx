import { beforeEach, describe, expect, it } from 'vitest'
import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { ThemeProvider } from '@/app/providers/theme-provider'
import { renderWithProviders } from '@/test/render-utils'
import { useSettingsDraftStore } from '@/stores/settingsDraftStore'
import { ThemeToggle } from './ThemeToggle'

beforeEach(() => {
  localStorage.clear()
  document.documentElement.removeAttribute('data-theme')
  useSettingsDraftStore.setState({
    dirtyFields: {},
    lastSavedAt: null,
    localUiPrefs: {
      showHints: true,
      theme: 'dark',
      colorPalette: 'default',
      lastPlayerSlugByTitle: {},
      allyTeamColor: null,
      enemyTeamColor: null,
      showWaypointColumn: true,
    },
  })
})

describe('ThemeToggle', () => {
  it('bascule le thème global depuis la nav', async () => {
    const user = userEvent.setup()

    renderWithProviders(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>,
    )

    const toggle = screen.getByRole('switch', { name: /Passer au thème clair/i })

    expect(toggle).toHaveAttribute('aria-checked', 'true')
    expect(document.documentElement.dataset.theme).toBe('dark')

    await user.click(toggle)

    expect(screen.getByRole('switch', { name: /Passer au thème sombre/i })).toHaveAttribute(
      'aria-checked',
      'false',
    )
    expect(document.documentElement.dataset.theme).toBe('light')
  })
})
