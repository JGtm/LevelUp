/**
 * useReplayViewMenu.test.tsx — LE BOUTON « VUE » branché sur les vrais réglages : choisir un
 * préréglage bascule les calques qui diffèrent (et eux seuls), la vue active se déduit de
 * l'état, une bascule à la main la passe en « Personnalisé », et rien d'autre que les calques
 * n'est touché.
 */
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, it } from 'vitest'

import { VIEW_PRESET_KEY } from './replayPreferences'
import { useReplaySettings } from './useReplaySettings'
import { useReplayViewMenu } from './useReplayViewMenu'
import type { FilmLayerAvailability } from './viewPresetLogic'
import type { ReplaySound } from '../sound/useReplaySound'

const FULL_FILM: FilmLayerAvailability = {
  zones: true,
  placements: { drawable: true, unnamed: true, dropped: true },
  weaponPads: true,
  groundWeapons: true,
  modeObjectives: true,
  flagCarries: true,
  vipCrown: true,
  skullCarrier: true,
  bombCarrier: true,
  vehicles: true,
  wallPlan: true,
}

/** Un Assassin : ni zone, ni socle, ni objectif ; la visée et la traînée restent. */
const SLAYER_FILM: FilmLayerAvailability = {
  zones: false,
  placements: { drawable: false, unnamed: false, dropped: false },
  weaponPads: false,
  groundWeapons: false,
  modeObjectives: false,
  flagCarries: false,
  vipCrown: false,
  skullCarrier: false,
  bombCarrier: false,
  vehicles: false,
  wallPlan: false,
}

const SOUND = { available: false, categories: {} } as unknown as ReplaySound
const HEAT = { mode: 'presence' as const, killsAvailable: true }
const ORIENTATION = { turns: 0 as const, setTurns: () => {}, available: true }

function mount(available: FilmLayerAvailability) {
  return renderHook(() => {
    const settings = useReplaySettings()
    const menu = useReplayViewMenu({ settings, orientation: ORIENTATION, sound: SOUND, available, heat: HEAT, locale: 'fr' })
    return { settings, menu }
  })
}

describe('useReplayViewMenu', () => {
  beforeEach(() => {
    window.localStorage.clear()
  })

  it('à l’ouverture par défaut, la vue est Complet (les défauts des calques sont tous allumés)', () => {
    const { result } = mount(FULL_FILM)
    expect(result.current.menu.active).toBe('complet')
  })

  it('le panneau est fermé par défaut, le bouton l’ouvre et le referme', () => {
    const { result } = mount(FULL_FILM)
    expect(result.current.menu.open).toBe(false)
    act(() => result.current.menu.toggle())
    expect(result.current.menu.open).toBe(true)
    act(() => result.current.menu.toggle())
    expect(result.current.menu.open).toBe(false)
  })

  it('choisir Combat éteint zones, socles, armes au sol et poses ; garde visée, traînée, véhicules, objectifs', () => {
    const { result } = mount(FULL_FILM)
    act(() => result.current.menu.panel.view.onSelect('combat'))
    const s = result.current.settings
    expect([s.showAim, s.showTrail, s.showVehicles, s.showModeObjectives, s.showFlagCarries]).toEqual([
      true, true, true, true, true,
    ])
    expect([s.showZones, s.showWeaponPads, s.showGroundWeapons, s.showPlacements]).toEqual([
      false, false, false, false,
    ])
    expect(s.showDroppedPlacements).toBe(false)
    expect(result.current.menu.active).toBe('combat')
  })

  it('basculer un calque à la main passe en Personnalisé, le rebasculer rend le préréglage', () => {
    const { result } = mount(FULL_FILM)
    act(() => result.current.menu.panel.view.onSelect('objectif'))
    expect(result.current.menu.active).toBe('objectif')
    act(() => result.current.settings.toggleAim())
    expect(result.current.menu.active).toBe('personnalise')
    act(() => result.current.settings.toggleAim())
    expect(result.current.menu.active).toBe('objectif')
  })

  it('un préréglage ne touche ni les effets, ni la chaleur, ni la couleur des points, ni les affinages', () => {
    const { result } = mount(FULL_FILM)
    act(() => result.current.settings.toggleHeatmap())
    act(() => result.current.settings.toggleKillFx())
    act(() => result.current.settings.setMarkerColors('player'))
    act(() => result.current.settings.toggleUnnamedPlacements())
    const avant = result.current.settings
    const attendu = {
      heatmap: avant.showHeatmap,
      killFx: avant.showKillFx,
      shotFx: avant.showShotFx,
      colors: avant.markerColors,
      unnamed: avant.showUnnamedPlacements,
      special: avant.showGroundWeaponsSpecialOnly,
      starting: avant.showGroundWeaponsStarting,
    }
    act(() => result.current.menu.panel.view.onSelect('deplacements'))
    const s = result.current.settings
    expect({
      heatmap: s.showHeatmap,
      killFx: s.showKillFx,
      shotFx: s.showShotFx,
      colors: s.markerColors,
      unnamed: s.showUnnamedPlacements,
      special: s.showGroundWeaponsSpecialOnly,
      starting: s.showGroundWeaponsStarting,
    }).toEqual(attendu)
  })

  it('un calque indisponible n’est jamais activé ni effacé : sa préférence reste', () => {
    const { result } = mount(SLAYER_FILM)
    // Le drapeau est mémorisé éteint (préférence d'un match précédent) ; Complet ne l'allume pas.
    act(() => result.current.settings.toggleFlagCarries())
    expect(result.current.settings.showFlagCarries).toBe(false)
    act(() => result.current.menu.panel.view.onSelect('complet'))
    expect(result.current.settings.showFlagCarries).toBe(false)
    // Et l'inverse : Déplacements n'éteint pas un calque que le match ne porte pas.
    act(() => result.current.settings.toggleVipCrown())
    act(() => result.current.settings.toggleVipCrown())
    act(() => result.current.menu.panel.view.onSelect('deplacements'))
    expect(result.current.settings.showVipCrown).toBe(true)
  })

  it('sur un Assassin, la vue se lit sur les seuls calques disponibles', () => {
    const { result } = mount(SLAYER_FILM)
    expect(result.current.menu.active).toBe('complet')
    act(() => result.current.settings.toggleAim())
    expect(result.current.menu.active).toBe('deplacements')
  })

  it('retient le dernier préréglage choisi, qui départage deux préréglages égaux', () => {
    const { result } = mount(SLAYER_FILM)
    // Sur un Assassin, Complet et Combat se confondent : le choisi gagne, et il est retenu.
    act(() => result.current.menu.panel.view.onSelect('combat'))
    expect(result.current.menu.active).toBe('combat')
    expect(window.localStorage.getItem(VIEW_PRESET_KEY)).toBe('combat')
    const reouvert = mount(SLAYER_FILM)
    expect(reouvert.result.current.menu.active).toBe('combat')
  })

  it('les préférences de calques d’avant ce lot restent lues telles quelles', () => {
    window.localStorage.setItem('replay-show-aim', 'false')
    window.localStorage.setItem('replay-show-zones', 'false')
    const { result } = mount(FULL_FILM)
    expect(result.current.settings.showAim).toBe(false)
    expect(result.current.settings.showZones).toBe(false)
    expect(result.current.menu.active).toBe('personnalise')
  })

  it('une valeur retenue invalide est ignorée : la vue se déduit de l’état', () => {
    window.localStorage.setItem(VIEW_PRESET_KEY, 'n-importe-quoi')
    const { result } = mount(FULL_FILM)
    expect(result.current.menu.active).toBe('complet')
  })
})
