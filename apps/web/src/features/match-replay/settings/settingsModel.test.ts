/**
 * settingsModel.test.ts — LA TABLE DES RÉGLAGES et ce qu'on en déduit : sa cohérence, ce qui est proposé ou
 * affiché selon le match, les interrupteurs maîtres, les préréglages souples (« modifié » déduit sur les seuls
 * réglages proposés) et l'accès rapide. Logique pure, sans rendu.
 */
import { describe, expect, it } from 'vitest'

import { FULL_FILM, makeSettingsContext, SLAYER_FILM } from '../test/settingsContextFixtures'
import {
  isProposed, isShown, masterState, masterWrites, matchModeKey, optionsOf, pinStatus, shownOfRubric,
} from './settingsAvailability'
import {
  defOf, defsOfRubric, factoryValues, PRESET_COVERED, RUBRIC_IDS, SETTING_DEFS, SETTING_IDS,
  type SettingValues,
} from './settingsTable'
import { changedFrom, PRESET_IDS, presetValues, snapshotValues, writesToReach } from './viewPresets'
import { DEFAULT_PINS, PIN_MAX, readPins, togglePin } from './quickAccess'

const FACTORY = factoryValues()
const with_ = (over: Partial<Record<string, unknown>>): SettingValues => ({ ...FACTORY, ...over }) as SettingValues

describe('la table des réglages', () => {
  it('porte une ligne par identifiant, sans doublon', () => {
    expect(SETTING_DEFS.map((d) => d.id).sort()).toEqual([...SETTING_IDS].sort())
  })

  it('range chaque réglage dans une rubrique connue, ou hors rubriques pour la barre fine seule', () => {
    const apart = SETTING_DEFS.filter((d) => d.rubric === null).map((d) => d.id)
    expect(apart).toEqual(['thinBar'])
    for (const d of SETTING_DEFS) if (d.rubric) expect(RUBRIC_IDS).toContain(d.rubric)
  })

  it('ne rattache une fille qu’à une mère de la table, qui la précède', () => {
    for (const d of SETTING_DEFS) {
      if (!d.parent) continue
      const parent = SETTING_DEFS.findIndex((p) => p.id === d.parent)
      expect(parent).toBeGreaterThanOrEqual(0)
      expect(parent).toBeLessThan(SETTING_DEFS.indexOf(d))
    }
  })

  it('ne propose en choix que des défauts qui sont dans ses options', () => {
    for (const d of SETTING_DEFS) if (d.options) expect(d.options).toContain(d.default)
  })

  it('les interrupteurs hors préréglage, hors maître ou hors accès rapide sont ceux que la maquette désigne', () => {
    const off = (key: 'preset' | 'master' | 'pinnable') => SETTING_DEFS.filter((d) => d.control === 'toggle' && !d[key]).map((d) => d.id)
    // Le plan des murs et les traits du relief sont partagés avec d'autres pages : hors préréglages (`SHARED_SETTING_IDS`).
    expect(off('preset')).toEqual(['reliefLines', 'wallPlan', 'floorFollow', 'sound', 'thinBar'])
    expect(off('master')).toEqual(['floorFollow', 'thinBar'])
    expect(off('pinnable')).toEqual([
      'floorFollow', 'groundSpecial', 'groundStarting', 'placeDropped', 'placeUnnamed',
      'vip', 'skull', 'bomb', 'catWeapon', 'catGrenade', 'catMelee', 'catEquip', 'thinBar',
    ])
  })

  it('les préréglages couvrent 29 réglages : les interrupteurs de rubrique sauf l’étage, le son, la barre fine et les réglages partagés, plus le filtre du fil', () => {
    expect(PRESET_COVERED).toHaveLength(29)
    expect(PRESET_COVERED).toContain('feedFilter')
    for (const hors of ['bg', 'orientation', 'floor', 'floorFollow', 'sound', 'volume', 'colors', 'heatMode', 'heatSpan', 'livesWho', 'livesWhen', 'forecastDraw', 'thinBar'] as const) {
      expect(PRESET_COVERED).not.toContain(hors)
    }
  })

  it('les valeurs d’usine sont exactement le préréglage Complet (l’état d’ouverture)', () => {
    expect(changedFrom(presetValues('complet'), FACTORY, makeSettingsContext())).toEqual([])
  })
})

