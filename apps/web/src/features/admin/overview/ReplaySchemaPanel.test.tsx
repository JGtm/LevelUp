/**
 * Tests — ReplaySchemaPanel : un compte par titre (courants, à l'ancien schéma, illisibles), l'état vide en bloc
 * pointillé, et les deux langues.
 */
import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { useAppShellStore } from '@/stores/appShellStore'

import { ReplaySchemaPanel } from './ReplaySchemaPanel'

const schema = vi.hoisted(() => ({ value: { data: undefined, isLoading: false, isError: false } as Record<string, unknown> }))

vi.mock('../monitoring/queries', () => ({ useReplaySchema: () => schema.value }))
vi.mock('../titles/queries', () => ({
  useAdminTitles: () => ({ data: { titles: [{ slug: 'halo_infinite', name: 'Halo Infinite', status: 'active' }] } }),
}))

function respond(over: Record<string, unknown>) {
  schema.value = { data: undefined, isLoading: false, isError: false, ...over }
}

describe('ReplaySchemaPanel', () => {
  beforeEach(() => {
    useAppShellStore.setState({ locale: 'fr' } as never)
    respond({})
  })

  it('un titre : le total, les rejeux au schéma courant, à l’ancien schéma et illisibles', () => {
    respond({
      data: {
        generated_at: '2026-11-24T10:00:00Z',
        current_schema: 127,
        titles: [{ title_slug: 'halo_infinite', total: 1500, current: 1450, behind: 40, unreadable: 10 }],
      },
    })
    render(<ReplaySchemaPanel />)
    expect(screen.getByText('Rejeux par version de schéma')).toBeTruthy()
    expect(screen.getByText('Halo Infinite')).toBeTruthy()
    expect(screen.getByText(/Schéma courant · 127/)).toBeTruthy()
    expect(screen.getByText(/Compté le/).textContent).toContain(new Date('2026-11-24T10:00:00Z').toLocaleString('fr-FR'))
    expect(screen.getByText("À l'ancien schéma").nextSibling?.textContent).toBe((40).toLocaleString('fr-FR'))
    expect(screen.getByText('Au schéma courant').nextSibling?.textContent).toBe((1450).toLocaleString('fr-FR'))
    expect(screen.getByText('Illisibles').nextSibling?.textContent).toBe('10')
  })

  it('un titre sans rejeu rangé : un bloc pointillé, jamais une rangée de zéros', () => {
    respond({
      data: { generated_at: 'x', current_schema: 127, titles: [{ title_slug: 'halo_infinite', total: 0, current: 0, behind: 0, unreadable: 0 }] },
    })
    const { container } = render(<ReplaySchemaPanel />)
    expect(screen.getByText('Aucun rejeu rangé pour ce titre.')).toBeTruthy()
    expect(container.querySelector('.border-dashed')).not.toBeNull()
    expect(screen.queryByText('Illisibles')).toBeNull()
  })

  it('aucun titre ne range de rejeux : un bloc pointillé', () => {
    respond({ data: { generated_at: 'x', current_schema: 127, titles: [] } })
    const { container } = render(<ReplaySchemaPanel />)
    expect(container.querySelector('.border-dashed')).not.toBeNull()
  })

  it('une lecture impossible le dit', () => {
    respond({ isError: true })
    render(<ReplaySchemaPanel />)
    expect(screen.getByText('Décompte des rejeux indisponible.')).toBeTruthy()
  })

  it('se dit en anglais', () => {
    useAppShellStore.setState({ locale: 'en' } as never)
    respond({
      data: { generated_at: '2026-11-24T10:00:00Z', current_schema: 127, titles: [{ title_slug: 'halo_infinite', total: 3, current: 1, behind: 2, unreadable: 0 }] },
    })
    render(<ReplaySchemaPanel />)
    expect(screen.getByText('Replays by schema version')).toBeTruthy()
    expect(screen.getByText(/Counted at/)).toBeTruthy()
    expect(screen.getByText('On an older schema')).toBeTruthy()
  })
})
