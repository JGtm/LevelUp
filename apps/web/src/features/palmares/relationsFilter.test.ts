import { describe, expect, it } from 'vitest'

import type { RelationBadge, RelationInsight } from '@/lib/api/types'

import {
  CROSS_GAME_BADGE_KEY,
  filterRelations,
  hasCrossGameRelations,
  isCrossGame,
} from './relationsFilter'

// Fabrique minimale : seuls les champs lus par les prédicats importent, le reste
// est bouché (cast assumé — le contrat OpenAPI garantit la forme au runtime).
function rel(partial: Partial<RelationInsight>): RelationInsight {
  return {
    xuid: 'x',
    gamertag: 'GT',
    total_matches: 1,
    teammate_matches: 0,
    enemy_matches: 0,
    is_core: false,
    last_seen_at: null,
    badges: null,
    ...partial,
  } as RelationInsight
}

const crossBadge: RelationBadge = {
  label_key: CROSS_GAME_BADGE_KEY,
  color_token: 'narrative-encounter-cross-game',
  style: 'solid',
  detail: { game: 'Halo 5', matches_together: 7 },
}
const ordinalBadge: RelationBadge = {
  label_key: 'narrative.encounter.ordinal',
  color_token: 'narrative-encounter-ordinal',
  style: 'tinted',
  detail: { ordinal: 5 },
}

describe('relationsFilter — filtre Multi-jeux (cross-jeu)', () => {
  const withCross = rel({ gamertag: 'Cross', badges: [ordinalBadge, crossBadge] })
  const noCross = rel({ gamertag: 'Mono', badges: [ordinalBadge] })
  const noBadges = rel({ gamertag: 'Bare', badges: null })

  it('isCrossGame détecte le badge cross-jeu et lui seul', () => {
    expect(isCrossGame(withCross)).toBe(true)
    expect(isCrossGame(noCross)).toBe(false)
    expect(isCrossGame(noBadges)).toBe(false)
  })

  it("filterRelations('cross') ne garde que les relations cross-jeu", () => {
    const out = filterRelations([withCross, noCross, noBadges], 'cross')
    expect(out.map((r) => r.gamertag)).toEqual(['Cross'])
  })

  it('hasCrossGameRelations : vrai ssi au moins une relation cross-jeu', () => {
    expect(hasCrossGameRelations([withCross, noCross])).toBe(true)
    expect(hasCrossGameRelations([noCross, noBadges])).toBe(false)
    expect(hasCrossGameRelations([])).toBe(false)
  })

  it("filterRelations('all') reste inchangé (filtre additif, aucune régression)", () => {
    const rows = [withCross, noCross, noBadges]
    expect(filterRelations(rows, 'all')).toHaveLength(3)
  })
})
