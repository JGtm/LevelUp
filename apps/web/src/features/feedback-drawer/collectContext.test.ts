import { describe, expect, it } from 'vitest'
import { collectContext, describeFocusedElement } from './collectContext'
import type { BrowserEnv, ShellSummary } from './collectContext'

const browser: BrowserEnv = {
  url: '/players/Foo/synthesis',
  pathname: '/players/Foo/synthesis',
  userAgent: 'Mozilla/5.0',
  viewportWidth: 1920,
  viewportHeight: 1080,
  locale: 'fr',
  theme: 'dark',
  timestampIso: '2026-05-05T12:00:00Z',
  focusedElement: 'button',
}

const shell: ShellSummary = {
  titleSlug: 'halo_infinite',
  playerSlug: 'Foo',
  appVersion: '7.0.0',
}

describe('collectContext', () => {
  it('agrège les inputs en gardant la structure', () => {
    const ctx = collectContext({
      browser,
      shell,
      filters: { filter_mode: 'period', period: { start_date: '2026-04-01' } },
      console: [{ level: 'error', message: 'boom', timestamp: 1 }],
      failedRequests: [{ url: '/api/v1/foo', method: 'GET', status: 500, timestamp: 1 }],
    })
    expect(ctx.browser).toEqual(browser)
    expect(ctx.shell.appVersion).toBe('7.0.0')
    expect(ctx.filters?.filter_mode).toBe('period')
    expect(ctx.console).toHaveLength(1)
    expect(ctx.failedRequests).toHaveLength(1)
  })

  it('accepte filters: null', () => {
    const ctx = collectContext({
      browser,
      shell,
      filters: null,
      console: [],
      failedRequests: [],
    })
    expect(ctx.filters).toBeNull()
  })
})

describe('describeFocusedElement', () => {
  it('renvoie null si rien de focus', () => {
    document.body.focus()
    expect(describeFocusedElement()).toBeNull()
  })

  it('renvoie tag.classes (top 3) si un élément est focus', () => {
    const btn = document.createElement('button')
    btn.className = 'a b c d e'
    btn.tabIndex = 0
    document.body.appendChild(btn)
    btn.focus()
    expect(describeFocusedElement()).toBe('button.a.b.c')
    document.body.removeChild(btn)
  })

  it('renvoie juste le tag si pas de classe', () => {
    const inp = document.createElement('input')
    inp.tabIndex = 0
    document.body.appendChild(inp)
    inp.focus()
    expect(describeFocusedElement()).toBe('input')
    document.body.removeChild(inp)
  })
})
