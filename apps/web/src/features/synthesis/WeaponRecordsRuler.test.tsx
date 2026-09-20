/**
 * WeaponRecordsRuler — la règle des records de distance, vue du DOM.
 *
 * Ce que ces tests verrouillent : chaque arme est dessinée avec son nom et son record ; la
 * légende est en pied et centrée ; ce qui est écarté est NOMMÉ ; un clic ouvre le REJEU du
 * match du record à l'instant du frag, sur l'horloge du match ; sans arme traçable la carte
 * reste et le dit.
 */
import { fireEvent, screen, within } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { SynthesisWeaponRecords } from '@/lib/api/types'
import { useAppShellStore } from '@/stores/appShellStore'
import { renderWithProviders } from '@/test/render-utils'

import { WeaponRecordsRuler } from './WeaponRecordsRuler'

const navigate = vi.fn()
vi.mock('@tanstack/react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@tanstack/react-router')>()
  return { ...actual, useNavigate: () => navigate }
})

const records: SynthesisWeaponRecords = {
  weapons: [
    {
      weapon_key: 'hinf_br75', label: 'BR75', label_en: 'BR75', class: 'shoulder',
      measured: 281, median_m: 13.6, record_m: 52.7,
      record: {
        match_id: 'm-br', time_ms: 77000, started_at: '2026-08-28T20:00:00Z',
        map_label: 'Fragmentation', map_label_en: 'Fragmentation',
      },
    },
    {
      weapon_key: 'hinf_s7_sniper', label: 'S7 Sniper', label_en: 'S7 Sniper', class: 'heavy',
      measured: 43, median_m: 28.3, record_m: 96.4,
      record: { match_id: 'm-s7', time_ms: 12345 },
    },
  ],
  measured_kills: 324,
  total_kills: 400,
  excluded: [
    { weapon_key: 'hinf_environment', label: 'Chute et environnement', label_en: 'Environment', class: 'environmental', measured: 11 },
  ],
}

/** Normalise les espaces insécables FR pour comparer du texte lisible. */
const flat = (s: string | null) => (s ?? '').replace(/[\u00a0\u202f]/g, ' ').replace(/\s+/g, ' ').trim()

describe('WeaponRecordsRuler', () => {
  beforeEach(() => {
    navigate.mockReset()
    useAppShellStore.setState({ locale: 'fr' })
  })

  it('dessine chaque arme avec son nom et son record, et le compte dans le bandeau', () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const ruler = screen.getByTestId('weapon-records-ruler')
    expect(within(ruler).getByText('BR75')).toBeInTheDocument()
    expect(within(ruler).getByText('S7 Sniper')).toBeInTheDocument()
    expect(within(ruler).getByText('52,7 m')).toBeInTheDocument()
    expect(within(ruler).getByText('96,4 m')).toBeInTheDocument()
    expect(within(ruler).getAllByTestId('weapon-record-item')).toHaveLength(2)
    expect(flat(screen.getByRole('heading', { level: 3 }).textContent)).toContain('2 armes · 324 frags mesurés sur 400')
  })

  it('pose la légende des classes présentes en pied, centrée', () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const foot = screen.getByTestId('weapon-records-legend')
    const legend = within(foot).getByTestId('chart-legend')
    expect(legend.className).toContain('justify-center')
    expect(within(legend).getAllByRole('listitem')).toHaveLength(2)
  })

  it('nomme ce qui est écarté, avec son effectif, et la réserve de couverture', () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const note = flat(screen.getByTestId('weapon-records-note').textContent)
    expect(note).toContain('Chute et environnement (11)')
    expect(note).toContain('position du tueur et de la victime')
  })

  it("ouvre le rejeu du match du record à l'instant du frag, sur l'horloge du match", () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const [br] = screen.getAllByTestId('weapon-record-item')
    fireEvent.click(br)
    expect(navigate).toHaveBeenCalledTimes(1)
    const arg = navigate.mock.calls[0][0] as { to: string; params: Record<string, string>; search: Record<string, string> }
    expect(arg.to).toBe('/{-$lang}/t/$titleSlug/players/$playerSlug/matches/$matchId/replay')
    expect(arg.params).toMatchObject({ playerSlug: 'JGtm', matchId: 'm-br' })
    expect(arg.search).toEqual({ t: '77000', clock: 'match' })
  })

  it('le clavier ouvre aussi le rejeu (Entrée)', () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const [, s7] = screen.getAllByTestId('weapon-record-item')
    fireEvent.keyDown(s7, { key: 'Enter' })
    expect(navigate).toHaveBeenCalledTimes(1)
    expect((navigate.mock.calls[0][0] as { params: { matchId: string } }).params.matchId).toBe('m-s7')
  })

  it("au survol, l'infobulle dit le record, la médiane, l'effectif et le match", () => {
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    const [br] = screen.getAllByTestId('weapon-record-item')
    fireEvent.mouseMove(br, { clientX: 100, clientY: 40 })
    const tip = flat(screen.getByTestId('weapon-records-tooltip').textContent)
    expect(tip).toContain('BR75')
    expect(tip).toContain('52,7 m')
    expect(tip).toContain('médiane 13,6 m · 281 frags mesurés')
    expect(tip).toContain('Fragmentation')
    expect(tip).toContain('28 août 2026')
  })

  it('sans arme traçable (toutes écartées), la carte reste et le dit', () => {
    renderWithProviders(
      <WeaponRecordsRuler records={{ ...records, weapons: [] }} playerSlug="JGtm" />,
    )
    expect(screen.getByTestId('weapon-records-empty')).toBeInTheDocument()
    expect(screen.queryByTestId('weapon-records-ruler')).not.toBeInTheDocument()
    expect(flat(screen.getByTestId('weapon-records-note').textContent)).toContain('Chute et environnement (11)')
  })

  it('en anglais, les libellés et les nombres suivent la locale', () => {
    useAppShellStore.setState({ locale: 'en' })
    renderWithProviders(<WeaponRecordsRuler records={records} playerSlug="JGtm" />)
    expect(screen.getByText('96.4 m')).toBeInTheDocument()
    expect(flat(screen.getByRole('heading', { level: 3 }).textContent)).toContain('Distance records by weapon')
  })
})
