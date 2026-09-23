/// <reference types="node" />
/**
 * Garde-rail (retours du rejeu 2026-09-23, lot L1.5) : LES TABLES D'ARMES DE VÉHICULE SONT CLÉES
 * PAR DES TAGS OBSERVÉS.
 *
 * LE DÉFAUT QU'IL FERME. Les trois tables client (style, son, montage) étaient clées par les tags
 * `weap` du lot V3F — lus dans le module du jeu, jamais confrontés à un document. Mesure du
 * 2026-09-23 sur les 111 documents du parc : 7 des 11 clés n'apparaissaient dans AUCUN document, et
 * 3 tags réellement publiés en étaient absents (`0BB6976B` lance-grenades du Falcon, 52 tirs
 * dont 51 en véhicule ; `49E40D17` canon du Scorpion, 13 dont 13 ; `850902EF`, 25 dont 15, non
 * identifié). Le Rockethog (`C7D50912`, 132 tirs dont 127 en véhicule) était muet : il ne sonnait
 * que pour une famille `rockethog` qu'aucun châssis ne porte.
 *
 * LES COMPTES : « N tirs dont M en véhicule » = `shots` / `shotsWithVehicle` de la fixture (tous les
 * tirs publiés sous le tag / ceux qui portent `v`). Le seuil ci-dessous porte sur `shots`.
 *
 * LA RÈGLE, DANS LES DEUX SENS :
 *  1. toute clé de table est OBSERVÉE (fixture datée `vehicle_weapon_tags_observed.json`, sortie de
 *     l'instrument `test/fixtures/sweep_vehicle_weapon_tags.mjs`) OU inscrite dans `ATTENDUS_NON_OBSERVES` — les armes à TIR CONTINU, que
 *     le décodeur ne lit pas encore (lot M4b), et leur voisine ;
 *  2. tout tag observé au moins `SEUIL_OBSERVE` fois a une entrée de STYLE, ou une ligne de
 *     `INCONNUS` motivée ; et un son, ou une ligne de `SILENCES_DECIDES`.
 *
 * DATE DE RETRAIT : `ATTENDUS_NON_OBSERVES` disparaît au lot M4a (registre `vehicle_weapons.toml`
 * côté Go, clé = tag observé, garde-rail Go équivalent). Critère : les trois tables client
 * supprimées.
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

import { VEHICLE_SHOT_SOUND_STEMS } from '../sound/vehicleShotSound'
import { featureRoot } from '../test/featureFiles'
import { VEHICLE_SHOT_FX } from './vehicleShotFx'
import { VEHICLE_WEAPON_MOUNTS, vehicleWeapTag } from './vehicleWeaponMounts'

interface TagObserve {
  tag: string
  shots: number
  docs: string[]
}

const FIXTURE = join(featureRoot(), 'test', 'fixtures', 'vehicle_weapon_tags_observed.json')
/** L'instrument VERSIONNÉ qui écrit la fixture (revue RR-L1-06) : `node <lui> <parc> [--check]`. */
const INSTRUMENT = 'sweep_vehicle_weapon_tags.mjs'
const OBSERVES: TagObserve[] = JSON.parse(readFileSync(FIXTURE, 'utf8')).tags

/** Un tag observé moins souvent peut attendre sa ligne (le Wasp `11725DC4` en a 4, et l'a). */
const SEUIL_OBSERVE = 5

/**
 * ATTENDUS_NON_OBSERVES — 2026-09-23, source `V3F_TIRS_COVENANT_2026-09-02.md` §3-4. Retrait : lot M4a.
 * Toutes des armes à TIR CONTINU (son `lsnd` en boucle) : le décodeur des tirs ne lit que le
 * record de tête des paquets `0xD2` et n'en a jamais publié une seule (0 sur 6 au parc, lot M4b) —
 * plus la bombe de la Banshee, dont le tag `0000aa69` n'est pas encore rapproché de `850902EF`.
 */
const ATTENDUS_NON_OBSERVES: ReadonlyMap<string, string> = new Map([
  [vehicleWeapTag('00015435'), 'Ghost — canons à plasma jumeaux, tir continu (M4b)'],
  [vehicleWeapTag('0000aa68'), 'Banshee M1 — canons à plasma, tir continu (M4b)'],
  [vehicleWeapTag('0000aa69'), 'Banshee M2 — bombe à combustible, jamais vue sous ce tag (850902EF ?)'],
  [vehicleWeapTag('b40e9618'), 'Chopper — canons avant, tir continu (M4b)'],
  [vehicleWeapTag('00015cd3'), 'Falcon — LMG de porte, tir continu (M4b)'],
  [vehicleWeapTag('d3c407ed'), 'Wasp — son en boucle, 600/min (M4b)'],
])

