/**
 * GARDE-RAIL — le vocabulaire de la Coordination (D19, 2026-09-21).
 *
 * La section disait la MÊME mesure sous quatre mots : « échange » (Go et carte par
 * session), « vengeance » (délai, matrice en infobulle, nuage), « riposte » (distribution)
 * et « assistance croisée » (deux titres). Le lecteur croyait lire quatre mesures et en
 * lisait une. L'utilisateur a tranché : « riposte » pour la mort vengée, « appui » pour
 * l'assistance entre coéquipiers.
 *
 * UNE FACTORISATION SANS GARDE-RAIL RE-DIVERGE (CLAUDE.md n°6) : ce test interdit le retour
 * des mots bannis dans les chaînes UI FR de `features/squad` et du manifest `squad.toml`.
 *
 * CE QU'IL NE GARDE PAS, et c'est délibéré : la statistique de jeu « assistances » (assists
 * du KDA, médailles, compteurs). C'est le chiffre officiel du jeu, pas la notion — seules
 * les formules qui DÉSIGNENT LA NOTION sont bannies.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const RACINE_SQUAD = join(process.cwd(), 'src', 'features', 'squad')
const MANIFEST = join(process.cwd(), 'src', 'lib', 'i18n', 'manifests', 'squad.toml')

/** Les formules bannies — chacune désigne la NOTION, jamais la statistique du jeu. */
const BANNIS: { motif: RegExp; remplacement: string }[] = [
  { motif: /vengeances?\b/i, remplacement: 'riposte' },
  { motif: /\bvenger\b/i, remplacement: 'riposter' },
  { motif: /taux d[’']échange/i, remplacement: 'taux de riposte' },
  { motif: /assistances crois[ée]es/i, remplacement: 'appui' },
]

/** Fichiers sources de `features/squad`, tests et fixtures exclus. */
function sourcesSquad(dir: string): string[] {
  const out: string[] = []
  for (const entree of readdirSync(dir, { withFileTypes: true })) {
    const chemin = join(dir, entree.name)
    if (entree.isDirectory()) {
      out.push(...sourcesSquad(chemin))
      continue
    }
    if (!/\.tsx?$/.test(entree.name)) continue
    if (/\.(test|guard\.test|fixtures)\.tsx?$/.test(entree.name)) continue
    out.push(chemin)
  }
  return out
}

/**
 * Les lignes de CHAÎNES seulement. Les commentaires gardent le droit de raconter d'où l'on
 * vient — un historique effacé est un historique reperdu — mais aucune chaîne affichable ne
 * doit porter un mot banni.
 */
function lignesDeChaines(source: string): string[] {
  return source
    .split(/\r?\n/)
    .filter((l) => !/^\s*(\/\/|\*|\/\*)/.test(l))
    .filter((l) => /['"`]/.test(l))
}

describe('vocabulaire de la Coordination — « riposte » et « appui »', () => {
  const fichiers = sourcesSquad(RACINE_SQUAD)

  it('balaye une arborescence NON VIDE (sentinelle : un garde qui ne lit rien ne garde rien)', () => {
    expect(fichiers.length).toBeGreaterThan(30)
  })

  for (const { motif, remplacement } of BANNIS) {
    it(`bannit ${motif} des chaînes de features/squad (dire : « ${remplacement} »)`, () => {
      const fautifs: string[] = []
      for (const f of fichiers) {
        for (const ligne of lignesDeChaines(readFileSync(f, 'utf8'))) {
          if (motif.test(ligne)) fautifs.push(`${f.split(/[\\/]/).pop()} : ${ligne.trim()}`)
        }
      }
      expect(fautifs, `dire « ${remplacement} » — ${fautifs.join(' ; ')}`).toEqual([])
    })

    it(`bannit ${motif} des chaînes FR de squad.toml (dire : « ${remplacement} »)`, () => {
      const fautifs = readFileSync(MANIFEST, 'utf8')
        .split(/\r?\n/)
        .filter((l) => l.startsWith('fr = ') && motif.test(l))
      expect(fautifs, `dire « ${remplacement} » — ${fautifs.join(' ; ')}`).toEqual([])
    })
  }
})
