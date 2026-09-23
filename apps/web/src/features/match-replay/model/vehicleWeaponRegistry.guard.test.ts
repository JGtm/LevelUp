/// <reference types="node" />
/**
 * Garde-rail (retours du rejeu 2026-09-23, lot M4a) : LE CLIENT NE CONNAÎT AUCUN TAG D'ARME DE
 * VÉHICULE.
 *
 * LE DÉFAUT QU'IL FERME. Trois tables client (style, son, montage) étaient clées par des tags `weap`
 * recopiés sous le gabarit `0x<tag>00000000` par `vehicleWeapTag(...)` : 7 des 11 clés
 * n'apparaissaient dans aucun document du parc. Le registre du TITRE les remplace
 * (`config/titles/{slug}/mappings/vehicle_weapons.toml`, garde-rail Go sur une fixture datée des
 * tags observés), publié dans le document et lu par UN fichier. Une table recréée ici, même d'une
 * ligne, re-divergerait du registre sans que rien ne le voie.
 *
 * LA RÈGLE, SUR LES SOURCES DE LA FEATURE (hors tests) :
 *  1. aucun littéral au gabarit d'une arme de véhicule (`0x` + 8 chiffres hex + `00000000`) ;
 *  2. aucune trace de l'ancien assembleur de gabarit (`vehicleWeapTag`) ;
 *  3. `vehicleWeapons` n'est lu que par le lecteur du registre ;
 *  4. aucun TAG du registre du titre, sous quelque forme que ce soit (tag nu, casse indifférente,
 *     concaténation comprise dès qu'un tag entier est écrit) — le gabarit seul laissait passer une
 *     table clée par `'c7d50912'` (revue adverse du lot M4a, F8). Les tags sont RELUS dans le
 *     registre versionné (`test/vehicleWeaponsTitre.ts`) : un tag ajouté au registre est interdit
 *     ici sans autre geste.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

import {
  cheminCourt,
  fichiersSous,
  nomDe,
  racineWeb,
  sourcesDeLaFeature,
  tousLesFichiers,
} from '../test/featureFiles'
import { registreDuTitre } from '../test/vehicleWeaponsTitre'

const LECTEUR = 'vehicleWeaponRegistry.ts'
const GABARIT_ARME_DE_VEHICULE = /0x[0-9A-Fa-f]{8}00000000/
const ASSEMBLEUR = /vehicleWeapTag/

describe('garde-rail : aucun tag d arme de véhicule côté client', () => {
  it('aucun littéral au gabarit `0x<tag>00000000` dans les sources du rejeu', () => {
    const fautifs = sourcesDeLaFeature().filter((f) => GABARIT_ARME_DE_VEHICULE.test(readFileSync(f, 'utf8')))
    expect(fautifs.map(cheminCourt)).toEqual([])
  })

  it('l ancien assembleur de gabarit n existe plus nulle part (tests compris)', () => {
    const moi = 'vehicleWeaponRegistry.guard.test.ts'
    const fautifs = tousLesFichiers()
      .filter((f) => nomDe(f) !== moi)
      .filter((f) => ASSEMBLEUR.test(readFileSync(f, 'utf8')))
    expect(fautifs.map(cheminCourt)).toEqual([])
  })

  it('seul le lecteur du registre lit `vehicleWeapons`', () => {
    const fautifs = sourcesDeLaFeature()
      .filter((f) => nomDe(f) !== LECTEUR)
      .filter((f) => /\.vehicleWeapons\b/.test(readFileSync(f, 'utf8')))
    expect(fautifs.map(cheminCourt)).toEqual([])
  })

  it('contre-test : le lecteur porte bien la lecture du registre', () => {
    const lecteur = sourcesDeLaFeature().find((f) => nomDe(f) === LECTEUR)
    expect(lecteur, LECTEUR).toBeDefined()
    expect(readFileSync(lecteur!, 'utf8')).toMatch(/\.vehicleWeapons\?\.\[/)
  })

  it('aucun tag du registre du titre dans les sources du rejeu (casse indifférente)', () => {
    const { tags } = registreDuTitre()
    const motif = new RegExp(tags.join('|'), 'i')
    const sources = [
      ...sourcesDeLaFeature(),
      ...fichiersSous(join(racineWeb(), 'src', 'lib', 'replay')).filter((f) => !/\.test\.(ts|tsx)$/.test(f)),
    ]
    const fautifs = sources.filter((f) => motif.test(readFileSync(f, 'utf8')))
    expect(fautifs.map((f) => nomDe(f))).toEqual([])
  })

  it('contre-test : le registre du titre est lu, et il porte des tags', () => {
    const { tags, armes: table } = registreDuTitre()
    const armes = Object.keys(table).map((cle) => cle.slice(2, 10))
    expect(armes.length).toBeGreaterThan(0)
    expect(tags).toEqual(expect.arrayContaining(armes))
    // Le motif prend un tag NU, en minuscules — la forme que le gabarit seul laissait passer.
    expect(new RegExp(tags.join('|'), 'i').test(`const t = '${tags[0].toLowerCase()}'`)).toBe(true)
  })
})