describe('ce que le match propose', () => {
  it('un Assassin ne propose ni zones, ni objets, ni objectifs, ni vies ; la visée et la traînée restent', () => {
    const c = makeSettingsContext({ film: SLAYER_FILM })
    for (const id of ['zones', 'weaponPads', 'groundWeapons', 'groundSpecial', 'vehicles', 'placements', 'objectives', 'flags', 'spawns', 'forecast'] as const) {
      expect(isProposed(id, c)).toBe(false)
    }
    expect(isProposed('aim', c)).toBe(true)
    expect(isProposed('trail', c)).toBe(true)
    expect(isProposed('heatmap', c)).toBe(true)
  })

  it('le plan des murs n’est pas proposé sur le fond Plan, où les murs sont déjà dans l’image', () => {
    expect(isProposed('wallPlan', makeSettingsContext({ drawn: 'plan' }))).toBe(false)
    expect(isProposed('wallPlan', makeSettingsContext({ drawn: 'relief' }))).toBe(true)
    expect(isProposed('wallPlan', makeSettingsContext({ drawn: 'rendu', film: { ...FULL_FILM, wallPlan: false } }))).toBe(false)
  })

  it('le fil posé sur le plan n’est proposé que là où le cadre le pose ; la barre fine, qu’au Plein cadre', () => {
    expect(isProposed('planFeed', makeSettingsContext({ planFeedAvailable: false }))).toBe(false)
    expect(isProposed('thinBar', makeSettingsContext())).toBe(false)
    expect(isProposed('thinBar', makeSettingsContext({ thinBarAvailable: true }))).toBe(true)
  })

  it('le fond ne se propose que s’il y a un choix, l’étage que sur une carte à maillage, le son que si la piste en porte', () => {
    const c = makeSettingsContext({ backgroundOptions: [], floor: 'none', soundAvailable: false, orientationAvailable: false })
    for (const id of ['bg', 'floor', 'floorFollow', 'orientation', 'sound', 'volume', 'catWeapon'] as const) expect(isProposed(id, c)).toBe(false)
    expect(isProposed('floor', makeSettingsContext({ floor: 'noMesh' }))).toBe(true)
    expect(isProposed('floorFollow', makeSettingsContext({ floor: 'noMesh' }))).toBe(false)
  })

  it('les options d’un choix se réduisent : pas d’« éliminations » sans mort localisable, pas de « point de vue » sans joueur suivi', () => {
    expect(optionsOf('heatMode', makeSettingsContext({ heatKillsAvailable: false }))).toEqual(['presence'])
    expect(optionsOf('livesWho', makeSettingsContext({ film: { ...FULL_FILM, lives: { ...FULL_FILM.lives, followedKnown: false } } }))).toEqual(['all', 'team'])
    expect(optionsOf('bg', makeSettingsContext({ backgroundOptions: ['plan', 'rendu'] }))).toEqual(['plan', 'rendu'])
  })
})

