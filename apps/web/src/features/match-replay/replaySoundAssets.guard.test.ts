/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : le manifeste sonore du rejeu (replaySound.ts) et le
 * dossier d'assets `static/sounds/halo_infinite/` sont la MÊME liste, rejouée ici.
 *
 * POURQUOI. Le client ne sonde jamais le serveur pour savoir si un son existe (pas de
 * 404 exploratoire) : il croit le manifeste. Un stem sans fichier serait un silence
 * inexpliqué en production ; un fichier sans stem, un asset mort que rien ne joue. Les
 * deux dérives cassent CE test, pas l'écoute.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync } from 'node:fs'
import { resolve } from 'node:path'

import { WEAPON_SOUND_STEMS, THROW_SOUND_STEMS } from './replaySound'

const SOUNDS_DIR = resolve(__dirname, '..', '..', '..', '..', '..', 'static', 'sounds', 'halo_infinite')

describe('garde-rail : manifeste sonore = dossier d assets', () => {
  const shipped = new Set(
    readdirSync(SOUNDS_DIR)
      .filter((f) => f.endsWith('.wav'))
      .map((f) => f.slice(0, -'.wav'.length)),
  )
  const referenced = new Set([...Object.values(WEAPON_SOUND_STEMS), ...THROW_SOUND_STEMS])

  it('chaque stem du manifeste a son fichier .wav', () => {
    const missing = [...referenced].filter((s) => !shipped.has(s))
    expect(missing).toEqual([])
  })

  it('chaque fichier .wav livré est joué par un stem (0 asset mort)', () => {
    const orphans = [...shipped].filter((s) => !referenced.has(s))
    expect(orphans).toEqual([])
  })
})
