/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : LES FAMILLES DE POSE SONT ÉNUMÉRÉES EN QUATRE ENDROITS.
 *
 * Une famille d'objet d'équipement existe à quatre endroits, et il n'y a pas de compilateur
 * entre eux :
 *   1. le VALIDEUR du serveur — `equipmentFamilies` (loader_replay_labels.go), liste fermée ;
 *   2. la TABLE du titre      — `[[equipment_objects]]` de replay_labels.toml (`family = …`) ;
 *   3. le RENDU du client     — `PLACEMENT_RENDER` (equipmentPlacementsLayer.ts) ;
 *   4. les LIBELLÉS du client — `placementFamily` (i18n.ts), FR et EN.
 *
 * CE QUI CASSE SANS CE TEST, ET SILENCIEUSEMENT. Une famille ajoutée au manifeste et acceptée
 * par le valideur, mais absente de la table de rendu, ne dessine RIEN : la pose est publiée,
 * décodée, transmise, et l'écran reste vide sans qu'aucune erreur ne soit levée. Une famille
 * dessinée mais sans libellé rendrait l'infobulle muette. Les deux dérives sont invisibles à
 * l'exécution — c'est exactement ce qu'un garde-rail doit attraper. Modèle : fxInk.guard.test.ts.
 *
 * `other` EST LE DÉFAUT DU SERVEUR, pas une famille nommée : il est admis par le valideur et
 * dessiné (le point neutre des objets non identifiés), mais il n'a pas de libellé de famille —
 * l'infobulle lui sert `placementUnnamedLabel`. Il est donc traité à part, explicitement.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

import { PLACEMENT_RENDER } from './equipmentPlacementsLayer'
import { REPLAY_TEXT } from './i18n'

const REPO = resolve(__dirname, '..', '..', '..', '..', '..')
const GO_LOADER = resolve(REPO, 'apps/go-api/internal/games/mappings/loader_replay_labels.go')
const TOML = resolve(REPO, 'config/titles/halo_infinite/mappings/replay_labels.toml')

/** La famille par défaut du serveur : admise et dessinée, jamais nommée. */
const DEFAULT_FAMILY = 'other'

/** Les familles admises par le VALIDEUR Go (la liste fermée d'`equipmentFamilies`). */
function goFamilies(): string[] {
  const src = readFileSync(GO_LOADER, 'utf8')
  const start = src.indexOf('var equipmentFamilies = map[string]bool{')
  expect(start, 'equipmentFamilies introuvable dans le loader Go').toBeGreaterThan(-1)
  const body = src.slice(start, src.indexOf('}', start))
  return [...body.matchAll(/"(\w+)":\s*true/g)].map((m) => m[1]).sort()
}

/** Les familles que la table du titre emploie réellement (`family = "…"`). */
function tomlFamilies(): string[] {
  const src = readFileSync(TOML, 'utf8')
  const start = src.indexOf('[[equipment_objects]]')
  expect(start, '[[equipment_objects]] introuvable dans le manifeste du titre').toBeGreaterThan(-1)
  const block = src.slice(start)
  return [...new Set([...block.matchAll(/^family\s*=\s*"(\w+)"/gm)].map((m) => m[1]))].sort()
}

describe('garde-rail : le vocabulaire des familles de pose', () => {
  it('le RENDU du client couvre exactement ce que le valideur Go admet', () => {
    expect(Object.keys(PLACEMENT_RENDER).sort()).toEqual(goFamilies())
  })

  it('chaque famille que le titre emploie est admise ET dessinée', () => {
    for (const f of tomlFamilies()) {
      expect(goFamilies(), `${f} n'est pas admise par le valideur Go`).toContain(f)
      expect(Object.keys(PLACEMENT_RENDER), `${f} ne dessine rien côté client`).toContain(f)
    }
  })

  it('chaque famille NOMMÉE a son libellé en FR et en EN', () => {
    const nommees = goFamilies().filter((f) => f !== DEFAULT_FAMILY)
    for (const locale of ['fr', 'en'] as const) {
      const labels = REPLAY_TEXT[locale].placementFamily as Record<string, string>
      expect(Object.keys(labels).sort(), `libellés ${locale} désynchronisés`).toEqual(nommees)
      for (const f of nommees) expect(labels[f], `${f} sans libellé ${locale}`).toBeTruthy()
    }
  })

  it('la famille par défaut est admise et dessinée, mais jamais nommée', () => {
    expect(goFamilies()).toContain(DEFAULT_FAMILY)
    expect(PLACEMENT_RENDER[DEFAULT_FAMILY]).toBe('unnamed')
    expect(Object.keys(REPLAY_TEXT.fr.placementFamily)).not.toContain(DEFAULT_FAMILY)
  })
})

/**
 * Garde-rail de TAILLE — le cliquet du registre des reports (2026-08-16 : « prochaine addition
 * sur l'un d'eux : extraire d'abord »). Le canvas du rejeu porte une dette de taille gelée ;
 * le lot des poses l'avait fait passer de 861 à 942 lignes sans extraction préalable, et c'est
 * ce que la revue a relevé. Le plafond n'est pas un idéal, c'est un CLIQUET : il ne remonte
 * jamais. Le franchir se corrige en extrayant, pas en relevant le nombre.
 */
describe('garde-rail : la taille du canvas du rejeu ne remonte pas', () => {
  it('ReplayCanvas.tsx reste sous son plafond', () => {
    const src = readFileSync(resolve(__dirname, 'ReplayCanvas.tsx'), 'utf8')
    expect(src.split('\n').length - 1).toBeLessThanOrEqual(861)
  })
})
