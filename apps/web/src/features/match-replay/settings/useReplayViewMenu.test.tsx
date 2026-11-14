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
  lives: { spawns: true, deaths: true, killers: true, followedKnown: true },
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
  lives: { spawns: false, deaths: false, killers: false, followedKnown: false },
}

const SOUND = { available: false, categories: {} } as unknown as ReplaySound
const HEAT = { mode: 'presence' as const, killsAvailable: true }
const ORIENTATION = { turns: 0 as const, setTurns: () => {}, available: true }
const NO_PLAN_BACKGROUND = { available: false, shown: false }

function mount(available: FilmLayerAvailability, planBackground = NO_PLAN_BACKGROUND) {
  return renderHook(() => {
    const settings = useReplaySettings()
    const menu = useReplayViewMenu({ settings, orientation: ORIENTATION, sound: SOUND, available, heat: HEAT, locale: 'fr', planBackground })
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

  it('le fond : la commande suit la présence d une image plan, et son choix est retenu', () => {
    const sans = mount(FULL_FILM)
    expect(sans.result.current.menu.panel.background.available).toBe(false)
    const { result } = mount(FULL_FILM, { available: true, shown: false })
    expect(result.current.menu.panel.background).toMatchObject({ available: true, value: 'rendu' })
    act(() => result.current.menu.panel.background.onSelect('plan'))
    expect(result.current.menu.panel.background.value).toBe('plan')
    expect(window.localStorage.getItem('replay-background-style')).toBe('plan')
  })

  it('plan demandé mais repli sur le rendu (image plan en échec) : le fond posé est le rendu, l’interrupteur des murs reste actif', () => {
    window.localStorage.setItem('replay-background-style', 'plan')
    const { result } = mount(FULL_FILM, { available: true, shown: false })
    expect(result.current.menu.panel.background.value).toBe('plan')
    expect(result.current.menu.panel.wallPlan.inPlan).toBe(false)
  })

  it('le plan des murs est désactivé quand le fond posé est le plan, et seulement alors', () => {
    expect(mount(FULL_FILM, { available: true, shown: false }).result.current.menu.panel.wallPlan.inPlan).toBe(false)
    expect(mount(FULL_FILM, { available: true, shown: true }).result.current.menu.panel.wallPlan.inPlan).toBe(true)
  })

  it('les trois calques des vies sont éteints à l’ouverture : Complet est l’état d’ouverture', () => {
    const { result } = mount(FULL_FILM)
    const s = result.current.settings
    expect([s.showSpawns, s.showDeaths, s.showKillers]).toEqual([false, false, false])
    expect(result.current.menu.active).toBe('complet')
  })

  it('Combat allume morts et tueurs, Déplacements les apparitions, Complet et Objectif les éteignent', () => {
    const { result } = mount(FULL_FILM)
    const vies = () => {
      const s = result.current.settings
      return [s.showSpawns, s.showDeaths, s.showKillers]
    }
    act(() => result.current.menu.panel.view.onSelect('combat'))
    expect(vies()).toEqual([false, true, true])
    act(() => result.current.menu.panel.view.onSelect('deplacements'))
    expect(vies()).toEqual([true, false, false])
    act(() => result.current.menu.panel.view.onSelect('combat'))
    act(() => result.current.menu.panel.view.onSelect('objectif'))
    expect(vies()).toEqual([false, false, false])
    act(() => result.current.menu.panel.view.onSelect('combat'))
    act(() => result.current.menu.panel.view.onSelect('complet'))
    expect(vies()).toEqual([false, false, false])
    expect(result.current.menu.active).toBe('complet')
  })

  it('un calque des vies sans marque n’est jamais activé par un préréglage', () => {
    const { result } = mount({ ...FULL_FILM, lives: { spawns: false, deaths: false, killers: false, followedKnown: false } })
    act(() => result.current.menu.panel.view.onSelect('combat'))
    act(() => result.current.menu.panel.view.onSelect('deplacements'))
    const s = result.current.settings
    expect([s.showSpawns, s.showDeaths, s.showKillers]).toEqual([false, false, false])
  })

  it('les affinages des vies ne sont pas des calques : un préréglage ne les touche pas, la vue non plus', () => {
    const { result } = mount(FULL_FILM)
    act(() => result.current.settings.setLivesWho('team'))
    act(() => result.current.settings.setLivesWhen('sofar'))
    act(() => result.current.menu.panel.view.onSelect('combat'))
    expect([result.current.settings.livesWho, result.current.settings.livesWhen]).toEqual(['team', 'sofar'])
    expect(result.current.menu.active).toBe('combat')
  })

  it('le panneau expose le groupe des vies : disponibilité, état, et « joueur suivi » seulement avec un joueur suivi', () => {
    const { result } = mount(FULL_FILM)
    expect(result.current.menu.panel.lives).toMatchObject({
      spawns: { available: true, show: false },
      deaths: { available: true, show: false },
      killers: { available: true, show: false },
      who: 'all',
      when: 'all',
      followedAvailable: true,
    })
    const sans = mount({ ...FULL_FILM, lives: { ...FULL_FILM.lives, followedKnown: false } })
    expect(sans.result.current.menu.panel.lives.followedAvailable).toBe(false)
  })

  it('« joueur suivi » mémorisé retombe sur « tous » dans un match sans joueur suivi, sans effacer la préférence', () => {
    window.localStorage.setItem('replay-lives-who', 'followed')
    const sans = mount({ ...FULL_FILM, lives: { ...FULL_FILM.lives, followedKnown: false } })
    expect(sans.result.current.menu.panel.lives.who).toBe('all')
    expect(sans.result.current.settings.livesWho).toBe('followed')
    const avec = mount(FULL_FILM)
    expect(avec.result.current.menu.panel.lives.who).toBe('followed')
  })

  it('les affinages et calques des vies sont retenus, et une valeur invalide est ignorée', () => {
    const { result } = mount(FULL_FILM)
    act(() => result.current.settings.toggleDeaths())
    act(() => result.current.settings.setLivesWho('team'))
    act(() => result.current.settings.setLivesWhen('sofar'))
    expect(window.localStorage.getItem('replay-show-deaths')).toBe('true')
    expect(window.localStorage.getItem('replay-lives-who')).toBe('team')
    expect(window.localStorage.getItem('replay-lives-when')).toBe('sofar')
    const reouvert = mount(FULL_FILM)
    expect([reouvert.result.current.settings.showDeaths, reouvert.result.current.settings.livesWho, reouvert.result.current.settings.livesWhen]).toEqual([true, 'team', 'sofar'])
    window.localStorage.setItem('replay-lives-who', 'n-importe-quoi')
    window.localStorage.setItem('replay-lives-when', '')
    const invalide = mount(FULL_FILM)
    expect([invalide.result.current.settings.livesWho, invalide.result.current.settings.livesWhen]).toEqual(['all', 'all'])
  })
})
