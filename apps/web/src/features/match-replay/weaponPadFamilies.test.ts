/// <reference types="node" />
// @vitest-environment node
/**
 * Tests — weaponPadFamilies : LA LISTE DES ARMES DE PUISSANCE, nommée une par une.
 *
 * CE QUE CE FICHIER VERROUILLE :
 *  - les HUIT familles que l'utilisateur a désignées le 2026-08-18 sont dans la liste, et
 *    aucune n'a disparu au fil d'une réécriture ;
 *  - les clés d'ARME existent réellement au registre du titre (`weapon_names.toml`) — une
 *    clé mal orthographiée ne ferait RIEN, silencieusement : le socle resterait petit et
 *    personne ne le verrait ;
 *  - les deux POWER-UPS sont des familles que le valideur Go admet — même piège, autre table ;
 *  - la règle de taille : puissance = grande, tout le reste (y compris l'inconnu) = petite.
 */
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { racineDuDepot } from './test/featureFiles'
import {
  PAD_EQUIPMENT_FAMILIES,
  POWER_PAD_KEYS,
  POWER_PAD_WEAPON_KEYS,
  padEquipmentFamilyOf,
  padScaleOf,
} from './weaponPadFamilies'

const REPO = racineDuDepot()
const WEAPON_NAMES = resolve(REPO, 'config/titles/halo_infinite/mappings/weapon_names.toml')
const REPLAY_LABELS = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')
const HUD_DIR = resolve(REPO, 'static/weapons-assets/halo_infinite/hud')
const GO_LOADER = resolve(
  REPO,
  'apps/go-api/internal/games/mappings/loader_replay_labels_equipment.go',
)

describe('POWER_PAD_KEYS — la liste que l’utilisateur a nommée', () => {
  it('porte les huit familles désignées le 18/08, sans en perdre une', () => {
    // Verbatim du bilan : « sniper, épée, marteau, roquettes, Skewer, Cindershot,
    // surbouclier, camo ». Les roquettes comptent leurs deux SPNKr.
    for (const key of [
      'hinf_s7_sniper',
      'hinf_energy_sword',
      'hinf_gravity_hammer',
      'hinf_m41_spnkr',
      'hinf_fuel_rod_spnkr',
      'hinf_skewer',
      'hinf_cindershot',
      'powerup_overshield',
      'powerup_camo',
    ]) {
      expect(POWER_PAD_KEYS, `${key} doit rester dans la liste`).toContain(key)
    }
  })

  it('chaque clé d’ARME existe au registre du titre (une faute de frappe ne dirait rien)', () => {
    const toml = readFileSync(WEAPON_NAMES, 'utf8')
    for (const key of POWER_PAD_WEAPON_KEYS) {
      expect(toml, `${key} absente de weapon_names.toml`).toContain(`${key} `)
    }
  })

  it('chaque POWER-UP est une famille que le valideur Go admet', () => {
    const src = readFileSync(GO_LOADER, 'utf8')
    for (const key of POWER_PAD_KEYS.filter((k) => k.startsWith('powerup_'))) {
      expect(src, `${key} n’est pas admise par le valideur Go`).toContain(`"${key}":`)
    }
  })
})

describe('padScaleOf — la règle de taille', () => {
  it('une arme de puissance est GRANDE', () => {
    expect(padScaleOf('hinf_s7_sniper')).toBe('power')
    expect(padScaleOf('hinf_gravity_hammer')).toBe('power')
    expect(padScaleOf('powerup_overshield')).toBe('power')
  })

  it('une arme classique est PETITE', () => {
    expect(padScaleOf('hinf_br75')).toBe('classic')
    expect(padScaleOf('hinf_cqs48_bulldog')).toBe('classic')
    expect(padScaleOf('hinf_vestige_carbine')).toBe('classic')
  })

  it('ce qu’on ne sait pas nommer reste PETIT — jamais promu par défaut', () => {
    expect(padScaleOf(undefined)).toBe('classic')
    expect(padScaleOf(null)).toBe('classic')
    expect(padScaleOf('')).toBe('classic')
    expect(padScaleOf('cle_d_un_titre_futur')).toBe('classic')
  })
})

describe('PAD_EQUIPMENT_FAMILIES — les familles NON-ARME d’un socle', () => {
  const cles = Object.keys(PAD_EQUIPMENT_FAMILIES)

  it('chaque famille est DÉJÀ dans la règle de taille — jamais une seconde règle', () => {
    for (const key of cles) {
      expect(POWER_PAD_KEYS, `${key} hors de POWER_PAD_KEYS : le socle resterait petit`).toContain(
        key,
      )
    }
  })

  it('chaque famille est admise par le valideur Go — même vocabulaire des deux côtés', () => {
    const src = readFileSync(GO_LOADER, 'utf8')
    for (const key of cles) {
      expect(src, `${key} n’est pas une famille d’équipement du manifeste`).toContain(`"${key}":`)
    }
  })

  it('chaque vignette de HUD est LIVRÉE, et nommée par le manifeste du titre', () => {
    // Le manifeste ne déclare aucune icône sur ses `[[equipment_objects]]` — il nomme en
    // revanche ces MÊMES masques pour les capacités de même nom (`icon = "hud/Overshield"`).
    // Les deux vérifications comptent : un fichier absent donnerait un cadre vide en
    // production, un stem inventé une image qui ne vient pas du jeu.
    const manifeste = readFileSync(REPLAY_LABELS, 'utf8')
    for (const [key, famille] of Object.entries(PAD_EQUIPMENT_FAMILIES)) {
      expect(
        existsSync(resolve(HUD_DIR, `${famille.icon}.png`)),
        `${key} : vignette hud/${famille.icon}.png absente des assets livrés`,
      ).toBe(true)
      expect(manifeste, `${key} : hud/${famille.icon} n’est nommée nulle part au manifeste`).toContain(
        `"hud/${famille.icon}"`,
      )
    }
  })

  it('c’est une TABLE, pas un test de préfixe : une famille inconnue n’en est pas une', () => {
    expect(padEquipmentFamilyOf('powerup_overshield')).toBe('powerup_overshield')
    expect(padEquipmentFamilyOf('powerup_inconnu')).toBeNull()
    expect(padEquipmentFamilyOf('0x0A1992BC')).toBeNull()
    expect(padEquipmentFamilyOf('wall')).toBeNull()
    // Une clé héritée du prototype n’est pas une famille (un socle peut porter n’importe quoi).
    expect(padEquipmentFamilyOf('constructor')).toBeNull()
    expect(padEquipmentFamilyOf('toString')).toBeNull()
  })
})
