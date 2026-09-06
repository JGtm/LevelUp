/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail des ENCRES du canvas : chaque `InkVar` que le rendu peut demander existe dans le
 * thème. Une variable absente rendrait '' (fillStyle inchangé) et un contour de nom
 * invisible, en silence — `readInk` le journalise, ce test le fait échouer avant.
 *
 * SA LISTE EST DÉRIVÉE DU TYPE DEPUIS LE 2026-09-06 (registre, M5). Elle était écrite à la
 * main et n'en contenait QU'UNE sur six, pendant que le docstring promettait « chaque
 * `InkVar` » : cinq encres du système de design pouvaient donc disparaître de la feuille sans
 * que rien ne le dise. La liste se lit maintenant dans l'union `InkVar` elle-même — une encre
 * ajoutée au type entre dans le garde le jour où on l'y écrit.
 *
 * DEUX EXIGENCES, ET ELLES NE SONT PAS LES MÊMES POUR TOUT LE MONDE :
 *
 *  - TOUTE encre du type doit être DÉCLARÉE quelque part dans la feuille, sans quoi `readInk`
 *    rend '' et le canvas peint avec l'encre précédente ;
 *  - les encres PROPRES AU REJEU (`--replay-*`) doivent l'être dans les DEUX thèmes : elles
 *    n'ont pas de valeur héritée du système de design sur laquelle retomber.
 */
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const CSS = resolve(__dirname, '..', '..', 'styles', 'globals.css')
const SOURCE = resolve(__dirname, 'canvasInk.ts')

/**
 * Les encres déclarées par l'union `InkVar`, LUES DANS LA SOURCE.
 *
 * Le motif suit la forme de l'union (`| '--nom'`), commentaires intercalés compris : c'est
 * exactement ce que le type autorise `readInk` à recevoir.
 */
function encresDuType(): string[] {
  const src = readFileSync(SOURCE, 'utf8')
  const union = src.slice(src.indexOf('export type InkVar ='))
  const fin = union.indexOf('\n\n')
  return [...union.slice(0, fin).matchAll(/\|\s*'(--[\w-]+)'/g)].map((m) => m[1])
}

describe('garde-rail : encres du canvas de rejeu', () => {
  it('la liste se dérive bien du type, et couvre les six encres', () => {
    // Le nombre n'est pas figé — c'est la DÉRIVATION qui l'est. Il est vérifié pour que le
    // jour où l'union change de forme, ce garde tombe au lieu de garder une liste vide.
    const encres = encresDuType()
    expect(encres).toContain('--replay-label-stroke')
    expect(encres.length).toBeGreaterThanOrEqual(6)
  })

  it('chaque encre du type est déclarée dans la feuille', () => {
    const css = readFileSync(CSS, 'utf8')
    const absentes = encresDuType().filter((v) => !css.includes(`  ${v}:`))
    expect(
      absentes,
      `ces encres sont demandées par le canvas et absentes de globals.css : [${absentes.join(', ')}]. ` +
        `readInk rendrait '' et le canvas peindrait avec l'encre précédente.`,
    ).toEqual([])
  })

  it('les encres PROPRES AU REJEU sont définies dans les DEUX thèmes', () => {
    const css = readFileSync(CSS, 'utf8')
    for (const v of encresDuType().filter((n) => n.startsWith('--replay-'))) {
      const decl = css.indexOf(`${v}:`)
      expect(decl, `${v} absente de globals.css`).toBeGreaterThan(-1)
      // Le bloc qui la porte doit viser :root, le thème sombre ET le thème clair.
      const block = css.slice(css.lastIndexOf('}', decl), decl)
      expect(block.includes(":root[data-theme='dark']"), `${v} : thème sombre`).toBe(true)
      expect(block.includes(":root[data-theme='light']"), `${v} : thème clair`).toBe(true)
    }
  })
})
