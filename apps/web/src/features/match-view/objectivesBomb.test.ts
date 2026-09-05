/**
 * Tests — LE MODE ASSAUT DANS LA SECTION « Objectifs ».
 *
 * CE QU'ILS PROTÈGENT, et pourquoi chaque cas est construit sur une RÈGLE et non sur une valeur
 * choisie :
 *
 *   1. LE DISCRIMINANT. `detectObjectiveMode` rend `'bomb'` dès qu'une ligne porte un compteur
 *      de bombe, et RIEN quand aucune ne le fait — c'est ce qui distingue « un titre sans la
 *      capability `film.bomb_stats` » (aucune colonne servie -> section masquée) d'un match
 *      d'Assaut mesuré. Sans lui, la section resterait invisible sur tout l'Assaut, comme
 *      avant le 2026-09-05. `bomb_carriers_killed` N'EN FAIT PAS PARTIE : le discriminant teste
 *      TROIS compteurs (mêmes que `HasBomb()` côté Go), la cinquième colonne n'en est pas un.
 *   2. LES COLONNES SUIVENT LE MODE. `objectiveColsFor('bomb')` rend les CINQ grandeurs
 *      affichées depuis l'arbitrage N-17 (2026-09-05, registre du plan d'intégration, seconde
 *      lecture E.2) : `bomb_carriers_killed` est mesuré depuis le lot G.6 (témoin `9f57c612` :
 *      3 porteurs tués) et la prémisse « colonne de tirets » qui la retenait n'était plus vraie.
 *   3. ABSENT N'EST PAS ZÉRO. `objectiveValue` rend `null` — jamais 0 — pour une mesure non
 *      publiée, et le total d'équipe d'une colonne que personne ne porte est `null` : la
 *      cellule affiche alors le repli « non mesuré » de `ValueGrid`, pas un chiffre. Vérifié à
 *      la fois sur une colonne ancienne (`bomb_detonations`) et sur la colonne neuve
 *      (`bomb_carriers_killed`).
 */
import { describe, expect, it } from 'vitest'

import type { MatchScoreboardRow } from '@/lib/api/types'

import {
  detectObjectiveMode,
  objectiveColsFor,
  objectiveTeamTotal,
} from './MatchScoreboard.logic'
import { objectiveValue } from './objectivesChart'

/** Une ligne de scoreboard réduite à ce que la section lit. */
function ligne(
  gamertag: string,
  side: string,
  objective: Record<string, number | null> | null,
): MatchScoreboardRow {
  return { xuid: gamertag, gamertag, team_side: side, objective } as unknown as MatchScoreboardRow
}

// Un match d'Assaut MESURÉ : deux joueurs d'un camp, un de l'autre. Aucune valeur n'est
// arbitraire — les comptes suivent la règle « une bombe explose au bout d'un armement ».
// `bomb_carriers_killed` est MESURÉ sur Alpha (2, dont potentiellement un tir ami — la réserve
// écrite en tête de `bomb_stats.go`) et à ZÉRO sur Bravo (une mesure, pas une absence) ; Charlie
// ne le porte PAS, pour garder un cas « non mesuré » dans le même match — comme la chaîne peut le
// publier aujourd'hui (source non lue sur cette ligne).
const ASSAUT = [
  ligne('Alpha', 't0', {
    bomb_detonations: 2,
    bomb_arms: 2,
    bomb_grabs: 3,
    time_as_bomb_carrier_seconds: 41.5,
    bomb_carriers_killed: 2,
  }),
  ligne('Bravo', 't0', {
    bomb_detonations: 0,
    bomb_arms: 0,
    bomb_grabs: 1,
    time_as_bomb_carrier_seconds: 4,
    bomb_carriers_killed: 0,
  }),
  ligne('Charlie', 't1', {
    bomb_detonations: 1,
    bomb_arms: 1,
    bomb_grabs: 2,
    time_as_bomb_carrier_seconds: 18,
  }),
]

