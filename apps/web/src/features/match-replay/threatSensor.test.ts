/**
 * Tests — threatSensor (le capteur de menaces : ses chiffres officiels, son ping, sa révélation).
 *
 * Ce que ces tests verrouillent :
 *  - LES CHIFFRES OFFICIELS, à la valeur exacte de la source (Halo Waypoint, « Sandbox Overview
 *    Season 4 ») : rayon 4,25, ping 1,8 s, révélation 0,75 s, durée 15 s. Un rendu peut se
 *    discuter, ces quatre nombres non — ils sont cités, pas choisis ;
 *  - LE PING COMME FONCTION DU TEMPS : même âge, même onde, donc un retour en arrière rejoue
 *    l'image ; aucune onde entre deux pings ni sous mouvement réduit ;
 *  - LA RÉVÉLATION, cas par cas : un adversaire dans le rayon AU PING est révélé 0,75 s ; un
 *    coéquipier non ; hors rayon non ; mort non ; sans poseur mesuré, aucune.
 */
import { describe, expect, it } from 'vitest'

import type { ReplayEquipmentPlacement, ReplayPoint } from '@/lib/api/types'

import type { ReplayTrackReady } from './replayNormalize'
import {
  REVEAL_ALPHA,
  SENSOR_DURATION_MS,
  SENSOR_PING_MS,
  SENSOR_PINGS_DECLARED,
  SENSOR_RADIUS_M,
  SENSOR_REVEAL_MS,
  SENSOR_SWEEP_MS,
  revealAlpha,
  sensorPing,
  sensorPingAgeMs,
  sensorReveals,
} from './threatSensor'

/** 100 ms par image : l'échelle réelle des artefacts du corpus (frameIntervalMs = 100). */
const FRAME_MS = 100

/** Le capteur du test : posé à l'origine, image 10, poseur = slot 1 (camp « t0 »). */
function sensor(over: Partial<ReplayEquipmentPlacement> = {}): ReplayEquipmentPlacement {
  return { t0: 10, t1: 300, x: 0, y: 0, family: 'sensor', id: '0xcapteur', owner: 1, ...over }
}

/** Une vie IMMOBILE : deux échantillons à la même position, sur toute la fenêtre. */
function life(slot: number, x: number, y: number, start = 0, end = 300): ReplayTrackReady {
  const points: ReplayPoint[] = [
    { t: start, x, y },
    { t: end, x, y },
  ]
  return { slot, team: -1, startFrame: start, endFrame: end, points }
}

/** Camps : slot 1 et 2 dans « t0 », slot 3 dans « t1 », slot 9 sans camp connu. */
const SIDES: Record<number, string | null> = { 1: 't0', 2: 't0', 3: 't1', 9: null }
const sideOfSlot = (slot: number) => SIDES[slot] ?? null

describe('les chiffres officiels du capteur', () => {
  it('sont exactement ceux de la source citée', () => {
    expect(SENSOR_RADIUS_M).toBe(4.25)
    expect(SENSOR_PING_MS).toBe(1_800)
    expect(SENSOR_REVEAL_MS).toBe(750)
    expect(SENSOR_DURATION_MS).toBe(15_000)
  })

  it('une vie officielle porte NEUF pings (0 s, 1,8 s, ... 14,4 s) — la borne de coût', () => {
    expect(SENSOR_PINGS_DECLARED).toBe(9)
    expect((SENSOR_PINGS_DECLARED - 1) * SENSOR_PING_MS).toBeLessThanOrEqual(SENSOR_DURATION_MS)
    expect(SENSOR_PINGS_DECLARED * SENSOR_PING_MS).toBeGreaterThan(SENSOR_DURATION_MS)
  })

  it('l’onde est brève devant la période : le capteur pinge, il ne gonfle pas en continu', () => {
    expect(SENSOR_SWEEP_MS).toBeLessThan(SENSOR_PING_MS / 4)
  })
})

describe('sensorPingAgeMs — le temps depuis le dernier ping', () => {
  it('le premier ping part à l’âge zéro : le capteur balaie dès qu’il est posé', () => {
    expect(sensorPingAgeMs(0)).toBe(0)
  })

  it('remet le compteur à zéro à chaque période, sans dérive', () => {
    expect(sensorPingAgeMs(SENSOR_PING_MS)).toBe(0)
    expect(sensorPingAgeMs(SENSOR_PING_MS * 3)).toBe(0)
    expect(sensorPingAgeMs(SENSOR_PING_MS * 3 + 250)).toBe(250)
  })

  it('un âge négatif (pose pas encore là) rend zéro, jamais une phase inversée', () => {
    expect(sensorPingAgeMs(-500)).toBe(0)
  })
})

