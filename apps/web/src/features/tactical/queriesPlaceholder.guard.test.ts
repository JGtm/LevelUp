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
 * CE QUE LE SCAN LIT (revue L2-R4) : l'ARGUMENT ENTIER de `queryKey:`, parenthèses
 * équilibrées, sur autant de lignes qu'il en faut — une clé que Prettier coupe sur plusieurs
 * lignes est une lecture filtrée comme une autre ; et `placeholderData:` HORS COMMENTAIRES — un
 * commentaire qui le nomme ne déclare rien. Les deux cas sont cadenassés sur des sources
 * synthétiques ci-dessous.
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

/** Le source sans ses commentaires : un commentaire ne déclare rien. */
function sansCommentaires(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

/** L'argument de `queryKey:` — jusqu'à la virgule ou la fermeture de niveau 0. */
function argumentQueryKey(corps: string): string | null {
  const debut = corps.indexOf('queryKey:')
  if (debut < 0) return null
  let profondeur = 0
  for (let i = debut + 'queryKey:'.length; i < corps.length; i++) {
    const c = corps[i]
    if ('([{'.includes(c)) profondeur++
    else if (')]}'.includes(c)) {
      if (profondeur === 0) return corps.slice(debut, i)
      profondeur--
    } else if (c === ',' && profondeur === 0) return corps.slice(debut, i)
  }
  return corps.slice(debut)
}

interface Lecture {
  nom: string
  filtree: boolean
  placeholder: boolean
}

/** Les fonctions exportées d'un source, chacune avec son corps (jusqu'à la suivante). */
function lectures(source: string): Lecture[] {
  const propre = sansCommentaires(source)
  const debuts = [...propre.matchAll(/^export function (\w+)/gm)].map((m) => ({
    nom: m[1],
    index: m.index ?? 0,
  }))
  return debuts.map((d, i) => {
    const corps = propre.slice(d.index, i + 1 < debuts.length ? debuts[i + 1].index : propre.length)
    return {
      nom: d.nom,
      filtree: argumentQueryKey(corps)?.includes('hashFiltre(') ?? false,
      placeholder: /\bplaceholderData\s*:/.test(corps),
    }
  })
}

function fautives(source: string): string[] {
  return lectures(source)
    .filter((l) => l.filtree && !l.placeholder && !(l.nom in EXEMPTEES))
    .map((l) => l.nom)
}

describe('garde-rail : une lecture tactique filtrée garde sa réponse précédente', () => {
  const source = readFileSync(resolve(process.cwd(), 'src/features/tactical/queries.ts'), 'utf8')
  const filtrees = lectures(source).filter((l) => l.filtree)

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
    const liste = fautives(source)
    expect(
      liste,
      `placeholderData manquant — la vue repasserait par « en attente » et démonterait le fond ` +
        `à chaque filtre : ${liste.join(', ')}`,
    ).toEqual([])
  })

  it('l’allowlist ne couvre que des lectures filtrées qui existent et en ont encore besoin', () => {
    for (const nom of Object.keys(EXEMPTEES)) {
      const lecture = filtrees.find((l) => l.nom === nom)
      expect(lecture, `${nom} n'est plus une lecture filtrée : retirer l'entrée`).toBeDefined()
      expect(lecture?.placeholder, `${nom} déclare désormais placeholderData : retirer l'entrée`).toBe(
        false,
      )
    }
  })
})

describe('garde-rail : le scan lui-même (sources synthétiques, revue L2-R4)', () => {
  it('une clé coupée sur plusieurs lignes, sans placeholderData, est fautive', () => {
    const src = [
      'export function useNouvelleLecture(a: string) {',
      '  return useQuery({',
      '    queryKey: queryKeys.tacticalMaps(',
      '      a,',
      '      b,',
      '      hashFiltre(corps),',
      '    ),',
      '    queryFn: () => api.post(a, corps),',
      '  })',
      '}',
    ].join('\n')
    expect(fautives(src)).toEqual(['useNouvelleLecture'])
  })

  it('un commentaire qui nomme placeholderData ne déclare rien', () => {
    const src = [
      'export function useNouvelleLecture(a: string) {',
      '  return useQuery({',
      '    queryKey: queryKeys.tacticalMaps(a, b, hashFiltre(corps)),',
      '    // placeholderData: keepPreviousData — à poser un jour',
      '    /* placeholderData: keepPreviousData */',
      '    queryFn: () => api.post(a, corps),',
      '  })',
      '}',
    ].join('\n')
    expect(fautives(src)).toEqual(['useNouvelleLecture'])
  })

  it('une clé sans empreinte de filtre n’est pas une lecture filtrée, même si hashFiltre sert ailleurs', () => {
    const src = [
      'export function useFond(a: string) {',
      '  const h = hashFiltre(a)',
      '  return useQuery({',
      '    queryKey: queryKeys.tacticalMapBackground(a, b),',
      '    queryFn: () => api.get(h),',
      '  })',
      '}',
    ].join('\n')
    expect(fautives(src)).toEqual([])
    expect(lectures(src)[0].filtree).toBe(false)
  })

  it('une clé filtrée qui déclare placeholderData (sur plusieurs lignes) passe', () => {
    const src = [
      'export function useNouvelleLecture(a: string) {',
      '  return useQuery({',
      '    queryKey: queryKeys.tacticalMaps(',
      '      a,',
      '      hashFiltre(corps),',
      '    ),',
      '    placeholderData: precedente(a),',
      '  })',
      '}',
    ].join('\n')
    expect(fautives(src)).toEqual([])
  })
})
