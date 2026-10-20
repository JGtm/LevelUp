/// <reference types="node" />
/**
 * tourelleVisee.mesure.test.ts — LOT 5.5, POINT 2 : LA VISÉE DU TOURELLEUR EST-ELLE LA DIRECTION
 * DU TIR DE TOURELLE ?
 *
 * ## Pourquoi cette mesure existe
 *
 * Un tir dont le montage est de classe `tourelle` sort SANS direction (`vehicleShotPlacement`
 * rend `angle: null`, donc `drawMuzzleFlash` tombe sur la bouffée RONDE). La règle était juste au
 * lot 5.2a.5 — la visée d'un tourelleur n'est pas le cap du châssis — mais le lot 5.5.1 a fermé
 * la voie « grammaire » par deux négatifs mesurés : `ti=40 i41`/`i42` (`vehicle-seats-override`)
 * sont une API de SCRIPT et ne sont déclarés par AUCUN record (0 sur 253 863, deux films), pas
 * plus que la visée de la tourelle automatique (`i31`) ni l'`i21` du véhicule lui-même.
 *
 * IL RESTE UNE SEULE DIRECTION MESURÉE : la visée du BIPÈDE assis au siège qui sert cette arme.
 * Le document la publie déjà — `vehicles[].rides[].aim`, un épisode par occupant, avec son
 * `seat` — et le rendu ne l'emploie aujourd'hui que pour le CÔNE, jamais pour le tir.
 *
 * ## Ce que la mesure rend, et avec quels oracles
 *
 * COUVERTURE : part des tirs de tourelle pour lesquels un occupant d'un siège donné a une lecture
 * de visée EN VIGUEUR à l'image du tir (même fenêtre de maintien que le cône,
 * `VEHICLE_AIM_HOLD_FRAMES`). Sans couverture, le port ne dessinerait rien de plus.
 *
 * UTILITÉ : écart angulaire entre cette visée et le cap auquel le CHÂSSIS est dessiné
 * (`vehicleChassisHeadingAt`). Un écart nul dirait que la tourelle suit le corps, donc que le
 * champ n'apporte rien ; un écart franc dit que l'éclair part aujourd'hui dans une direction
 * qu'aucune mesure ne soutient.
 *
 * VENTILATION PAR SIÈGE, et ce n'est pas un détail : le tourelleur du Warthog est un PASSAGER,
 * alors que le canon du Scorpion est servi par le CONDUCTEUR. La mesure ne suppose donc aucun
 * siège — elle les compte tous, arme par arme, et c'est elle qui désigne l'opérateur.
 *
 * ## Régime
 *
 *     TOURELLE_MESURE=1 npx vitest run src/features/match-replay/model/tourelleVisee.mesure.test.ts
 *     TOURELLE_MESURE_DIR=<abs>/data/cache/replays/halo_infinite   (défaut : celui du dépôt)
 *
 * Lecture seule sur des documents DÉJÀ CUITS : aucun décodage de film, aucune base ouverte.
 * Sans `TOURELLE_MESURE`, la suite est ignorée — même porte que `ReplayTeams.perf.test.tsx`.
 * DEPUIS LE SCHÉMA 69 (lot M4a) les montages viennent du registre `vehicleWeapons`, RÉSOLU À LA
 * REQUÊTE : un artefact lu sur disque n'en porte pas. L'instrument y repose donc la table telle
 * que l'API la sert, relue dans le registre versionné du titre (`test/vehicleWeaponsTitre.ts`) —
 * sans elle, chaque tir tombait dans « sans montage » et la mesure était muette sans le dire
 * (revue adverse du lot M4a, F7). Un document qui porte déjà la table (lu depuis l'API) la garde.
 */
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

import type { ReplayDocument, ReplayVehicleRide } from '@/lib/api/types'

import { racineDuDepot } from '../test/featureFiles'
import { testReplayDoc } from '../test/testDoc'
import { registreDuTitre } from '../test/vehicleWeaponsTitre'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import type { ReplayVehicleTrackReady } from '../../../lib/replay/replayNormalize'
import { buildShotFx, type VehicleShotSource } from './shotFx'
import { vehicleActiveRides } from './vehiclesLayer'
import {
  VEHICLE_AIM_HOLD_FRAMES,
  vehicleChassisHeadingAt,
  vehicleRideAimReading,
} from './vehiclesAim'
import { vehicleShotOrigin } from './vehicleWeaponMounts'
import { vehicleWeaponMountOf } from './vehicleWeaponRegistry'

/** Les témoins : un BTB à Warthogs, et deux films où un Scorpion tire. */
const TEMOINS = ['4f77afc1', '8a485699', '0a44c6cc'] as const