describe('sensorPing — l’onde, fonction du temps', () => {
  it('le même âge rend toujours la même onde (un retour en arrière rejoue l’image)', () => {
    const a = sensorPing(120, false)
    const b = sensorPing(120 + SENSOR_PING_MS * 4, false)
    expect(a).not.toBeNull()
    expect(b).toEqual(a)
  })

  it('part du centre au ping et atteint le rayon au bout de sa course', () => {
    expect(sensorPing(0, false)?.reach).toBeCloseTo(0, 10)
    const late = sensorPing(SENSOR_SWEEP_MS - 1, false)
    expect(late?.reach).toBeGreaterThan(0.99)
    expect(late?.reach).toBeLessThan(1)
  })

  it('s’efface en s’ouvrant : l’opacité décroît sur toute la course', () => {
    const alphas = [0, 100, 200, 300, 399].map((ms) => sensorPing(ms, false)?.alpha ?? 0)
    for (let i = 1; i < alphas.length; i++) expect(alphas[i]).toBeLessThan(alphas[i - 1])
  })

  it('entre deux pings, aucune onde : le capteur retombe sur son disque', () => {
    expect(sensorPing(SENSOR_SWEEP_MS, false)).toBeNull()
    expect(sensorPing(SENSOR_PING_MS - 1, false)).toBeNull()
  })

  it('mouvement réduit : aucune onde, jamais', () => {
    for (const ms of [0, 100, 900, 1_800, 3_600]) expect(sensorPing(ms, true)).toBeNull()
  })
})

describe('revealAlpha — la marque s’éteint à l’heure dite', () => {
  it('franche au ping, nulle à 0,75 s', () => {
    expect(revealAlpha(0, false)).toBeCloseTo(REVEAL_ALPHA, 10)
    expect(revealAlpha(SENSOR_REVEAL_MS, false)).toBeCloseTo(0, 10)
  })

  it('mouvement réduit : l’information reste, l’estompage part', () => {
    expect(revealAlpha(0, true)).toBe(REVEAL_ALPHA)
    expect(revealAlpha(SENSOR_REVEAL_MS - 1, true)).toBe(REVEAL_ALPHA)
  })
})

