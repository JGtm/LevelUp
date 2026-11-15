/**
 * Tests AnalyseTab — focus sur la card « Progression long-terme »
 * (toggle Objectifs/Prestige + lien glossaire).
 */
import { afterEach, describe, it, expect, vi } from 'vitest'
import { fireEvent, screen } from '@testing-library/react'
import { renderWithProviders } from '@/test/render-utils'
import { useAppShellStore } from '@/stores/appShellStore'
import { AnalyseTab } from './AnalyseTab'
import { getSettingsText } from './i18n'
import type { SettingsResponse } from '@/lib/api/types'

vi.mock('@tanstack/react-router', () => ({
  Link: ({
    to,
    search,
    children,
    className,
  }: {
    to: string
    search?: Record<string, string>
    children: React.ReactNode
    className?: string
  }) => {
    const qs = search
      ? '?' +
        Object.entries(search)
          .map(([k, v]) => `${k}=${v}`)
          .join('&')
      : ''
    return (
      <a href={`${to}${qs}`} className={className}>
        {children}
      </a>
    )
  },
}))

vi.mock('@/features/settings/queries', () => ({
  useRecalculateSessions: () => ({ mutate: vi.fn(), isPending: false }),
}))

const t = getSettingsText('fr')

function renderTab(merged: Partial<SettingsResponse>, onChange = vi.fn()) {
  renderWithProviders(<AnalyseTab merged={merged} handleChange={onChange} t={t} />)
  return onChange
}

describe('AnalyseTab — Progression long-terme', () => {
  it('affiche la card avec son titre et son hint', () => {
    renderTab({ show_progression: true })
    expect(screen.getByText('Progression long-terme')).toBeInTheDocument()
    expect(screen.getByText('Afficher Objectifs & Prestige')).toBeInTheDocument()
    expect(screen.getByText(/Points de Prestige/)).toBeInTheDocument()
  })

  it('active par défaut quand show_progression est absent', () => {
    renderTab({})
    const toggle = screen
      .getByText('Afficher Objectifs & Prestige')
      .closest('div')!
      .querySelector('button')!
    // bg-primary => actif (true)
    expect(toggle.className).toMatch(/bg-primary/)
  })

  it('reflète show_progression=false (toggle inactif)', () => {
    renderTab({ show_progression: false })
    const toggle = screen
      .getByText('Afficher Objectifs & Prestige')
      .closest('div')!
      .querySelector('button')!
    expect(toggle.className).not.toMatch(/bg-primary/)
  })

  it('appelle handleChange("show_progression", false) quand on clique le toggle activé', () => {
    const onChange = renderTab({ show_progression: true })
    const toggle = screen
      .getByText('Afficher Objectifs & Prestige')
      .closest('div')!
      .querySelector('button')!
    fireEvent.click(toggle)
    expect(onChange).toHaveBeenCalledWith('show_progression', false)
  })

  it('expose un lien vers /help?tab=glossary', () => {
    renderTab({ show_progression: true })
    const link = screen.getByRole('link', { name: /glossaire/i })
    expect(link).toHaveAttribute('href', '/help?tab=glossary')
  })
})

describe('AnalyseTab — rendement sans assistances : vie du bootstrap', () => {
  const title = (hp: number) => ({
    slug: 'un_titre',
    name: 'T',
    status: 'active' as const,
    capabilities: [],
    is_default: true,
    effective_hp_to_kill: hp,
    provides_damage_taken: true,
    provides_team_mmr: true,
    provides_max_killing_spree: true,
    offensive_conversion_p80: 0.9,
    defensive_resistance_p80: 1.65,
  })

  afterEach(() => {
    useAppShellStore.setState({ availableTitles: [], currentTitleSlug: 'halo_infinite' })
  })

  it('rend la vie du titre courant dans la formule', () => {
    useAppShellStore.setState({ availableTitles: [title(115)], currentTitleSlug: 'un_titre' })
    renderTab({})
    expect(screen.getByText(/le rendement devient 115 × éliminations/)).toBeInTheDocument()
  })

  it("n'affiche aucune valeur avant le bootstrap", () => {
    useAppShellStore.setState({ availableTitles: [], currentTitleSlug: 'un_titre' })
    renderTab({})
    expect(screen.getByText(/le rendement devient — × éliminations/)).toBeInTheDocument()
  })
})