describe('ce que la rubrique affiche', () => {
  it('une fille disparaît quand sa mère est éteinte, mais pas une couche indépendante sous « Objectifs du mode »', () => {
    const c = makeSettingsContext()
    const off = with_({ groundWeapons: false, placements: false, heatmap: false, forecast: false, objectives: false })
    expect(isShown('groundSpecial', c, off)).toBe(false)
    expect(isShown('placeDropped', c, off)).toBe(false)
    expect(isShown('heatSpan', c, off)).toBe(false)
    expect(isShown('forecastDraw', c, off)).toBe(false)
    // Les drapeaux restent dessinés quand les objectifs sont éteints : leur commande reste atteignable.
    expect(isShown('flags', c, off)).toBe(true)
    expect(isShown('groundSpecial', c, FACTORY)).toBe(true)
  })

  it('« qui » et « quand » des vies ne s’affichent qu’avec une trace allumée', () => {
    const c = makeSettingsContext()
    expect(isShown('livesWho', c, FACTORY)).toBe(false)
    expect(isShown('livesWho', c, with_({ deaths: true }))).toBe(true)
    expect(isShown('livesWhen', c, with_({ spawns: true }))).toBe(true)
    // Une trace allumée mais sans marque sur ce match n’y compte pas.
    const sansMorts = makeSettingsContext({ film: { ...FULL_FILM, lives: { ...FULL_FILM.lives, deaths: false } } })
    expect(isShown('livesWho', sansMorts, with_({ deaths: true }))).toBe(false)
  })

  it('range les lignes dans l’ordre de la table, sous leur sous-titre', () => {
    const c = makeSettingsContext()
    const joueurs = shownOfRubric('joueurs', c, FACTORY).map((d) => d.id)
    expect(joueurs).toEqual(['aim', 'trail', 'colors', 'shotFx', 'killFx', 'spawns', 'deaths', 'killers', 'forecast', 'forecastDraw'])
    expect(defsOfRubric('son').map((d) => d.sub)).toEqual([undefined, undefined, 'categories', 'categories', 'categories', 'categories'])
  })
})

describe('l’interrupteur maître d’une rubrique', () => {
  it('compte « N sur M » les interrupteurs affichés, et dit tout, une partie ou rien', () => {
    const c = makeSettingsContext()
    const objets = masterState('objets', c, FACTORY)
    // Trois filles (armes spéciales, armes de départ, objets non identifiés) sont éteintes d'usine.
    expect(objets).toEqual({ on: 7, total: 10, state: 'some' })
    expect(masterState('objets', c, with_(Object.fromEntries(masterWrites('objets', c, true)))).state).toBe('all')
    expect(masterState('objets', c, with_({ vehicles: false })).state).toBe('some')
    const tout = Object.fromEntries(masterWrites('objets', c, false)) as Record<string, boolean>
    expect(masterState('objets', c, with_(tout)).state).toBe('none')
  })

  it('n’écrit que des interrupteurs proposés par le match', () => {
    const slayer = makeSettingsContext({ film: SLAYER_FILM })
    expect(masterWrites('objets', slayer, true)).toEqual([])
    expect(masterWrites('joueurs', slayer, true).map(([id]) => id)).toEqual(['aim', 'trail', 'shotFx', 'killFx'])
  })

  it('le maître du son compte l’interrupteur du son et les quatre catégories', () => {
    expect(masterState('son', makeSettingsContext(), FACTORY).total).toBe(5)
    expect(masterWrites('son', makeSettingsContext(), true).map(([id]) => id)).toEqual(['sound', 'catWeapon', 'catGrenade', 'catMelee', 'catEquip'])
  })
})

