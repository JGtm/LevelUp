/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6, même patron que `weaponFullIcon.guard.test.ts`) : la vignette
 * d'assistance codée en dur dans `ReplayKillFeed.tsx` (`ASSIST_ICON_STEM`, cf. `AssistMark`)
 * doit exister sur disque ET rester CE que `jeu/index.json` déclare à cet index.
 *
 * POURQUOI LE SECOND GARDE-RAIL, PAS SEULEMENT LE PREMIER. `weapon-icons-build` régénère cet
 * atlas depuis les archives du jeu (cf. cmd/weapon-icons-build) : si une future extraction
 * réassigne l'index 62 à un autre pictogramme (le style « killfeed » n'est pas numéroté par
 * arme, cf. main.go), le fichier `killfeed-62.png` continuerait d'exister — mais il ne
 * dessinerait plus une assistance. Une icône FAUSSE est un mensonge, pas un cadre vide : ce
 * test la rend rouge, l'écran ne le montrerait pas.
 */
import { describe, expect, it } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { ASSIST_ICON_STEM } from './ReplayKillFeed'

import { racineDuDepot } from '../test/featureFiles'

const REPO_ROOT = racineDuDepot()
const JEU_DIR = resolve(REPO_ROOT, 'static', 'weapons-assets', 'halo_infinite', 'jeu')

interface JeuIndexEntry {
  index: number
  style: string
  file: string
}

describe('garde-rail : vignette d assistance du kill feed (ASSIST_ICON_STEM)', () => {
  it('le fichier référencé existe sur disque', () => {
    expect(existsSync(resolve(JEU_DIR, `${ASSIST_ICON_STEM}.png`))).toBe(true)
  })

  it('jeu/index.json déclare toujours ce fichier au style killfeed, index 62', () => {
    const entries = JSON.parse(readFileSync(resolve(JEU_DIR, 'index.json'), 'utf8')) as JeuIndexEntry[]
    const entry = entries.find((e) => e.style === 'killfeed' && e.file === `${ASSIST_ICON_STEM}.png`)
    expect(entry, `aucune entree killfeed pour ${ASSIST_ICON_STEM}.png`).toBeTruthy()
    expect(entry?.index).toBe(62)
  })
})
