/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail vocabulaire « Hauteur d'engagement » (decision D26, 2026-09-22 ; CLAUDE.md n°1
 * « FR sans anglicismes » + n°6 anti-divergence).
 *
 * « Denivele » a ete retire de TOUTES les chaines d'interface qui parlent de la hauteur des
 * engagements : titres de carte (Timeseries, Match view), axe des ordonnees (« Hauteur (m) »),
 * legendes et infobulles. Le mot ne doit pas revenir par une nouvelle chaine : c'est un terme
 * de topographie, la grandeur mesuree est un ecart de hauteur entre deux joueurs.
 *
 * Perimetre : les chaines des trois features qui la publient + les deux manifestes i18n qui
 * les servent. Les COMMENTAIRES sont exclus — ils gardent le droit de raconter l'ancien nom
 * et l'histoire de la carte — ainsi que les fichiers de test, qui citent des libelles.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

/** Le mot proscrit, accentue ou non, a n'importe quelle casse et au pluriel. */
const INTERDIT = /d[ée]nivel[ée]?e?s?/i

const ROOT = resolve(process.cwd(), 'src')
const FEATURES = ['timeseries', 'match-view', 'squad']
const MANIFESTES = ['synthesis.toml', 'squad.toml']

/** Retire les commentaires : `//` et `/* *\/` en TS/TSX, `#` en TOML. */
function sansCommentaires(contenu: string, toml: boolean): string {
  // Les lignes sont BLANCHIES, jamais retirees : le numero rapporte reste celui du fichier.
  if (toml) {
    return contenu
      .split('\n')
      .map((ligne) => (/^\s*#/.test(ligne) ? '' : ligne))
      .join('\n')
  }
  return contenu
    .replace(/\/\*[\s\S]*?\*\//g, (bloc) => bloc.replace(/[^\n]/g, ' '))
    .split('\n')
    .map((ligne) => ligne.replace(/^\s*\/\/.*$/, ''))
    .join('\n')
}

function sources(): string[] {
  const out = MANIFESTES.map((f) => join(ROOT, 'lib', 'i18n', 'manifests', f))
  for (const feature of FEATURES) {
    const dir = join(ROOT, 'features', feature)
    const pile = [dir]
    while (pile.length > 0) {
      const courant = pile.pop() as string
      for (const entree of readdirSync(courant, { withFileTypes: true })) {
        const chemin = join(courant, entree.name)
        if (entree.isDirectory()) {
          pile.push(chemin)
          continue
        }
        if (!/\.(ts|tsx)$/.test(entree.name)) continue
        if (/\.test\.tsx?$/.test(entree.name)) continue
        if (statSync(chemin).isFile()) out.push(chemin)
      }
    }
  }
  return out
}

describe('garde-rail vocabulaire « Hauteur d’engagement » (D26)', () => {
  it('aucune chaine d’interface ne dit « denivele »', () => {
    const fautifs: string[] = []
    for (const fichier of sources()) {
      const toml = fichier.endsWith('.toml')
      const contenu = sansCommentaires(readFileSync(fichier, 'utf8'), toml)
      for (const [i, ligne] of contenu.split('\n').entries()) {
        if (INTERDIT.test(ligne)) fautifs.push(`${relative(ROOT, fichier)}:${i + 1}`)
      }
    }
    expect(
      fautifs,
      `« Denivele » proscrit des chaines d’interface (D26) — dire « hauteur d’engagement » : ${fautifs.join(', ')}`,
    ).toEqual([])
  })

  it('le perimetre balaye est non vide (le garde-rail ne se vide pas en silence)', () => {
    expect(sources().length).toBeGreaterThan(20)
  })
})
