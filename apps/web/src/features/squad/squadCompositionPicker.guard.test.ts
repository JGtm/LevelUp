/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail : le sélecteur de composition d'escouade a UNE définition, `SquadCompositionPicker`.
 *
 * LE DÉFAUT QU'IL FERME : le `GamertagCombobox` monté avec la pastille du joueur en tête et les
 * escouades enregistrées (`useSquadPresets`) avait deux copies (Escouade, Tendances) quand l'onglet
 * Tactique en montait une troisième, différente — liste de bots, aucune escouade enregistrée,
 * libellé « Coéquipiers » méconnaissable. Une page qui choisit une composition monte le sélecteur
 * partagé ; aucune ne réassemble ses pièces.
 *
 * CE QUE LE SCAN LIT : chaque source de `src/` hors tests ; `useSquadPresets(` (l'appel, pas
 * l'export), `presetGroups=` et `leadingPill=` ne s'écrivent que dans le sélecteur partagé.
 */
import { describe, expect, it } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

const SRC = resolve(__dirname, '../..')
const SEUL_AUTORISE = 'features/squad/SquadCompositionPicker.tsx'
const MOTIFS = [/(?<!function )useSquadPresets\(/, /\bpresetGroups=/, /\bleadingPill=/]

function* sources(dossier: string): Generator<string> {
  for (const entree of readdirSync(dossier)) {
    const chemin = join(dossier, entree)
    if (statSync(chemin).isDirectory()) yield* sources(chemin)
    else if (/\.tsx?$/.test(entree) && !/\.(test|spec)\.tsx?$/.test(entree)) yield chemin
  }
}

/** Le source sans ses commentaires : un commentaire qui nomme un motif ne l'utilise pas. */
function sansCommentaires(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

function fautifs(fichiers: Iterable<[string, string]>): string[] {
  const out: string[] = []
  for (const [chemin, source] of fichiers) {
    if (chemin === SEUL_AUTORISE) continue
    const code = sansCommentaires(source)
    if (MOTIFS.some((m) => m.test(code))) out.push(chemin)
  }
  return out
}

describe('le sélecteur de composition a une seule définition', () => {
  it('aucun autre fichier ne monte les escouades enregistrées dans un GamertagCombobox', () => {
    const fichiers = [...sources(SRC)].map(
      (f) => [relative(SRC, f).replaceAll('\\', '/'), readFileSync(f, 'utf8')] as [string, string],
    )
    expect(fichiers.length).toBeGreaterThan(100)
    expect(fautifs(fichiers)).toEqual([])
  })

  it('le scan voit un réassemblage, et ignore un commentaire ou l’export du hook', () => {
    expect(
      fautifs([
        ['features/x/A.tsx', 'const p = useSquadPresets({ playerSlug })'],
        ['features/x/B.tsx', '<GamertagCombobox leadingPill={{ label }} />'],
        ['features/x/C.tsx', '// presetGroups= est réservé au sélecteur partagé'],
        ['features/squad/useSquadPresets.tsx', 'export function useSquadPresets({'],
      ]),
    ).toEqual(['features/x/A.tsx', 'features/x/B.tsx'])
  })
})
