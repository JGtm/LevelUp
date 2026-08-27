/**
 * Tests — equipmentZones (le joueur dans une zone d'équipement).
 *
 * CE QU'ILS PROTÈGENT : la fiche ne dit jamais autre chose que ce que la carte dessine —
 * mêmes portes (famille déployée, fenêtre active), mêmes rayons, même règle de camp que la
 * révélation du capteur. Chaque porte est éprouvée sur ses deux faces.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement } from '@/lib/api/types'

import { NO_ZONES, zonePresenceAt, type ZoneScene } from './equipmentZones'
import { FIELD_ID, pose, SENSOR_ID } from './test/placementFixtures'

/** 100 ms la frame, 600 images : le même axe de temps que la fixture du calque. */
const TIME = { frameMs: 100, frames: 600 }

/** Une scène : des poses, et une table de camps par slot (absent = camp inconnu). */
function sceneOf(
  placements: ReplayEquipmentPlacement[],
  sides: Record<number, string> = {},
): ZoneScene {
  return { placements, sideOfSlot: (slot) => sides[slot] ?? null }
}

/** Le joueur interrogé : slot 1, à (x, y), à l'image donnée. */
function at(x: number, y: number, frame = 50) {
  return { slot: 1, x, y, frame }
}

const field = (over: Partial<ReplayEquipmentPlacement> = {}) =>
  pose({ family: 'repair_field', id: FIELD_ID, x: 5, y: 5, ...over })
const shroud = (over: Partial<ReplayEquipmentPlacement> = {}) =>
  pose({ family: 'shroud_screen', id: '0x5eeb1a13', x: 5, y: 5, ...over })
const sensor = (over: Partial<ReplayEquipmentPlacement> = {}) =>
  pose({ family: 'sensor', id: SENSOR_ID, x: 5, y: 5, owner: 2, ...over })

describe('zonePresenceAt — champ de réparation et écran occultant', () => {
  it('dedans : la zone est dite ; dehors : rien — au rayon du calque', () => {
    // Champ : rayon 3 m. À 1 m du centre : dedans ; à 3,5 m : dehors.
    expect(zonePresenceAt(sceneOf([field()]), at(6, 5), TIME).repair).toBe(true)
    expect(zonePresenceAt(sceneOf([field()]), at(8.5, 5), TIME).repair).toBe(false)
    // Écran : rayon 6 m. À 5 m : dedans ; à 7 m : dehors.
    expect(zonePresenceAt(sceneOf([shroud()]), at(10, 5), TIME).shroud).toBe(true)
    expect(zonePresenceAt(sceneOf([shroud()]), at(5, 12.5), TIME).shroud).toBe(false)
  })

  it('aucun camp requis : le dôme du jeu soigne et cache tout le monde', () => {
    // La table des camps est VIDE (tout le monde à null) : les deux zones valent quand même.
    const p = zonePresenceAt(sceneOf([field(), shroud()]), at(5, 5), TIME)
    expect(p.repair).toBe(true)
    expect(p.shroud).toBe(true)
  })

  it('avant t0, rien — la pose n’existe pas encore', () => {
    expect(zonePresenceAt(sceneOf([field({ t0: 80 })]), at(5, 5, 50), TIME)).toBe(NO_ZONES)
  })

  it('après t1, la zone TIENT — t1 est une mise au repos, pas une disparition (règle du calque)', () => {
    // Le champ n'a pas de durée officielle : il reste, comme sur la carte, jusqu'à la fin.
    expect(zonePresenceAt(sceneOf([field({ t1: 20 })]), at(5, 5, 400), TIME).repair).toBe(true)
  })

  it('un objet LÂCHÉ ou d’origine inconnue n’exerce aucune zone — la porte du calque', () => {
    expect(zonePresenceAt(sceneOf([field({ origin: 'dropped' })]), at(5, 5), TIME)).toBe(NO_ZONES)
    expect(zonePresenceAt(sceneOf([field({ origin: undefined })]), at(5, 5), TIME)).toBe(NO_ZONES)
  })

  it('une famille sans zone (le mur) ne déclenche rien, même sous le joueur', () => {
    expect(zonePresenceAt(sceneOf([pose()]), at(5, 5), TIME)).toBe(NO_ZONES)
  })
})

describe('zonePresenceAt — capteur de menaces adverse', () => {
  const CAMPS = { 1: 't0', 2: 't1' }

  it('capteur adverse couvrant le joueur : l’âge du dernier ping est publié', () => {
    // t0 = 10, image 50 : âge 40 frames = 4 000 ms, soit 400 ms après le 3e ping (période 1,8 s).
    const p = zonePresenceAt(sceneOf([sensor()], CAMPS), at(5, 5), TIME)
    expect(p.sensorSincePingMs).toBe(400)
  })

  it('hors de portée (4,25 m officiel) : rien', () => {
    const p = zonePresenceAt(sceneOf([sensor()], CAMPS), at(10, 5), TIME)
    expect(p.sensorSincePingMs).toBeNull()
  })

  it('même camp, poseur non mesuré, ou camp inconnu d’un côté : AUCUNE inimitié affirmée', () => {
    const memeCamp = { 1: 't0', 2: 't0' }
    expect(zonePresenceAt(sceneOf([sensor()], memeCamp), at(5, 5), TIME)).toBe(NO_ZONES)
    expect(zonePresenceAt(sceneOf([sensor({ owner: -1 })], CAMPS), at(5, 5), TIME)).toBe(NO_ZONES)
    const poseurSansCamp = { 1: 't0' }
    expect(zonePresenceAt(sceneOf([sensor()], poseurSansCamp), at(5, 5), TIME)).toBe(NO_ZONES)
    const joueurSansCamp = { 2: 't1' }
    expect(zonePresenceAt(sceneOf([sensor()], joueurSansCamp), at(5, 5), TIME)).toBe(NO_ZONES)
  })

  it('au-delà des 15 s officielles, le capteur n’existe plus — quand le champ, lui, tient', () => {
    // Fin officielle du capteur : t0 + 150 frames = 160. À l'image 170 : éteint.
    const p = zonePresenceAt(sceneOf([sensor(), field()], CAMPS), at(5, 5, 170), TIME)
    expect(p.sensorSincePingMs).toBeNull()
    expect(p.repair).toBe(true)
  })

  it('deux capteurs superposés : l’horloge du plus fraîchement pingé', () => {
    // t0 = 10 : 400 ms après son ping ; t0 = 48 : 200 ms après le sien. Le plus frais gagne.
    const deux = [sensor(), sensor({ t0: 48 })]
    const p = zonePresenceAt(sceneOf(deux, CAMPS), at(5, 5), TIME)
    expect(p.sensorSincePingMs).toBe(200)
  })
})

describe('zonePresenceAt — hors de tout', () => {
  it('rend la valeur partagée NO_ZONES, jamais un objet neuf', () => {
    expect(zonePresenceAt(sceneOf([field()]), at(0, 0), TIME)).toBe(NO_ZONES)
    expect(zonePresenceAt(sceneOf([]), at(5, 5), TIME)).toBe(NO_ZONES)
  })
})