function dossierTemoins(): string {
  return (
    process.env.TOURELLE_MESURE_DIR ??
    join(racineDuDepot(), 'data', 'cache', 'replays', 'halo_infinite')
  )
}

function charger(court: string): ReplayDocumentReady | null {
  const p = join(dossierTemoins(), `${court}.json`)
  if (!existsSync(p)) return null
  const brut = JSON.parse(readFileSync(p, 'utf8')) as Partial<ReplayDocument>
  return testReplayDoc({ ...brut, vehicleWeapons: brut.vehicleWeapons ?? registreDuTitre().armes })
}

/** Écart angulaire absolu entre deux caps en degrés, replié sur [0, 180]. */
function ecartDeg(a: number, b: number): number {
  const d = Math.abs(((a - b) % 360 + 360) % 360)
  return d > 180 ? 360 - d : d
}

function quantile(v: readonly number[], q: number): number {
  if (v.length === 0) return Number.NaN
  const t = [...v].sort((x, y) => x - y)
  return t[Math.min(t.length - 1, Math.max(0, Math.ceil(q * t.length) - 1))]
}

interface LigneSiege {
  /** Nombre de tirs de tourelle où ce siège est OCCUPÉ à l'image du tir. */
  occupe: number
  /** Parmi ceux-là, combien portent une lecture de visée EN VIGUEUR. */
  avecVisee: number
  /** Écarts visée-châssis, en degrés, pour les tirs à visée en vigueur. */
  ecarts: number[]
}

interface Mesure {
  tirs: number
  tirsVehicule: number
  montageConnu: number
  montageTourelle: number
  montageFixe: number
  sansMontage: number
  /**
   * AVANT / APRÈS, SUR LA MÊME POPULATION ET LE MÊME CODE. Les deux décomptes passent par la
   * chaîne de rendu RÉELLE (`buildShotFx` puis `vehicleShotOrigin`) ; l'AVANT n'est obtenu qu'en
   * forçant `shooterHeadingDeg` à `null`, c'est-à-dire l'état d'avant ce lot. Recopier la règle
   * ici en donnerait une seconde version, qui divergerait du rendu sans que rien ne le voie.
   *
   * `populationRendu` est le dénominateur commun : les tirs que le rendu traite comme des tirs de
   * VÉHICULE (donc hors mêlée, et hors arme de JOUEUR tirée depuis un siège de passager — celle-là
   * garde son propre regard, décision du lot 5.2a.5).
   */
  populationRendu: number
  avantAvecDirection: number
  apresAvecDirection: number
  avantTourelleAvecDirection: number
  apresTourelleAvecDirection: number
  /** Par tag d'arme de classe `tourelle`, puis par siège. */
  parArme: Map<string, Map<number | 'inconnu', LigneSiege>>
  /** LE TIREUR LUI-MÊME : `Shot.slot` apparié à `VehicleRide.slot`, par classe de montage. */
  tireur: LigneTireur
  tireurTourelle: LigneTireur
}

/**
 * LigneTireur — L'APPARIEMENT QUI NE DEVINE AUCUN SIÈGE. Un tir porte le slot de son TIREUR
 * (`Shot.slot`, celui qui alimente déjà `heldReading`) et un épisode d'occupation porte le slot de
 * son OCCUPANT (`VehicleRide.slot`). Les apparier rend l'épisode du tireur — donc SA visée —
 * quel que soit son siège, et sans supposer qui sert quelle arme (le tourelleur du Warthog est un
 * passager ; le canon du Scorpion est servi par le conducteur).
 */
interface LigneTireur {
  /** Tirs de véhicule dont le slot tireur trouve un épisode couvrant SUR CE VÉHICULE. */
  episodeTrouve: number
  /** Parmi eux, ceux dont l'épisode porte une lecture de visée EN VIGUEUR. */
  avecVisee: number
  /** Écarts visée du tireur - cap du châssis, en degrés. */
  ecarts: number[]
  /** Sièges observés pour l'épisode DU TIREUR (qui sert l'arme, mesuré). */
  sieges: Map<number | 'inconnu', number>
  /** Tirs de tourelle dont le slot tireur ne trouve AUCUN épisode sur ce véhicule. */
  sansEpisode: number
}

