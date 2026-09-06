/// <reference types="node" />
// @vitest-environment node
/**
 * Tests — gameChangers : LA LISTE VOTÉE du repli des bilans, et ses garde-rails.
 *
 * CE QUE CE FICHIER VERROUILLE (patron `weaponPadFamilies.test.ts`) :
 *  - chaque FAMILLE élue existe au manifeste du titre (`replay_labels.toml`) — une famille mal
 *    orthographiée ne ferait RIEN, silencieusement : sa colonne resterait repliée ;
 *  - chaque `weapon_key` élu existe au registre du titre (`weapon_names.toml`) — même piège ;
 *  - le PONT D5 tient dans les deux sens : ses clés sont des familles de socle ÉLUES, ses
 *    valeurs sont des familles d'ÉPISODE que la mesure connaît (`EPISODE_FAMILIES`) ;
 *  - la COÏNCIDENCE cindershot avec `POWER_PAD_KEYS` (promotion utilisateur du 05/09) est
 *    un FAIT daté : ce test la fige pour qu'une réécriture ne la défasse pas en douce ;
 *  - les prédicats : élu = en avant, tout le reste (y compris l'inconnu et la clé absente,
 *    décision D6) = replié.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import { EPISODE_FAMILIES } from './equipmentUsageLogic'
import {
  EPISODE_FAMILY_OF_POWERUP,
  GAME_CHANGER_EQUIPMENT_FAMILIES,
  GAME_CHANGER_WEAPON_KEYS,
  isGameChangerFamily,
  isGameChangerWeaponKey,
} from './gameChangers'
import { PAD_EQUIPMENT_FAMILIES, POWER_PAD_KEYS } from './weaponPadFamilies'

import { racineDuDepot } from './test/featureFiles'

const REPO = racineDuDepot()
const WEAPON_NAMES = resolve(REPO, 'config/titles/halo_infinite/mappings/weapon_names.toml')
const REPLAY_LABELS = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')

describe('GAME_CHANGER_EQUIPMENT_FAMILIES — les cinq familles élues', () => {
  it('porte exactement les cinq élues du vote du 05/09, sans en perdre une', () => {
    expect([...GAME_CHANGER_EQUIPMENT_FAMILIES].sort()).toEqual(
      ['powerup_camo', 'powerup_overshield', 'sensor', 'shroud_screen', 'threat_seeker'].sort(),
    )
  })

  it('chaque famille existe au manifeste du titre (une faute de frappe ne dirait rien)', () => {
    const manifeste = readFileSync(REPLAY_LABELS, 'utf8')
    for (const family of GAME_CHANGER_EQUIPMENT_FAMILIES) {
      expect(
        new RegExp(`family\\s*=\\s*"${family}"`).test(manifeste),
        `${family} absente de replay_labels.toml`,
      ).toBe(true)
    }
  })

  it('les repliés du vote ne se sont pas glissés dans la liste', () => {
    for (const family of [
      'grapple',
      'thruster',
      'repulsor',
      'wall',
      'repair_field',
      'translocator_beacon',
    ]) {
      expect(GAME_CHANGER_EQUIPMENT_FAMILIES, `${family} a été voté replié`).not.toContain(family)
    }
  })
})

describe('GAME_CHANGER_WEAPON_KEYS — les armes élues, et la coïncidence datée', () => {
  it('chaque clé existe au registre du titre (une faute de frappe ne dirait rien)', () => {
    const toml = readFileSync(WEAPON_NAMES, 'utf8')
    for (const key of GAME_CHANGER_WEAPON_KEYS) {
      expect(toml, `${key} absente de weapon_names.toml`).toContain(`${key} `)
    }
  })

  it('fige la COÏNCIDENCE cindershot : grand sur la carte ET en avant dans les bilans (D1)', () => {
    // POWER_PAD_KEYS (échelle des socles du rejeu 2D) garde le cindershot ; le vote du 05/09
    // l'avait replié, puis l'utilisateur l'a PROMU le jour même (« le cindershot peut être un
    // game changer »). Les deux affirmations comptent : si l'une tombe, c'est une décision
    // utilisateur à consigner, pas une réécriture silencieuse.
    expect(POWER_PAD_KEYS).toContain('hinf_cindershot')
    expect(GAME_CHANGER_WEAPON_KEYS).toContain('hinf_cindershot')
  })

  it('les deux SPNKr voyagent ensemble : la variante est élue avec son socle', () => {
    expect(GAME_CHANGER_WEAPON_KEYS).toContain('hinf_m41_spnkr')
    expect(GAME_CHANGER_WEAPON_KEYS).toContain('hinf_fuel_rod_spnkr')
  })
})

describe('EPISODE_FAMILY_OF_POWERUP — le pont D5, dans les deux sens', () => {
  it('chaque clé du pont est une famille de SOCLE de bonus connue ET élue', () => {
    for (const powerup of Object.keys(EPISODE_FAMILY_OF_POWERUP)) {
      expect(Object.keys(PAD_EQUIPMENT_FAMILIES), `${powerup} n'est pas un socle de bonus`)
        .toContain(powerup)
      expect(GAME_CHANGER_EQUIPMENT_FAMILIES, `${powerup} n'est pas élu`).toContain(powerup)
    }
  })

  it('chaque valeur du pont est une famille d’ÉPISODE que la mesure connaît', () => {
    for (const episode of Object.values(EPISODE_FAMILY_OF_POWERUP)) {
      expect(
        EPISODE_FAMILIES as readonly string[],
        `${episode} n'est pas une famille d'épisode mesurée`,
      ).toContain(episode)
    }
  })
})

describe('isGameChangerFamily — le prédicat, dans les deux vocabulaires', () => {
  it('une famille élue est EN AVANT, socle comme épisode (le pont D5 fait foi)', () => {
    expect(isGameChangerFamily('sensor')).toBe(true)
    expect(isGameChangerFamily('powerup_camo')).toBe(true)
    // Vocabulaire d'ÉPISODE : atteint UNIQUEMENT par le pont — retirer le pont fait tomber
    // ces deux assertions (mutation du plan, G1.3).
    expect(isGameChangerFamily('camo')).toBe(true)
    expect(isGameChangerFamily('overshield')).toBe(true)
  })

  it('une famille votée repliée, ou inconnue, reste REPLIÉE — jamais promue', () => {
    expect(isGameChangerFamily('grapple')).toBe(false)
    expect(isGameChangerFamily('wall')).toBe(false)
    expect(isGameChangerFamily('repair_field')).toBe(false)
    expect(isGameChangerFamily('other')).toBe(false)
    expect(isGameChangerFamily('famille_d_un_titre_futur')).toBe(false)
  })
})

describe('isGameChangerWeaponKey — le prédicat d’arme (D6)', () => {
  it('une arme élue est EN AVANT', () => {
    expect(isGameChangerWeaponKey('hinf_s7_sniper')).toBe(true)
    expect(isGameChangerWeaponKey('hinf_gravity_hammer')).toBe(true)
    expect(isGameChangerWeaponKey('hinf_cindershot')).toBe(true)
  })

  it('une arme non élue, ou une clé ABSENTE (label sans key), reste REPLIÉE', () => {
    expect(isGameChangerWeaponKey('hinf_vk78_commando')).toBe(false)
    expect(isGameChangerWeaponKey('hinf_br75')).toBe(false)
    expect(isGameChangerWeaponKey(undefined)).toBe(false)
    expect(isGameChangerWeaponKey(null)).toBe(false)
    expect(isGameChangerWeaponKey('')).toBe(false)
  })
})
