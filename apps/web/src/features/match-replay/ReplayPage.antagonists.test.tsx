/**
 * Tests — LA FABRIQUE DES ANTAGONISTES arrive à chaque cadre de la page du rejeu.
 *
 * La page l'appelle une fois (`useReplayAntagonists`) et la pose dans la vue qu'elle remet aux
 * cadres (`ReplayFrameView.antagonists`). La scène des trois cadres (Standard, Cinéma, Plein
 * cadre) reçoit cette même vue : on l'observe à cet endroit, sans remplacer la scène.
 */
import { act, cleanup } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReplayFrame } from './model/replayFrame'
import { goToFrame, mountReplayPage, stubReplayPage, unstubReplayPage } from './test/replayPageHarness'
import type { ReplayFrameView } from './ui/frames/replayFrameTypes'

const vues: ReplayFrameView[] = []

vi.mock('@tanstack/react-router', async (importOriginal) =>
  (await import('./test/replayPageRouter')).routerMock(await importOriginal<typeof import('@tanstack/react-router')>()),
)

// La scène réelle est gardée : elle est seulement enveloppée pour relever la vue qu'elle reçoit.
vi.mock('./ui/frames/ReplayStage', async (importOriginal) => {
  const reel = await importOriginal<typeof import('./ui/frames/ReplayStage')>()
  return {
    ...reel,
    ReplayStage: (props: Parameters<typeof reel.ReplayStage>[0]) => {
      vues.push(props.view)
      return reel.ReplayStage(props)
    },
  }
})

beforeEach(() => {
  vues.length = 0
  stubReplayPage()
})

afterEach(() => {
  cleanup()
  unstubReplayPage()
})

describe('ReplayPage : les cadres reçoivent la fabrique des antagonistes', () => {
  it('la vue de chaque cadre porte la même fabrique, qui sert un index par sujet', async () => {
    await mountReplayPage()
    const frames: ReplayFrame[] = ['standard', 'cinema', 'plein-cadre']
    const parCadre = new Map<ReplayFrame, ReplayFrameView>()
    for (const frame of frames) {
      act(() => goToFrame(frame))
      const vue = vues[vues.length - 1]
      expect(vue).toBeDefined()
      parCadre.set(frame, vue)
      expect(typeof vue.antagonists.forSubject).toBe('function')
      expect(typeof vue.antagonists.forViewpoint).toBe('function')
      expect(vue.antagonists.forSubject(null).at(0)).toEqual({ nemesis: null, victim: null })
    }
    // Une seule fabrique pour la page : changer de cadre ne la reconstruit pas.
    const [standard, cinema, pleinCadre] = frames.map((f) => parCadre.get(f)!.antagonists)
    expect(cinema).toBe(standard)
    expect(pleinCadre).toBe(standard)
  })
})