function mesurer(doc: ReplayDocumentReady): Mesure {
  const m: Mesure = {
    tirs: doc.shots.length,
    tirsVehicule: 0,
    montageConnu: 0,
    montageTourelle: 0,
    montageFixe: 0,
    sansMontage: 0,
    populationRendu: 0,
    avantAvecDirection: 0,
    apresAvecDirection: 0,
    avantTourelleAvecDirection: 0,
    apresTourelleAvecDirection: 0,
    parArme: new Map(),
    tireur: ligneTireurVide(),
    tireurTourelle: ligneTireurVide(),
  }
  mesurerRenduReel(doc, m)
  for (const s of doc.shots) {
    if (s.v === undefined) continue
    m.tirsVehicule++
    const track = doc.vehicles.find((v) => v.slot === s.v)
    if (track) ventilerTireur(m.tireur, track, s.slot, s.t)
    const mount = vehicleWeaponMountOf(doc, s.w)
    if (!mount) {
      m.sansMontage++
      continue
    }
    m.montageConnu++
    if (mount.classe !== 'tourelle') {
      m.montageFixe++
      continue
    }
    m.montageTourelle++
    if (!track) continue
    ventilerTireur(m.tireurTourelle, track, s.slot, s.t)
    ventilerParSiege(m, s.w ?? 'sans-tag', track, s.t)
  }
  return m
}

function ligneTireurVide(): LigneTireur {
  return { episodeTrouve: 0, avecVisee: 0, ecarts: [], sieges: new Map(), sansEpisode: 0 }
}

/**
 * mesurerRenduReel passe par `buildShotFx` puis `vehicleShotOrigin` — LA CHAÎNE QUE LE CANEVAS
 * APPELLE — et compte les tirs de véhicule qui en sortent AVEC un angle. `sizeOf: undefined`
 * exerce la branche « sprite pas encore chargé », dont la règle de DIRECTION est la même que celle
 * du cas nominal (foyer unique `vehicleMountAngle`) ; seul le décalage y manque, et il ne pèse pas
 * sur ce décompte.
 */
function mesurerRenduReel(doc: ReplayDocumentReady, m: Mesure): void {
  for (const e of buildShotFx(doc, VEHICLE_AIM_HOLD_FRAMES)) {
    const src = e.vehicleShot
    if (!src) continue
    m.populationRendu++
    const tourelle = src.mount?.classe === 'tourelle'
    const apres = angleDeRendu(e.h, src)
    const avant = angleDeRendu(e.h, { ...src, shooterHeadingDeg: null })
    if (avant !== null) {
      m.avantAvecDirection++
      if (tourelle) m.avantTourelleAvecDirection++
    }
    if (apres !== null) {
      m.apresAvecDirection++
      if (tourelle) m.apresTourelleAvecDirection++
    }
  }
}

/** angleDeRendu appelle la chaîne du canevas et ne garde que sa DIRECTION. */
function angleDeRendu(h: number | null, src: VehicleShotSource): number | null {
  return vehicleShotOrigin({
    h,
    vehicleShot: src,
    center: { x: 0, y: 0 },
    // `sizeOf: undefined` exerce la branche « sprite pas encore chargé », dont la règle de
    // DIRECTION est la même que celle du cas nominal (foyer unique `vehicleMountAngle`) : seul le
    // décalage y manque, et il ne pèse pas sur ce décompte.
    sizeOf: undefined,
    k: 1,
    scalePxPerM: 20,
  }).angle
}

/** ventilerTireur apparie le slot du TIREUR a un episode d occupation de CE vehicule. */
function ventilerTireur(
  l: LigneTireur,
  track: ReplayVehicleTrackReady,
  slotTireur: number,
  frame: number,
): void {
  const sien = vehicleActiveRides(track, frame).find((r) => r.slot === slotTireur)
  if (!sien) {
    l.sansEpisode++
    return
  }
  l.episodeTrouve++
  const cle: number | 'inconnu' = sien.seat ?? 'inconnu'
  l.sieges.set(cle, (l.sieges.get(cle) ?? 0) + 1)
  const lu = lectureDeVisee(sien, frame)
  if (lu === null) return
  l.avecVisee++
  l.ecarts.push(ecartDeg(lu, vehicleChassisHeadingAt(track, frame)))
}

function ventilerParSiege(
  m: Mesure,
  tag: string,
  track: ReplayVehicleTrackReady,
  frame: number,
): void {
  let parSiege = m.parArme.get(tag)
  if (!parSiege) {
    parSiege = new Map()
    m.parArme.set(tag, parSiege)
  }
  const cap = vehicleChassisHeadingAt(track, frame)
  for (const ride of vehicleActiveRides(track, frame)) {
    const cle: number | 'inconnu' = ride.seat ?? 'inconnu'
    let ligne = parSiege.get(cle)
    if (!ligne) {
      ligne = { occupe: 0, avecVisee: 0, ecarts: [] }
      parSiege.set(cle, ligne)
    }
    ligne.occupe++
    const lu = lectureDeVisee(ride, frame)
    if (lu === null) continue
    ligne.avecVisee++
    ligne.ecarts.push(ecartDeg(lu, cap))
  }
}

