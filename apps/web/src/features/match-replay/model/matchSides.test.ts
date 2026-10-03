/**
 * matchSides — CARACTÉRISATION DU 2026-09-06, écrite AVANT le point de vue du rejeu.
 *
 * POURQUOI CE FICHIER, ET POURQUOI MAINTENANT. Ce module dit QUI EST DE QUEL CÔTÉ à quatre
 * calques du rejeu (bombe, drapeaux, zones) et aux sons d'objectif, et il n'avait AUCUN test
 * direct : il n'était couvert que par ses consommateurs, c'est-à-dire par des tests qui
 * pourraient tous rester verts pendant qu'il rend un camp faux (chacun d'eux fixe une couleur
 * ou un son, pas la lecture du tableau de score). Le chantier « point de vue de la frise »
 * (plan `.ai/PLAN_FRISE_POINT_DE_VUE_2026-09-06.md`) va router ces deux lectures par un point
 * de vue sélectionnable ; avant de les toucher, on photographie ce qu'elles rendent.
 *
 * CE QUE CES TESTS FIXENT : le comportement CONSTATÉ aujourd'hui, sur la seule entrée que le
 * module accepte — le tableau de score. Les deux règles qu'il refuse de deviner (« pas de
 * ligne moi, donc pas de camp », « pas de camp lisible, donc pas d'entrée ») sont ici des
 * assertions, plus seulement un commentaire d'en-tête.
 *
 * CONTRAT POUR LE LOT SUIVANT : ajouts seulement. Aucune ligne supprimée, aucune modifiée —
 * un appel sans point de vue doit rendre après le chantier exactement ce qu'il rend ici.
 *
 * La fixture est ÉCRITE ICI, volontairement, et non partagée avec `xuidMeta.test.ts` : les
 * deux modules lisent le même tableau de score mais n'ont aucune dépendance l'un à l'autre,
 * et une fixture commune ferait bouger deux photographies d'un seul geste.
 */
import { describe, expect, it } from 'vitest'

import {
  allyTeamFromScoreboard,
  teamOfXuidFromScoreboard,
  type ScoreboardSide,
} from './matchSides'

/**
 * Le lobby témoin : deux camps, une ligne « moi » dans `t0`, un bot (clé `bid(N.0)`) rangé
 * dans `t1`, et une ligne sans camp transmis.
 */
const BOARD: ScoreboardSide[] = [
  { xuid: 'me-1', team_side: 't0', is_me: true },
  { xuid: 'ally-2', team_side: 't0', is_me: false },
  { xuid: 'foe-1', team_side: 't1', is_me: false },
  { xuid: 'foe-2', team_side: 't1', is_me: false },
  { xuid: 'bid(1.0)', team_side: 't1', is_me: false },
  { xuid: 'nomad-9', team_side: null, is_me: false },
]

/** Le même lobby SANS aucune ligne « moi ». */
const BOARD_SANS_MOI: ScoreboardSide[] = BOARD.map((r) => ({ ...r, is_me: false }))

describe('allyTeamFromScoreboard — le camp de référence, ou rien', () => {
  it('(a) ligne « moi » avec camp : rend l’identifiant de CE camp', () => {
    expect(allyTeamFromScoreboard(BOARD)).toBe(0)
  })

  it('rend l’identifiant réel, jamais un rang : un camp `t5` rend 5', () => {
    expect(allyTeamFromScoreboard([{ xuid: 'me-1', team_side: 't5', is_me: true }])).toBe(5)
  })

  it('(d) aucune ligne « moi » : null — pas de camp deviné à partir du premier venu', () => {
    expect(allyTeamFromScoreboard(BOARD_SANS_MOI)).toBeNull()
  })

  it('ligne « moi » sans camp transmis : null', () => {
    const sansCamp: ScoreboardSide[] = [
      { xuid: 'me-1', team_side: null, is_me: true },
      { xuid: 'foe-1', team_side: 't1', is_me: false },
    ]
    expect(allyTeamFromScoreboard(sansCamp)).toBeNull()
  })

  it('camp illisible sur la ligne « moi » : null, et surtout pas 0', () => {
    // `null` n'est PAS « équipe 0 » : les appelants s'en servent pour se taire.
    expect(allyTeamFromScoreboard([{ xuid: 'me-1', team_side: 'red', is_me: true }])).toBeNull()
  })

  it('tableau de score absent : null', () => {
    expect(allyTeamFromScoreboard(null)).toBeNull()
    expect(allyTeamFromScoreboard(undefined)).toBeNull()
    expect(allyTeamFromScoreboard([])).toBeNull()
  })
})

