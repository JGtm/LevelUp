/**
 * canvasTestKit — LE MONTAGE DE TEST DE LA TOILE `ReplayCanvas` : un document minimal, les propriétés requises
 * (mémorisées par document : deux montages du même document reçoivent les mêmes valeurs) et les doubles du
 * navigateur qu'un rendu de la toile suppose. Il FOURNIT, il ne juge pas : aucune assertion.
 */
import type { ComponentProps, ReactElement } from 'react'
import { vi } from 'vitest'

import { buildFilmAllegiance } from '@/lib/replay/filmAllegiance'
import type { ReplayDocumentReady } from '@/lib/replay/replayNormalize'

import { buildPlayers } from '../../../lib/replay/rosterLogic'
import { createPlaybackStore } from '../model/playbackStore'
import { ReplayCanvas } from '../ui/ReplayCanvas'
import type { ReplayCommandsRender } from '../ui/replayTransportCommands'
import { makeFrameControl } from './frameControlStub'
import { goFixtureEntries, loadGoFixture } from './goFixtures'
import { recordingContext } from './recordingContext'
import { testReplayDoc } from './testDoc'

export function smallDoc(): ReplayDocumentReady {
  const smallest = [...goFixtureEntries()].sort((a, b) => a.bytes - b.bytes)[0]
  return testReplayDoc(loadGoFixture(smallest).doc)
}

type BaseProps = Omit<ComponentProps<typeof ReplayCanvas>, 'transport' | 'blockOverlays'>
const BASE_PROPS = new WeakMap<ReplayDocumentReady, BaseProps>()

/** Les propriétés requises de la toile, pour un document. */
export function baseProps(doc: ReplayDocumentReady): BaseProps {
  const known = BASE_PROPS.get(doc)
  if (known) return known
  const players = buildPlayers(doc, [])
  const viewpoint = players.find((p) => p.team !== undefined && p.team >= 0 && !p.xuid.startsWith('bot:'))?.xuid ?? null
  const props: BaseProps = {
    doc,
    locale: 'fr',
    playWindow: null,
    playbackStore: createPlaybackStore(),
    allegiance: buildFilmAllegiance(players, viewpoint),
    players,
    viewpoint,
    fullscreen: { active: false, disabled: false, toggle: () => {} },
    frameControl: makeFrameControl(),
  }
  BASE_PROPS.set(doc, props)
  return props
}

/** Monte la toile avec les emplacements donnés ; le reste des propriétés est le minimum requis. */
export function canvasWith(doc: ReplayDocumentReady, transport: ReplayCommandsRender, blockOverlays?: ReplayCommandsRender): ReactElement {
  return <ReplayCanvas {...baseProps(doc)} transport={transport} blockOverlays={blockOverlays} />
}

/**
 * Les doubles du navigateur : aucune boucle d'animation (la lecture change d'état, l'image ne bouge que sur un
 * saut), une largeur mesurée (sans elle la toile ne peint rien et l'horloge reste vide) et un contexte 2D
 * enregistreur. À appeler dans un `beforeEach` ; `vi.restoreAllMocks()` et `vi.unstubAllGlobals()` les défont.
 */
export function installCanvasDoubles(): void {
  try {
    localStorage.clear()
  } catch {
    // Pas de stockage : les réglages gardent leurs défauts, ce que ces tests supposent déjà.
  }
  vi.stubGlobal('requestAnimationFrame', () => 0)
  vi.stubGlobal('cancelAnimationFrame', () => {})
  vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockReturnValue({
    x: 0, y: 0, top: 0, left: 0, bottom: 540, right: 960, width: 960, height: 540, toJSON: () => ({}),
  })
  vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockImplementation(() => recordingContext().ctx as never)
}
