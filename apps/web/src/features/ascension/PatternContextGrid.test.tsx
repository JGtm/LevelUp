/**
 * Tests PatternContextGrid — les liens pattern→Solo utilisent filter_key (F7).
 *
 * Le backend sert `filter_key` = la clé de filtrage STABLE (FR-first, locale-
 * indépendante) que matche le pipeline de filtres. Le lien DOIT se construire sur
 * filter_key, JAMAIS sur `label` (localisé → ne matcherait pas en EN). Sans
 * filter_key → pas de lien (carte statique).
 */
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen, cleanup } from '@testing-library/react'
import { PatternContextGrid } from './PatternContextGrid'
import { getAscensionText } from './i18n'
import type { ContextualPattern } from './types'

afterEach(() => cleanup())

function pattern(p: Partial<ContextualPattern>): ContextualPattern {
  return {
    type: 'by_map',
    key: 'guid-xxxx',
    match_count: 20,
    win_rate: 0.3,
    avg_kda: 1,
    avg_oc: 0.5,
    avg_dr: 1.2,
    delta: -0.2,
    signal: 'weakness',
    ...p,
  }
}

// Décode le contexte de filtre encodé dans un href `?f=…` (enveloppe v2 { t, c }).
function decodeCascade(href: string): { modes?: string[]; maps?: string[] } {
  const f = new URL(href, 'http://localhost').searchParams.get('f') ?? ''
  const payload = JSON.parse(decodeURIComponent(atob(f)))
  return payload.c.cascade
}

const BASE = { t: getAscensionText('fr'), minMatchesForSignal: 5, playerSlug: 'demo', titleSlug: 'halo_infinite' }

describe('PatternContextGrid — liens filter_key (F7)', () => {
  it('by_map : le lien encode filter_key (FR-first), pas le label localisé', () => {
    render(
      <PatternContextGrid
        {...BASE}
        patterns={[pattern({ type: 'by_map', key: 'guid-1', label: 'Recharge', filter_key: 'Décharge' })]}
      />,
    )
    const link = screen.getByRole('link')
    const cascade = decodeCascade(link.getAttribute('href') ?? '')
    expect(cascade.maps).toEqual(['Décharge'])
    expect(cascade.maps).not.toContain('Recharge')
  })

  it('by_mode : le lien encode filter_key (mode normalisé)', () => {
    render(
      <PatternContextGrid
        {...BASE}
        patterns={[pattern({ type: 'by_mode', key: 'Slayer', filter_key: 'Slayer', label: undefined })]}
      />,
    )
    const link = screen.getByRole('link')
    const cascade = decodeCascade(link.getAttribute('href') ?? '')
    expect(cascade.modes).toEqual(['Slayer'])
  })

  it('by_map sans filter_key : aucune carte cliquable (pas de lien vers un label localisé)', () => {
    render(
      <PatternContextGrid
        {...BASE}
        patterns={[pattern({ type: 'by_map', key: 'guid-2', label: 'Recharge', filter_key: undefined })]}
      />,
    )
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
    // La carte reste affichée (label localisé), juste non cliquable.
    expect(screen.getByText('Recharge')).toBeInTheDocument()
  })
})
