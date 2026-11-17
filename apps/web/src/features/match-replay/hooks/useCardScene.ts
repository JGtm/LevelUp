/**
 * useCardScene — CE QUE TOUTES LES FICHES D'UN DOCUMENT PARTAGENT : les âges de lecture (en
 * images), la scène des effets d'équipement et le calque de score. Une lecture de DOCUMENT, donc
 * calculée une fois par document (et par jeu de joueurs) et jamais par image ni par fiche : la
 * colonne l'appelle une fois et donne le résultat à chacune de ses tuiles, quelle que soit la
 * forme de la tuile (fiche, tuile compacte, ligne).
 *
 * Aucun état propre : seulement des `useMemo` sur `doc` et `players`.
 */
import { useMemo } from 'react'

import { scoreTimelineOf, type ReplayScoreTimelineReady } from '@/lib/replay/scoreTimeline'

import { teleportMoments } from '../model/placementTeleport'
import type { CardFxScene } from '../model/playerCardReadings'
import { drawablePlacements } from '../model/spawnedObjects'
import { frameToMs, msToFrames } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import {
  buildSlotOwnership,
  campResolver,
  vitalityPresence,
  type ReplayPlayer,
  type VitalityPresence,
} from '../../../lib/replay/rosterLogic'

/**
 * Estompage COMPLET d'une lecture de vitalité à 6 s : la même graduation que le bouclier
 * sur la carte. Le REPORT, lui, n'a pas de limite dans une vie — le flux est différentiel,
 * non retransmis veut dire inchangé — et les points appartiennent à la vie, donc il ne
 * franchit jamais une mort. Ce qui vieillit pâlit ; ce qui n'a jamais été mesuré reste
 * une lacune dite.
 */
const VITALITY_FADE_MS = 6_000
/**
 * Au-delà, une lecture d'inventaire est au plancher d'opacité. 20 s est l'écart médian entre
 * deux images-clés du film : c'est donc l'âge maximal ordinaire d'une lecture.
 */
const READING_FULL_MS = 20_000
/**
 * Durée des éclats d'événement (coup fatal, réapparition, translocation), en temps réel —
 * assez pour être vus sans être subis, calée sur la rémanence des lancers. L'état de mort,
 * lui, est porté en continu par le fond de la fiche. LA COMPOSITION DES EFFETS (éclats,
 * verre du camouflage, encadrés, voile de l'écran occultant) vit dans `playerCardFx.ts` : le
 * hook ne fait que donner les âges.
 */
const FLASH_MS = 1_400

/** Ce que le hook rend : exactement les entrées de lecture d'une fiche qui ne dépendent que du document. */
export interface CardScene {
  /** Quelles vitalités le document publie (bouclier, santé) : sans elles, aucune jauge n'existe. */
  presence: VitalityPresence
  /** Âge, en images, auquel une lecture de vitalité est entièrement estompée. */
  vitalityFade: number
  /** Âge, en images, auquel une lecture d'inventaire est au plancher d'opacité. */
  readingFull: number
  /** Durée des éclats d'événement, en images (au moins 1). */
  flashFrames: number
  /**
   * La scène des effets d'équipement : les poses et les camps par vie (le capteur adverse en a
   * besoin, même contrat que le calque), l'axe de temps des fenêtres de pose, et les instants de
   * translocation. L'éclat de translocation est daté par l'événement du film (schéma 38) ; sur
   * un artefact antérieur, `teleportMoments` retombe sur son repli daté.
   */
  fxScene: CardFxScene
  /**
   * Le calque de score passé par sa garde d'horloge, une seule fois : absent = artefact antérieur
   * au schéma 12, mode sans compteur, ou origine non recalée (cf. `filmClockTrusted`). Les
   * tuiles n'ont alors rien de plus à dire qu'avant — aucune ligne ne se vide, aucun zéro.
   */
  scoreTimeline: ReplayScoreTimelineReady | undefined
}

/** Ce que lit une fiche pour estomper ses lectures : les vitalités publiées et les deux âges d'estompage. */
export type ReadingAges = Pick<CardScene, 'presence' | 'vitalityFade' | 'readingFull'>

/**
 * useReadingAges — les âges de lecture et la présence des vitalités, pour toute surface qui rend
 * l'état d'UN joueur sans monter la colonne des fiches (le panneau de joueur). Même source que
 * `useCardScene` : une surface ne recalcule ni ne redéfinit ces seuils.
 */
export function useReadingAges(doc: ReplayDocumentReady): ReadingAges {
  const vitalityFade = useMemo(() => msToFrames(VITALITY_FADE_MS, doc), [doc])
  const readingFull = useMemo(() => msToFrames(READING_FULL_MS, doc), [doc])
  const presence = useMemo(() => vitalityPresence(doc), [doc])
  return useMemo(() => ({ presence, vitalityFade, readingFull }), [presence, vitalityFade, readingFull])
}

export function useCardScene(doc: ReplayDocumentReady, players: readonly ReplayPlayer[]): CardScene {
  const { presence, vitalityFade, readingFull } = useReadingAges(doc)
  const flashFrames = useMemo(() => Math.max(1, msToFrames(FLASH_MS, doc)), [doc])
  // LE CAMP D'UNE VIE PAR SLOT ET PAR IMAGE (résolveur frame-aware) : un slot de biped est
  // réattribué entre manches, le camp doit suivre l'occupant. Le capteur adverse le lit à
  // l'image du joueur interrogé / à la pose du capteur (cf. equipmentZones).
  const campOfSlot = useMemo(() => campResolver(buildSlotOwnership(players)), [players])
  const teleports = useMemo(() => teleportMoments(doc), [doc])
  // Le canvas lit le même calque de son côté, par son propre besoin : lui veut les POSITIONS, la
  // fiche ne veut que les INSTANTS.
  const fxScene = useMemo<CardFxScene>(
    () => ({
      zones: {
        placements: drawablePlacements(doc),
        campOfSlot,
      },
      time: { frameMs: frameToMs(1, doc), frames: doc.frameCount },
      teleports,
    }),
    [doc, campOfSlot, teleports],
  )
  const scoreTimeline = useMemo(() => scoreTimelineOf(doc), [doc])
  return { presence, vitalityFade, readingFull, flashFrames, fxScene, scoreTimeline }
}
