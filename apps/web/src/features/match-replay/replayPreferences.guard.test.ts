/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n°6, « à la 3e copie, centraliser » + garde-rail) : la lecture et
 * l'écriture localStorage des préférences du rejeu (son, calques, vitesse, catégories —
 * tiroir de réglages, décision du 16/08) passent par replayPreferences.ts, patron unique
 * try/catch né dans useReplaySound. Un accès direct ailleurs recopierait ce patron une
 * fois de plus — le jour où la copie oublie le try, la préférence plante au lieu de
 * dégrader en silence (navigation privée notamment).
 *
 * UNE EXCEPTION CONNUE, NOMMÉE : useReplaySound.ts lit SOUND_CATEGORIES_OFF_KEY en JSON
 * (un ensemble de catégories coupées, pas un booléen ni un nombre) — forme que
 * replayPreferences.ts ne couvre pas, usage unique à ce jour (pas encore de 2e copie à
 * centraliser). Autorisée nommément plutôt qu'ignorée en silence.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'

const DIRECT_ACCESS = /\blocalStorage\.(getItem|setItem|removeItem)\(/

// replayPreferences.ts EST le patron (légitime) ; useReplaySound.ts porte l'exception JSON
// documentée ci-dessus.
const ALLOWED = new Set(['replayPreferences.ts', 'useReplaySound.ts'])

describe('garde-rail : localStorage des préférences du rejeu passe par replayPreferences.ts', () => {
  it('aucun accès direct ailleurs dans la feature (hors tests, hors patron, hors exception nommée)', () => {
    const offenders = readdirSync(__dirname)
      .filter((f) => /\.tsx?$/.test(f) && !/\.test\.tsx?$/.test(f) && !ALLOWED.has(f))
      .filter((f) => DIRECT_ACCESS.test(readFileSync(join(__dirname, f), 'utf8')))
    expect(offenders).toEqual([])
  })
})
