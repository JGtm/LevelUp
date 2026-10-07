/**
 * sessionEmprise.logic.test.ts — l'index des matchs de la page Sessions et LE prédicat de présence
 * des cartes A à L (plan PLAN_SESSIONS_EMPRISE_2026-10-06, S4.2), sur les sessions témoins des
 * relevés (`sessionEmprise.fixtures.ts`).
 */
import { describe, expect, it } from 'vitest'

import { soloEmpriseSansFilm } from '@/features/timeseries/usages/usages.fixtures'
import type { SessionCompareEntry, SessionPageResponse } from '@/lib/api/types'

import { sessionRowKeys } from './_sections'
import { MATCHES_2209, ME_GAMERTAG, session2209, sessionSolo } from './sessionEmprise.fixtures'
import {
  buildSessionEmpriseModels,
  sessionCardsPresence,
  sessionColumnBlocks,
  sessionColumnPresence,
  sessionMatchIndex,
  sessionPlayerName,
  type SessionColumnBlocks,
} from './sessionEmprise.logic'

const byKey = (o: { key?: string }) => o.key ?? ''

describe('sessionMatchIndex — ce que la page sait de chaque match', () => {
  it('carte, mode, résultat, score et drapeau de dominance des lignes de match', () => {
    const index = sessionMatchIndex(MATCHES_2209)
    const withDom = MATCHES_2209.find((m) => (m.dominance_flag ?? 0) > 0)
    expect(withDom).toBeDefined()
    const info = index.get(withDom!.match_id)
    expect(info?.map).toBe(withDom!.map_name)
    expect(info?.mode).toBe(withDom!.mode_ui)
    expect(info?.score).toBe(withDom!.score_label)
    expect(info?.dominance).toBe(withDom!.dominance_flag)
    expect(info?.outcome).not.toBeNull()
  })

  it('sans score ni drapeau : null et indéfini, jamais une valeur inventée', () => {
    const index = sessionMatchIndex([{ match_id: 'x', start_time: '' } as never])
    expect(index.get('x')?.score).toBeNull()
    expect(index.get('x')?.dominance).toBeUndefined()
  })
})

describe('sessionColumnBlocks — la colonne affichée et la colonne comparée', () => {
  it('chaque côté lit SES blocs ; l’emblème et la référence de portée sont communs', () => {
    const data = {
      current_session: { session_label: 'a' },
      compare_session: { session_label: 'b' },
      emprise: { matches_total: 1 },
      compare_emprise: { matches_total: 2 },
      lives_near_teammate: { near: { lives: 1 } },
      compare_lives_near_teammate: { near: { lives: 2 } },
      formes_retenues: { matches_total: 1 },
      compare_formes_retenues: { matches_total: 2 },
      player_emblem_url: 'e.png',
      range_reference: { matches: [] },
    } as unknown as SessionPageResponse
    const cur = sessionColumnBlocks(data, 'current')
    const cmp = sessionColumnBlocks(data, 'compare')
    expect(cur.entry?.session_label).toBe('a')
    expect(cmp.entry?.session_label).toBe('b')
    expect(cur.emprise?.matches_total).toBe(1)
    expect(cmp.emprise?.matches_total).toBe(2)
    expect(cmp.lives?.near.lives).toBe(2)
    expect(cmp.formes?.matches_total).toBe(2)
    expect(cmp.emblemUrl).toBe('e.png')
    expect(cmp.rangeReference).toBe(cur.rangeReference)
  })
})

describe('sessionCardsPresence — une carte n’existe que si elle dessine quelque chose', () => {
  it('soirée d’escouade du 22/09 : tout sauf la précision (Halo 5) et l’équipement (non servi)', () => {
    expect(sessionColumnPresence(session2209())).toEqual({
      frag_donut: true,
      tools: true,
      weapon_accuracy: false,
      control: true,
      fil: true,
      grid: true,
      mine: true,
      production: true,
      yield: true,
      lives: true,
      objective_balance: true,
      objective_sheet: true,
      equipment: false,
    })
  })

  it('session solo sans match à objectif : ni rapport de force ni fiche ; aucune prise : ni C, ni D, ni F', () => {
    const p = sessionColumnPresence(sessionSolo())
    expect(p.objective_balance).toBe(false)
    expect(p.objective_sheet).toBe(false)
    expect(p.control).toBe(false)
    expect(p.fil).toBe(false)
    expect(p.mine).toBe(false)
    expect(p.production).toBe(true)
    expect(p.lives).toBe(true)
  })

  it('Halo 5 (sans film) : A, B, B’ et la barre des armes spéciales de G, rien d’autre', () => {
    const col: SessionColumnBlocks = {
      ...session2209(),
      entry: {
        ...session2209().entry,
        weapon_accuracy: [{ weapon: 'BR', shots_fired: 10, shots_hit: 5 }],
      } as unknown as SessionCompareEntry,
      emprise: soloEmpriseSansFilm(),
      lives: null,
      formes: null,
    }
    const p = sessionColumnPresence(col)
    expect(Object.entries(p).filter(([, v]) => v).map(([k]) => k)).toEqual([
      'frag_donut',
      'tools',
      'weapon_accuracy',
      'production',
    ])
  })

  it('« Non attribué » seul ne fait pas une carte d’outils', () => {
    const col = session2209()
    col.entry = {
      ...col.entry,
      weapon_tools: { players: [ME_GAMERTAG], lines: [{ kind: 'unattributed', class: 'unattributed', kills_by_player: { [ME_GAMERTAG]: 4 }, total_squad: 4 }] },
    } as unknown as SessionCompareEntry
    expect(sessionColumnPresence(col).tools).toBe(false)
  })

  it('les modèles bâtis par la page donnent la même présence (un seul prédicat)', () => {
    const col = session2209()
    expect(sessionCardsPresence(col, buildSessionEmpriseModels(col, byKey))).toEqual(sessionColumnPresence(col))
  })
})

describe('sessionPlayerName', () => {
  it('le nom que portent les outils, sinon l’Emprise, sinon la repli', () => {
    expect(sessionPlayerName(session2209(), 'slug')).toBe(ME_GAMERTAG)
    expect(sessionPlayerName({ entry: null, matches: [] }, 'slug')).toBe('slug')
  })
})

describe('sessionRowKeys — les rangées partagées de la comparaison', () => {
  it('soirée à objectif face à une session solo : l’union garde les cartes d’objectif', () => {
    const a = session2209()
    const b = sessionSolo()
    const data = {
      current_session: a.entry,
      matches: a.matches,
      emprise: a.emprise,
      lives_near_teammate: a.lives,
      formes_retenues: a.formes,
      compare_session: b.entry,
      compare_matches: b.matches,
      compare_emprise: b.emprise,
      compare_lives_near_teammate: b.lives,
      compare_formes_retenues: b.formes,
    } as unknown as SessionPageResponse
    const keys = sessionRowKeys(data)
    expect(keys).toContain('objective_balance')
    expect(keys).toContain('objective_sheet')
    expect(keys).toContain('control')
    expect(keys).not.toContain('weapon_accuracy')
    expect(keys).not.toContain('coordination')
  })
})
