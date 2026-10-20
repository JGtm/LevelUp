/**
 * wavFile.ts — LA LECTURE DES CHUNKS d'un WAV livré, pour les garde-rails d'assets sonores.
 *
 * Né le 2026-09-27 (item 7, A4.8) quand un second fichier de garde-rails
 * (`sound/replaySoundLevels.guard.test.ts`) a eu besoin de lire les échantillons : plutôt qu'une
 * copie de plus du parcours des chunks RIFF, les deux garde-rails passent par ici. Aucune
 * tolérance : un fichier sans le chunk demandé est une erreur, jamais une valeur devinée.
 */
/// <reference types="node" />
import { readFileSync } from 'node:fs'

/** Le contenu d'un chunk RIFF (`fmt `, `data`…) d'un WAV, ou une erreur s'il manque. */
export function wavChunk(file: string, cherche: string): Buffer {
  const buf = readFileSync(file)
  for (let at = 12; at + 8 <= buf.length; ) {
    const id = buf.toString('latin1', at, at + 4)
    const size = buf.readUInt32LE(at + 4)
    if (id === cherche) return buf.subarray(at + 8, at + 8 + size)
    at += 8 + size + (size % 2)
  }
  throw new Error(`${file} : chunk ${cherche} introuvable`)
}

/** Cadence, canaux et profondeur, lus dans le chunk `fmt `. */
export function wavFormat(file: string): { canaux: number; cadence: number; bits: number } {
  const fmt = wavChunk(file, 'fmt ')
  return { canaux: fmt.readUInt16LE(2), cadence: fmt.readUInt32LE(4), bits: fmt.readUInt16LE(14) }
}
