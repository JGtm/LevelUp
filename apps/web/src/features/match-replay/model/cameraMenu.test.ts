/**
 * Tests — cameraMenu (ce que le menu de la caméra lit) : le nom du joueur suivi, par sa clé de roster, et la
 * caméra par défaut d'une toile montée sans page autour.
 */
import { describe, expect, it } from 'vitest'

import { followedLabelOf, NO_CAMERA_FOLLOW } from './cameraMenu'
import type { ViewpointOptionGroup } from './viewpointOptions'

const GROUPS: readonly ViewpointOptionGroup[] = [
  {
    key: 'camp:0',
    label: 'Cobalt',
    options: [
      { value: 'x1', rosterKey: 'x1', label: 'JGtm', disabled: false, title: 'JGtm' },
      { value: 'bid(3.0)', rosterKey: 'bot:Cortana', label: 'Cortana', disabled: false, title: 'Cortana' },
    ],
  },
  { key: 'camp:1', label: 'Ambre', options: [{ value: 'x2', rosterKey: 'x2', label: 'Rival', disabled: false, title: 'Rival' }] },
]

describe('followedLabelOf — le nom du joueur suivi', () => {
  it('se retrouve par la clé de roster, dans n’importe quel camp, bot compris', () => {
    expect(followedLabelOf(GROUPS, 'x2')).toBe('Rival')
    expect(followedLabelOf(GROUPS, 'bot:Cortana')).toBe('Cortana')
  })

  it('la valeur du point de vue d’un bot n’est PAS sa clé de roster : elle ne le retrouve pas', () => {
    expect(followedLabelOf(GROUPS, 'bid(3.0)')).toBeNull()
  })

  it('personne n’est suivi, ou le joueur n’a aucune entrée : null', () => {
    expect(followedLabelOf(GROUPS, null)).toBeNull()
    expect(followedLabelOf(GROUPS, 'inconnu')).toBeNull()
    expect(followedLabelOf([], 'x1')).toBeNull()
  })
})

describe('NO_CAMERA_FOLLOW — une toile sans page', () => {
  it('ne suit personne, et le geste ne fait rien', () => {
    expect(NO_CAMERA_FOLLOW.player).toBeNull()
    expect(() => NO_CAMERA_FOLLOW.onFollow('x1')).not.toThrow()
  })
})
