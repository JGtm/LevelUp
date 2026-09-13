/**
 * objectivesOptional.test.ts — UNE GRANDEUR OPTIONNELLE N'ENTRE DANS AUCUN AGRÉGAT.
 *
 * CE FICHIER EXISTE À CAUSE D'UN DÉFAUT MESURÉ (revue adversariale du 2026-09-13, constat 3) :
 * `row.values?.[c] ?? 0` transformait une clé ABSENTE en zéro, et faisait entrer les prises
 * nettes — mesurées sur les seuls matchs dont le film a été lu — dans la somme du rôle
 * « prendre », dans les jauges, dans les écarts et dans la piste du lobby. Un match sans film
 * y comptait comme un match sans prise : exactement le faux zéro que le serveur refuse en
 * n'écrivant AUCUNE clé.
 *
 * La règle tenue ici : une grandeur `optional` absente sort de l'agrégat, NUMÉRATEUR ET
 * DÉNOMINATEUR ; et les agrégats de RÔLE ne la portent pas du tout — un rôle est une somme
 * ramenée à une part, et y mêler deux couvertures rendrait cette part incomparable d'une
 * session à l'autre.
 */
import { describe, expect, it } from 'vitest'

import type { SquadFormesBlock } from '@/lib/api/types'

import { aggregateColumns, aggregateRole, roleLobbyParts } from './objectives'

const MOI = 'x-moi'
const ALLIE = 'x-allie'

/** Les colonnes de la famille drapeau, dont UNE optionnelle (lue du film). */
const COLONNES = [
  { key: 'flag_captures', role: 'take' },
  { key: 'flag_grabs_net', role: 'take', optional: true },
]

/**
 * blocDeuxMatchs — m1 est mesuré par le film, m2 ne l'est pas (aucune clé
 * `flag_grabs_net`). Les deux portent la MÊME grille de colonnes : c'est le contrat du
 * serveur, la grille est par famille et non par match.
 */
function blocDeuxMatchs(): SquadFormesBlock {
  return {
    available: true,
    main_xuid: MOI,
    matches_total: 2,
    matches_measured: 2,
    matches: [
      {
        match_id: 'm1',
        measured: true,
        player_team: 0,
        team_size: 4,
        lobby_size: 8,
        objective: {
          family: 'ctf',
          columns: COLONNES,
          flag_juggle_window_seconds: 1.5,
          players: [
            { xuid: MOI, team_id: 0, values: { flag_captures: 1, flag_grabs_net: 3 } },
            { xuid: ALLIE, team_id: 0, values: { flag_captures: 2, flag_grabs_net: 1 } },
          ],
        },
      },
      {
        match_id: 'm2',
        measured: true,
        player_team: 0,
        // EFFECTIFS DIFFERENTS DE m1, ET C'EST VOLONTAIRE : la parité d'un agrégat est la
        // moyenne des effectifs des matchs qui y ENTRENT. C'est elle qui prouve que m2 sort
        // bel et bien de l'agrégat de la grandeur optionnelle — un total resté identique
        // parce que m2 y vaut zéro ne prouverait rien.
        team_size: 8,
        lobby_size: 16,
        objective: {
          family: 'ctf',
          columns: COLONNES,
          players: [
            // AUCUNE clé `flag_grabs_net` : le film de ce match n'a pas été lu.
            { xuid: MOI, team_id: 0, values: { flag_captures: 5 } },
            { xuid: ALLIE, team_id: 0, values: { flag_captures: 1 } },
          ],
        },
      },
    ],
  } as SquadFormesBlock
}

describe('agrégats et grandeurs optionnelles', () => {
  it("le rôle « prendre » ne porte QUE les colonnes non optionnelles", () => {
    const take = aggregateRole(blocDeuxMatchs(), 'take')
    // flag_captures seul : moi 1 + 5 = 6, mon camp 1+2+5+1 = 9.
    expect(take.me).toBe(6)
    expect(take.team).toBe(9)
  })

  it("la piste du lobby d'un rôle ignore elle aussi la grandeur optionnelle", () => {
    const parts = roleLobbyParts(blocDeuxMatchs(), 'take', [MOI])
    expect(parts.bySquad[MOI]).toBe(6)
    expect(parts.teamRest).toBe(3)
  })

  it("agrégée SEULE, la grandeur optionnelle ne compte QUE les matchs mesurés", () => {
    const agg = aggregateColumns(blocDeuxMatchs().matches ?? [], MOI, [COLONNES[1]])
    // Seul m1 est mesuré : moi 3, mon camp 4. m2 n'entre ni au numérateur ni au dénominateur.
    expect(agg.me).toBe(3)
    expect(agg.team).toBe(4)
    expect(agg.myShareOfTeamPct).toBeCloseTo(75, 6)
    // Et la dispersion ne porte que sur le match mesuré.
    expect(agg.teamSpread.measured).toBe(1)
    // LA PREUVE QUE m2 EST SORTI : la parité est celle de m1 SEUL (équipe de 4 -> 25 %),
    // pas la moyenne des deux matchs (4 et 8 -> 16,7 %).
    expect(agg.teamParity).toBeCloseTo(25, 6)
  })

  it('une colonne ordinaire garde tous ses matchs, y compris ses zéros', () => {
    const agg = aggregateColumns(blocDeuxMatchs().matches ?? [], MOI, [COLONNES[0]])
    expect(agg.me).toBe(6)
    expect(agg.teamSpread.measured).toBe(2)
    // Les deux matchs entrent : la parité est la moyenne de leurs effectifs (4 et 8).
    expect(agg.teamParity).toBeCloseTo(100 / 6, 6)
  })

  it('un scope où RIEN ne mesure la grandeur rend un agrégat vide, pas des zéros trompeurs', () => {
    const bloc = blocDeuxMatchs()
    // On retire la mesure de m1 : plus aucun match ne porte la clé.
    const m1 = (bloc.matches ?? [])[0]
    for (const p of m1.objective?.players ?? []) delete p.values?.flag_grabs_net
    const agg = aggregateColumns(bloc.matches ?? [], MOI, [COLONNES[1]])
    expect(agg.me).toBe(0)
    expect(agg.team).toBe(0)
    expect(agg.myShareOfTeamPct).toBeNull()
    expect(agg.teamSpread.measured).toBe(0)
  })
})
