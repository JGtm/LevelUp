/**
 * Tests — buildViewpointOptions (ce que le menu de point de vue propose).
 *
 * DEUX RÈGLES Y SONT INVISIBLES À LA RELECTURE, et ce sont elles que ces cas tiennent :
 *
 *  1. LA VALEUR D'UNE OPTION EST CELLE DE LA BASE. Un bot porte deux identités — la clé
 *     synthétique `bot:<nom>` côté film, un xuid `bid(N.0)` côté base — et les marques de la
 *     frise s'apparient sur la seconde. Rendre la première changerait bien le point de vue,
 *     vers un joueur que rien ne reconnaît, et laisserait la piste VIDE en silence. C'est le
 *     défaut nommé au §3.3 du plan, et il ne se voit ni au typage ni à l'écran.
 *  2. UN JOUEUR SANS LIGNE DE TABLEAU DE SCORE EST LISTÉ MAIS INERTE (décision 7 bis) : sans
 *     cette ligne il n'a ni camp ni xuid de base, et `collectKillEvents` jette déjà les kills
 *     d'un acteur hors tableau. Le retirer de la liste se lirait « il n'était pas là » (faux) ;
 *     le laisser cliquable ferait un choix sans effet.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { buildViewpointOptions, type ViewpointOptionLabels } from './viewpointOptions'

const LABELS: ViewpointOptionLabels = {
  teamLabelOf: (id) => (id === 0 ? 'Cobalt' : `Équipe ${id}`),
  noTeam: 'Sans équipe',
  noData: 'Aucune donnée de match pour ce joueur',
}

function board(over: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return { xuid: 'x', gamertag: 'GT', team_side: 't0', ...over } as MatchScoreboardRow
}

function player(over: Partial<ReplayPlayer>): ReplayPlayer {
  return { xuid: 'x', lives: [], ...over }
}

describe('buildViewpointOptions — les sections', () => {
  it('un camp par section, nommé par la cascade du tableau de score', () => {
    const groups = buildViewpointOptions(
      [
        player({ xuid: '1', board: board({ xuid: '1', gamertag: 'JGtm', team_side: 't0' }) }),
        player({ xuid: '2', board: board({ xuid: '2', gamertag: 'Rival', team_side: 't1' }) }),
      ],
      LABELS,
    )
    expect(groups.map((g) => [g.label, g.options.map((o) => o.label)])).toEqual([
      ['Cobalt', ['JGtm']],
      ['Équipe 1', ['Rival']],
    ])
  })

  it('les joueurs SANS ligne de tableau de score ont leur propre section, nommée', () => {
    const groups = buildViewpointOptions(
      [
        player({ xuid: '1', board: board({ xuid: '1', gamertag: 'JGtm' }) }),
        player({ xuid: 'bot:Fantome', bot: true, filmName: 'Fantome [bot]' }),
      ],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Cobalt', 'Sans équipe'])
  })

  it('une section VIDE n’est pas rendue : un groupe sans nom affichable n’a rien à dire', () => {
    // Une trace anonyme (caméra, spectateur de fin de partie) n'a ni base ni nom de film.
    expect(buildViewpointOptions([player({ xuid: 'anonyme' })], LABELS)).toEqual([])
  })

  it('un `team_side` illisible ne se traduit pas en camp inventé : la valeur brute passe', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: '1', board: board({ xuid: '1', gamertag: 'JGtm', team_side: 'bizarre' }) })],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['bizarre'])
  })
})

describe('buildViewpointOptions — la valeur d’une option (le piège des bots)', () => {
  it('LE BOT JOINT REND SON XUID DE BASE, pas la clé `bot:<nom>` du film', () => {
    const groups = buildViewpointOptions(
      [
        player({
          xuid: 'bot:Cortana',
          bot: true,
          filmName: 'Cortana [bot]',
          board: board({ xuid: 'bid(3.0)', gamertag: 'Cortana' }),
        }),
      ],
      LABELS,
    )
    expect(groups[0].options).toEqual([
      { value: 'bid(3.0)', label: 'Cortana', disabled: false, title: 'Cortana' },
    ])
  })

  it('un joueur ORDINAIRE rend son xuid, qui est le même des deux côtés', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: '2533274', board: board({ xuid: '2533274', gamertag: 'JGtm' }) })],
      LABELS,
    )
    expect(groups[0].options[0].value).toBe('2533274')
  })

  it('le SUFFIXE « [bot] » du film ne sort jamais à l’écran', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: 'bot:Fantome', bot: true, filmName: 'Fantome [bot]' })],
      LABELS,
    )
    expect(groups[0].options[0].label).toBe('Fantome')
  })

  it('LE NOM VIENT DE LA BASE D’ABORD : c’est elle qui suit un changement de pseudo', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: '1', filmName: 'AncienNom', board: board({ xuid: '1', gamertag: 'NouveauNom' }) })],
      LABELS,
    )
    expect(groups[0].options[0].label).toBe('NouveauNom')
  })
})

describe('buildViewpointOptions — l’option inerte (décision 7 bis)', () => {
  it('sans ligne de tableau de score : DÉSACTIVÉE, et l’infobulle dit pourquoi', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: 'bot:Fantome', bot: true, filmName: 'Fantome' })],
      LABELS,
    )
    expect(groups[0].options[0]).toEqual({
      value: 'bot:Fantome',
      label: 'Fantome',
      disabled: true,
      title: 'Aucune donnée de match pour ce joueur',
    })
  })

  it('AVEC une ligne mais SANS camp : active — elle a un xuid de base, sa piste peut se peupler', () => {
    // Le groupe reste « Sans équipe » (le camp n'est pas transmis), mais l'option, elle, marche :
    // les deux conditions ne sont pas la même, et les confondre priverait ce joueur du menu.
    const groups = buildViewpointOptions(
      [player({ xuid: '9', board: board({ xuid: '9', gamertag: 'Nomade', team_side: null }) })],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Sans équipe'])
    expect(groups[0].options[0]).toEqual({
      value: '9',
      label: 'Nomade',
      disabled: false,
      title: 'Nomade',
    })
  })
})
