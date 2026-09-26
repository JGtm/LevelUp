/**
 * usageRegularityBandModel.test.ts — buildRegularityBand, une case par match mesuré.
 * Extrait de `usageLogic.test.ts` le 2026-09-09 (étape E5.1bis, scission de taille —
 * CLAUDE.md n°5) au moment du déménagement du bloc vers `features/_shared/usage/`.
 */
import { describe, expect, it } from 'vitest'

import { USAGE_TEXT } from './usageI18n'
import { buildRegularityBand } from './usageRegularityBandModel'

const t = USAGE_TEXT.fr

describe('buildRegularityBand — une case par match mesuré', () => {
  it('teinte chaque match par son écart à la parité, tiret pour le non-mesuré', () => {
    const cells = buildRegularityBand(
      [
        { match_id: 'a', player_share_of_team_pct: 40 },
        { match_id: 'b', player_share_of_team_pct: 25.5 },
        { match_id: 'c', player_share_of_team_pct: 10 },
        { match_id: 'd' },
      ],
      25,
      t,
      'fr',
    )
    expect(cells.map((c) => c.tone)).toEqual(['above', 'near', 'below', 'unmeasured'])
    expect(cells[3].tooltip).toBe(t.bandTipUnmeasured(4))
  })

  it('une part de 0 % MESURÉE est « sous la parité », pas « non mesurée »', () => {
    const cells = buildRegularityBand([{ match_id: 'a', player_share_of_team_pct: 0 }], 25, t, 'fr')
    expect(cells[0].tone).toBe('below')
  })

  it('sans parité de session, tout est non mesuré', () => {
    const cells = buildRegularityBand(
      [{ match_id: 'a', player_share_of_team_pct: 40 }],
      undefined,
      t,
      'fr',
    )
    expect(cells[0].tone).toBe('unmeasured')
  })
})
