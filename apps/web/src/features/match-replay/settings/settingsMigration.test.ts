/**
 * settingsMigration.test.ts — LA REPRISE DES PRÉFÉRENCES D'AVANT, ancien état -> nouvel état : valeurs absentes,
 * invalides, clé du préréglage, idempotence. Les clés des réglages ne changent pas de nom : seul ce qui change de
 * sens est repris (les traces des vies que le préréglage allumait d'office, le groupe « Vies » qui ne se replie plus).
 * Les traits du relief, retenus par fond avant, sont repris par leur lecture (`lib/replay/useReliefLines.test.ts`).
 */
import { beforeEach, describe, expect, it } from 'vitest'

import { migrateReplayPreferences, PREFS_VERSION_KEY } from './settingsMigration'
import { persistViewsState, readPinnedSettings, readViewsState, persistPinnedSettings, PINS_KEY, VIEWS_KEY, VIEW_ACTIVE_KEY, VIEW_DEFAULT_KEY } from './settingsStore'
import { EMPTY_VIEWS, saveView, setDefaultView } from './savedViews'
import { DEFAULT_PINS } from './quickAccess'
import { factoryValues } from './settingsTable'
import { snapshotValues } from './viewPresets'

const get = (k: string) => window.localStorage.getItem(k)
const set = (k: string, v: string) => window.localStorage.setItem(k, v)

/** Les treize calques d'avant les vies, à l'état que le préréglage d'avant leur donnait. */
const LAYERS = ['aim', 'trail', 'zones', 'weapon-pads', 'ground-weapons', 'vehicles', 'placements', 'dropped-placements', 'mode-objectives', 'flag-carries', 'vip-crown', 'skull-carrier', 'bomb-carrier']
function layers(on: readonly string[]) {
  for (const l of LAYERS) set(`replay-show-${l}`, String(on.includes(l)))
}
const COMBAT_ON = ['aim', 'trail', 'vehicles', 'mode-objectives', 'flag-carries', 'vip-crown', 'skull-carrier', 'bomb-carrier']

beforeEach(() => window.localStorage.clear())

describe('les traces des vies que le préréglage allumait d’office', () => {
  it('Combat retenu et treize calques encore à Combat : morts et tueurs sont écrits, pas les apparitions', () => {
    set(VIEW_ACTIVE_KEY, 'combat')
    layers(COMBAT_ON)
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBe('true')
    expect(get('replay-show-killers')).toBe('true')
    expect(get('replay-show-spawns')).toBeNull()
  })

  it('Déplacements retenu : les apparitions seules', () => {
    set(VIEW_ACTIVE_KEY, 'deplacements')
    layers(['trail', 'zones'])
    migrateReplayPreferences()
    expect(get('replay-show-spawns')).toBe('true')
    expect(get('replay-show-deaths')).toBeNull()
  })

  it('Complet et Objectif n’allumaient aucune trace : rien n’est écrit', () => {
    set(VIEW_ACTIVE_KEY, 'complet')
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBeNull()
    window.localStorage.clear()
    set(VIEW_ACTIVE_KEY, 'objectif')
    layers(['trail', 'zones', 'mode-objectives', 'flag-carries', 'vip-crown', 'skull-carrier', 'bomb-carrier'])
    migrateReplayPreferences()
    expect(get('replay-show-killers')).toBeNull()
  })

  it('un calque changé depuis le choix du préréglage : la vue n’est plus la sienne, rien n’est écrit', () => {
    set(VIEW_ACTIVE_KEY, 'combat')
    layers(COMBAT_ON)
    set('replay-show-aim', 'false')
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBeNull()
  })

  it('une trace déjà écrite par le lecteur n’est jamais réécrite', () => {
    set(VIEW_ACTIVE_KEY, 'combat')
    layers(COMBAT_ON)
    set('replay-show-deaths', 'false')
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBe('false')
    expect(get('replay-show-killers')).toBe('true')
  })

  it('une clé de préréglage absente, inconnue ou « personnalise » : rien n’est écrit', () => {
    for (const raw of [null, 'inconnu', 'personnalise', '']) {
      window.localStorage.clear()
      if (raw !== null) set(VIEW_ACTIVE_KEY, raw)
      layers(COMBAT_ON)
      migrateReplayPreferences()
      expect(get('replay-show-deaths')).toBeNull()
    }
  })

  it('une valeur de calque invalide se lit comme éteinte, comme partout dans les préférences', () => {
    set(VIEW_ACTIVE_KEY, 'combat')
    layers(COMBAT_ON)
    set('replay-show-vehicles', 'peut-etre')
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBeNull()
  })
})

