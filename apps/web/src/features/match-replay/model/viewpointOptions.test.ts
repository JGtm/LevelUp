/**
 * Tests — buildViewpointOptions (ce que le menu de point de vue propose).
 *
 * TROIS RÈGLES Y SONT INVISIBLES À LA RELECTURE, et ce sont elles que ces cas tiennent :
 *
 *  1. LA VALEUR D'UNE OPTION EST CELLE DE LA BASE. Un bot porte deux identités — la clé
 *     synthétique `bot:<nom>` côté film, un xuid `bid(N.0)` côté base — et les marques de la
 *     frise s'apparient sur la seconde. Rendre la première changerait bien le point de vue,
 *     vers un joueur que rien ne reconnaît, et laisserait la piste VIDE en silence. C'est le
 *     défaut nommé au §3.3 du plan, et il ne se voit ni au typage ni à l'écran.
 *  2. UN JOUEUR SANS LIGNE DE TABLEAU DE SCORE EST LISTÉ MAIS INERTE (décision 7 bis) : sans
 *     cette ligne il n'a pas de xuid de base, et `collectKillEvents` jette déjà les kills d'un
 *     acteur hors tableau. Le retirer de la liste se lirait « il n'était pas là » (faux) ; le
 *     laisser cliquable ferait un choix sans effet.
 *  3. LES SECTIONS SONT LES CAMPS DU FILM (décision du 2026-10-06) : un camp par désignateur,
 *     nommé par la cascade des colonnes de fiches, et JAMAIS une section « sans équipe » — un
 *     joueur dont le film tait l'équipe n'est dans aucune.
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'
import { campLabel } from '@/lib/replay/replayCamps'
import type { ReplayPlayer } from '@/lib/replay/rosterLogic'

import { REPLAY_TEXT } from '../i18n/i18n'
import { buildViewpointOptions, type ViewpointOptionLabels } from './viewpointOptions'

const T = REPLAY_TEXT.fr

/** Les libellés tels que la frise les passe : la cascade des colonnes, la raison d'inertie. */
const LABELS: ViewpointOptionLabels = {
  campLabelOf: (camp) => campLabel(camp, camp.players.map((p) => p.board), T),
  noData: T.viewpointNoData,
}

function board(over: Partial<MatchScoreboardRow>): MatchScoreboardRow {
  return { xuid: 'x', gamertag: 'GT', team_side: 't0', ...over } as MatchScoreboardRow
}

/** Un joueur du rejeu, de l'équipe 0 du film par défaut — comme toute entrée publiée en a une. */
function player(over: Partial<ReplayPlayer>): ReplayPlayer {
  return { xuid: 'x', team: 0, lives: [], ...over }
}

describe('buildViewpointOptions — les sections sont les camps du film', () => {
  it('un camp par section, dans l’ordre des désignateurs, nommé comme la colonne de fiches', () => {
    const groups = buildViewpointOptions(
      [
        player({ xuid: '2', team: 1, board: board({ xuid: '2', gamertag: 'Rival', team_side: 't1' }) }),
        player({ xuid: '1', team: 0, board: board({ xuid: '1', gamertag: 'JGtm', team_side: 't0' }) }),
      ],
      LABELS,
    )
    expect(groups.map((g) => [g.key, g.label, g.options.map((o) => o.label)])).toEqual([
      ['camp:0', 'Équipe Eagle', ['JGtm']],
      ['camp:1', 'Équipe Cobra', ['Rival']],
    ])
  })

  it('un joueur SANS ligne de tableau de score reste dans le camp que le film lui donne', () => {
    const groups = buildViewpointOptions(
      [
        player({ xuid: '1', board: board({ xuid: '1', gamertag: 'JGtm' }) }),
        player({ xuid: 'bot:Fantome', bot: true, filmName: 'Fantome [bot]' }),
      ],
      LABELS,
    )
    expect(groups).toHaveLength(1)
    expect(groups[0].options.map((o) => o.label)).toEqual(['JGtm', 'Fantome'])
  })

  it('un joueur dont le film TAIT l’équipe n’est dans aucune section — et n’en ouvre aucune', () => {
    const groups = buildViewpointOptions(
      [
        player({ xuid: '1', board: board({ xuid: '1', gamertag: 'JGtm' }) }),
        player({ xuid: 'bot:Sandwolf', bot: true, filmName: 'Sandwolf [bot]', team: undefined }),
      ],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Équipe Eagle'])
    expect(groups.flatMap((g) => g.options.map((o) => o.label))).toEqual(['JGtm'])
  })

  it('un camp qu’aucune ligne de feuille ne nomme garde « Équipe N » de son désignateur', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: 'bot:Fantome', bot: true, filmName: 'Fantome [bot]', team: 1 })],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Équipe 1'])
  })

  it('une section VIDE n’est pas rendue : un groupe sans nom affichable n’a rien à dire', () => {
    // Une trace anonyme (caméra, spectateur de fin de partie) n'a ni base ni nom de film.
    expect(buildViewpointOptions([player({ xuid: 'anonyme' })], LABELS)).toEqual([])
  })

  it('un `team_side` illisible ne se traduit pas en camp inventé : « Équipe N » du désignateur', () => {
    const groups = buildViewpointOptions(
      [player({ xuid: '1', team: 1, board: board({ xuid: '1', gamertag: 'JGtm', team_side: 'bizarre' }) })],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Équipe 1'])
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

  it('AVEC une ligne mais SANS côté de feuille : active — elle a un xuid de base, sa piste peut se peupler', () => {
    // Le camp est celui du film ; la ligne sans côté ne le nomme pas (plancher « Équipe 0 »),
    // mais l'option, elle, marche : les deux conditions ne sont pas la même.
    const groups = buildViewpointOptions(
      [player({ xuid: '9', board: board({ xuid: '9', gamertag: 'Nomade', team_side: null }) })],
      LABELS,
    )
    expect(groups.map((g) => g.label)).toEqual(['Équipe 0'])
    expect(groups[0].options[0]).toEqual({
      value: '9',
      label: 'Nomade',
      disabled: false,
      title: 'Nomade',
    })
  })
})
