/**
 * Tests — MatchLivesCard (« Isolement, par joueur », carte I) : une ligne par joueur de l'équipe sur la
 * forme de la carte des Séries temporelles, le nom à l'encre du joueur, une ligne qui le dit pour un
 * joueur sans vie rangée, les vies écartées comptées dans l'aide.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { STARBOARD, STARBOARD_LIVES, XUID } from './matchEmprise.fixtures'
import { buildMatchLives } from './matchEmprise.logic'
import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'
import { MatchLivesCard } from './MatchLivesCard'

const T = MATCH_EMPRISE_TEXT.fr

function afficher(lives = buildMatchLives(STARBOARD_LIVES, STARBOARD.players!)!) {
  return render(<MatchLivesCard lives={lives} inkOf={(x) => `var(--ink-${x})`} meXUID={XUID.jgtm} ut={T.cards} noRankedLife={T.own.noRankedLife} />)
}

describe('MatchLivesCard', () => {
  it('le joueur de la page en gras, chaque nom précédé de son encre', () => {
    afficher()
    const jgtm = screen.getByText('JGtm')
    expect(jgtm.className).toContain('font-semibold')
    expect(screen.getByText('XL JACOB').className).not.toContain('font-semibold')
    expect((jgtm.previousElementSibling as HTMLElement).style.backgroundColor).toBe(`var(--ink-${XUID.jgtm})`)
  })

  it('un joueur sans vie rangée garde sa ligne, qui le dit', () => {
    afficher(buildMatchLives({ players: STARBOARD_LIVES.players!.filter((p) => p.xuid !== XUID.madina) }, STARBOARD.players!)!)
    expect(screen.getByTestId(`match-emprise-lives-none-${XUID.madina}`).textContent).toContain(T.own.noRankedLife)
  })

  it('l’aide compte les vies écartées des trois causes, après la portée « une ligne par joueur »', () => {
    const players = STARBOARD_LIVES.players!.map((p) => ({ ...p, excluded_unlocated: 1, excluded_no_radar: 0, excluded_unpublishable: 0 }))
    afficher(buildMatchLives({ players }, STARBOARD.players!)!)
    fireEvent.mouseEnter(screen.getByRole('button', { name: /informations/i }))
    expect(screen.getByRole('tooltip').textContent).toBe(T.cards.lives.info(4, 0, 0))
  })
})
