/**
 * Garde-rail : LE NIVEAU LIVRÉ DE LA CONCLUSION ET DE L'INTRO (item 7, A4.8, 2026-09-27).
 *
 * UN FICHIER À PART de `replaySoundAssets.guard.test.ts` : ce dernier est à son plafond de taille
 * (`max-lines`). Mêmes assets, même dossier, lecture des chunks partagée (`test/wavFile.ts`).
 *
 * POURQUOI CE GARDE-RAIL. Les trois fanfares de fin ont été livrées à −18 LUFS en 4 canaux, puis
 * réduites en stéréo APRÈS la normalisation. Le dépôt les a donc portées pendant un mois à
 * −25 LUFS, soit 9 LU sous la voix de l'annonceur au lieu de 2, et rien ne le signalait. Une
 * fanfare aussi basse se perd sous les derniers tirs, ce qui colle au constat de l'item 11
 * (« je n'ai plus la musique »).
 *
 * LE CRITÈRE N'EST PAS UN LUFS, et il ne prétend pas l'être. Il se calcule sur les échantillons
 * PCM, sans pondération K ni porte : c'est le RMS MAXIMAL sur une fenêtre glissante de 400 ms (la
 * durée du « momentary » EBU R128), moyenne des carrés sur les canaux, en dBFS. Mesures du
 * 2026-09-27 sur ce catalogue :
 *  - musiques à −18 LUFS : −16,7 à −17,3 dBFS, intro comprise. Les fanfares à −25 LUFS d'avant
 *    le correctif donnaient −23,8 à −24,2 ;
 *  - voix à −16 LUFS : −16,0 à −18,7 dBFS.
 * Les bandes laissent environ 3 dB de part et d'autre de ces mesures. Elles ne jugent pas un
 * dixième de LU : elles attrapent une chaîne de livraison qui a perdu (ou ajouté) plusieurs
 * décibels. La mesure exacte en LUFS reste le travail d'ffmpeg (`ebur128`, recette au journal
 * du plan `.ai/PLAN_BACKLOG_2026-09-26.md`, lot A4).
 */
/// <reference types="node" />
import { describe, expect, it } from 'vitest'
import { resolve } from 'node:path'

import { END_FFA_WIN_VOICE_STEMS, END_MUSIC_STEMS, END_VOICE_STEMS } from './endMatchSound'
import { INTRO_MUSIC_STEM } from './introSound'
import { racineDuDepot } from '../test/featureFiles'
import { wavChunk, wavFormat } from '../test/wavFile'

const SOUNDS_DIR = resolve(racineDuDepot(), 'static', 'sounds', 'halo_infinite')

/** RMS maximal sur 400 ms glissantes, en dBFS, lu sur le PCM 16 bits du fichier. */
function rmsMax400Db(stem: string): number {
  const file = resolve(SOUNDS_DIR, `${stem}.wav`)
  const { canaux, cadence, bits } = wavFormat(file)
  expect(bits, stem).toBe(16)
  const data = wavChunk(file, 'data')
  const n = Math.floor(data.length / 2 / canaux)
  const fenetre = Math.round(0.4 * cadence)
  const carres = new Float64Array(n)
  for (let i = 0; i < n; i++) {
    let s = 0
    for (let c = 0; c < canaux; c++) {
      const v = data.readInt16LE((i * canaux + c) * 2) / 32768
      s += v * v
    }
    carres[i] = s / canaux
  }
  let somme = 0
  let max = 0
  for (let i = 0; i < n; i++) {
    somme += carres[i]
    if (i >= fenetre) somme -= carres[i - fenetre]
    if (i >= fenetre - 1) max = Math.max(max, somme / fenetre)
  }
  return 10 * Math.log10(max)
}

/** Les stems dont le niveau mesuré sort de la bande [bas ; haut] dBFS. */
function horsBande(stems: readonly string[], bas: number, haut: number) {
  return [...new Set(stems)]
    .map((stem) => ({ stem, db: rmsMax400Db(stem) }))
    .filter((m) => m.db < bas || m.db > haut)
}

describe('garde-rail : niveau livré des musiques et des voix de fin (A4.8)', () => {
  it('musiques (fanfares et intro, −18 LUFS) : entre −20 et −14 dBFS', () => {
    expect(horsBande([...Object.values(END_MUSIC_STEMS), INTRO_MUSIC_STEM], -20, -14)).toEqual([])
  })

  it('voix de l annonceur (−16 LUFS) : entre −21 et −13 dBFS', () => {
    const voix = [
      ...Object.values(END_VOICE_STEMS).flatMap((byLocale) => Object.values(byLocale).flat()),
      ...Object.values(END_FFA_WIN_VOICE_STEMS).flat(),
    ]
    expect(horsBande(voix, -21, -13)).toEqual([])
  })
})
