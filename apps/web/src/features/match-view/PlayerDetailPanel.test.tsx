/**
 * PlayerDetailPanel.test.tsx — drawer (expander) du scoreboard, section Médailles.
 *
 * GH-5a (2026-07-08) — RÉGRESSION : les médailles Halo 5 (sprite, sans PNG par-médaille)
 * s'affichaient vides car le drawer rendait un `<img src={image_url}>` brut et image_url
 * était "" pour H5. Le fix migre vers <MedalIcon> (title-agnostic : PNG Infinite OU
 * sprite Halo 5). Ce test aurait attrapé la régression : il vérifie qu'une médaille
 * sprite (H5) rend bien une icône (role=img via MedalIcon), pas un fallback texte, et
 * qu'une médaille PNG (Infinite) rend un <img> à src non vide.
 */
import { describe, it, expect } from 'vitest'
import { screen } from '@testing-library/react'

import { renderWithProviders } from '@/test/render-utils'
import type { MatchScoreboardRow, PlayerMedalRow } from '@/lib/api/types'
import { MATCH_VIEW_TEXT } from './i18n'
import { PlayerDetailPanel } from './PlayerDetailPanel'

// Médaille Infinite : PNG (image_url), pas de sprite.
const INFINITE_MEDAL: PlayerMedalRow = {
  medal_id: 1,
  count: 3,
  label: 'Killing Spree',
  image_url: '/static/medals/halo_infinite/1.png',
  difficulty: 'normal',
}

// Médaille Halo 5 : sprite (feuille + offset), image_url vide.
const H5_MEDAL: PlayerMedalRow = {
  medal_id: 2,
  count: 1,
  label: 'Double Kill',
  image_url: '',
  difficulty: 'common',
  sprite_sheet: '/static/medals/halo_5/sheet.png',
  sprite_left: 74,
  sprite_top: 0,
  sprite_width: 74,
  sprite_height: 74,
}

function buildRow(medals: PlayerMedalRow[]): MatchScoreboardRow {
  return {
    xuid: 'x1',
    gamertag: 'Tester',
    team_side: '0',
    is_me: false,
    rank: 1,
    score: 100,
    kills: 10,
    deaths: 5,
    assists: 2,
    shots_fired: null,
    shots_hit: null,
    accuracy: null,
    damage_dealt: null,
    damage_taken: null,
    average_life: null,
    headshot_kills: null,
    max_killing_spree: null,
    perfect_kills: null,
    power_weapon_kills: null,
    melee_kills: null,
    outcome_label: 'Victoire',
    medals,
  }
}

describe('PlayerDetailPanel — médailles (GH-5a)', () => {
  it('rend une médaille Infinite comme <img> à src non vide', () => {
    renderWithProviders(<PlayerDetailPanel row={buildRow([INFINITE_MEDAL])} t={MATCH_VIEW_TEXT.fr} />)
    const img = screen.getByAltText('Killing Spree') as HTMLImageElement
    expect(img.tagName).toBe('IMG')
    expect(img.getAttribute('src')).toBeTruthy()
    expect(img.getAttribute('src')).not.toBe('')
  })

  it('rend une médaille Halo 5 (sprite) comme icône, pas un fallback texte', () => {
    // Sur le code cassé (img brut + image_url vide), cette médaille tombait sur le
    // fallback texte : aucun élément role=img n'existait → ce getByRole échoue.
    renderWithProviders(<PlayerDetailPanel row={buildRow([H5_MEDAL])} t={MATCH_VIEW_TEXT.fr} />)
    const sprite = screen.getByRole('img', { name: 'Double Kill' })
    expect(sprite).toBeInTheDocument()
    // Le sprite est rendu via background-image (MedalIcon), pas un <img src>.
    expect(sprite.querySelector('div')?.style.backgroundImage).toContain('sheet.png')
  })
})
