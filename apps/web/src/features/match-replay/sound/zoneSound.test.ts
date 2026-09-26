import { describe, expect, it } from 'vitest'

import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  ZONE_SOUND_STEMS,
  ZONE_TICK_PERIOD_MS,
  zoneSoundEvents,
} from './zoneSound'

/**
 * zoneSound.test.ts — les trois règles des sons d'état de zone, une par une.
 *
 * CE QUE CES TESTS ÉPINGLENT EN PRIORITÉ, ce sont les SILENCES. Un son d'objectif qui part au
 * mauvais moment ou du mauvais camp annonce un gain quand on perd une base : la règle du
 * chantier est que le rejeu se TAIT plutôt que de deviner. Trois silences sont donc testés
 * nommément — camp allié non résolu, rampe de jauge sans changement de propriétaire, et tics
 * en Roi de la colline.
 */

/** Un document minimal : seuls le pas de grille et les états de zone comptent ici. */
function doc(zoneStates: unknown[]): ReplayDocumentReady {
  return { frameIntervalMs: 100, zoneStates } as unknown as ReplayDocumentReady
}

/**
 * Le même document, PLUS l'horloge de score du jeu (lot 5.8.6) : `scoreTimeline.teams[].total`
 * est l'escalier cumulatif d'un camp, ses `t` sont des FRAMES.
 */
function docAvecScore(
  zoneStates: unknown[],
  teams: { teamId: number; total: { t: number; v: number }[] }[],
): ReplayDocumentReady {
  return {
    frameIntervalMs: 100,
    zoneStates,
    scoreTimeline: { teams },
  } as unknown as ReplayDocumentReady
}

/** Trois zones tenues par le même camp de `t0` à `t1` : une DOMINATION. */
function domination(t0: number, t1: number, owner: number) {
  return [{ spans: [span(t0, t1, owner)] }, { spans: [span(t0, t1, owner)] }, { spans: [span(t0, t1, owner)] }]
}

/** Une zone tenue par `owner` de `t0` à `t1`. */
function span(t0: number, t1: number, owner: number | null, active = false) {
  return { t0, t1, owner, active }
}

