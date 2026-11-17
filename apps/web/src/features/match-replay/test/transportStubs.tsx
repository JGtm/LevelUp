/**
 * transportStubs.tsx — L'OBJET DE COMMANDES DE LA TOILE, EN DOUBLE DE TEST, pour les tests des barres de lecture
 * qui ne montent pas la toile : une frise vide, un son présent, un export absent, un cadrage libre, et des
 * gestes qui sont des espions. Chaque cas ne surcharge que ce qu'il regarde.
 *
 * Il FOURNIT, il ne juge pas : aucune assertion, aucun état entre deux appels.
 */
import { createRef } from 'react'
import { vi } from 'vitest'

import { NO_ALLEGIANCE } from '@/lib/replay/filmAllegiance'

import type { ReplayExportState } from '../export/useReplayExport'
import type { ReplayExport } from '../export/useReplayExport'
import type { ReplayTimeline } from '../hooks/useReplayTimeline'
import type { ReplayZoom } from '../hooks/useReplayZoom'
import type { ReplaySlotCommands } from '../ui/replayTransportCommands'
import type { ReplaySound } from '../sound/useReplaySound'
import { makeFrameControl } from './frameControlStub'

/** Un son présent et coupé : les commandes de son s'affichent telles qu'un match les montre. */
export function makeSound(over: Partial<ReplaySound> = {}): ReplaySound {
  return {
    available: true,
    on: false,
    toggle: vi.fn(),
    wake: vi.fn(),
    volume: 0.7,
    setVolume: vi.fn(),
    mutedBySpeed: false,
    categories: { weapon: true, grenade: true, melee: true, equipment: true, objective: true },
    toggleCategory: vi.fn(),
    tick: vi.fn(),
    seek: vi.fn(),
    setTransportPlaying: vi.fn(),
    endMatch: vi.fn(),
    intro: vi.fn(),
    recordingTrack: () => null,
    ...over,
  } as unknown as ReplaySound
}

/** La frise servie par la toile (cf. `useReplayTimeline`) : vide de repères par défaut. */
export function makeTimeline(over: Partial<ReplayTimeline> = {}): ReplayTimeline {
  return {
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
    viewpoint: null,
    viewpointGroups: [],
    onSelectViewpoint: vi.fn(),
    dominance: [],
    fragMarks: [],
    score: null,
    allyOf: () => null,
    labelOf: () => '',
    media: [],
    showMediaTrack: true,
    tracksExpanded: true,
    onToggleTracks: vi.fn(),
    playing: true,
    onRequestPause: vi.fn(),
    locale: 'fr',
    ...over,
  }
}

/** Un exporteur vidéo pris dans l'état donné : supporté, sans effet de bord. */
export function makeExporter(state: ReplayExportState): ReplayExport {
  return {
    supported: true,
    state,
    defaultBounds: () => ({ startFrame: 0, endFrame: 100 }),
    run: vi.fn(),
    zoomLevel: 1,
    cancel: vi.fn(),
    clockOf: () => '0:00',
    lengthClock: () => '0:00',
  } as unknown as ReplayExport
}

/** Un cadrage libre : tous les gestes sont des espions. */
export function makeZoom(over: Partial<ReplayZoom> = {}): ReplayZoom {
  return {
    level: 1,
    center: { x: 0, y: 0 },
    canZoomIn: true,
    canZoomOut: true,
    canPan: false,
    zoomIn: vi.fn(),
    zoomOut: vi.fn(),
    reset: vi.fn(),
    panStep: vi.fn(),
    zoomAt: vi.fn(),
    panBy: vi.fn(),
    focusOn: vi.fn(),
    ...over,
  } as unknown as ReplayZoom
}

/** L'objet de commandes complet, en lecture, sans export. */
export function makeCommands(over: Partial<ReplaySlotCommands> = {}): ReplaySlotCommands {
  return {
    playing: true,
    onTogglePlay: vi.fn(),
    onRestart: vi.fn(),
    onSeekBy: vi.fn(),
    clockRef: createRef<HTMLSpanElement>(),
    autoPlay: false,
    onToggleAutoPlay: vi.fn(),
    timeline: makeTimeline(),
    speed: 1,
    onSetSpeed: vi.fn(),
    sound: makeSound(),
    capture: {
      captureImage: vi.fn(),
      recordingSupported: true,
      recording: false,
      toggleRecording: vi.fn(),
      videoExport: null,
    },
    locale: 'fr',
    fullscreen: { active: false, disabled: false, toggle: vi.fn() },
    frameControl: makeFrameControl('cinema'),
    zoom: makeZoom(),
    cameraFollow: { player: null, onFollow: vi.fn() },
    planFeedShown: false,
    ...over,
  }
}
