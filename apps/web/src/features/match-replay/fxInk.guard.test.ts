/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LES TROIS SOURCES DU VOCABULAIRE DE TEINTES SONT LA MÊME.
 *
 * Une nature de décharge existe à trois endroits, et il n'y a pas de compilateur entre eux :
 *   1. le VALIDEUR du serveur   — `shotTintKinds` (loader_replay_labels.go), liste fermée ;
 *   2. la TABLE du titre        — `[shot_tints]` de replay_labels.toml ;
 *   3. le CLIENT                — `SERVED_FX_TINTS` (fxInk.ts) et sa variable de thème.
 *
 * Ce qui casse sans ce test, et silencieusement : une teinte ajoutée côté serveur retombe en
 * « neutral » côté client (le rendu est entier, personne ne le voit), et une teinte déclarée
 * côté client sans variable dans le thème peint une couleur vide (donc rien). Les deux
 * dérives sont invisibles à l'écran — c'est exactement ce qu'un garde-rail doit attraper.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { fxTintVar, SERVED_FX_TINTS } from './fxInk'

const REPO = resolve(__dirname, '..', '..', '..', '..', '..')
const GO_LOADER = resolve(REPO, 'apps/go-api/internal/games/mappings/loader_replay_labels.go')
const TOML = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')
const CSS = resolve(REPO, 'apps/web/src/styles/globals.css')

/** Les natures admises par le VALIDEUR Go (la liste fermée de `shotTintKinds`). */
function goTintKinds(): string[] {
  const src = readFileSync(GO_LOADER, 'utf8')
  const start = src.indexOf('var shotTintKinds = map[string]bool{')
  expect(start, 'shotTintKinds introuvable dans le loader Go').toBeGreaterThan(-1)
  const body = src.slice(start, src.indexOf('}', start))
  return [...body.matchAll(/"(\w+)":\s*true/g)].map((m) => m[1]).sort()
}

/** Les natures effectivement employées par la table du titre. */
function tomlTints(): string[] {
  const src = readFileSync(TOML, 'utf8')
  const block = src.slice(src.indexOf('[shot_tints]'))
  return [...new Set([...block.matchAll(/^\w+\s*=\s*"(\w+)"/gm)].map((m) => m[1]))].sort()
}

describe('garde-rail : le vocabulaire des teintes de tir', () => {
  it('est le MÊME côté serveur et côté client', () => {
    expect([...SERVED_FX_TINTS].sort()).toEqual(goTintKinds())
  })

  it('couvre toutes les teintes que le titre emploie', () => {
    const inconnues = tomlTints().filter((t) => !(SERVED_FX_TINTS as readonly string[]).includes(t))
    expect(inconnues, `teintes du titre que le client ne sait pas peindre : ${inconnues}`).toEqual([])
  })

  it('a une variable de thème pour CHAQUE teinte, dans les deux thèmes', () => {
    const css = readFileSync(CSS, 'utf8')
    const clair = css.slice(css.indexOf(":root[data-theme='light']"))
    for (const tint of [...SERVED_FX_TINTS, 'neutral' as const]) {
      const v = fxTintVar(tint)
      expect(css.includes(`${v}:`), `${v} absente du thème sombre`).toBe(true)
      expect(clair.includes(`${v}:`), `${v} absente du thème clair`).toBe(true)
    }
    expect(css.includes('--replay-fx-core:')).toBe(true)
    expect(clair.includes('--replay-fx-core:')).toBe(true)
  })

  it('n’écrit AUCUNE couleur dans la feature : les teintes ne vivent que dans le thème', () => {
    // La règle du dépôt est étendue, pas contournée (item 2.6c du plan).
    for (const f of ['fxInk.ts', 'muzzleFlash.ts', 'shotFx.ts']) {
      const src = readFileSync(resolve(__dirname, f), 'utf8')
      expect(/#[0-9a-fA-F]{6}\b/.test(src), `${f} porte une valeur hex`).toBe(false)
      expect(/oklch\(/.test(src), `${f} porte une couleur littérale`).toBe(false)
    }
  })
})