/** INCONNUS — tags observés que rien n'identifie encore : ni style, ni son, ni montage. */
const INCONNUS: ReadonlyMap<string, string> = new Map([
  [
    vehicleWeapTag('850902ef'),
    '25 tirs dont 15 en véhicule, un seul document (5676a9ba), porteurs Banshee 11 / Warthog 4 : aucune arme ne '
      + 'le nomme (candidat : bombe de la Banshee 0000aa69) — à identifier au lot M4a',
  ],
])

/** SILENCES_DECIDES — tags observés qui ont un style mais AUCUNE reconstruction sonore. */
const SILENCES_DECIDES: ReadonlyMap<string, string> = new Map([
  [
    vehicleWeapTag('0bb6976b'),
    'Falcon lance-grenades : aucune reconstruction (manifeste V3, planches, static/sounds) — '
      + 'silence décidé le 2026-09-23, question posée à l’utilisateur',
  ],
])

const observe = (tag: string) => OBSERVES.find((o) => o.tag === tag)

describe('garde-rail : tables d’armes de véhicule clées par des tags OBSERVÉS', () => {
  it('la fixture est datée et porte le parc entier', () => {
    const brut = JSON.parse(readFileSync(FIXTURE, 'utf8'))
    expect(brut.generatedAt).toBe('2026-09-23')
    expect(brut.corpus.documents).toBe(111)
    expect(OBSERVES.length).toBeGreaterThan(0)
  })

  it('la fixture se régénère depuis un instrument versionné, posé à côté d’elle', () => {
    const brut = JSON.parse(readFileSync(FIXTURE, 'utf8'))
    expect(brut.source, 'la fixture doit citer son instrument').toContain(INSTRUMENT)
    expect(existsSync(join(featureRoot(), 'test', 'fixtures', INSTRUMENT)), INSTRUMENT).toBe(true)
  })

  const tables: Array<[string, Iterable<string>]> = [
    ['style (vehicleShotFx)', VEHICLE_SHOT_FX.keys()],
    ['son (vehicleShotSound)', VEHICLE_SHOT_SOUND_STEMS.keys()],
    ['montage (vehicleWeaponMounts)', VEHICLE_WEAPON_MOUNTS.keys()],
  ]
  for (const [nom, cles] of tables) {
    it(`toute clé de la table ${nom} est observée ou attendue`, () => {
      for (const tag of cles) {
        const ok = observe(tag) !== undefined || ATTENDUS_NON_OBSERVES.has(tag)
        expect(ok, `${nom} : ${tag} n'est ni observé ni attendu`).toBe(true)
      }
    })
  }

  it('un tag attendu n’est pas observé (sinon, le retirer de la liste)', () => {
    for (const tag of ATTENDUS_NON_OBSERVES.keys()) expect(observe(tag), tag).toBeUndefined()
  })

  it(`tout tag observé au moins ${SEUIL_OBSERVE} fois a un style, ou une ligne « inconnu »`, () => {
    for (const o of OBSERVES.filter((x) => x.shots >= SEUIL_OBSERVE)) {
      const ok = VEHICLE_SHOT_FX.has(o.tag) || INCONNUS.has(o.tag)
      expect(ok, `${o.tag} (${o.shots} tirs) sans style ni ligne inconnue`).toBe(true)
    }
  })

  it(`tout tag observé au moins ${SEUIL_OBSERVE} fois a un son, un silence décidé, ou une ligne « inconnu »`, () => {
    for (const o of OBSERVES.filter((x) => x.shots >= SEUIL_OBSERVE)) {
      const ok = VEHICLE_SHOT_SOUND_STEMS.has(o.tag) || SILENCES_DECIDES.has(o.tag) || INCONNUS.has(o.tag)
      expect(ok, `${o.tag} (${o.shots} tirs) sans son ni silence décidé`).toBe(true)
    }
  })

  it('une ligne « inconnu » ou « silence décidé » ne double pas une entrée réelle', () => {
    for (const tag of INCONNUS.keys()) {
      expect(VEHICLE_SHOT_FX.has(tag), tag).toBe(false)
      expect(VEHICLE_SHOT_SOUND_STEMS.has(tag), tag).toBe(false)
    }
    for (const tag of SILENCES_DECIDES.keys()) expect(VEHICLE_SHOT_SOUND_STEMS.has(tag), tag).toBe(false)
  })
})