function lectureDeVisee(ride: ReplayVehicleRide, frame: number): number | null {
  const lu = vehicleRideAimReading(ride, frame)
  return lu?.h ?? null
}

/**
 * sortie — UN SEUL FOYER D'ÉCRITURE, et c'est `process.stdout` et non `console.log` : vitest
 * intercepte la console et n'en rend rien en `run`, alors que le flux passe (même geste que
 * `ReplayTeams.perf.test.tsx`, seul autre instrument de mesure de la feature).
 */
function sortie(ligne: string): void {
  process.stdout.write(`${ligne}\n`)
}

/** ligneTireur ecrit la ligne d appariement par slot d une population de tirs. */
function ligneTireur(
  titre: string,
  t: LigneTireur,
  total: number,
  pc: (n: number, d: number) => string,
): void {
  if (total === 0) return
  const sieges = [...t.sieges.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([s, n]) => `${s}:${n}`)
    .join(' ')
  sortie(
    `    ${titre} — TIREUR APPARIE PAR SLOT : episode trouve ${t.episodeTrouve} / ${total} ` +
      `(${pc(t.episodeTrouve, total)}), sans episode ${t.sansEpisode} ; ` +
      `visee en vigueur ${t.avecVisee} (${pc(t.avecVisee, total)}) | ` +
      `ecart au chassis med ${quantile(t.ecarts, 0.5).toFixed(1)} q75 ${quantile(t.ecarts, 0.75).toFixed(1)} ` +
      `q90 ${quantile(t.ecarts, 0.9).toFixed(1)} deg | sieges du tireur ${sieges || '-'}`,
  )
}

function journaliser(court: string, m: Mesure): void {
  const pc = (n: number, d: number) => (d === 0 ? '-' : `${((100 * n) / d).toFixed(1)} %`)
  sortie(
    `\n=== ${court} : ${m.tirs} tirs, ${m.tirsVehicule} en vehicule ` +
      `(montage connu ${m.montageConnu}, dont tourelle ${m.montageTourelle} / fixe ${m.montageFixe} ; ` +
      `sans montage ${m.sansMontage})`,
  )
  sortie(
    `    CHAINE DE RENDU REELLE, population ${m.populationRendu} tirs de vehicule` +
      ` — AVEC DIRECTION : AVANT ${m.avantAvecDirection} (${pc(m.avantAvecDirection, m.populationRendu)})` +
      ` -> APRES ${m.apresAvecDirection} (${pc(m.apresAvecDirection, m.populationRendu)})`,
  )
  sortie(
    `        dont TIRS DE TOURELLE : AVANT ${m.avantTourelleAvecDirection} / ${m.montageTourelle}` +
      ` -> APRES ${m.apresTourelleAvecDirection} / ${m.montageTourelle} ` +
      `(${pc(m.apresTourelleAvecDirection, m.montageTourelle)})`,
  )
  ligneTireur('TOUS TIRS DE VEHICULE', m.tireur, m.tirsVehicule, pc)
  ligneTireur('TIRS DE TOURELLE SEULS', m.tireurTourelle, m.montageTourelle, pc)
  for (const [tag, parSiege] of m.parArme) {
    sortie(`    --- arme ${tag} (classe tourelle)`)
    const sieges = [...parSiege.keys()].sort((a, b) =>
      a === 'inconnu' ? 1 : b === 'inconnu' ? -1 : a - b,
    )
    for (const s of sieges) {
      const l = parSiege.get(s)
      if (!l) continue
      const med = quantile(l.ecarts, 0.5)
      const q75 = quantile(l.ecarts, 0.75)
      const q90 = quantile(l.ecarts, 0.9)
      sortie(
        `        siege ${String(s).padStart(7)} : occupe ${l.occupe}, visee en vigueur ${l.avecVisee} ` +
          `(${pc(l.avecVisee, l.occupe)}) | ecart au chassis med ${med.toFixed(1)} q75 ${q75.toFixed(1)} q90 ${q90.toFixed(1)} deg`,
      )
    }
  }
}

describe.skipIf(!process.env.TOURELLE_MESURE)(
  'lot 5.5 — la visee du tourelleur, mesuree sur documents cuits (TOURELLE_MESURE)',
  () => {
    it('ventile les tirs de tourelle par siege, avec couverture et ecart au chassis', () => {
      let vus = 0
      for (const court of TEMOINS) {
        const doc = charger(court)
        if (!doc) {
          sortie(`=== ${court} : ABSENT de ${dossierTemoins()}`)
          continue
        }
        vus++
        journaliser(court, mesurer(doc))
      }
      expect(vus).toBeGreaterThan(0)
    })
  },
)
