/**
 * Tests — L'ENCRE DES COLONNES EST L'ALLÉGEANCE DU FILM (2026-10-06).
 *
 * Le titre d'une colonne se teint de l'allégeance de son CAMP : son désignateur du film comparé à
 * l'équipe du film du joueur regardé (`FilmAllegiance.ofTeam`). Il se teignait par les occupants
 * présents que la table d'identité de la feuille reconnaissait : un camp de bots restait neutre
 * (la table est clée par xuid de base, le film désigne un bot par `bot:<nom>`).
 */
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import { filmAllegianceOf } from '@/lib/replay/filmAllegiance'

import { ReplayTeams } from './ReplayTeams'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** Une vie de 0 à 200, sur un slot ; un bot n'a pas de xuid de trace, le film le nomme. */
function vie(slot: number, who: { xuid?: string; bot?: string }) {
  return { slot, team: -1, ...who, startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] }
}

/**
 * Alpha (humain, la référence) et le bot Sandwolf au camp 0 ; les bots Ritzy et Oscar au camp 1.
 * La feuille porte les bots (`bid(N.0)`, `is_bot`) comme la base les publie.
 */
function partie(muet = false) {
  const doc = testReplayDoc({
    roster: [
      { xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', ...(muet ? {} : { team: 0 }) },
      { xuid: '', bot: true, filmIndex: 8, seat: 1, name: 'Sandwolf [bot]', team: 0 },
      { xuid: '', bot: true, filmIndex: 9, seat: 0, name: 'Ritzy [bot]', team: 1 },
      { xuid: '', bot: true, filmIndex: 10, seat: 1, name: 'Oscar [bot]', team: 1 },
    ],
    tracks: [
      vie(512, { xuid: 'A' }),
      vie(513, { bot: 'Sandwolf [bot]' }),
      vie(514, { bot: 'Ritzy [bot]' }),
      vie(515, { bot: 'Oscar [bot]' }),
    ],
  })
  const board = [
    scoreboardRow('A', 'Alpha', 't0', { is_me: true }),
    scoreboardRow('bid(44.0)', 'Sandwolf', 't0', { is_bot: true }),
    scoreboardRow('bid(0.0)', 'Ritzy', 't1', { is_bot: true }),
    scoreboardRow('bid(50.0)', 'Oscar', 't1', { is_bot: true }),
  ]
  return { doc, board }
}

function titres(reference: string, muet = false) {
  const { doc, board } = partie(muet)
  const vue = render(
    <ReplayTeams allegiance={filmAllegianceOf(doc, board, reference)} doc={doc} scoreboard={board} frame={10} locale="fr" />,
  )
  const titre = (libelle: string) => vue.getByText(libelle).parentElement as HTMLElement
  return { eagle: titre('Équipe Eagle'), cobra: titre('Équipe Cobra') }
}

describe('ReplayTeams — l’encre des colonnes vient de l’équipe du film', () => {
  it('un camp de BOTS adverse prend l’encre ADVERSE (il restait neutre) ; celui de la référence l’encre alliée', () => {
    const { eagle, cobra } = titres('A')
    expect(eagle.style.borderLeft).toContain('var(--ac-team-ally)')
    expect(cobra.style.borderLeft).toContain('var(--ac-team-enemy)')
    expect(cobra.className).toContain('text-foreground')
  })

  it('vu d’un bot (sa ligne de feuille le relie au film), les encres s’échangent', () => {
    const { eagle, cobra } = titres('bid(0.0)')
    expect(eagle.style.borderLeft).toContain('var(--ac-team-enemy)')
    expect(cobra.style.borderLeft).toContain('var(--ac-team-ally)')
  })

  it('référence dont le film tait l’équipe : AUCUNE colonne n’a d’encre de camp — jamais devinée par la feuille', () => {
    const { eagle, cobra } = titres('A', true)
    for (const titre of [eagle, cobra]) {
      expect(titre.style.borderLeft).toContain('var(--border)')
      expect(titre.style.borderLeft).not.toContain('team')
      expect(titre.className).toContain('text-muted-foreground')
    }
  })
})