describe('capture en cours — la jauge dit QUAND, le propriétaire d arrivée dit QUI', () => {
  it('une rampe suivie d une prise sonne au DÉBUT de la rampe, du camp qui prend', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [
          { t: 10, v: 0.1 },
          { t: 20, v: 0.4 },
          { t: 30, v: 0.9 },
        ],
      },
    ])
    const evs = zoneSoundEvents(d, 1)
    expect(evs).toEqual([{ ms: 1000, stem: ZONE_SOUND_STEMS.capturing.ally }])
  })

  it('le camp est celui du NOUVEAU propriétaire, pas de l ancien', () => {
    const d = doc([
      {
        spans: [span(0, 49, 1), span(50, 200, 0)],
        gauge: [
          { t: 10, v: 0.2 },
          { t: 30, v: 0.8 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 1000, stem: ZONE_SOUND_STEMS.capturing.enemy },
    ])
  })

  it('une rampe qui retombe sans changer le propriétaire sonne la CONTESTATION', () => {
    // Définition de l utilisateur : on prend une zone adverse, un adversaire entre et la
    // conteste. Le son part à l instant où la jauge CESSE de monter, pas au début de la rampe.
    const d = doc([
      {
        spans: [span(0, 200, 0)],
        gauge: [
          { t: 10, v: 0.2 },
          { t: 30, v: 0.8 },
          { t: 40, v: 0 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([{ ms: 3000, stem: ZONE_SOUND_STEMS.contested }])
  })

  it('la contestation sonne SANS camp allié résolu : le jeu n en a qu un son', () => {
    const d = doc([
      {
        spans: [span(0, 200, 0)],
        gauge: [
          { t: 10, v: 0.2 },
          { t: 30, v: 0.8 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, null)).toEqual([{ ms: 3000, stem: ZONE_SOUND_STEMS.contested }])
  })

  it('la CAPTURE EN COURS est muette sans camp allié résolu : on ne devine jamais un camp', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [
          { t: 10, v: 0.1 },
          { t: 30, v: 0.9 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, null)).toEqual([])
  })

  it('un seul point de jauge n est pas une rampe', () => {
    const d = doc([{ spans: [span(0, 49, null), span(50, 200, 1)], gauge: [{ t: 10, v: 0.9 }] }])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  /**
   * LE ZÉRO DE FERMETURE N'EST PAS LE DÉBUT DE LA RAMPE SUIVANTE (correction du 2026-09-20).
   *
   * TÉMOIN `396cfc92`, ZONE A, recopié tel quel du cache : la rampe publiée court de 73 600 à
   * 103 900 ms, mais son premier point vaut ZÉRO — c'est le retour à zéro que le Go pose à la
   * fin de la capture PRÉCÉDENTE (le camp adverse vient d aboutir à 73 500 ms). La poussée
   * réelle commence à 97 000 ms. Le son partait donc 23,4 s trop tôt, et il annonçait « une
   * capture démarre » à l instant où une capture se terminait.
   */
  it('témoin 396cfc92 : la rampe 73 600 -> 103 900 ms sonne à 97 000, pas à 73 600', () => {
    const d = doc([
      {
        spans: [span(735, 1038, 1), span(1039, 1284, 0)],
        gauge: [
          { t: 736, v: 0 },
          { t: 970, v: 0.012 },
          { t: 972, v: 0.04 },
          { t: 1038, v: 0.983 },
          { t: 1039, v: 0.998 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 97_000, stem: ZONE_SOUND_STEMS.capturing.enemy },
    ])
  })

  it('une rampe qui ne commence PAS par un zéro garde son premier point', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [
          { t: 10, v: 0.1 },
          { t: 20, v: 0.4 },
          { t: 30, v: 0.9 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([{ ms: 1000, stem: ZONE_SOUND_STEMS.capturing.ally }])
  })

  /**
   * LE CAMP MESURÉ (schéma 64). `proprietaireApres` rend `null` quand l intervalle qui s ouvre
   * après la rampe est NEUTRE — le son se taisait alors complètement, alors que la rampe a bien
   * été poussée par quelqu un. Le document le nomme désormais.
   */
  it('rampe suivie d un intervalle NEUTRE : le camp vient du document, et le son sonne', () => {
    const d = doc([
      {
        spans: [span(0, 49, 1), span(50, 200, null)],
        gauge: [
          { t: 10, v: 0 },
          { t: 20, v: 0.4 },
          { t: 49, v: 0.99 },
        ],
        gaugeRamps: [{ t0: 10, t1: 49, capturingTeam: 0 }],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 2000, stem: ZONE_SOUND_STEMS.capturing.enemy },
    ])
  })

  it('le camp MESURÉ prime sur le propriétaire d arrivée', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [{ t: 10, v: 0.1 }, { t: 30, v: 0.9 }],
        gaugeRamps: [{ t0: 10, t1: 30, capturingTeam: 0 }],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 1000, stem: ZONE_SOUND_STEMS.capturing.enemy },
    ])
  })

  it('rampe SANS camp mesuré : on retombe sur le propriétaire d arrivée (artefact <= 63)', () => {
    const d = doc([
      {
        spans: [span(0, 49, null), span(50, 200, 1)],
        gauge: [{ t: 10, v: 0.1 }, { t: 30, v: 0.9 }],
        gaugeRamps: [{ t0: 10, t1: 30 }],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 1000, stem: ZONE_SOUND_STEMS.capturing.ally },
    ])
  })

  it('la CONTESTATION garde l instant du sommet : la correction ne déplace que le début', () => {
    const d = doc([
      {
        spans: [span(0, 600, 0)],
        gauge: [
          { t: 100, v: 0 },
          { t: 300, v: 0.2 },
          { t: 400, v: 0.8 },
          { t: 410, v: 0 },
        ],
      },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([{ ms: 40_000, stem: ZONE_SOUND_STEMS.contested }])
  })
})

describe('tic de score — un par seconde tant qu un camp tient TOUTES les zones', () => {
  it('trois zones au même camp : un tic par seconde, du bon côté', () => {
    const d = doc([
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 1)] },
    ])
    const evs = zoneSoundEvents(d, 1)
    // 0 à 30 frames à 100 ms = 3 s : les tics de 0, 1000, 2000 et 3000 ms.
    expect(evs.map((e) => e.ms)).toEqual([0, ZONE_TICK_PERIOD_MS, 2000, 3000])
    expect(new Set(evs.map((e) => e.stem))).toEqual(new Set([ZONE_SOUND_STEMS.tick.ally]))
  })

  /**
   * LOT 5.8.6 (a) — L'INSTANT DU TIC VIENT DE L'HORLOGE DE SCORE DU JEU, PAS D'UNE CADENCE.
   *
   * Le tic était ancré sur le DÉBUT de la domination et battait la seconde : un tic de score qui
   * ne tombe pas sur un point est un tic qui mentait — le jeu ne marque pas toutes les secondes,
   * et la cadence dépend de la variante. `scoreTimeline.teams[].total` la porte.
   */
  it('l horloge de score du jeu donne les instants : un tic par MARCHE du camp qui domine', () => {
    const d = docAvecScore(domination(0, 100, 1), [
      // Escalier cumulatif : la marche est une montée de `v`, son `t` est une FRAME.
      { teamId: 1, total: [{ t: 0, v: 0 }, { t: 25, v: 1 }, { t: 63, v: 2 }, { t: 90, v: 3 }] },
    ])
    const evs = zoneSoundEvents(d, 1)
    // 2 500 / 6 300 / 9 000 ms : les trois marches, et RIEN entre elles.
    expect(evs.map((e) => e.ms)).toEqual([2_500, 6_300, 9_000])
    expect(new Set(evs.map((e) => e.stem))).toEqual(new Set([ZONE_SOUND_STEMS.tick.ally]))
  })

  it('le premier point d une série à zéro n est pas une marche : il décrit l état initial', () => {
    const d = docAvecScore(domination(0, 100, 1), [
      { teamId: 1, total: [{ t: 0, v: 0 }, { t: 50, v: 1 }] },
    ])
    expect(zoneSoundEvents(d, 1).map((e) => e.ms)).toEqual([5_000])
  })

  it('un point marqué HORS domination ne sonne pas : la fenêtre est celle de la domination', () => {
    const d = docAvecScore(domination(0, 50, 1), [
      { teamId: 1, total: [{ t: 20, v: 1 }, { t: 80, v: 2 }] },
    ])
    // La domination s'arrête à la frame 50 (5 000 ms) : la seconde marche est dehors.
    expect(zoneSoundEvents(d, 1).map((e) => e.ms)).toEqual([2_000])
  })

  it('une série SANS marche dans la fenêtre est une RÉPONSE : aucun tic, et aucun repli', () => {
    const d = docAvecScore(domination(0, 100, 1), [
      { teamId: 1, total: [{ t: 500, v: 1 }] },
    ])
    // Le camp dominait et n'a rien marqué : inventer une cadence le contredirait.
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('le calque de score qui ne couvre PAS ce camp laisse le repli synthétique répondre', () => {
    const d = docAvecScore(domination(0, 30, 1), [
      { teamId: 0, total: [{ t: 10, v: 1 }] },
    ])
    expect(zoneSoundEvents(d, 1).map((e) => e.ms)).toEqual([0, 1000, 2000, 3000])
  })

  /**
   * LOT 5.8.6 (a) — LE PLAFOND DUR NE TRONQUE PLUS LA FIN DE L'INTERVALLE.
   *
   * La boucle s'arrêtait au 180e tic : au-delà de trois minutes de domination continue, le son se
   * TAISAIT jusqu'à la fin — précisément quand la domination devient l'information la plus utile.
   */
  it('une domination plus longue que le budget ÉTIRE ses tics au lieu de s arrêter', () => {
    // 6 000 frames à 100 ms = 600 s. L'ancien plafond s'arrêtait à 180 s.
    const evs = zoneSoundEvents(doc(domination(0, 6_000, 1)), 1)
    expect(evs.length).toBeLessThanOrEqual(181)
    // LE DERNIER TIC ATTEINT LA FIN : c'est tout le défaut corrigé (il tombait à 179 000 ms).
    expect(evs[evs.length - 1].ms).toBeGreaterThan(590_000)
    // Et la cadence est régulière : l'écart entre deux tics vaut la période dérivée.
    expect(evs[1].ms - evs[0].ms).toBeCloseTo(600_000 / 180, 6)
  })

  it('sous le budget, la période reste EXACTEMENT la seconde demandée (rendu inchangé)', () => {
    const evs = zoneSoundEvents(doc(domination(0, 300, 1)), 1)
    expect(evs.map((e) => e.ms)).toEqual(
      Array.from({ length: 31 }, (_, i) => i * ZONE_TICK_PERIOD_MS),
    )
  })

  it('une seule zone au camp adverse suffit à faire taire les tics', () => {
    const d = doc([
      { spans: [span(0, 30, 1)] },
      { spans: [span(0, 30, 0)] },
      { spans: [span(0, 30, 1)] },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('une zone NEUTRE fait taire les tics', () => {
    const d = doc([{ spans: [span(0, 30, 1)] }, { spans: [span(0, 30, null)] }])
    expect(zoneSoundEvents(d, 1)).toEqual([])
  })

  it('AUCUN TIC en Roi de la colline — le marqueur `active` est la garde', () => {
    const d = doc([
      { spans: [span(0, 30, 1, true)] },
      { spans: [span(0, 30, 1, true)] },
    ])
    const tics: readonly string[] = [ZONE_SOUND_STEMS.tick.ally, ZONE_SOUND_STEMS.tick.enemy]
    expect(zoneSoundEvents(d, 1).filter((e) => tics.includes(e.stem))).toEqual([])
  })

  it('MUET sur une zone unique : une zone n est pas une domination', () => {
    expect(zoneSoundEvents(doc([{ spans: [span(0, 30, 1)] }]), 1)).toEqual([])
  })
})

describe('nouvelle colline — chaque déplacement, jamais le premier intervalle', () => {
  it('deux collines successives ne sonnent qu une fois', () => {
    const d = doc([
      { spans: [span(0, 100, null, true)] },
      { spans: [span(101, 200, null, true)] },
    ])
    expect(zoneSoundEvents(d, 1)).toEqual([
      { ms: 10100, stem: ZONE_SOUND_STEMS.newZone },
    ])
  })

  it('sonne même sans camp allié : le déplacement n affirme aucun camp', () => {
    const d = doc([
      { spans: [span(0, 100, null, true)] },
      { spans: [span(101, 200, null, true)] },
    ])
    expect(zoneSoundEvents(d, null)).toHaveLength(1)
  })

  it('une seule colline ne sonne pas — ce n est pas un déplacement', () => {
    expect(zoneSoundEvents(doc([{ spans: [span(0, 200, null, true)] }]), 1)).toEqual([])
  })

  /**
   * UNE COLLINE QUI CHANGE DE MAINS N EST PAS UNE NOUVELLE COLLINE (correction du 2026-09-20).
   * Le Go découpe une période de colline en sous-intervalles `active` CONTIGUS, un par
   * changement de propriétaire (`hillSpansOf`) : les sonner un par un faisait entendre le
   * déplacement à chaque reprise de la même colline.
   */
  it('une colline qui change trois fois de mains ne sonne AUCUN déplacement', () => {
    const d = doc([
      { spans: [span(0, 99, 1, true), span(100, 199, 0, true), span(200, 300, 1, true)] },
    ])
    const news = zoneSoundEvents(d, 1).filter((e) => e.stem === ZONE_SOUND_STEMS.newZone)
    expect(news).toEqual([])
  })

  it('la colline qui se DÉPLACE sur une autre zone sonne, changements de mains compris', () => {
    const d = doc([
      { spans: [span(0, 99, 1, true), span(100, 199, 0, true)] },
      { spans: [span(300, 399, 0, true), span(400, 499, 1, true)] },
    ])
    const news = zoneSoundEvents(d, 1).filter((e) => e.stem === ZONE_SOUND_STEMS.newZone)
    expect(news).toEqual([{ ms: 30_000, stem: ZONE_SOUND_STEMS.newZone }])
  })

  it('la MÊME zone reprise APRÈS un trou est un déplacement : la colline est revenue', () => {
    const d = doc([{ spans: [span(0, 99, 1, true), span(300, 399, 1, true)] }])
    const news = zoneSoundEvents(d, 1).filter((e) => e.stem === ZONE_SOUND_STEMS.newZone)
    expect(news).toEqual([{ ms: 30_000, stem: ZONE_SOUND_STEMS.newZone }])
  })
})

describe('cas dégénérés', () => {
  it('aucun état de zone : aucun son', () => {
    expect(zoneSoundEvents(doc([]), 1)).toEqual([])
  })

  it('jauge absente : aucune capture en cours, mais les tics restent possibles', () => {
    const d = doc([{ spans: [span(0, 10, 1)] }, { spans: [span(0, 10, 1)] }])
    const evs = zoneSoundEvents(d, 1)
    expect(evs.every((e) => e.stem === ZONE_SOUND_STEMS.tick.ally)).toBe(true)
    expect(evs.length).toBeGreaterThan(0)
  })
})

/**
 * SÉCURISATION DE LA COLLINE (2026-08-30) — la sirène de garde.
 *
 * CE QUE CES TESTS ÉPINGLENT, encore une fois, ce sont les SILENCES : la colline neutre, le
 * camp non résolu, et le transfert trop court. Le quatrième test tient la disjonction avec la
 * capture en cours — les deux règles ne peuvent pas sonner ensemble parce que la jauge n'existe
 * jamais sur une colline, et il vaut mieux le prouver que le supposer.
 */
describe('sécurisation de la colline — l intervalle `active` possédé', () => {
  it('sonne au DÉBUT de la garde, du camp qui tient la colline', () => {
    const d = doc([{ spans: [span(40, 400, 1, true)] }])
    const evts = zoneSoundEvents(d, 1).filter(
      (e) => e.stem === ZONE_SOUND_STEMS.securing.ally,
    )
    expect(evts).toEqual([{ ms: 4000, stem: ZONE_SOUND_STEMS.securing.ally }])
  })

  it('distingue les deux camps', () => {
    const d = doc([{ spans: [span(0, 400, 2, true)] }])
    const stems = zoneSoundEvents(d, 1).map((e) => e.stem)
    expect(stems).toContain(ZONE_SOUND_STEMS.securing.enemy)
    expect(stems).not.toContain(ZONE_SOUND_STEMS.securing.ally)
  })

  it('MUET sur une colline NEUTRE : personne ne sécurise', () => {
    const d = doc([{ spans: [span(0, 400, null, true)] }])
    expect(zoneSoundEvents(d, 1).some((e) => e.stem.startsWith('objective_zone_securing'))).toBe(
      false,
    )
  })

  it('MUET sans camp allié résolu : le rejeu ne devine pas un camp', () => {
    const d = doc([{ spans: [span(0, 400, 1, true)] }])
    expect(zoneSoundEvents(d, null).some((e) => e.stem.startsWith('objective_zone_securing'))).toBe(
      false,
    )
  })

  it('MUET sur un TRANSFERT trop court : le plancher de 3 s tient', () => {
    const d = doc([{ spans: [span(0, 20, 1, true)] }])
    expect(zoneSoundEvents(d, 1).some((e) => e.stem.startsWith('objective_zone_securing'))).toBe(
      false,
    )
  })

  it('MUET hors colline : un intervalle possédé SANS `active` ne sécurise rien', () => {
    const d = doc([{ spans: [span(0, 400, 1)] }, { spans: [span(0, 400, null)] }])
    expect(zoneSoundEvents(d, 1).some((e) => e.stem.startsWith('objective_zone_securing'))).toBe(
      false,
    )
  })
})