describe('le mode Assaut est reconnu par la section Objectifs', () => {
  it('rend « bomb » dès qu’une ligne porte un compteur de bombe', () => {
    expect(detectObjectiveMode(ASSAUT)).toBe('bomb')
  })

  it('rend null quand AUCUNE ligne n’en porte — le titre sans la capability, ou un autre mode', () => {
    const sansBombe = [ligne('Alpha', 't0', null), ligne('Charlie', 't1', null)]
    expect(detectObjectiveMode(sansBombe)).toBeNull()
  })

  it('ne confond pas un autre mode avec l’Assaut : CTF reste CTF', () => {
    const ctf = [ligne('Alpha', 't0', { flag_captures: 1, flag_grabs: 3 })]
    expect(detectObjectiveMode(ctf)).toBe('ctf')
  })

  it('reconnaît un compteur MESURÉ À ZÉRO — c’est une mesure, pas une absence', () => {
    const zeros = [ligne('Alpha', 't0', { bomb_detonations: 0, bomb_arms: 0 })]
    expect(detectObjectiveMode(zeros)).toBe('bomb')
  })

  // LE PORTAGE SEUL SUFFIT, et c'est le cas que le discriminant à deux compteurs perdait
  // (revue adversariale de branche, 2026-09-05). Les colonnes d'Assaut ne sortent pas toutes de
  // la même lecture : `bomb_detonations` demande le statborg, `bomb_arms` demande l'anneau ET le
  // portage, `bomb_grabs` ne demande QUE le portage. Un film dont seul le canal des armes tenues
  // a été lu ne publie donc que celui-là — le Go le déclare Assaut (`ObjectiveRaw.HasBomb()`,
  // MÊME liste de trois), et le web faisait disparaître la section entière.
  it('rend « bomb » quand SEUL le portage a été lu — la même liste que HasBomb() côté Go', () => {
    const portageSeul = [ligne('Alpha', 't0', { bomb_grabs: 2, time_as_bomb_carrier_seconds: 18 })]
    expect(detectObjectiveMode(portageSeul)).toBe('bomb')
  })
})

describe('les colonnes d’Assaut', () => {
  it('sont les CINQ grandeurs publiées, dans l’ordre du mode — arbitrage N-17 (2026-09-05)', () => {
    expect(objectiveColsFor('bomb').map((c) => c.key)).toEqual([
      'bomb_detonations',
      'bomb_arms',
      'bomb_grabs',
      'time_as_bomb_carrier_seconds',
      'bomb_carriers_killed',
    ])
  })

  it('exposent bomb_carriers_killed : mesuré depuis G.6, la prémisse « colonne de tirets » qui la retenait n’était plus vraie', () => {
    expect(objectiveColsFor('bomb').map((c) => c.key)).toContain('bomb_carriers_killed')
  })

  it('marquent le temps de portage comme une DURÉE (mm:ss), les quatre autres comme des comptes', () => {
    const cols = objectiveColsFor('bomb')
    const duree = cols.filter((c) => c.duration === true).map((c) => c.key)
    expect(duree).toEqual(['time_as_bomb_carrier_seconds'])
    expect(cols.every((c) => c.agg === 'sum')).toBe(true)
  })
})

describe('absent n’est pas zéro', () => {
  it('rend null — jamais 0 — pour une mesure que la ligne ne porte pas', () => {
    const col = objectiveColsFor('bomb')[0] // bomb_detonations
    const sansMesure = ligne('Echo', 't0', { bomb_grabs: 1 })
    expect(objectiveValue(sansMesure, col)).toBeNull()
  })

  it('rend la valeur quand elle est mesurée, ZÉRO COMPRIS', () => {
    const col = objectiveColsFor('bomb')[0]
    expect(objectiveValue(ASSAUT[1], col)).toBe(0)
  })

  it('laisse le total d’équipe à null quand personne du camp ne porte la colonne', () => {
    const col = objectiveColsFor('bomb')[0]
    const camp = [ligne('Echo', 't0', { bomb_grabs: 1 })]
    expect(objectiveTeamTotal(camp, col)).toBeNull()
  })

  it('cumule le camp sur les lignes qui portent la colonne', () => {
    const col = objectiveColsFor('bomb')[0]
    const t0 = ASSAUT.filter((r) => r.team_side === 't0')
    // 2 + 0 : le zéro MESURÉ entre dans la somme, il n'est pas écarté comme une absence.
    expect(objectiveTeamTotal(t0, col)).toBe(2)
  })

  // La colonne NEUVE (N-17, arbitrage 2026-09-05) suit la MÊME règle que les quatre premières —
  // ce n'est pas un cas particulier, et le test le vérifie explicitement sur elle.
  it('rend null pour bomb_carriers_killed sur une ligne qui ne le porte pas (Charlie)', () => {
    const col = objectiveColsFor('bomb')[4] // bomb_carriers_killed
    expect(objectiveValue(ASSAUT[2], col)).toBeNull()
  })

  it('rend la valeur mesurée de bomb_carriers_killed, zéro compris (Alpha : 2, Bravo : 0)', () => {
    const col = objectiveColsFor('bomb')[4]
    expect(objectiveValue(ASSAUT[0], col)).toBe(2)
    expect(objectiveValue(ASSAUT[1], col)).toBe(0)
  })

  it('cumule bomb_carriers_killed sur le camp qui le porte (2 + 0, Charlie hors mesure exclu du camp adverse)', () => {
    const col = objectiveColsFor('bomb')[4]
    const t0 = ASSAUT.filter((r) => r.team_side === 't0')
    expect(objectiveTeamTotal(t0, col)).toBe(2)
  })
})
