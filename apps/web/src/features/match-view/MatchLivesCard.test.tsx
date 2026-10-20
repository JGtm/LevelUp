/**
 * Tests — MatchLivesCard (« Isolement, par joueur », carte I) : une ligne par joueur de l'équipe sur la
 * forme de la carte des Séries temporelles, le nom à l'encre du joueur ; un joueur sans vie rangée
 * n'a pas de ligne, et l'aide ne compte aucune vie écartée.
 */
import { describe, expect, it } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'

import { STARBOARD, STARBOARD_LIVES, XUID } from './matchEmprise.fixtures'
import { buildMatchLives } from './matchEmprise.logic'
import { MATCH_EMPRISE_TEXT } from './matchEmpriseText'
import { MatchLivesCard } from './MatchLivesCard'

const T = MATCH_EMPRISE_TEXT.fr

function afficher(lives = buildMatchLives(STARBOARD_LIVES, STARBOARD.players!)!) {
  return render(<MatchLivesCard lives={lives} inkOf={(x) => `var(--ink-${x})`} meXUID={XUID.jgtm} ut={T.cards} />)
}

describe('MatchLivesCard', () => {
  it('le joueur de la page en gras, chaque nom précédé de son encre', () => {
    afficher()
    const jgtm = screen.getByText('JGtm')
    expect(jgtm.className).toContain('font-semibold')
    expect(screen.getByText('XL JACOB').className).not.toContain('font-semibold')
    expect((jgtm.previousElementSibling as HTMLElement).style.backgroundColor).toBe(`var(--ink-${XUID.jgtm})`)
  })

  it('un joueur sans vie rangée n’a pas de ligne', () => {
    afficher(buildMatchLives({ players: STARBOARD_LIVES.players!.filter((p) => p.xuid !== XUID.madina) }, STARBOARD.players!)!)
    expect(screen.queryByText('Madina97294')).toBeNull()
  })

  it('l’aide dit ce que la carte range, après la portée « une ligne par joueur », sans compte d’écartées', () => {
    afficher()
    fireEvent.mouseEnter(screen.getByRole('button', { name: /informations/i }))
    expect(screen.getByRole('tooltip').textContent).toBe(T.cards.lives.info)
  })
})
