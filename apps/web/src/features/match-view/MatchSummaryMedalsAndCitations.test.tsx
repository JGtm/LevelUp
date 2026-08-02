/**
 * MatchSummaryMedalsAndCitations.test.tsx — hauteur ADAPTATIVE des cartes
 * Médailles / Citations (v7.3 lot 2, item 2.1).
 *
 * Le plancher fixe de 280 px réservait la hauteur de 3 rangées de vignettes même
 * sur un match à 2 médailles. Ces tests épinglent la règle de remplacement :
 *   - le corps de carte ne porte plus qu'un plancher bas (MEDALS_CARD_MIN_BODY_HEIGHT),
 *     identique quel que soit le nombre d'éléments (pauvre comme riche) ;
 *   - la carte est un `flex-col` dont le corps est `flex-1` : l'égalisation des
 *     deux cartes d'une rangée revient à la grille CSS du parent, ce qui est ce
 *     qui permet aux DEUX cartes de se compacter ensemble quand elles sont pauvres.
 */
import { describe, expect, it } from 'vitest'
import { render, screen } from '@testing-library/react'

import {
  MEDALS_CARD_MIN_BODY_HEIGHT,
  MatchCitationsSection,
  MatchMedalsSection,
} from './MatchSummaryMedalsAndCitations'
import type { MatchViewText } from './i18n'
import type { MatchCitationSnippet, MatchMedal } from '@/lib/api/types'

const T = {
  sectionMedals: 'Médailles',
  sectionCitations: 'Citations',
  noMedals: 'Aucune médaille',
  noCitations: 'Aucune citation',
  newlyMastered: 'Maîtrisé !',
} as MatchViewText

function medal(id: string): MatchMedal {
  return {
    medal_name_id: id,
    name: `Médaille ${id}`,
    description: null,
    count: 1,
    difficulty: null,
    image_url: null,
    sprite_sheet: null,
  } as unknown as MatchMedal
}

function citation(key: string): MatchCitationSnippet {
  return {
    key,
    name: `Citation ${key}`,
    description: null,
    delta: 2,
    progress_pct: 40,
    image_url: null,
    tier_count: 3,
    tier_index: 1,
    cumulative: 4,
    next_tier_target: 10,
    is_newly_mastered: false,
  } as unknown as MatchCitationSnippet
}

/** minHeight (px) posé sur le corps de la carte rendue. */
function bodyMinHeight(): string {
  return (screen.getByTestId('medals-pane-body') as HTMLElement).style.minHeight
}

describe('MatchSummaryMedalsAndCitations — hauteur adaptative', () => {
  it('2 médailles : plancher bas, pas les 280 px de l ancienne carte', () => {
    render(<MatchMedalsSection medals={[medal('a'), medal('b')]} t={T} />)
    expect(bodyMinHeight()).toBe(`${MEDALS_CARD_MIN_BODY_HEIGHT}px`)
    expect(MEDALS_CARD_MIN_BODY_HEIGHT).toBeLessThan(280)
  })

  it('match riche : MÊME plancher, la hauteur vient du contenu (pas de régression)', () => {
    const many = Array.from({ length: 14 }, (_, i) => medal(`m${i}`))
    render(<MatchMedalsSection medals={many} t={T} />)
    expect(bodyMinHeight()).toBe(`${MEDALS_CARD_MIN_BODY_HEIGHT}px`)
    expect(screen.getAllByText(/^Médaille m/)).toHaveLength(14)
  })

  it('carte vide : message centré sur le plancher bas', () => {
    render(<MatchMedalsSection medals={[]} t={T} />)
    expect(bodyMinHeight()).toBe(`${MEDALS_CARD_MIN_BODY_HEIGHT}px`)
    expect(screen.getByText('Aucune médaille')).toBeInTheDocument()
  })

  it('la carte s étire dans sa cellule de grille (flex-col + corps flex-1)', () => {
    render(<MatchCitationsSection citations={[citation('c1')]} t={T} />)
    const body = screen.getByTestId('medals-pane-body')
    expect(body.className).toContain('flex-1')
    expect(body.parentElement?.className).toContain('flex-col')
  })
})
