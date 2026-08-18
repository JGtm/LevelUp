/**
 * useReplayTiming — TOUTES LES DURÉES DU REJEU, converties une fois par document.
 *
 * POURQUOI CE FICHIER EXISTE. Le canvas du rejeu porte une dette de taille GELÉE par un
 * cliquet (`placementFamily.guard.test.ts`) : toute addition s'y fait par EXTRACTION, jamais
 * par empilement — c'est la manœuvre que le cliquet existe pour imposer, et c'est la
 * quatrième fois qu'elle sert (useReplayInks, useGrenadeIcons, useReplayWeaponPads avant lui).
 * Le lot R3 allonge la croix de mort et devait écrire pourquoi : les réglages temporels et
 * leur conversion partent donc ici, avec leurs justifications.
 *
 * TOUT EST DÉCLARÉ EN TEMPS RÉEL, JAMAIS EN NOMBRE D'IMAGES. La cadence d'échantillonnage du
 * film est choisie au build et peut changer sans que la lecture change : une durée écrite en
 * frames se mettrait à mentir le jour où elle bouge. La conversion vit ici, une fois par
 * document, et personne d'autre ne la refait.
 */
import { useMemo } from 'react'

import { DYNAMO_REST_HOLD_MS, GRENADE_REST_HOLD_MS } from './grenadeFx'
import { framesPerSecond, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import type { MarkerTiming } from './replayMarkers'

/**
 * Rémanences des événements ponctuels, en temps réel — celles du POC, et elles DIFFÈRENT :
 * un TIR est un éclat bref (0,6 s — c'est sa brièveté qui le rend lisible : à 1,4 s le trait
 * traînait dim et se fondait dans la carte, mesure du recalage 2.2), un LANCER et une MORT
 * tiennent 1,4 s parce qu'ils portent plus de sens qu'une détonation.
 */
const SHOT_HOLD_MS = 600
const EVENT_HOLD_MS = 1_400

/**
 * Réglages temporels du calque des joueurs. Valeurs reprises du POC, où elles ont été réglées
 * à l'écran ; leur justification mesurée est en tête de replayMarkers.ts.
 */
const TIMING_MS = {
  trail: 7_000,
  aimHold: 5_000,
  /**
   * LA CROIX DE MORT PERSISTE 2,5 s depuis le 2026-08-18 (A1 : « une persistance plus
   * longue »). 1,5 s était la valeur du POC : à 4x, une croix vivait moins de dix images et
   * la mort passait inaperçue. DEUX durées sont proposées sur la planche (2,5 s et 4 s) et
   * c'est la PLUS COURTE qui est livrée — allonger un repère qui reste à l'écran sans que
   * rien ne l'y appelle encombre la carte, et le pas suivant se juge avant d'être pris.
   */
  death: 2_500,
  spawn: 800,
} as const

/** La durée proposée EN PLUS sur la planche (item R3-1) : elle n'est pas livrée. */
export const DEATH_HOLD_LONG_MS = 4_000

/** Les durées de la fin de vol d'une grenade, en frames : halo court, nappe Dynamo longue. */
export interface GrenadeRestWindow {
  holdHalo: number
  holdDynamo: number
}

export interface ReplayTiming {
  /** Cadence NATIVE du document, avant multiplicateur de vitesse. */
  baseFps: number
  /** Traînée, maintien du cône, croix de mort, anneau d'apparition — en frames. */
  timing: MarkerTiming
  /** Rémanence d'un lancer ou d'une mort, en frames. */
  eventHoldFrames: number
  /** Rémanence d'un tir, en frames. */
  shotHoldFrames: number
  /** Fenêtres de la fin de vol d'une grenade, en frames. */
  restWindow: GrenadeRestWindow
}

/** useReplayTiming convertit toutes les durées du rejeu pour ce document. */
export function useReplayTiming(doc: ReplayDocumentReady): ReplayTiming {
  return useMemo(
    () => ({
      baseFps: framesPerSecond(doc),
      timing: {
        trail: msToFrames(TIMING_MS.trail, doc),
        aimHold: msToFrames(TIMING_MS.aimHold, doc),
        death: msToFrames(TIMING_MS.death, doc),
        spawn: msToFrames(TIMING_MS.spawn, doc),
      },
      eventHoldFrames: msToFrames(EVENT_HOLD_MS, doc),
      shotHoldFrames: msToFrames(SHOT_HOLD_MS, doc),
      restWindow: {
        holdHalo: msToFrames(GRENADE_REST_HOLD_MS, doc),
        holdDynamo: msToFrames(DYNAMO_REST_HOLD_MS, doc),
      },
    }),
    [doc],
  )
}

/** Les durées DÉCLARÉES, en ms — exposées pour les tests et la planche, jamais pour le rendu. */
export const REPLAY_TIMING_MS = { ...TIMING_MS, event: EVENT_HOLD_MS, shot: SHOT_HOLD_MS } as const