describe('sensorReveals — qui le ping révèle, et qui il ne révèle pas', () => {
  const time = { frame: 10, frameMs: FRAME_MS }
  /** L'adversaire (slot 3, camp t1) à 2 m du capteur : dans le rayon de 4,25 m. */
  const foeInside = life(3, 2, 0)

  it('un adversaire dans le rayon au ping est révélé', () => {
    const out = sensorReveals([sensor()], { lives: [foeInside], sideOfSlot }, time)
    expect(out).toHaveLength(1)
    expect(out[0]).toMatchObject({ slot: 3, owner: 1, sinceMs: 0, x: 2, y: 0 })
  })

  it('un coéquipier du poseur ne l’est pas : le capteur ne montre pas son propre camp', () => {
    const mate = life(2, 2, 0)
    expect(sensorReveals([sensor()], { lives: [mate], sideOfSlot }, time)).toEqual([])
  })

  it('un adversaire hors du rayon ne l’est pas — la portée est celle de la source', () => {
    const far = life(3, SENSOR_RADIUS_M + 0.01, 0)
    expect(sensorReveals([sensor()], { lives: [far], sideOfSlot }, time)).toEqual([])
    // Et juste à l'intérieur, il l'est : la borne est bien là où la source la met.
    const edge = life(3, SENSOR_RADIUS_M - 0.01, 0)
    expect(sensorReveals([sensor()], { lives: [edge], sideOfSlot }, time)).toHaveLength(1)
  })

  it('une vie qui ne couvre pas l’image n’est pas révélée : un mort ne se détecte pas', () => {
    const dead = life(3, 2, 0, 0, 5) // vie close à l'image 5, le ping est à l'image 10
    expect(sensorReveals([sensor()], { lives: [dead], sideOfSlot }, time)).toEqual([])
  })

  it('sans poseur mesuré, AUCUNE révélation : on ignore le camp du capteur', () => {
    const orphan = sensor({ owner: -1 })
    expect(sensorReveals([orphan], { lives: [foeInside], sideOfSlot }, time)).toEqual([])
  })

  it('sans camp connu (ni pour le poseur, ni pour la cible), aucune révélation non plus', () => {
    // Poseur sans ligne de scoreboard.
    expect(sensorReveals([sensor({ owner: 9 })], { lives: [foeInside], sideOfSlot }, time)).toEqual([])
    // Cible sans ligne de scoreboard : on ne l'appelle pas « ennemie » pour autant.
    const unknown = life(9, 2, 0)
    expect(sensorReveals([sensor()], { lives: [unknown], sideOfSlot }, time)).toEqual([])
  })

  it('la marque dure 0,75 s après le ping, puis le capteur attend le suivant', () => {
    const scene = { lives: [foeInside], sideOfSlot }
    const at = (ms: number) => sensorReveals([sensor()], scene, { frame: 10 + ms / FRAME_MS, frameMs: FRAME_MS })
    expect(at(0)).toHaveLength(1)
    expect(at(SENSOR_REVEAL_MS)).toHaveLength(1)
    expect(at(SENSOR_REVEAL_MS + FRAME_MS)).toEqual([])
    // Au ping suivant, elle revient — et son âge est reparti de zéro.
    expect(at(SENSOR_PING_MS)).toHaveLength(1)
    expect(at(SENSOR_PING_MS)[0].sinceMs).toBe(0)
  })

  it('L’APPARTENANCE SE MESURE AU PING, pas à l’image courante', () => {
    // Une vie qui SORT du rayon : dans la zone au ping (image 10), à 20 m 0,5 s plus tard.
    const leaving: ReplayTrackReady = {
      slot: 3,
      team: -1,
      startFrame: 0,
      endFrame: 300,
      points: [
        { t: 10, x: 1, y: 0 },
        { t: 15, x: 20, y: 0 },
      ],
    }
    const out = sensorReveals([sensor()], { lives: [leaving], sideOfSlot }, { frame: 15, frameMs: FRAME_MS })
    // Elle reste marquée (elle était là au ping) ET la marque a suivi jusqu'à sa position.
    expect(out).toHaveLength(1)
    expect(out[0].x).toBeCloseTo(20, 6)
    expect(out[0].sinceMs).toBe(500)

    // Le symétrique : une vie qui ENTRE après le ping n'est pas marquée.
    const entering: ReplayTrackReady = {
      slot: 3,
      team: -1,
      startFrame: 0,
      endFrame: 300,
      points: [
        { t: 10, x: 20, y: 0 },
        { t: 15, x: 1, y: 0 },
      ],
    }
    expect(sensorReveals([sensor()], { lives: [entering], sideOfSlot }, { frame: 15, frameMs: FRAME_MS })).toEqual([])
  })

  it('deux capteurs sur la même vie ne posent qu’UNE marque, la plus fraîche', () => {
    // Le second est posé 5 images plus tard : son ping est donc plus récent à l'image 12.
    const a = sensor({ id: '0xa', t0: 10, owner: 1 })
    const b = sensor({ id: '0xb', t0: 12, owner: 2 })
    const out = sensorReveals([a, b], { lives: [foeInside], sideOfSlot }, { frame: 12, frameMs: FRAME_MS })
    expect(out).toHaveLength(1)
    expect(out[0].owner).toBe(2)
    expect(out[0].sinceMs).toBe(0)
  })

  it('la marque suit le joueur : elle est posée à sa position COURANTE', () => {
    const moving: ReplayTrackReady = {
      slot: 3,
      team: -1,
      startFrame: 0,
      endFrame: 300,
      points: [
        { t: 10, x: 1, y: 1 },
        { t: 13, x: 1, y: 4 },
      ],
    }
    const out = sensorReveals([sensor()], { lives: [moving], sideOfSlot }, { frame: 13, frameMs: FRAME_MS })
    expect(out).toHaveLength(1)
    expect(out[0].y).toBeCloseTo(4, 6)
  })

  // MULTI-MANCHE : un slot de biped est réattribué entre manches. Le camp doit se lire à
  // l'image qui identifie le bon joueur — le POSEUR à sa pose, la CIBLE au ping — jamais à
  // l'image courante, où le slot peut appartenir à quelqu'un d'un autre camp. Les deux tests
  // ci-dessous le prouvent par CONTRE-ÉPREUVE : le double `sideOfSlot` renvoie, à l'image
  // courante, un camp qui FERAIT ÉCHOUER la révélation ; elle n'a lieu que si la bonne image
  // a été employée.
  it('le POSEUR se lit à sa POSE (t0), jamais à l’image courante', () => {
    // Slot 1 : poseur de camp « t0 » en manche 0 ; le slot revient à un joueur de camp « t1 »
    // plus tard. La cible (slot 3) est de camp « t1 ». Résolu à t0=10 → t0, adversaire de la
    // cible → révélation. Résolu à l'image courante 30 → t1 = la cible → aucune.
    const sideAt = (slot: number, frame: number) =>
      slot === 1 ? (frame < 20 ? 't0' : 't1') : slot === 3 ? 't1' : null
    const out = sensorReveals([sensor()], { lives: [foeInside], sideOfSlot: sideAt }, { frame: 30, frameMs: FRAME_MS })
    expect(out).toHaveLength(1)
    // L'image de la pose voyage avec la révélation : le calque colore la marque par le poseur.
    expect(out[0].ownerFrame).toBe(10)
  })

  it('la CIBLE se lit au PING, jamais à l’image courante', () => {
    // Slot 3 : adversaire (« t1 ») au ping (image 28) ; le slot passe au camp du poseur (« t0 »)
    // à l'image courante 30. Résolue au ping → t1, adversaire → révélation ; résolue à 30 → t0 =
    // poseur → aucune.
    const sideAt = (slot: number, frame: number) =>
      slot === 1 ? 't0' : slot === 3 ? (frame < 29 ? 't1' : 't0') : null
    const out = sensorReveals([sensor()], { lives: [foeInside], sideOfSlot: sideAt }, { frame: 30, frameMs: FRAME_MS })
    expect(out).toHaveLength(1)
  })
})
