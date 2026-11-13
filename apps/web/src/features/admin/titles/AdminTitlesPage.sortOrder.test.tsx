/**
 * Ordre des lignes de la liste des titres (AdminTitlesPage) après chaque clic d'en-tête : figé avant et
 * après le passage du tri sur TanStack Table (étape 7.7).
 */
import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { recordSortOrders } from '@/lib/table/sortOrder.testutil'

vi.mock('../useAdminText', () => ({
  useAdminT: () => (key: string) => key,
  useDateLocale: () => 'fr-FR',
}))

function title(name: string, status: string, capabilities: string[] | null, mappings: boolean) {
  return { slug: name, name, provider: name, icon_url: '', status, capabilities, is_default: false, xbox_title_id: '', steam_app_id: '', has_mappings: mappings }
}

vi.mock('./queries', () => ({
  useAdminTitles: () => ({
    data: {
      titles: [
        title('delta', 'active', ['a', 'b'], true),
        title('Alpha', 'archived', ['a'], false),
        title('charlie', 'coming_soon', null, true),
        title('bravo', 'active', ['a', 'b', 'c'], false),
        title('echo', 'archived', ['a', 'b'], true),
      ],
      count: 5,
    },
    isLoading: false,
    isError: false,
  }),
  useAdminTitleDetail: () => ({ data: undefined, isLoading: true, isError: false }),
  useAdminTitleDiagnostic: () => ({ data: undefined, isLoading: true, isError: false }),
  fetchTitleTomlDraft: () => Promise.resolve(''),
}))

import { AdminTitlesPage } from './AdminTitlesPage'

describe('AdminTitlesPage — ordre des lignes par en-tête', () => {
  it('ordre initial, puis trois clics par en-tête', () => {
    const { container } = render(<AdminTitlesPage />)
    const table = container.querySelector('table') as HTMLElement
    const recording = recordSortOrders(table, [
      { name: 'admin.titles.col_title' },
      { name: 'admin.titles.col_status' },
      { name: 'admin.titles.col_capabilities' },
      { name: 'admin.titles.col_mappings' },
    ])
    expect(recording).toMatchInlineSnapshot(`
      {
        "admin.titles.col_capabilities": "1: bravobravo > deltadelta > echoecho > AlphaAlpha > charliecharlie | 2: charliecharlie > AlphaAlpha > deltadelta > echoecho > bravobravo | 3: bravobravo > deltadelta > echoecho > AlphaAlpha > charliecharlie | aria-sort: none,none,descending,none",
        "admin.titles.col_mappings": "1: deltadelta > charliecharlie > echoecho > AlphaAlpha > bravobravo | 2: AlphaAlpha > bravobravo > deltadelta > charliecharlie > echoecho | 3: deltadelta > charliecharlie > echoecho > AlphaAlpha > bravobravo | aria-sort: none,none,none,descending",
        "admin.titles.col_status": "1: deltadelta > bravobravo > AlphaAlpha > echoecho > charliecharlie | 2: charliecharlie > AlphaAlpha > echoecho > deltadelta > bravobravo | 3: deltadelta > bravobravo > AlphaAlpha > echoecho > charliecharlie | aria-sort: none,ascending,none,none",
        "admin.titles.col_title": "1: AlphaAlpha > bravobravo > charliecharlie > deltadelta > echoecho | 2: echoecho > deltadelta > charliecharlie > bravobravo > AlphaAlpha | 3: AlphaAlpha > bravobravo > charliecharlie > deltadelta > echoecho | aria-sort: ascending,none,none,none",
        "initial": "deltadelta > AlphaAlpha > charliecharlie > bravobravo > echoecho",
      }
    `)
  })
})
