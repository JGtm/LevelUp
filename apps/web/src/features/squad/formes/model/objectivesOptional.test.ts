/**
 * objectivesOptional.test.ts — LA LECTURE DU BLOC `formes_retenues` PAR LES CARTES D'OBJECTIF, et
 * la règle des grandeurs optionnelles.
 *
 * UNE GRANDEUR OPTIONNELLE N'ENTRE DANS AUCUN AGRÉGAT QUAND ELLE MANQUE (revue adversariale du
 * 2026-09-13, constat 3) : `row.values?.[c] ?? 0` transformait une clé ABSENTE en zéro et faisait
 * compter un match sans film comme un match sans prise — exactement le faux zéro que le serveur
 * refuse en n'écrivant AUCUNE clé. Un match qui ne mesure pas une colonne optionnelle sort donc de
 * l'agrégat pour TOUTES les colonnes de l'ensemble.
 *
 * Les cas « objectifs » de l'ancien `padsObjectives.test.ts` (sélection des matchs, colonnes
 * publiées, valeur d'un joueur) vivent ici depuis la suppression des cartes de formes (plan
 * PLAN_TIMESERIES_USAGES_EMPRISE_2026-10-05, §4.B).
 */
import { describe, expect, it } from 'vitest'

import type { SquadFormesBlock } from '@/lib/api/types'

import { aggregateColumns, columnsOfFamily, matchesOfFamily, objectiveCell, objectiveFamilies, objectiveMatches } from './objectives'

const MOI = 'x-moi'
const ALLIE = 'x-allie'
const ADV = 'x-adv'

/** Les colonnes de la famille drapeau, dont UNE optionnelle (lue du film) et une durée. */
const COLONNES = [
  { key: 'flag_captures', role: 'take' },
  { key: 'flag_grabs_net', role: 'take', optional: true },
  { key: 'time_as_flag_carrier_seconds', role: 'hold', duration: true },
]

/**
 * blocDeuxMatchs — m1 est mesuré par le film, m2 ne l'est pas (aucune clé `flag_grabs_net`). Les
 * deux portent la MÊME grille de colonnes : c'est le contrat du serveur. Un troisième match, sans
 * objectif, ne doit compter nulle part.
 */
function blocDeuxMatchs(): SquadFormesBlock {
  return {
    available: true,
    main_xuid: MOI,
    matches_total: 3,
    matches_measured: 3,
    matches: [
      {
        match_id: 'm1',
        player_team: 0,
        objective: {
          family: 'ctf',
          columns: COLONNES,
          players: [
            { xuid: MOI, team_id: 0, values: { flag_captures: 1, flag_grabs_net: 3, time_as_flag_carrier_seconds: 20 } },
            { xuid: ALLIE, team_id: 0, values: { flag_captures: 2, flag_grabs_net: 1, time_as_flag_carrier_seconds: 10 } },
            { xuid: ADV, team_id: 1, values: { flag_captures: 4, flag_grabs_net: 5, time_as_flag_carrier_seconds: 30 } },
          ],
        },
      },
      {
        match_id: 'm2',
        player_team: 0,
        objective: {
          family: 'ctf',
          columns: COLONNES,
          players: [
            // AUCUNE clé `flag_grabs_net` : le film de ce match n'a pas été lu.
            { xuid: MOI, team_id: 0, values: { flag_captures: 5, time_as_flag_carrier_seconds: 0 } },
            { xuid: ALLIE, team_id: 0, values: { flag_captures: 1, time_as_flag_carrier_seconds: 7 } },
            { xuid: ADV, team_id: 1, values: { flag_captures: 2, time_as_flag_carrier_seconds: 9 } },
          ],
        },
      },
      { match_id: 'm3', player_team: 0 },
    ],
  } as SquadFormesBlock
}

describe('lecture du bloc', () => {
  it('ne retient que les matchs qui portent un objectif ; familles dans l’ordre d’apparition', () => {
    const b = blocDeuxMatchs()
    expect(objectiveMatches(b).map((m) => m.match_id)).toEqual(['m1', 'm2'])
    expect(objectiveFamilies(b)).toEqual(['ctf'])
    expect(matchesOfFamily(b, 'ctf')).toHaveLength(2)
  })

  it('sert les colonnes publiées par le serveur, avec leur unité', () => {
    const cols = columnsOfFamily(blocDeuxMatchs(), 'ctf')
    expect(cols.map((c) => c.key)).toEqual(['flag_captures', 'flag_grabs_net', 'time_as_flag_carrier_seconds'])
    expect(cols[2].duration).toBe(true)
  })

  it('rend 0 pour un joueur absent de la feuille d’objectif', () => {
    const m1 = matchesOfFamily(blocDeuxMatchs(), 'ctf')[0]
    expect(objectiveCell(m1, 'x-inconnu', COLONNES[0])).toBe(0)
  })

  it('rend « non mesuré » (null), pas 0, pour une grandeur optionnelle absente', () => {
    const m2 = matchesOfFamily(blocDeuxMatchs(), 'ctf')[1]
    expect(objectiveCell(m2, MOI, COLONNES[1])).toBeNull()
    expect(objectiveCell(m2, MOI, { ...COLONNES[1], optional: false })).toBe(0)
  })
})

describe('aggregateColumns — mon camp et le lobby', () => {
  it('une colonne ordinaire : tous les matchs, les deux camps', () => {
    // Captures : mon camp 1+2+5+1 = 9, lobby 9+4+2 = 15.
    expect(aggregateColumns(blocDeuxMatchs().matches ?? [], [COLONNES[0]])).toEqual({ team: 9, lobby: 15 })
  })

  it('la grandeur optionnelle seule : les seuls matchs qui la mesurent', () => {
    expect(aggregateColumns(blocDeuxMatchs().matches ?? [], [COLONNES[1]])).toEqual({ team: 4, lobby: 9 })
  })

  it('mêlée à une colonne ordinaire, elle fait sortir le match non mesuré pour TOUTES les colonnes', () => {
    // m1 seul : mon camp (1+3) + (2+1) = 7, lobby 7 + (4+5) = 16. Avec m2 on lirait 13 / 24.
    expect(aggregateColumns(blocDeuxMatchs().matches ?? [], [COLONNES[0], COLONNES[1]])).toEqual({ team: 7, lobby: 16 })
  })

  it('un scope où RIEN ne mesure la grandeur rend un agrégat vide, pas des zéros trompeurs', () => {
    const bloc = blocDeuxMatchs()
    for (const p of bloc.matches![0].objective?.players ?? []) delete p.values?.flag_grabs_net
    expect(aggregateColumns(bloc.matches ?? [], [COLONNES[0], COLONNES[1]])).toEqual({ team: 0, lobby: 0 })
  })

  it('camp inconnu : rien pour mon camp, tout pour le lobby', () => {
    const bloc = blocDeuxMatchs()
    bloc.matches![0].player_team = undefined
    expect(aggregateColumns([bloc.matches![0]], [COLONNES[0]])).toEqual({ team: 0, lobby: 7 })
  })
})
