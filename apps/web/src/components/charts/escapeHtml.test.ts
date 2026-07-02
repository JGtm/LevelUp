import { describe, it, expect } from 'vitest'

import { escapeHtml } from './_utils'

describe('escapeHtml', () => {
  it('échappe les métacaractères HTML, apostrophe incluse', () => {
    expect(escapeHtml(`<img src=x onerror="a">&'`)).toBe(
      '&lt;img src=x onerror=&quot;a&quot;&gt;&amp;&#39;',
    )
  })

  // Garde-rail (CLAUDE.md règle #6 : ≤2 copies → centraliser + garde-rail).
  // escapeHtml vit UNIQUEMENT dans components/charts/_utils.ts. Toute
  // redéfinition locale re-diverge (la version BarStackedChart oubliait
  // l'apostrophe) et rouvre la surface XSS des tooltips — interdit.
  it('n’est défini nulle part ailleurs que dans components/charts/_utils.ts', () => {
    const files = import.meta.glob('/src/**/*.{ts,tsx}', {
      query: '?raw',
      import: 'default',
      eager: true,
    }) as Record<string, string>
    const offenders = Object.entries(files)
      .filter(
        ([path, src]) =>
          !path.endsWith('/components/charts/_utils.ts') &&
          /function\s+escapeHtml\b/.test(src),
      )
      .map(([path]) => path)
    expect(offenders, `escapeHtml redéfini hors _utils.ts :\n${offenders.join('\n')}`).toEqual([])
  })
})
