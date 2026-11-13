/**
 * viewPresetLogic.test.ts — LES PRÉRÉGLAGES DE VUE : la table, ce qu'un préréglage fait d'un
 * état de calques, et comment l'état dit quelle vue est active. Un test par règle.
 */
import { describe, expect, it } from 'vitest'

import {
  CUSTOM_VIEW,
  LAYER_KEYS,
  PRESET_LAYERS,
  VIEW_PRESET_IDS,
  activeView,
  layerAvailability,
  layersToToggle,
  presetFlags,
  type FilmLayerAvailability,
  type LayerFlags,
  type LayerKey,
} from './viewPresetLogic'

const ALL_AVAILABLE = Object.fromEntries(LAYER_KEYS.map((k) => [k, true])) as LayerFlags

function flags(on: readonly LayerKey[]): LayerFlags {
  return Object.fromEntries(LAYER_KEYS.map((k) => [k, on.includes(k)])) as LayerFlags
}

/** L'état d'un match dont ne sont disponibles que les calques de `keys`. */
function availableOnly(keys: readonly LayerKey[]): LayerFlags {
  return flags(keys)
}

/** Applique les bascules demandées : ce que fait la page, calque par calque. */
function applied(current: LayerFlags, toggled: readonly LayerKey[]): LayerFlags {
  const next = { ...current }
  for (const k of toggled) next[k] = !next[k]
  return next
}

describe('la table des préréglages (pièce à valider : une ligne par calque réel)', () => {
  it('Complet allume TOUS les calques, Déplacements seulement traînée et zones', () => {
    expect([...PRESET_LAYERS.complet].sort()).toEqual([...LAYER_KEYS].sort())
    expect([...PRESET_LAYERS.deplacements].sort()).toEqual(['trail', 'zones'])
  })

  it('Combat : visée, traînée, véhicules et tous les objectifs ; ni terrain ni poses', () => {
    expect([...PRESET_LAYERS.combat].sort()).toEqual(
      [
        'aim', 'trail', 'vehicles',
        'modeObjectives', 'flagCarries', 'vipCrown', 'skullCarrier', 'bombCarrier',
      ].sort(),
    )
  })

  it('Objectif : traînée, zones et tous les objectifs ; ni visée, ni véhicules, ni poses', () => {
    expect([...PRESET_LAYERS.objectif].sort()).toEqual(
      [
        'trail', 'zones',
        'modeObjectives', 'flagCarries', 'vipCrown', 'skullCarrier', 'bombCarrier',
      ].sort(),
    )
  })

  it('chaque préréglage ne nomme que des calques connus, sans doublon', () => {
    for (const id of VIEW_PRESET_IDS) {
      const keys = PRESET_LAYERS[id]
      expect(new Set(keys).size).toBe(keys.length)
      for (const k of keys) expect(LAYER_KEYS).toContain(k)
    }
  })

  it('presetFlags donne une valeur à CHAQUE calque (la table est totale)', () => {
    for (const id of VIEW_PRESET_IDS) {
      const target = presetFlags(id)
      expect(Object.keys(target).sort()).toEqual([...LAYER_KEYS].sort())
    }
  })
})

describe('layerAvailability : ce que le film porte, calque par calque', () => {
  const film: FilmLayerAvailability = {
    zones: true,
    placements: { drawable: true, unnamed: false, dropped: false },
    weaponPads: false,
    groundWeapons: true,
    modeObjectives: false,
    flagCarries: true,
    vipCrown: false,
    skullCarrier: false,
    bombCarrier: false,
    vehicles: true,
    wallPlan: true,
  }

  it('la visée et la traînée sont toujours disponibles', () => {
    const a = layerAvailability({ ...film, zones: false, vehicles: false })
    expect(a.aim).toBe(true)
    expect(a.trail).toBe(true)
  })

  it('les calques lus du film suivent leur disponibilité', () => {
    const a = layerAvailability(film)
    expect(a.zones).toBe(true)
    expect(a.weaponPads).toBe(false)
    expect(a.groundWeapons).toBe(true)
    expect(a.modeObjectives).toBe(false)
    expect(a.flagCarries).toBe(true)
    expect(a.vehicles).toBe(true)
  })

  it('les objets lâchés n existent que si les poses sont dessinables ET qu un objet est lâché', () => {
    expect(layerAvailability(film).placements).toBe(true)
    expect(layerAvailability(film).droppedPlacements).toBe(false)
    const lacher = { ...film, placements: { drawable: true, unnamed: false, dropped: true } }
    expect(layerAvailability(lacher).droppedPlacements).toBe(true)
    const sansPose = { ...film, placements: { drawable: false, unnamed: false, dropped: true } }
    expect(layerAvailability(sansPose).placements).toBe(false)
    expect(layerAvailability(sansPose).droppedPlacements).toBe(false)
  })
})

