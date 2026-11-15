/**
 * replayViewPanelFixtures.tsx — LES FABRIQUES ET LE RENDU DU PANNEAU DE LA VUE, partagés par les
 * fichiers de test du panneau (`ReplayViewPanel.test.tsx`, `ReplayViewPanel.terrain.test.tsx`).
 * Rien ici n'asserte : chaque fabrique rend des commandes valides, que le test surcharge.
 */
import { vi } from 'vitest'
import { render } from '@testing-library/react'

import {
  ReplayViewPanel,
  type ReplayGroundWeaponControls,
  type ReplayHeatmapControls,
  type ReplayModeObjectivesControls,
  type ReplayOrientationControls,
  type ReplayPlacementControls,
  type ReplayFlagControls,
  type ReplayBackgroundControls,
  type ReplayLivesControls,
  type ReplayWallPlanControls,
  type ReplayWeaponPadControls,
} from './ReplayViewPanel'
import type { ReplaySound } from '../sound/useReplaySound'

export function makePlacements(over: Partial<ReplayPlacementControls> = {}): ReplayPlacementControls {
  return {
    available: true,
    show: true,
    onToggle: vi.fn(),
    unnamedAvailable: true,
    showUnnamed: false,
    onToggleUnnamed: vi.fn(),
    droppedAvailable: true,
    showDropped: true,
    onToggleDropped: vi.fn(),
    ...over,
  }
}

export function makeWeaponPads(over: Partial<ReplayWeaponPadControls> = {}): ReplayWeaponPadControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

export function makeGroundWeapons(
  over: Partial<ReplayGroundWeaponControls> = {},
): ReplayGroundWeaponControls {
  return {
    available: true,
    show: true,
    onToggle: vi.fn(),
    showSpecialOnly: false,
    onToggleSpecialOnly: vi.fn(),
    showStarting: false,
    onToggleStarting: vi.fn(),
    ...over,
  }
}

export function makeWallPlan(over: Partial<ReplayWallPlanControls> = {}): ReplayWallPlanControls {
  return { available: true, show: true, onToggle: vi.fn(), inPlan: false, ...over }
}

export function makeLives(over: Partial<ReplayLivesControls> = {}): ReplayLivesControls {
  return {
    spawns: { available: true, show: false, onToggle: vi.fn() },
    deaths: { available: true, show: false, onToggle: vi.fn() },
    killers: { available: true, show: false, onToggle: vi.fn() },
    who: 'all',
    onSelectWho: vi.fn(),
    followedAvailable: true,
    when: 'all',
    onSelectWhen: vi.fn(),
    ...over,
  }
}

export function makeBackground(over: Partial<ReplayBackgroundControls> = {}): ReplayBackgroundControls {
  return { available: true, value: 'rendu', onSelect: vi.fn(), ...over }
}

export function makeOrientation(over: Partial<ReplayOrientationControls> = {}): ReplayOrientationControls {
  return { available: true, turns: 0, onSelect: vi.fn(), ...over }
}

export function makeFlagCarries(over: Partial<ReplayFlagControls> = {}): ReplayFlagControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

export function makeModeObjectives(
  over: Partial<ReplayModeObjectivesControls> = {},
): ReplayModeObjectivesControls {
  return { available: true, show: true, onToggle: vi.fn(), ...over }
}

export function makeHeatmap(over: Partial<ReplayHeatmapControls> = {}): ReplayHeatmapControls {
  return {
    show: false,
    onToggle: vi.fn(),
    mode: 'presence',
    onSetMode: vi.fn(),
    span: 'match',
    onSetSpan: vi.fn(),
    killsAvailable: true,
    ...over,
  }
}

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
  exportTrack: () => ({ timeline: [], endMatchStems: [], variationPercent: 0, distancePercent: 0, families: { voice: [], music: [] }, engines: [] }),
    ...over,
  }
}

export function renderPanel(over: Partial<Parameters<typeof ReplayViewPanel>[0]> = {}) {
  const onClose = vi.fn()
  const onSelect = vi.fn()
  const onToggleAim = vi.fn()
  const onToggleZones = vi.fn()
  const onToggleTrail = vi.fn()
  const onSetSpeed = vi.fn()
  const onToggleShotFx = vi.fn()
  const onToggleKillFx = vi.fn()
  const onSetMarkerColors = vi.fn()
  const utils = render(
    <ReplayViewPanel
      locale="fr"
      onClose={onClose}
      view={{ active: 'complet', onSelect }}
      showAim
      onToggleAim={onToggleAim}
      showZones
      onToggleZones={onToggleZones}
      showTrail
      onToggleTrail={onToggleTrail}
      zonesAvailable
      placements={makePlacements()}
      weaponPads={makeWeaponPads()}
      groundWeapons={makeGroundWeapons()}
      modeObjectives={makeModeObjectives()}
      flagCarries={makeFlagCarries()}
      vipCrown={makeFlagCarries()}
      skullCarrier={makeFlagCarries()}
      bombCarrier={makeFlagCarries()}
      vehicles={makeWeaponPads()}
      wallPlan={makeWallPlan()}
      background={makeBackground()}
      lives={makeLives()}
      orientation={makeOrientation()}
      heatmap={makeHeatmap()}
      showShotFx
      onToggleShotFx={onToggleShotFx}
      showKillFx={false}
      onToggleKillFx={onToggleKillFx}
      sound={makeSound()}
      markerColors="team"
      onSetMarkerColors={onSetMarkerColors}
      {...over}
    />,
  )
  return {
    ...utils, onClose, onSelect, onToggleAim, onToggleZones, onToggleTrail, onSetSpeed,
    onToggleShotFx, onToggleKillFx, onSetMarkerColors,
  }
}
