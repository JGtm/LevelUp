/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (retours rejeu L2, 2026-09-23) : toute lecture de `features/tactical/queries.ts`
 * dont la clé de cache embarque l'empreinte d'un filtre (`hashFiltre(`) DÉCLARE
 * `placeholderData`.
 *
 * LE DÉFAUT QU'IL FERME : une clé qui change à chaque filtre ou question, sans donnée
 * précédente, repasse par « en attente » ; la vue démontait alors son corps, fond de carte
 * compris, puis le reconstruisait (constat utilisateur « le fond clignote »). Même classe
 * que l'Escouade du 2026-09-20. Le test de COMPORTEMENT est
 * `TacticalAnalysisView.fond.test.tsx` ; celui-ci empêche qu'une NOUVELLE lecture filtrée
 * réintroduise la classe sans qu'on le décide.
 *
 * ALLOWLIST DATÉE (2026-09-23), UNE ENTRÉE : `useTacticalCellule`. Le détail d'une cellule
 * ne garde PAS la réponse précédente — ce serait afficher les contributions d'une AUTRE
 * cellule (ou d'un autre périmètre) sous la cellule choisie. Ajouter une entrée exige une
 * raison datée du même ordre.
 */
import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const EXEMPTEES: Readonly<Record<string, string>> = {
  useTacticalCellule:
    '2026-09-23 — les contributions d’une autre cellule mentiraient : le détail repart à vide',
}

/** Les fonctions exportées de `queries.ts`, chacune avec son corps (jusqu'à la suivante). */
function lectures(source: string): { nom: string; corps: string }[] {
  const re = /^export function (\w+)/gm
  const debuts = [...source.matchAll(re)].map((m) => ({ nom: m[1], index: m.index ?? 0 }))
  return debuts.map((d, i) => ({
    nom: d.nom,
    corps: source.slice(d.index, i + 1 < debuts.length ? debuts[i + 1].index : source.length),
  }))
}

describe('garde-rail : une lecture tactique filtrée garde sa réponse précédente', () => {
  const source = readFileSync(resolve(process.cwd(), 'src/features/tactical/queries.ts'), 'utf8')
  const filtrees = lectures(source).filter((l) => /queryKey:[^\n]*hashFiltre\(/.test(l.corps))

  it('le scan voit les lectures filtrées (périmètre, grille, raster, cellule)', () => {
    expect(filtrees.map((l) => l.nom)).toEqual(
      expect.arrayContaining([
        'useTacticalMatchIDs',
        'useTacticalMaps',
        'useTacticalRaster',
        'useTacticalCellule',
      ]),
    )
  })

  it('toute lecture filtrée hors allowlist déclare placeholderData', () => {
    const fautives = filtrees
      .filter((l) => !(l.nom in EXEMPTEES))
      .filter((l) => !/placeholderData:/.test(l.corps))
      .map((l) => l.nom)
    expect(
      fautives,
      `placeholderData: keepPreviousData manquant — la vue repasserait par « en attente » et ` +
        `démonterait le fond à chaque filtre : ${fautives.join(', ')}`,
    ).toEqual([])
  })

  it('l’allowlist ne couvre que des lectures filtrées qui existent et en ont encore besoin', () => {
    for (const nom of Object.keys(EXEMPTEES)) {
      const lecture = filtrees.find((l) => l.nom === nom)
      expect(lecture, `${nom} n'est plus une lecture filtrée : retirer l'entrée`).toBeDefined()
      expect(
        /placeholderData:/.test(lecture?.corps ?? ''),
        `${nom} déclare désormais placeholderData : retirer l'entrée`,
      ).toBe(false)
    }
  })
})