describe('le groupe « Vies » ne se replie plus', () => {
  it('l’ancienne clé de son état est effacée', () => {
    set('replay-lives-group-open', 'true')
    migrateReplayPreferences()
    expect(get('replay-lives-group-open')).toBeNull()
  })
})

describe('la reprise est idempotente', () => {
  it('écrit sa marque de version, et ne touche plus à rien ensuite', () => {
    set(VIEW_ACTIVE_KEY, 'combat')
    layers(COMBAT_ON)
    migrateReplayPreferences()
    expect(get(PREFS_VERSION_KEY)).toBe('2')
    // Le lecteur éteint les morts : un second passage ne les rallume pas.
    set('replay-show-deaths', 'false')
    set('replay-lives-group-open', 'true')
    migrateReplayPreferences()
    expect(get('replay-show-deaths')).toBe('false')
    expect(get('replay-lives-group-open')).toBe('true')
  })

  it('sur un navigateur neuf : ne crée que sa marque', () => {
    migrateReplayPreferences()
    expect(window.localStorage.length).toBe(1)
    expect(get(PREFS_VERSION_KEY)).toBe('2')
  })

  it('la clé du préréglage reste lue telle quelle par la mémoire des vues', () => {
    set(VIEW_ACTIVE_KEY, 'objectif')
    migrateReplayPreferences()
    expect(readViewsState().activeId).toBe('objectif')
  })
})

describe('la mémoire des vues', () => {
  it('relit ce qu’elle a retenu : vues, vue d’ouverture, vue active', () => {
    const r = saveView(EMPTY_VIEWS, 'Ma vue', snapshotValues(factoryValues()), 'v1')
    if (!r.ok) throw new Error('attendu : ok')
    persistViewsState(setDefaultView(r.state, 'v1'))
    const back = readViewsState()
    expect(back.views.map((v) => v.name)).toEqual(['Ma vue'])
    expect(back.defaultId).toBe('v1')
    expect(back.activeId).toBe('v1')
  })

  it('une vue active ou d’ouverture qui n’existe plus retombe sur Complet et sur « dernière vue »', () => {
    set(VIEW_ACTIVE_KEY, 'v-disparue')
    set(VIEW_DEFAULT_KEY, 'v-disparue')
    expect(readViewsState()).toEqual({ views: [], defaultId: null, activeId: 'complet' })
  })

  it('un JSON illisible ou étranger donne aucune vue', () => {
    set(VIEWS_KEY, '{pas du json')
    expect(readViewsState().views).toEqual([])
    set(VIEWS_KEY, '{"a":1}')
    expect(readViewsState().views).toEqual([])
  })

  it('les épingles : usine sans choix, relues et validées avec un choix, un JSON illisible donne l’usine', () => {
    expect(readPinnedSettings()).toEqual(DEFAULT_PINS)
    persistPinnedSettings(['zones', 'vehicles'])
    expect(readPinnedSettings()).toEqual(['zones', 'vehicles'])
    persistPinnedSettings([])
    expect(readPinnedSettings()).toEqual([])
    set(PINS_KEY, '[[')
    expect(readPinnedSettings()).toEqual(DEFAULT_PINS)
  })
})