describe('les préréglages souples', () => {
  it('Complet à l’état d’usine : rien n’est modifié ; chaque réglage changé le devient', () => {
    const c = makeSettingsContext()
    expect(changedFrom(presetValues('complet'), FACTORY, c)).toEqual([])
    expect(changedFrom(presetValues('complet'), with_({ aim: false, heatmap: true }), c)).toEqual(['aim', 'heatmap'])
  })

  it('un réglage sans objet sur ce match ne rend pas la vue modifiée, et l’application ne le touche pas', () => {
    const c = makeSettingsContext({ film: SLAYER_FILM })
    // Combat éteint les zones : sans zone sur ce match, ni écriture ni écart.
    const writes = writesToReach(presetValues('combat'), FACTORY, c).map(([id]) => id)
    expect(writes).not.toContain('zones')
    expect(changedFrom(presetValues('combat'), with_({ zones: true }), c)).not.toContain('zones')
  })

  it('appliquer Combat écrit les seuls réglages qui diffèrent, filtre du fil compris', () => {
    const c = makeSettingsContext()
    const writes = Object.fromEntries(writesToReach(presetValues('combat'), FACTORY, c))
    expect(writes).toMatchObject({ deaths: true, killers: true, zones: false, forecast: false, feedFilter: 'kills' })
    expect(writes).not.toHaveProperty('aim')
    expect(writes).not.toHaveProperty('sound')
    expect(writes).not.toHaveProperty('bg')
    expect(writes).not.toHaveProperty('orientation')
  })

  it('le filtre « objectifs » d’Objectif ne se vise pas sur un match qui n’en propose pas', () => {
    const c = makeSettingsContext({ feedFilters: ['tout', 'kills'] })
    expect(writesToReach(presetValues('objectif'), FACTORY, c).map(([id]) => id)).not.toContain('feedFilter')
  })

  it('une vue enregistrée qui ne dit rien d’un réglage ne le compare ni ne l’écrit', () => {
    const c = makeSettingsContext()
    const view = snapshotValues(FACTORY)
    const sans = { ...view } as Record<string, unknown>
    delete sans.heatmap
    expect(changedFrom(sans, with_({ heatmap: true }), c)).toEqual([])
  })

  it('les quatre préréglages ont tous les réglages couverts, vip, crâne et bombe suivant les drapeaux', () => {
    for (const id of PRESET_IDS) {
      const t = presetValues(id)
      expect(Object.keys(t).sort()).toEqual([...PRESET_COVERED].sort())
      for (const porte of ['vip', 'skull', 'bomb'] as const) expect(t[porte]).toBe(t.flags)
    }
  })

  it('Déplacements allume la chaleur et les apparitions, Objectif filtre le fil sur les objectifs', () => {
    expect(presetValues('deplacements')).toMatchObject({ heatmap: true, spawns: true, trail: true, aim: false })
    expect(presetValues('objectif')).toMatchObject({ feedFilter: 'objectives', objectives: true, aim: false })
  })
})

describe('le mode de jeu lu du film', () => {
  it('l’enjeu porté décide, dans l’ordre drapeau, crâne, VIP, bombe, zones, sinon aucun', () => {
    expect(matchModeKey(FULL_FILM)).toBe('flag')
    expect(matchModeKey({ ...FULL_FILM, flagCarries: false })).toBe('zones')
    expect(matchModeKey({ ...FULL_FILM, flagCarries: false, skullCarrier: true })).toBe('skull')
    expect(matchModeKey({ ...FULL_FILM, flagCarries: false, vipCrown: true })).toBe('vip')
    expect(matchModeKey({ ...FULL_FILM, flagCarries: false, bombCarrier: true })).toBe('bomb')
    expect(matchModeKey(SLAYER_FILM)).toBe('open')
  })
})

describe('l’accès rapide', () => {
  it('épingle jusqu’à cinq réglages et dit quand il n’y a plus de place', () => {
    let pins = DEFAULT_PINS
    pins = togglePin(pins, 'zones').pins
    expect(pins).toHaveLength(PIN_MAX)
    const full = togglePin(pins, 'vehicles')
    expect(full).toEqual({ pins, full: true })
    expect(togglePin(pins, 'zones').pins).not.toContain('zones')
  })

  it('refuse d’épingler ce qui ne s’épingle pas', () => {
    expect(togglePin(['aim'], 'groundSpecial')).toEqual({ pins: ['aim'], full: false })
    expect(togglePin(['aim'], 'bg').pins).toEqual(['aim'])
  })

  it('relit les épingles du stockage en écartant l’inconnu, l’inépinglable et les doublons', () => {
    expect(readPins(['aim', 'aim', 'nimporte', 'groundSpecial', 'zones'])).toEqual(['aim', 'zones'])
    expect(readPins(undefined)).toEqual(DEFAULT_PINS)
    expect(readPins(['aim', 'trail', 'zones', 'heatmap', 'vehicles', 'planFeed'])).toHaveLength(PIN_MAX)
    expect(readPins([])).toEqual([])
  })

  it('un réglage épinglé sans objet reste affiché, grisé, avec la raison', () => {
    expect(pinStatus('planFeed', makeSettingsContext({ planFeedAvailable: false }))).toEqual({ enabled: false, reason: 'frame' })
    expect(pinStatus('zones', makeSettingsContext({ film: SLAYER_FILM }))).toEqual({ enabled: false, reason: 'match' })
    expect(pinStatus('aim', makeSettingsContext())).toEqual({ enabled: true, reason: null })
    expect(defOf('aim').pinnable).toBe(true)
  })
})