describe('teamOfXuidFromScoreboard — la table xuid → équipe, en entier', () => {
  it('(a) rend une entrée par ligne AU CAMP LISIBLE, et aucune pour les autres', () => {
    const table = teamOfXuidFromScoreboard(BOARD)
    expect(Object.fromEntries(table)).toEqual({
      'me-1': 0,
      'ally-2': 0,
      'foe-1': 1,
      'foe-2': 1,
      'bid(1.0)': 1,
    })
    expect(table.size).toBe(5)
  })

  it('(d) la table ne dépend PAS de la ligne « moi » : sans elle, elle est identique', () => {
    expect(Object.fromEntries(teamOfXuidFromScoreboard(BOARD_SANS_MOI))).toEqual(
      Object.fromEntries(teamOfXuidFromScoreboard(BOARD)),
    )
  })

  it('xuid connu : son propre camp, bot compris — un bot rangé dans un camp en a un', () => {
    const table = teamOfXuidFromScoreboard(BOARD)
    expect(table.get('foe-1')).toBe(1)
    expect(table.get('bid(1.0)')).toBe(1)
  })

  it('xuid inconnu : undefined, jamais une équipe par défaut', () => {
    const table = teamOfXuidFromScoreboard(BOARD)
    expect(table.get('xuid-jamais-vu')).toBeUndefined()
    expect(table.has('xuid-jamais-vu')).toBe(false)
  })

  it('ligne sans camp lisible : ABSENTE de la table, pas présente à undefined', () => {
    const table = teamOfXuidFromScoreboard(BOARD)
    expect(table.has('nomad-9')).toBe(false)
    expect(table.get('nomad-9')).toBeUndefined()
  })

  it('bot sans camp : absent lui aussi — aucun camp deviné pour un bot', () => {
    const table = teamOfXuidFromScoreboard([{ xuid: 'bid(2.0)', team_side: null, is_me: false }])
    expect(table.has('bid(2.0)')).toBe(false)
    expect(table.size).toBe(0)
  })

  it('tableau de score absent : une table vide, jamais une exception', () => {
    expect(teamOfXuidFromScoreboard(null).size).toBe(0)
    expect(teamOfXuidFromScoreboard(undefined).size).toBe(0)
    expect(teamOfXuidFromScoreboard([]).size).toBe(0)
  })
})

/**
 * AJOUT DU 2026-09-06 (lot L2b) — le SUJET, c'est-à-dire le point de vue de la page de rejeu.
 *
 * Ajouts seulement : aucun cas ci-dessus n'a bougé. Ce que ces cas fixent, c'est la bascule
 * groupée de quatre calques (bombe, drapeaux, zones, sons d'objectif) : ils lisent tous cette
 * seule valeur, donc ils suivent le point de vue ou aucun ne le suit.
 */
describe('allyTeamFromScoreboard — avec un sujet (le point de vue)', () => {
  it('sujet = la ligne « moi » : identique à l’appel sans sujet', () => {
    expect(allyTeamFromScoreboard(BOARD, 'me-1')).toBe(allyTeamFromScoreboard(BOARD))
  })

  it('sujet = un adversaire : SON camp devient le camp de référence', () => {
    expect(allyTeamFromScoreboard(BOARD, 'foe-1')).toBe(1)
  })

  it('sujet = un bot rangé dans un camp : ce camp (décision 7 du plan)', () => {
    expect(allyTeamFromScoreboard(BOARD, 'bid(1.0)')).toBe(1)
  })

  it('sujet sans camp transmis : null — et surtout pas un repli sur la ligne « moi »', () => {
    // Le repli serait le pire des comportements : le calque prendrait le camp de quelqu'un
    // d'autre que celui qu'on regarde, sans que rien ne le dise.
    expect(allyTeamFromScoreboard(BOARD, 'nomad-9')).toBeNull()
  })

  it('sujet absent du tableau de score : null, pas de repli non plus', () => {
    expect(allyTeamFromScoreboard(BOARD, 'xuid-jamais-vu')).toBeNull()
  })

  it('sujet situable alors qu’aucune ligne « moi » n’existe : son camp quand même', () => {
    expect(allyTeamFromScoreboard(BOARD_SANS_MOI, 'foe-2')).toBe(1)
  })

  it('sujet à null ou undefined : le comportement d’origine, la ligne « moi »', () => {
    expect(allyTeamFromScoreboard(BOARD, null)).toBe(0)
    expect(allyTeamFromScoreboard(BOARD, undefined)).toBe(0)
  })

  it('tableau de score absent : null, sujet ou pas', () => {
    expect(allyTeamFromScoreboard(null, 'me-1')).toBeNull()
    expect(allyTeamFromScoreboard(undefined, 'me-1')).toBeNull()
    expect(allyTeamFromScoreboard([], 'me-1')).toBeNull()
  })
})