describe('layersToToggle : un préréglage ne touche que ce qui diffère', () => {
  it('depuis un état déjà égal au préréglage : aucune bascule', () => {
    for (const id of VIEW_PRESET_IDS) {
      expect(layersToToggle(id, presetFlags(id), ALL_AVAILABLE)).toEqual([])
    }
  })

  it('Combat depuis tout allumé : éteint exactement ce que Combat n allume pas', () => {
    const toggled = layersToToggle('combat', flags(LAYER_KEYS), ALL_AVAILABLE)
    expect([...toggled].sort()).toEqual(
      ['zones', 'weaponPads', 'groundWeapons', 'placements', 'droppedPlacements'].sort(),
    )
  })

  it('Complet depuis tout éteint : allume tout', () => {
    const toggled = layersToToggle('complet', flags([]), ALL_AVAILABLE)
    expect([...toggled].sort()).toEqual([...LAYER_KEYS].sort())
  })

  it('JAMAIS un calque indisponible : aucun n est basculé, quel que soit le préréglage', () => {
    for (const id of VIEW_PRESET_IDS) {
      for (const missing of LAYER_KEYS) {
        const available = { ...ALL_AVAILABLE, [missing]: false }
        for (const start of [flags([]), flags(LAYER_KEYS)]) {
          expect(layersToToggle(id, start, available)).not.toContain(missing)
        }
      }
    }
  })

  it('un calque indisponible reste TEL QUEL (sa préférence mémorisée n est pas écrasée)', () => {
    const available = availableOnly(['aim', 'trail'])
    const start = flags(['aim', 'flagCarries'])
    const next = applied(start, layersToToggle('deplacements', start, available))
    expect(next.flagCarries).toBe(true)
    expect(next.zones).toBe(false)
  })

  it('un préréglage n active pas un calque indisponible même s il l inclut', () => {
    const available = availableOnly(['aim', 'trail', 'vehicles'])
    const start = flags([])
    const next = applied(start, layersToToggle('complet', start, available))
    expect(next.aim && next.trail && next.vehicles).toBe(true)
    expect(next.zones).toBe(false)
    expect(next.flagCarries).toBe(false)
  })
})

describe('activeView : le préréglage actif se déduit de l état', () => {
  it('un état égal à un préréglage nomme ce préréglage', () => {
    expect(activeView(presetFlags('complet'), ALL_AVAILABLE)).toBe('complet')
    expect(activeView(presetFlags('combat'), ALL_AVAILABLE)).toBe('combat')
    expect(activeView(presetFlags('deplacements'), ALL_AVAILABLE)).toBe('deplacements')
    expect(activeView(presetFlags('objectif'), ALL_AVAILABLE)).toBe('objectif')
  })

  it('un calque qui diffère d un préréglage passe la vue en Personnalisé', () => {
    const state = { ...presetFlags('combat'), zones: true }
    expect(activeView(state, ALL_AVAILABLE)).toBe(CUSTOM_VIEW)
  })

  it('basculer un calque depuis un préréglage donne Personnalisé, le rebasculer le rend', () => {
    const start = presetFlags('objectif')
    const once = applied(start, ['aim'])
    expect(activeView(once, ALL_AVAILABLE)).toBe(CUSTOM_VIEW)
    expect(activeView(applied(once, ['aim']), ALL_AVAILABLE)).toBe('objectif')
  })

  it('l égalité ne porte que sur les calques DISPONIBLES', () => {
    // Un Assassin : ni zones ni objectifs ni socles. Déplacements = traînée seule.
    const available = availableOnly(['aim', 'trail'])
    const state = { ...flags(['trail']), flagCarries: true, zones: true }
    expect(activeView(state, available)).toBe('deplacements')
  })

  it('un calque indisponible ne rend pas un état personnalisé', () => {
    const available = availableOnly(['aim', 'trail', 'vehicles'])
    const state = { ...presetFlags('complet'), vipCrown: false, zones: false }
    expect(activeView(state, available)).toBe('complet')
  })

  it('égalité entre plusieurs préréglages : le mémorisé gagne, sinon le premier de la liste', () => {
    // Sans objectif, zone ni véhicule : Complet et Combat se confondent (visée + traînée).
    const available = availableOnly(['aim', 'trail'])
    const state = flags(['aim', 'trail'])
    expect(activeView(state, available)).toBe('complet')
    expect(activeView(state, available, 'combat')).toBe('combat')
  })

  it('le mémorisé ne gagne que s il correspond encore à l état', () => {
    const state = { ...presetFlags('combat'), zones: true }
    expect(activeView(state, ALL_AVAILABLE, 'combat')).toBe(CUSTOM_VIEW)
    expect(activeView(presetFlags('objectif'), ALL_AVAILABLE, 'combat')).toBe('objectif')
  })

  it('un match où rien ne distingue les préréglages ne bascule jamais en Personnalisé', () => {
    const available = availableOnly(['trail'])
    expect(activeView(flags(['trail']), available)).toBe('complet')
    expect(activeView(flags([]), available)).toBe(CUSTOM_VIEW)
  })
})
