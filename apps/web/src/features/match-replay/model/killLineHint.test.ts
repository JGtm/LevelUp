/**
 * killLineHint.test.ts — L'INFOBULLE D'UNE LIGNE DE FRAG : les éliminations liées à l'objectif repliées sur la ligne
 * s'y lisent, dans la langue de la page, sans jargon, et seulement quand il y en a.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT } from '../i18n/i18n'
import type { ReplayKill } from './killFeedLogic'
import { killLineHint } from './killLineHint'

const kill = { assistState: '', killerDamagePct: null } as unknown as ReplayKill

describe('killLineHint — éliminations liées à l’objectif', () => {
  it('sans rien à dire, pas d’infobulle', () => {
    expect(killLineHint(kill, false, REPLAY_TEXT.fr, false)).toBeUndefined()
    expect(killLineHint(kill, false, REPLAY_TEXT.fr, false, [])).toBeUndefined()
  })

  it('dit chaque élimination dans la langue de la page', () => {
    expect(killLineHint(kill, false, REPLAY_TEXT.fr, false, ['zone_offensive_kills', 'vip_kills'])).toBe(
      'a éliminé un défenseur de base — a éliminé le VIP adverse',
    )
    expect(killLineHint(kill, false, REPLAY_TEXT.en, false, ['flag_kills_as_carrier'])).toBe(
      'got a kill while carrying the flag',
    )
  })

  it('suit le badge du porteur éliminé', () => {
    expect(killLineHint(kill, true, REPLAY_TEXT.fr, false, ['flag_returners_killed'])).toBe(
      'La victime portait le drapeau — a éliminé un joueur qui rendait le drapeau',
    )
  })
})
