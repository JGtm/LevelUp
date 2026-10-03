/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : LA FRONTIÈRE ENTRE LE REJEU ET LA MATCH VIEW RESTE NOMMÉE, MODULE PAR MODULE.
 *
 * POURQUOI (2026-09-06, lot v2 D.13, constat N4 de l'audit v7.5). L'allowlist du ratchet P8.5
 * (`tools/lint-cross-feature-imports.mjs`) portait sur des PAIRES DE FEATURES :
 * `match-view=>match-replay` autorisait n'importe lequel des ~370 fichiers du rejeu, et sa
 * justification écrite — « STRICTEMENT bornée au chargement de l'artefact » — était démentie par
 * quatre imports. Une exception qui ne dit pas ce qu'elle autorise ne garde rien : le lint reste
 * vert pendant que la dépendance grossit.
 *
 * CE QUE CE GARDE TIENT : la forme de l'exception, pas son contenu. Ajouter un module à la liste
 * reste possible (avec sa raison écrite, comme les huit d'aujourd'hui) ; revenir à la PAIRE ne
 * l'est pas — c'est le geste qui rouvrirait la feature entière d'un seul mot.
 *
 * CE QU'IL NE PRÉTEND PAS : il ne dit rien des autres paires de l'allowlist (elles restent des
 * paires, et c'est un autre chantier). Il tient la frontière que ce lot a nommée.
 */
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const REPO = resolve(dirname(__filename), '..', '..', '..', '..', '..')
const LINT = resolve(REPO, 'tools/lint-cross-feature-imports.mjs')

/** Les entrées de l'allowlist, telles qu'écrites dans le fichier. */
function entrees(): string[] {
  const src = readFileSync(LINT, 'utf8')
  return [...src.matchAll(/^\s*'([a-z0-9-]+=>[^']+)',/gm)].map((m) => m[1])
}

describe('garde-rail : la frontière rejeu <-> Match View est nommée par module', () => {
  it('aucune des deux paires nues ne figure dans l’allowlist', () => {
    const nues = entrees().filter(
      (e) => e === 'match-view=>match-replay' || e === 'match-replay=>match-view',
    )
    expect(
      nues,
      `une PAIRE de features rouvrirait la voisine entière : ${nues.join(', ')}. ` +
        `Nommer le module importé (\`match-view=>match-replay/<module>\`) avec sa raison.`,
    ).toEqual([])
  })

  it('et les exceptions par module, elles, existent — sans quoi ce garde ne garderait rien', () => {
    const modules = entrees().filter((e) => /^match-(view|replay)=>match-(replay|view)\//.test(e))
    expect(modules.length).toBeGreaterThanOrEqual(6)
  })
})
