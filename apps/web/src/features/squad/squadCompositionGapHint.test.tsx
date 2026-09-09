/**
 * Tests squadCompositionGapHint — contenu de l'info-bulle qui EXPLIQUE l'ecart
 * "composition exacte" publie sur la L2 (ADR 0033, D1 : phase A3 du plan
 * `.ai/PLAN_ESCOUADE_HORS_CADRE_2026-09-09.md`).
 *
 * TDD ROUGE : le module `./squadCompositionGapHint` n'existe pas encore.
 * Echec attendu (avant tout code) : echec de RESOLUTION DE MODULE (Vite).
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'
import { buildCompositionGapHint } from './squadCompositionGapHint'
import { FR_TEXT, EN_TEXT } from './i18n'
import type { SquadSessionExcludedMatch } from './squadSessionCounts'

const NILTON_MATCH: SquadSessionExcludedMatch = {
  match_id: 'm1',
  start_time: '2026-08-27T20:00:00Z',
  map_ui: 'Aquarius',
  extra_gamertags: ['Nilton410'],
}

const DUO_MATCH: SquadSessionExcludedMatch = {
  match_id: 'm2',
  start_time: '2026-08-27T21:00:00Z',
  map_ui: 'Recharge',
  extra_gamertags: ['Nilton410', 'passivemarquise'],
}

const UNKNOWN_MATCH: SquadSessionExcludedMatch = {
  match_id: 'm3',
  start_time: '2026-08-27T22:00:00Z',
  map_ui: 'Streets',
  extra_gamertags: null,
}

describe('buildCompositionGapHint', () => {
  it('aucun match ecarte : rend undefined (pas de tooltip a afficher)', () => {
    expect(buildCompositionGapHint([], 'fr', FR_TEXT.compositionGap)).toBeUndefined()
  })

  it('un match ecarte, un coequipier nomme : la ligne porte la date, la carte et le coequipier (FR)', () => {
    const node = buildCompositionGapHint([NILTON_MATCH], 'fr', FR_TEXT.compositionGap)
    render(<>{node}</>)
    expect(screen.getByText(/1 match écarté/)).toBeTruthy()
    expect(screen.getByText(/Aquarius/)).toBeTruthy()
    expect(screen.getByText(/Nilton410 était dans ton équipe/)).toBeTruthy()
  })

  it('deux coequipiers responsables du meme match : accord pluriel "etaient"', () => {
    const node = buildCompositionGapHint([DUO_MATCH], 'fr', FR_TEXT.compositionGap)
    render(<>{node}</>)
    expect(screen.getByText(/Nilton410 et passivemarquise étaient dans ton équipe/)).toBeTruthy()
  })

  it('coequipier non resolu (extra_gamertags null) : repli textuel, jamais une ligne vide', () => {
    const node = buildCompositionGapHint([UNKNOWN_MATCH], 'fr', FR_TEXT.compositionGap)
    render(<>{node}</>)
    expect(screen.getByText(/un coéquipier non identifié était dans ton équipe/)).toBeTruthy()
  })

  it('plusieurs matchs ecartes : une ligne par match, le titre compte le total', () => {
    const node = buildCompositionGapHint([NILTON_MATCH, DUO_MATCH], 'fr', FR_TEXT.compositionGap)
    render(<>{node}</>)
    expect(screen.getByText(/2 matchs écartés/)).toBeTruthy()
  })

  it('EN : accord singulier/pluriel "was"/"were"', () => {
    const node = buildCompositionGapHint([NILTON_MATCH, DUO_MATCH], 'en', EN_TEXT.compositionGap)
    render(<>{node}</>)
    expect(screen.getByText(/Nilton410 was in your team/)).toBeTruthy()
    expect(screen.getByText(/Nilton410 and passivemarquise were in your team/)).toBeTruthy()
    expect(screen.getByText(/2 matches excluded/)).toBeTruthy()
  })
})
