/**
 * Tests — la barre de lecture relaie la bande des titres à la frise (C.2 du plan des antagonistes).
 *
 * La bande n'est pas une commande de la toile : c'est le cadre qui la fournit (`canvasSlotsOf`), avec ce
 * que seule la page connaît (fabrique des antagonistes, roster, allégeance). La barre la remet telle
 * quelle à la frise à pistes ; absente, la barre est celle d'avant.
 */
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { render } from '@testing-library/react'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import { makeFrameControl } from '../test/frameControlStub'
import { GROUPES } from '../test/timelineTracksHarness'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplaySound } from '../sound/useReplaySound'
import { ReplayTransport } from './ReplayTransport'

const FRISE: ReplayTimeline = {
  sliderRef: createRef<HTMLInputElement>(),
  minFrame: 0,
  maxFrame: 600,
  onScrub: vi.fn(),
  own: [],
  teammates: [],
  shades: [],
  absence: [],
  allegiance: NO_ALLEGIANCE,
  onSeekFrame: vi.fn(),
  viewpoint: 'me-1',
  viewpointGroups: GROUPES,
  onSelectViewpoint: vi.fn(),
  dominance: [],
  score: null,
  allyOf: () => null,
  labelOf: (id) => `Équipe ${id}`,
  media: [],
  showMediaTrack: false,
  tracksExpanded: true,
  onToggleTracks: vi.fn(),
  playing: false,
  onRequestPause: vi.fn(),
  locale: 'fr',
}

function rendre(antagonistBand?: React.ReactNode) {
  return render(
    <ReplayTransport
      playing={false}
      autoPlay={false}
      onToggleAutoPlay={vi.fn()}
      onTogglePlay={vi.fn()}
      onRestart={vi.fn()}
      onSeekBy={vi.fn()}
      clockRef={createRef<HTMLSpanElement>()}
      timeline={FRISE}
      speed={1}
      onSetSpeed={vi.fn()}
      sound={{ available: false } as unknown as ReplaySound}
      capture={{ captureImage: vi.fn(), recordingSupported: false, recording: false, toggleRecording: vi.fn(), videoExport: null }}
      locale="fr"
      fullscreen={{ active: false, disabled: false, toggle: vi.fn() }}
      frameControl={makeFrameControl()}
      antagonistBand={antagonistBand}
    />,
  )
}

describe('ReplayTransport — la bande des titres', () => {
  it('la remet à la frise à pistes', () => {
    const { getByTestId } = rendre(<div data-testid="bande-test" />)
    expect(getByTestId('bande-test')).toBeTruthy()
  })

  it('absente, rien n est posé', () => {
    const { queryByTestId } = rendre()
    expect(queryByTestId('bande-test')).toBeNull()
  })
})
