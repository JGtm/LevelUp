/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (même patron que `assistMarkIcon.guard.test.ts`) : les deux vignettes du kill feed du jeu que
 * `ReplayFragMarks.tsx` désigne en dur (`HEADSHOT_ICON_STEM`, `PERFECT_ICON_STEM`) doivent exister sur disque ET rester
 * CE que `jeu/index.json` déclare à leur index. Une icône FAUSSE est un mensonge, pas un cadre vide : une extraction
 * qui réassignerait l'index à un autre pictogramme rend ce test rouge, l'écran ne le montrerait pas.
 */
import { describe, expect, it } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { HEADSHOT_ICON_STEM, PERFECT_ICON_STEM } from './ReplayFragMarks'

import { racineDuDepot } from '../test/featureFiles'

const JEU_DIR = resolve(racineDuDepot(), 'static', 'weapons-assets', 'halo_infinite', 'jeu')

interface JeuIndexEntry {
  index: number
  style: string
  file: string
  nom_jeu?: string
}

const entries = (): JeuIndexEntry[] => JSON.parse(readFileSync(resolve(JEU_DIR, 'index.json'), 'utf8')) as JeuIndexEntry[]

describe.each([
  { stem: HEADSHOT_ICON_STEM, index: 64, nom: 'headshot' },
  { stem: PERFECT_ICON_STEM, index: 83, nom: 'perfect' },
])('garde-rail : vignette « $nom » du kill feed ($stem)', ({ stem, index, nom }) => {
  it('le fichier référencé existe sur disque', () => {
    expect(existsSync(resolve(JEU_DIR, `${stem}.png`))).toBe(true)
  })

  it('jeu/index.json déclare toujours ce fichier au style killfeed, à son index, sous son nom du jeu', () => {
    const entry = entries().find((e) => e.style === 'killfeed' && e.file === `${stem}.png`)
    expect(entry, `aucune entree killfeed pour ${stem}.png`).toBeTruthy()
    expect(entry?.index).toBe(index)
    expect(entry?.nom_jeu).toBe(nom)
  })
})
