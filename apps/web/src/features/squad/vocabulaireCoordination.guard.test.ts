/**
 * GARDE-RAIL — le vocabulaire de la section « Appui et portée » de l'Escouade.
 *
 * L'assistance entre coéquipiers se dit « appui ». La mort reprise par un coéquipier sur son
 * tueur (« riposte », « vengeance », « échange » ; en anglais « payback », « trade »,
 * « retaliation », « avenged ») n'est plus une notion de l'Escouade (décision de l'utilisateur
 * du 2026-10-05) : aucune chaîne ne la nomme, sous aucun de ses noms ni de ses formes.
 *
 * Ce test interdit les formules bannies dans les chaînes UI de `features/squad` (lignes de
 * chaînes des sources, FR et EN confondus) et dans les chaînes du manifeste `squad.toml`
 * (lignes `fr =` pour les motifs français, lignes `en =` pour les motifs anglais).
 *
 * LES DEUX RÔLES DE L'APPUI SE DISENT « ASSISTANT » ET « FINISSEUR » (décision du 2026-11-14 ; en
 * anglais « assister » et « finisher ») : « larbin » et « patron » (en anglais « minion », « lackey »,
 * « boss ») sont bannis, sur l'Escouade comme sur la page Match qui montre le même diagramme
 * (même forme, mêmes mots). Les phrases sont à la voix active, sujet = l'assistant : « X a été
 * assisté par » est banni aussi.
 *
 * « ÉCHANGE » EST BANNI EN MOT ENTIER : aucune chaîne de l'Escouade ne l'emploie dans un
 * autre sens (relevé du 2026-10-07). Un usage légitime à venir (« échange de position »…)
 * resserrera le motif sur les formules de la notion (« taux d'échange », « échanges de
 * frags »), pas l'inverse.
 *
 * CE QU'IL NE GARDE PAS, et c'est délibéré : la statistique de jeu « assistances » (assists
 * du KDA, médailles, compteurs) et les noms d'armes (« MA5K Avenger » : `avenge` n'est banni
 * que comme mot entier). Seules les formules qui DÉSIGNENT LA NOTION sont bannies.
 */
import { readFileSync, readdirSync } from 'node:fs'
import { join } from 'node:path'

import { describe, expect, it } from 'vitest'

const RACINE_SQUAD = join(process.cwd(), 'src', 'features', 'squad')
const MANIFEST = join(process.cwd(), 'src', 'lib', 'i18n', 'manifests', 'squad.toml')

/** Bornes de mot Unicode : « vengé » ou « échanges » ne se coupent pas sur un accent. */
const AVANT = '(?<![\\p{L}\\p{N}])'
const APRES = '(?![\\p{L}\\p{N}])'
const mot = (corps: string) => new RegExp(`${AVANT}(?:${corps})${APRES}`, 'iu')

/** Les formules bannies — chacune désigne une NOTION, jamais la statistique du jeu. */
const NOTION_RETIREE = 'rien : notion retirée de l’Escouade'
const BANNIS: { motif: RegExp; langue: 'fr' | 'en'; remplacement: string; flux?: true }[] = [
  { motif: /ripost/i, langue: 'fr', remplacement: NOTION_RETIREE },
  // venger, vengé(e)(s), vengeance(s), vengeur(s) : toute forme du verbe et de ses dérivés.
  { motif: new RegExp(`${AVANT}veng\\p{L}*`, 'iu'), langue: 'fr', remplacement: NOTION_RETIREE },
  { motif: mot('[ée]changes?'), langue: 'fr', remplacement: NOTION_RETIREE },
  { motif: /assistances crois[ée]es/i, langue: 'fr', remplacement: 'appui' },
  // Les deux rôles du diagramme de flux (2026-11-14) : « larbin » / « patron » -> « assistant » / « finisseur ».
  { motif: mot('larbins?'), langue: 'fr', remplacement: 'assistant', flux: true },
  { motif: mot('patrons?'), langue: 'fr', remplacement: 'finisseur', flux: true },
  { motif: /(?:a|ont) été assistée?s?\b/i, langue: 'fr', remplacement: 'voix active, sujet = l’assistant : « X a assisté Y »', flux: true },
  { motif: /ripost/i, langue: 'en', remplacement: NOTION_RETIREE },
  { motif: mot('paybacks?|retaliat\\p{L}*|trades?|traded|trading|avenge[ds]?|revenge'), langue: 'en', remplacement: NOTION_RETIREE },
  { motif: mot('minions?|lackeys?|boss(?:es)?'), langue: 'en', remplacement: 'assister / finisher', flux: true },
  { motif: /\b(?:was|were|been) assisted\b/i, langue: 'en', remplacement: 'active voice, subject = the assister: “X assisted Y”', flux: true },
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

describe('vocabulaire de la section « Appui et portée »', () => {
  const fichiers = sourcesSquad(RACINE_SQUAD)

  it('balaye une arborescence NON VIDE (sentinelle : un garde qui ne lit rien ne garde rien)', () => {
    expect(fichiers.length).toBeGreaterThan(30)
  })

  for (const { motif, langue, remplacement } of BANNIS) {
    it(`bannit ${motif} des chaînes de features/squad (dire : « ${remplacement} »)`, () => {
      const fautifs: string[] = []
      for (const f of fichiers) {
        for (const ligne of lignesDeChaines(readFileSync(f, 'utf8'))) {
          if (motif.test(ligne)) fautifs.push(`${f.split(/[\\/]/).pop()} : ${ligne.trim()}`)
        }
      }
      expect(fautifs, `dire « ${remplacement} » — ${fautifs.join(' ; ')}`).toEqual([])
    })

    it(`bannit ${motif} des chaînes ${langue.toUpperCase()} de squad.toml (dire : « ${remplacement} »)`, () => {
      const fautifs = readFileSync(MANIFEST, 'utf8')
        .split(/\r?\n/)
        .filter((l) => l.startsWith(`${langue} = `) && motif.test(l))
      expect(fautifs, `dire « ${remplacement} » — ${fautifs.join(' ; ')}`).toEqual([])
    })
  }
})

// La page Match montre le même diagramme (« à question identique, forme identique ») et les textes du
// flux vivent dans un dossier partagé : les mêmes mots y valent, sinon deux vocabulaires reviennent.
describe('vocabulaire du flux « assistant -> finisseur » — page Match et textes partagés', () => {
  const racines = [
    join(process.cwd(), 'src', 'features', 'match-view'),
    join(process.cwd(), 'src', 'features', '_shared', 'assists'),
  ]
  const fichiers = racines.flatMap((r) => sourcesSquad(r))

  it('balaye une arborescence NON VIDE, textes partagés compris', () => {
    expect(fichiers.length).toBeGreaterThan(20)
    expect(fichiers.some((f) => f.endsWith('assistFlowStrings.ts'))).toBe(true)
  })

  for (const { motif, remplacement } of BANNIS.filter((b) => b.flux)) {
    it(`bannit ${motif} des chaînes de features/match-view et des textes partagés (dire : « ${remplacement} »)`, () => {
      const fautifs: string[] = []
      for (const f of fichiers) {
        for (const ligne of lignesDeChaines(readFileSync(f, 'utf8'))) {
          if (motif.test(ligne)) fautifs.push(`${f.split(/[\\/]/).pop()} : ${ligne.trim()}`)
        }
      }
      expect(fautifs, `dire « ${remplacement} » — ${fautifs.join(' ; ')}`).toEqual([])
    })
  }
})
