/**
 * Tests — le NOM et l'ENCRE d'un camp dans la colonne des équipes (`ReplayRosterColumn`).
 *
 * Le titre d'un camp porte le libellé résolu — la cascade de la feuille, « Équipe N » du désignateur quand
 * elle se tait, jamais « sans équipe » — et sa pastille se teint de l'allégeance du CAMP DU FILM : son
 * désignateur comparé à l'équipe du film du joueur regardé (`FilmAllegiance.ofTeam`). La feuille de la base
 * ne reconnaît pas un bot par son xuid (le film le désigne par `bot:<nom>`) : un camp de bots garde pourtant
 * son encre. Un camp dont l'allégeance n'est pas connue n'emprunte aucune des deux couleurs.
 */
import { useMemo } from 'react'
import { describe, expect, it } from 'vitest'
import { render } from '@testing-library/react'

import type { MatchScoreboardRow, ReplayDocument } from '@/lib/api/types'
import { filmAllegianceOf, NO_ALLEGIANCE, type FilmAllegiance } from '@/lib/replay/filmAllegiance'
import { buildPlayers } from '@/lib/replay/rosterLogic'

import { ReplayRosterColumn } from './ReplayRosterColumn'
import { scoreboardRow } from '../test/scoreboardRow'
import { testReplayDoc } from '../test/testDoc'

/** La colonne montée depuis une feuille, comme la page : jointure film <-> base, personne de suivi. */
function Colonne({ doc, scoreboard, allegiance, locale = 'fr' }: {
  doc: ReturnType<typeof testReplayDoc>
  scoreboard: MatchScoreboardRow[]
  allegiance: FilmAllegiance
  locale?: 'fr' | 'en'
}) {
  const players = useMemo(() => buildPlayers(doc, scoreboard), [doc, scoreboard])
  return <ReplayRosterColumn doc={doc} players={players} frame={10} locale={locale} allegiance={allegiance} tracked={null} />
}

/** Une vie de 0 à 200, sur un slot ; un bot n'a pas de xuid de trace, le film le nomme. */
function vie(slot: number, who: { xuid?: string; bot?: string }): NonNullable<ReplayDocument['tracks']>[number] {
  return { slot, team: -1, ...who, startFrame: 0, endFrame: 200, points: [{ t: 0, x: 0, y: 0 }] } as never
}

const titres = (c: HTMLElement) => [...c.querySelectorAll('h3')] as HTMLElement[]
const pastille = (titre: HTMLElement) => titre.firstElementChild as HTMLElement

describe('ReplayRosterColumn — le nom des camps', () => {
  const twoTeams = () =>
    testReplayDoc({
      roster: [
        { xuid: 'A', filmIndex: 0, name: 'Alpha', team: 0 },
        { xuid: 'B', filmIndex: 1, name: 'Bravo', team: 1 },
      ],
      tracks: [vie(512, { xuid: 'A' }), vie(513, { xuid: 'B' })],
    })
  const board = [scoreboardRow('A', 'Alpha', 't0'), scoreboardRow('B', 'Bravo', 't1')]

  it('résout `t0` en « Équipe Eagle » (FR) et `t1` en « Team Cobra » (EN), jamais l’identifiant de transport', () => {
    const fr = render(<Colonne doc={twoTeams()} scoreboard={board} allegiance={NO_ALLEGIANCE} />)
    expect(titres(fr.container).map((h) => h.textContent)).toEqual(['Équipe Eagle', 'Équipe Cobra'])
    expect(fr.queryByText('t0')).toBeNull()
    fr.unmount()
    const en = render(<Colonne doc={twoTeams()} scoreboard={board} allegiance={NO_ALLEGIANCE} locale="en" />)
    expect(en.getByText('Team Cobra')).toBeTruthy()
  })

  it('numérote une équipe hors référentiel plutôt que d’inventer un nom', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 }],
      tracks: [vie(512, { xuid: 'A' })],
    })
    const vue = render(<Colonne doc={doc} scoreboard={[scoreboardRow('A', 'Alpha', 't12')]} allegiance={NO_ALLEGIANCE} />)
    expect(vue.getByText('Équipe 12')).toBeTruthy()
  })

  it('camp que la feuille ne nomme pas : « Équipe N » de son désignateur, jamais « Sans équipe »', () => {
    const doc = testReplayDoc({
      roster: [{ xuid: 'A', filmIndex: 0, seat: 0, name: 'Alpha', team: 0 }],
      tracks: [vie(512, { xuid: 'A' })],
    })
    const vue = render(<Colonne doc={doc} scoreboard={[]} allegiance={NO_ALLEGIANCE} />)
    expect(vue.getByText('Équipe 0')).toBeTruthy()
    expect(vue.queryByText(/Sans équipe/)).toBeNull()
  })
})

describe('ReplayRosterColumn — l’encre des camps vient de l’équipe du film', () => {
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

  function pastilles(reference: string, muet = false) {
    const { doc, board } = partie(muet)
    const vue = render(<Colonne doc={doc} scoreboard={board} allegiance={filmAllegianceOf(doc, board, reference)} />)
    const [eagle, cobra] = titres(vue.container)
    return { eagle: pastille(eagle).style.background, cobra: pastille(cobra).style.background }
  }

  it('un camp de BOTS adverse prend l’encre ADVERSE ; celui de la référence l’encre alliée', () => {
    const { eagle, cobra } = pastilles('A')
    expect(eagle).toContain('var(--ac-team-ally)')
    expect(cobra).toContain('var(--ac-team-enemy)')
  })

  it('vu d’un bot (sa ligne de feuille le relie au film), les encres s’échangent', () => {
    const { eagle, cobra } = pastilles('bid(0.0)')
    expect(eagle).toContain('var(--ac-team-enemy)')
    expect(cobra).toContain('var(--ac-team-ally)')
  })

  it('référence dont le film tait l’équipe : AUCUN camp n’a d’encre d’allégeance — jamais devinée par la feuille', () => {
    const { eagle, cobra } = pastilles('A', true)
    for (const encre of [eagle, cobra]) {
      expect(encre).toContain('var(--border)')
      expect(encre).not.toContain('team')
    }
  })
})
