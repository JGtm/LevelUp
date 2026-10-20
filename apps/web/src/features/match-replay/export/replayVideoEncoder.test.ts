/**
 * replayVideoEncoder.test.ts — LES DÉCISIONS DE L'ENCODEUR, testées sans navigateur.
 *
 * Ce qui est testé ici ne dépend d'AUCUNE API WebCodecs : ce sont les quatre calculs qui
 * décident du fichier (dimensions paires, niveau H.264, débit, contre-pression). L'ouverture
 * réelle de l'encodeur, elle, n'a rien à faire sous jsdom — elle se vérifie dans le navigateur,
 * et c'est le gate manuel de l'étape E1 du plan.
 */
import { describe, expect, it } from 'vitest'

import {
  ENCODE_QUEUE_MAX,
  EXPORT_FPS,
  KEYFRAME_EVERY,
  avcLevelFor,
  canExportVideo,
  evenSize,
  isKeyFrame,
  shouldDrain,
  videoBitrate,
  videoEncoderConfig,
} from './replayVideoEncoder'

describe('evenSize — H.264 refuse les dimensions impaires', () => {
  it('laisse un nombre pair intact', () => {
    expect(evenSize(1280)).toBe(1280)
  })

  it('arrondit VERS LE BAS, jamais vers le haut', () => {
    // Rogner une colonne est invisible ; étirer l'image d'un pixel ne l'est pas.
    expect(evenSize(1281)).toBe(1280)
  })

  it('tronque une largeur fractionnaire (toile à DPR non entier)', () => {
    expect(evenSize(1439.7)).toBe(1438)
  })

  it('remonte à 2 tout ce qui n’a pas de sens pour un encodeur', () => {
    expect(evenSize(0)).toBe(2)
    expect(evenSize(-40)).toBe(2)
    expect(evenSize(Number.NaN)).toBe(2)
  })
})

describe('avcLevelFor — le PLUS BAS niveau qui accepte l’image', () => {
  it('720p à 30 im/s tient au niveau 3.1', () => {
    // 80x45 macroblocs = 3600 FS, soit exactement le plafond du niveau 3.1.
    expect(avcLevelFor(1280, 720, 30)).toBe('avc1.64001f')
  })

  it('1080p à 30 im/s monte au niveau 4.0', () => {
    expect(avcLevelFor(1920, 1080, 30)).toBe('avc1.640028')
  })

  it('la CADENCE compte autant que la surface', () => {
    // Même image qu'au-dessus : à 60 im/s le débit de macroblocs dépasse le niveau 4.0.
    expect(avcLevelFor(1920, 1080, 60)).toBe('avc1.64002a')
  })

  it('une toile démesurée rend le niveau le plus haut plutôt que rien', () => {
    expect(avcLevelFor(16_000, 16_000, 30)).toBe('avc1.64003c')
  })
})

describe('videoBitrate — borné aux deux bouts', () => {
  it('une petite toile ne descend pas sous le plancher', () => {
    // 320x180x30x0,1 = 172 800 b/s : le texte du HUD y baverait.
    expect(videoBitrate(320, 180, 30)).toBe(2_000_000)
  })

  it('une toile courante suit la surface', () => {
    expect(videoBitrate(1280, 720, 30)).toBe(2_764_800)
  })

  it('une toile démesurée ne dépasse pas le plafond', () => {
    expect(videoBitrate(7680, 4320, 60)).toBe(40_000_000)
  })
})

describe('videoEncoderConfig — ce qu’on présente au navigateur', () => {
  it('assemble codec, dimensions paires, cadence et débit', () => {
    expect(videoEncoderConfig(1281, 721)).toEqual({
      codec: 'avc1.64001f',
      width: 1280,
      height: 720,
      framerate: EXPORT_FPS,
      bitrate: 2_764_800,
      avc: { format: 'avc' },
    })
  })

  it('la cadence par défaut est celle de l’export', () => {
    expect(videoEncoderConfig(640, 360).framerate).toBe(30)
  })
})

describe('contre-pression et images-clés', () => {
  it('n’attend pas tant que la file reste sous le seuil', () => {
    expect(shouldDrain(0)).toBe(false)
    expect(shouldDrain(ENCODE_QUEUE_MAX)).toBe(false)
  })

  it('attend dès que la file dépasse le seuil', () => {
    // Sans cette attente, un match de dix minutes empile 18 000 images et l'onglet meurt.
    expect(shouldDrain(ENCODE_QUEUE_MAX + 1)).toBe(true)
  })

  it('ouvre le clip sur une image-clé, puis une toutes les 2 secondes', () => {
    expect(isKeyFrame(0)).toBe(true)
    expect(isKeyFrame(1)).toBe(false)
    expect(isKeyFrame(KEYFRAME_EVERY)).toBe(true)
    expect(KEYFRAME_EVERY).toBe(60)
  })
})

describe('canExportVideo — la capacité du navigateur', () => {
  it('rend false là où WebCodecs n’existe pas (jsdom)', () => {
    // C'est ce qui fait retomber l'UI sur l'enregistreur temps réel (décision D5 du plan).
    expect(canExportVideo()).toBe(false)
  })
})
