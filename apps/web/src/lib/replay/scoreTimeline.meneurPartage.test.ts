/// <reference types="node" />
// @vitest-environment node
/**
 * LE MENEUR DU SCORE, JEU DE CAS PARTAGÉ AVEC LE GO.
 *
 * La Mainmise d'un match (part du temps en tête, retournements) est calculée côté serveur par
 * `internal/analysis/leadshare`, qui doit dire EXACTEMENT ce que `leaderStates` et `leadChanges`
 * disent ici : une seule définition du meneur (égalité = personne en tête, la première prise de
 * tête n'est pas un retournement). Les deux côtés lisent le même fichier, c'est lui qui les
 * empêche de diverger ; le test Go est `leader_test.go`.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import {
  leadChanges,
  leaderStates,
  type ReplayScoreTimelineReady,
  type ReplayTeamScoreReady,
} from './scoreTimeline'

const JEU_DE_CAS = resolve(
  __dirname,
  '../../../../go-api/internal/analysis/leadshare/testdata/meneur_cas.json',
)

interface CasMeneur {
  nom: string
  camp_count: number
  teams: Array<{ team_id: number | null; total: Array<{ t: number; v: number }> }>
  etats: Array<{ frame: number; team_id: number | null }>
  retournements: Array<{ frame: number; team_id: number }>
}

const jeu = JSON.parse(readFileSync(JEU_DE_CAS, 'utf8')) as { cas: CasMeneur[] }

/** Le calque de score du cas, tel que le document le porte. Un camp sans identité n'a pas de `teamId`. */
function timelineOf(cas: CasMeneur): ReplayScoreTimelineReady {
  const teams = cas.teams.map((t): ReplayTeamScoreReady => {
    const total = t.total.map((p) => ({ t: p.t, v: p.v }))
    const team: ReplayTeamScoreReady = { rounds: [], total }
    if (t.team_id != null) team.teamId = t.team_id
    return team
  })
  return { teams, players: [], holdTicks: [] } as ReplayScoreTimelineReady
}

describe('leaderStates et leadChanges — le jeu de cas partagé avec le Go', () => {
  it('le jeu de cas n est pas vide', () => {
    expect(jeu.cas.length).toBeGreaterThan(0)
  })

  for (const cas of jeu.cas) {
    it(cas.nom, () => {
      const timeline = timelineOf(cas)
      expect(leaderStates(timeline, cas.camp_count)).toEqual(
        cas.etats.map((e) => ({ frame: e.frame, teamId: e.team_id })),
      )
      // `leadChanges` n'est que la projection de `leaderStates`, et il reçoit le même nombre de
      // camps que le calcul du serveur (`leadshare.Measure` le passe à `LeaderStates` puis à
      // `Changes`) : sans lui, un camp muet ne concourrait pas à zéro et les cas à camp muet
      // divergeraient du Go.
      expect(leadChanges(timeline, cas.camp_count)).toEqual(
        cas.retournements.map((r) => ({ frame: r.frame, teamId: r.team_id })),
      )
    })
  }
})
