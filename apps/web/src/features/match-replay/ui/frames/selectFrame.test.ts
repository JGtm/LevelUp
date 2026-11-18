/**
 * Tests — selectFrame et leaveFullFrame (les gestes des boutons de cadre).
 *
 * `leaveFullFrame` est le chemin UNIQUE de la sortie du Plein cadre par un geste de l'utilisateur : « Quitter »
 * de l'en-tête et l'icône du Plein cadre courant. Il rend le cadre et, comme Échap, referme le plein écran du
 * système quand il est atteint.
 */
import { describe, expect, it, vi } from 'vitest'

import { makeFrameControl } from '../../test/frameControlStub'
import { leaveFullFrame, selectFrame } from './selectFrame'

describe('leaveFullFrame', () => {
  it('sort du cadre, et referme le plein écran du système quand il est atteint', () => {
    const control = makeFrameControl('plein-cadre')
    const fullscreen = { active: true, toggle: vi.fn() }
    leaveFullFrame(control, fullscreen)
    expect(control.exit).toHaveBeenCalledTimes(1)
    expect(fullscreen.toggle).toHaveBeenCalledTimes(1)
  })

  it('sans plein écran du système atteint, il ne le touche pas', () => {
    const control = makeFrameControl('plein-cadre')
    const fullscreen = { active: false, toggle: vi.fn() }
    leaveFullFrame(control, fullscreen)
    expect(control.exit).toHaveBeenCalledTimes(1)
    expect(fullscreen.toggle).not.toHaveBeenCalled()
  })
})

describe('selectFrame', () => {
  it('sans effet sur le cadre courant', () => {
    const control = makeFrameControl('cinema')
    selectFrame(control, 'cinema')
    expect(control.toggleCinema).not.toHaveBeenCalled()
  })
})
