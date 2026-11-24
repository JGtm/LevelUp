/**
 * Les socles d'ÉQUIPEMENT, lus sur les documents réels produits par Go (schéma 127).
 *
 * Les familles testées sont ÉNUMÉRÉES depuis les fixtures (`weaponPads[].weapon` qui n'est pas un
 * identifiant d'arme hexadécimal), jamais d'une liste écrite : une famille qui entrerait dans un
 * film futur et resterait sans nom, sans encre d'équipement ou sans son d'équipement ferait
 * rougir ce fichier.
 *
 * Défaut d'origine : l'infobulle d'un socle de grappin, de répulseur ou de propulseur affichait
 * sa clé brute, et tout socle d'équipement autre qu'un bonus jouait le son d'une arme et prenait
 * l'encre des râteliers.
 */
import { describe, expect, it } from 'vitest'

import { REPLAY_TEXT, type ReplayLocale } from '../i18n/i18n'
import { PLACEMENT_RENDER } from './equipmentPlacementsLayer'
import { EQUIPMENT_PAD_SPAWN_SOUND_STEM, PAD_SPAWN_MAX_PAR_MATCH, padSpawnSoundEvents } from '../sound/padSpawnSound'
import { goFixtureEntries, loadGoFixture } from '../test/goFixtures'
import { testReplayDoc } from '../test/testDoc'
import {
  PAD_EQUIPMENT_FAMILIES,
  PAD_PLACED_EQUIPMENT_FAMILIES,
  padFamilyOf,
  padIsEquipment,
} from '../model/weaponPadFamilies'
import { padNameFor } from './useReplayWeaponPads'

const HEX = /^0x[0-9a-f]{8}$/i
const LOCALES = Object.keys(REPLAY_TEXT) as ReplayLocale[]

/** Les socles d'équipement de chaque fixture : [film, document, familles]. */
function equipmentPadsByFixture() {
  return goFixtureEntries().map((entry) => {
    const doc = testReplayDoc(loadGoFixture(entry).doc)
    const families = [...new Set(doc.weaponPads.map((p) => p.weapon).filter((w) => !HEX.test(w)))]
    return { film: entry.film, doc, families }
  })
}

describe('socles d équipement des documents réels', () => {
  const fixtures = equipmentPadsByFixture()

  it('les fixtures portent des socles de grappin, de répulseur et de propulseur (le test n est pas vide)', () => {
    const all = new Set(fixtures.flatMap((f) => f.families))
    for (const f of ['grapple', 'repulsor', 'thruster']) expect(all.has(f), f).toBe(true)
  })

  it('chaque famille est reconnue comme équipement, de l encre des bonus', () => {
    for (const { film, families } of fixtures) {
      for (const w of families) {
        expect(padIsEquipment(w), `${film} ${w}`).toBe(true)
        expect(padFamilyOf(w, undefined), `${film} ${w}`).toBe('powerup')
      }
    }
  })

  it('chaque famille a un nom qui n est pas sa clé, dans chaque langue', () => {
    expect(LOCALES.length).toBeGreaterThanOrEqual(2)
    for (const { film, doc, families } of fixtures) {
      for (const w of families) {
        for (const locale of LOCALES) {
          const name = padNameFor(w, doc.weaponLabels, REPLAY_TEXT[locale], locale, doc.abilityLabels)
          expect(name.length, `${film} ${w} ${locale}`).toBeGreaterThan(0)
          expect(name, `${film} ${w} ${locale}`).not.toBe(w)
        }
      }
    }
  })

  it('chaque apparition d un socle d équipement joue le son d équipement, jamais celui d une arme', () => {
    for (const { film, doc, families } of fixtures) {
      const pads = doc.weaponPads.filter((p) => families.includes(p.weapon))
      const attendu = pads.reduce((n, p) => n + (p.spawns?.length ?? 0), 0)
      const equipement = padSpawnSoundEvents({ ...doc, weaponPads: pads })
      expect(equipement.length, film).toBe(Math.min(attendu, PAD_SPAWN_MAX_PAR_MATCH))
      for (const e of equipement) expect(e.stem, film).toBe(EQUIPMENT_PAD_SPAWN_SOUND_STEM)
    }
  })
})

describe('table écrite des familles d équipement de socle', () => {
  it('elle égale les familles de rendu du client plus les bonus (aucune famille ne manque, aucune n est en trop)', () => {
    const ecrites = [...PAD_PLACED_EQUIPMENT_FAMILIES, ...Object.keys(PAD_EQUIPMENT_FAMILIES)].sort()
    const attendues = [...Object.keys(PLACEMENT_RENDER), ...Object.keys(PAD_EQUIPMENT_FAMILIES)].sort()
    expect(ecrites).toEqual(attendues)
  })

  it('un identifiant d arme n est jamais un équipement', () => {
    expect(padIsEquipment('0x0A1992BC')).toBe(false)
    expect(padFamilyOf('0x0A1992BC', 'hinf_br75')).toBe('classic')
  })
})
