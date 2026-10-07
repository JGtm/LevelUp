/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (CLAUDE.md n° 6) : UNE SEULE attribution des couleurs de joueurs sur la page
 * Escouade — l'ordre de la SÉLECTION, posé par `squadPlayerPalette` (`colors.ts`) et lu par les
 * onglets via `useSquadPlayerPalette`.
 *
 * Défaut corrigé (2026-10-07) : Synergies et Contributions coloraient selon l'ordre de la
 * sélection, l'Emprise (fiches, nuage des vies, objectif) selon l'ordre du bloc servi — le même
 * joueur changeait de teinte d'un onglet à l'autre. Ce test échoue si un fichier de
 * `features/squad/` (hors module canonique) attribue lui-même une couleur de joueur : jetons
 * `squad-player-2..4` écrits en dur, liste des jetons coéquipiers, construction directe d'une
 * table de couleurs ou de la palette.
 *
 * Exceptions (une ligne datée et justifiée chacune) dans `PERMIS`.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

/** Les modules canoniques : l'attribution et sa lecture sur le contexte de la page. */
const CANONIQUES = new Set(['colors.ts', 'useSquadPlayerPalette.ts'])

/** Fichier → identifiants permis, avec la date et la raison. */
const PERMIS: Readonly<Record<string, { motifs: readonly string[]; raison: string }>> = {
  'SquadFilterBar.tsx': {
    motifs: ['getSquadTeammateColors'],
    raison:
      '2026-10-07 — les slots du combobox de sélection SONT l’ordre de la sélection : leurs couleurs définissent l’attribution, elles ne la recopient pas',
  },
}

/** Ce qui attribue une couleur de joueur. */
const MOTIFS: readonly { nom: string; re: RegExp }[] = [
  { nom: 'getSquadPlayerColors', re: /\bgetSquadPlayerColors\b/ },
  { nom: 'getSquadTeammateColors', re: /\bgetSquadTeammateColors\b/ },
  { nom: 'SQUAD_TEAMMATE_COLOR_TOKENS', re: /\bSQUAD_TEAMMATE_COLOR_TOKENS\b/ },
  { nom: 'squadPlayerPalette', re: /\bsquadPlayerPalette\b/ },
  { nom: 'squad-player-2..4', re: /squad-player-[2-9]/ },
]

/** Le source sans ses commentaires : un commentaire n'attribue rien. */
function sansCommentaires(source: string): string {
  return source.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')
}

/** Les motifs interdits présents dans un source, hors identifiants permis. */
function fautes(source: string, permis: readonly string[] = []): string[] {
  const propre = sansCommentaires(source)
  return MOTIFS.filter((m) => !permis.includes(m.nom) && m.re.test(propre)).map((m) => m.nom)
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name) && !/\.test\.tsx?$/.test(entry.name)) out.push(full)
  }
  return out
}

describe('garde-rail : une seule attribution des couleurs de joueurs (Escouade)', () => {
  const racine = resolve(process.cwd(), 'src', 'features', 'squad')

  it('aucun fichier hors colors.ts / useSquadPlayerPalette.ts n’attribue une couleur de joueur', () => {
    const fautifs: string[] = []
    for (const f of walk(racine)) {
      const rel = relative(racine, f).replace(/\\/g, '/')
      if (CANONIQUES.has(rel)) continue
      const trouvees = fautes(readFileSync(f, 'utf8'), PERMIS[rel]?.motifs)
      if (trouvees.length > 0) fautifs.push(`${rel} : ${trouvees.join(', ')}`)
    }
    expect(
      fautifs,
      'couleurs de joueurs : lire la palette de la page (`useSquadPlayerPalette` : colorByPlayer, tokenOf, inkOf), jamais une attribution locale',
    ).toEqual([])
  })

  it('chaque exception vise un fichier existant qui en a encore besoin', () => {
    for (const [rel, { motifs }] of Object.entries(PERMIS)) {
      const source = readFileSync(join(racine, rel), 'utf8')
      expect(fautes(source), rel).toEqual(expect.arrayContaining([...motifs]))
    }
  })

  it('le détecteur voit une attribution locale par rang et ignore les commentaires', () => {
    expect(fautes('const c = SQUAD_TEAMMATE_COLOR_TOKENS[i - 1]')).toEqual(['SQUAD_TEAMMATE_COLOR_TOKENS'])
    expect(fautes("const t = i === 1 ? 'squad-player-2' : 'squad-player-3'")).toEqual(['squad-player-2..4'])
    expect(fautes('const m = getSquadPlayerColors(blocs[0], reste)')).toEqual(['getSquadPlayerColors'])
    expect(fautes('// getSquadPlayerColors(a, b)\n/* SQUAD_TEAMMATE_COLOR_TOKENS */ const x = 1')).toEqual([])
    expect(fautes("const t = tokenCssVar('squad-player-1')")).toEqual([])
  })
})
