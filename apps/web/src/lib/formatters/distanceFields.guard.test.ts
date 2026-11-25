/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail — AUCUNE DISTANCE DU CONTRAT N'EST LUE BRUTE.
 *
 * Les champs de distance du contrat (`median_m`, `pas_m`, `record_m`, ...) sont servis en UNITÉS DU JEU
 * (1 unité = 3,048 m) ; leur description dans `generated.ts` le dit (« Distance en unités du jeu »).
 * LA LISTE DE CES CHAMPS N'EST ÉCRITE NULLE PART À LA MAIN : elle est lue dans les descriptions du
 * contrat généré, source unique.
 *
 * Un fichier de `features/` ou de `components/` qui lit un de ces champs doit passer par le helper de
 * conversion (`lib/formatters/distance`) ; sinon il affiche des unités comme des mètres. Les lectures qui
 * ne servent pas à afficher (adresse d'une cellule, cadrage du plan) sont listées ci-dessous, datées et
 * justifiées : la liste ne grandit pas sans raison écrite.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import ts from 'typescript'

import { SCAN_TIMEOUT_MS, scanSources, visitAll } from '../archScan.testutil'

const GENERATED = join(process.cwd(), 'src', 'lib', 'api', 'generated.ts')
const DOC_UNITES = /@description (?:Distance|Rayon de la zone) en unités du jeu/

/** Les champs `*_m` documentés « unités du jeu » dans le contrat généré. */
function champsEnUnites(): Set<string> {
  const lignes = readFileSync(GENERATED, 'utf8').split('\n')
  const out = new Set<string>()
  for (let i = 0; i < lignes.length; i++) {
    if (!DOC_UNITES.test(lignes[i])) continue
    for (let j = i + 1; j < Math.min(i + 4, lignes.length); j++) {
      const m = /^\s*([a-z0-9_]+)\??:/.exec(lignes[j])
      if (m) {
        if (m[1].endsWith('_m')) out.add(m[1])
        break
      }
    }
  }
  return out
}

/**
 * Lectures qui n'écrivent aucune distance : le fichier lit le champ pour ADRESSER ou CADRER, jamais pour
 * l'afficher (2026-11-25, B.6). Un fichier ajouté ici dit pourquoi.
 */
const LECTURES_SANS_AFFICHAGE: Record<string, string> = {
  'features/tactical/queries.ts': 'adresse d’une cellule : le pas servi voyage avec la requête',
  'features/tactical/TacticalAnalysisView.tsx': 'adresse et clic sur une cellule, au pas servi',
  'features/tactical/TacticalMapTile.tsx': 'cadrage du mini-plan au pas servi',
  'features/tactical/TacticalPlanCard.tsx': 'cadrage du plan au pas servi',
  'features/tactical/TacticalReapparitionsPlan.tsx': 'cadrage du plan au pas servi',
  'features/tactical/mort.logic.ts': 'tri des coéquipiers par distance (ordre, pas affichage)',
}

describe('garde-rail : une distance du contrat est lue par un fichier qui convertit', () => {
  const champs = champsEnUnites()

  it('la liste lue dans le contrat n’est pas vide, et porte les champs attendus', () => {
    expect(champs.size).toBeGreaterThanOrEqual(15)
    for (const attendu of ['median_m', 'pas_m', 'record_m', 'lobby_delta_m', 'distance_mort_m']) {
      expect(champs.has(attendu), attendu).toBe(true)
    }
    expect(champs.has('rayon_m')).toBe(false) // vrais mètres
  })

  it('aucun fichier de features/ ou components/ ne lit un champ en unités sans le helper', () => {
    const fautifs: string[] = []
    for (const src of scanSources({ includeTests: false })) {
      if (!src.rel.startsWith('features/') && !src.rel.startsWith('components/')) continue
      if (src.rel.includes('/test/') || /\.(fixtures|testutil)\.ts$/.test(src.rel)) continue
      let lit = false
      visitAll(src, (n) => {
        if (ts.isIdentifier(n) && champs.has(n.text)) lit = true
      })
      if (!lit) continue
      const convertit = /formatters\/distance|weaponRangeSideInMeters|weaponRecordRowInMeters|gameUnitsToMeters/.test(src.text)
      if (!convertit && !(src.rel in LECTURES_SANS_AFFICHAGE)) fautifs.push(src.rel)
    }
    expect(fautifs).toEqual([])
  }, SCAN_TIMEOUT_MS)
})
