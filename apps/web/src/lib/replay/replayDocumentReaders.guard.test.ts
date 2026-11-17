/// <reference types="node" />
// @vitest-environment node
/**
 * LES CHAMPS DU DOCUMENT DE REJEU QUE PERSONNE NE LIT.
 *
 * Une donnée servie a un lecteur, ou une ligne datée qui nomme son lecteur prévu. Ce garde-rail lit les clés de
 * PREMIER NIVEAU du schéma du document (`replayDocumentSchema.ts`) et cherche, pour chacune, un lecteur de
 * production dans `apps/web/src`. Une clé sans lecteur doit figurer dans `SANS_LECTEUR` avec la date où elle y est
 * entrée et le lecteur prévu ; la liste ne fait que rétrécir : une clé qui trouve un lecteur doit en sortir (le
 * test échoue tant qu'elle y reste), une clé qui sort du schéma aussi.
 *
 * MÉTHODE. Une clé est « lue » si son nom apparaît comme accès de propriété du document (`doc.clé`, `doc?.clé`,
 * `ready.clé`, `data.clé`) dans un fichier de production du rejeu (`features/match-replay` et `lib/replay`, les seuls
 * paquets qui consomment le document), hors : le schéma, la normalisation (`replayNormalize.ts`), les types
 * (`replayReadyTypes.ts`), les fichiers générés, les tests et leurs dossiers de support, et hors commentaires.
 *
 * LIMITES. La recherche est textuelle : elle ne suit pas les types. Le récepteur doit s'appeler `doc`, `ready` ou `data`
 * (le document, le document normalisé et le résultat de la requête dans le code du rejeu) : sans cette condition, un
 * `kill.medals` ou un `.hits` d'un tout autre objet passe pour un lecteur. Un document lu sous un autre nom, par
 * destructuration (`const { clé } = doc`) ou par crochets (`doc['clé']`) n'est pas vu : la mesure signale alors à
 * tort la clé, et la lecture se réécrit en accès de propriété sur `doc`.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, relative, resolve, sep } from 'node:path'
import { describe, expect, it } from 'vitest'

const SRC = resolve(__dirname, '../..')

/** Les paquets qui consomment le document. */
const PAQUETS_LECTEURS = ['features/match-replay', 'lib/replay'].map((p) => resolve(SRC, p))
const SCHEMA = resolve(__dirname, 'replayDocumentSchema.ts')

/** Les fichiers qui décrivent le document sans le lire. */
const FICHIERS_EXCLUS = new Set(
  ['replayDocumentSchema.ts', 'replayNormalize.ts', 'replayReadyTypes.ts'].map((f) => resolve(__dirname, f)),
)

/** Les dossiers de support de test et les fichiers générés. */
const DOSSIERS_EXCLUS = new Set(['test', 'tests', '__tests__', '__mocks__', 'fixtures', 'generated'])

/** Une clé sans lecteur : `AAAA-MM-JJ : lecteur prévu`. La liste ne fait que rétrécir. */
const SANS_LECTEUR: Record<string, string> = {
  medals:
    "2026-11-17 : fiche à l'instant et temps forts ; la liste du film porte les médailles que les événements du match perdent",
  scoreActions:
    "2026-11-17 : fiche à l'instant, score ; seulement quand la lecture est dite complète",
  hits: "2026-11-17 : fiche à l'instant, tirs et dégâts ; seulement pour un joueur à lecture suffisante",
  damage: "2026-11-17 : fiche à l'instant, tirs et dégâts ; seulement pour un joueur à lecture suffisante",
  shotStats: "2026-11-17 : fiche à l'instant, tirs et dégâts ; seulement pour un joueur à lecture suffisante",
  roundClock: '2026-11-17 : chrono fidèle au jeu sur les modes à manches',
  rules: '2026-11-17 : confirmation des lectures de réapparition',
  // Mesurés sans lecteur le 2026-11-17, non décidés : à statuer (un lecteur prévu, ou le retrait du document servi).
  bombEvents: '2026-11-17 : à statuer par le user',
  bombStats: '2026-11-17 : à statuer par le user',
  identity: '2026-11-17 : à statuer par le user',
  stances: '2026-11-17 : à statuer par le user',
  structureBounds: '2026-11-17 : à statuer par le user',
  vehicleScenery: '2026-11-17 : à statuer par le user',
}

/** Le format exact d'une entrée : une date réelle, un séparateur, un lecteur nommé. */
const ENTREE_DATEE = /^(\d{4})-(\d{2})-(\d{2}) : \S/

/** Les clés de premier niveau du schéma : les lignes `  clé:` du corps de `z.strictObject({ ... })` de la racine. */
function clesDuSchema(): string[] {
  const lignes = readFileSync(SCHEMA, 'utf8').split('\n')
  const debut = lignes.findIndex((l) => l.startsWith('export const replayDocumentSchema = z.strictObject({'))
  expect(debut, 'le schéma de la racine est introuvable').toBeGreaterThanOrEqual(0)
  const cles: string[] = []
  for (const ligne of lignes.slice(debut + 1)) {
    if (ligne.startsWith('})')) break
    const m = /^ {2}(\w+):/.exec(ligne)
    if (m) cles.push(m[1])
  }
  return cles
}

/** Les fichiers de production sous `src` : ni tests, ni dossiers de support, ni fichiers décrivant le document. */
function fichiersDeProduction(dossier: string): string[] {
  const out: string[] = []
  for (const entree of readdirSync(dossier, { withFileTypes: true })) {
    const chemin = join(dossier, entree.name)
    if (entree.isDirectory()) {
      if (!DOSSIERS_EXCLUS.has(entree.name)) out.push(...fichiersDeProduction(chemin))
      continue
    }
    if (!/\.tsx?$/.test(entree.name) || /\.test\.tsx?$/.test(entree.name)) continue
    if (entree.name === 'generated.ts' || FICHIERS_EXCLUS.has(chemin)) continue
    out.push(chemin)
  }
  return out
}

/** Le texte d'un fichier sans ses commentaires (un `//` n'en est un qu'en début de ligne ou après une espace). */
function sansCommentaires(texte: string): string {
  return texte.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|\s)\/\/.*$/gm, '$1')
}

const LECTURES = PAQUETS_LECTEURS.flatMap(fichiersDeProduction).map((f) => ({
  fichier: relative(SRC, f).split(sep).join('/'),
  texte: sansCommentaires(readFileSync(f, 'utf8')),
}))

/** Les fichiers de production du rejeu où la clé est lue comme propriété du document. */
function lecteursDe(cle: string): string[] {
  const acces = new RegExp(`\\b(?:doc|ready|data)\\??\\.${cle}\\b`)
  return LECTURES.filter((l) => acces.test(l.texte)).map((l) => l.fichier)
}

describe('les champs du document de rejeu sans lecteur', () => {
  const cles = clesDuSchema()

  it('le schéma de la racine est lu en entier', () => {
    expect(cles.length).toBeGreaterThan(60)
    expect(new Set(cles).size).toBe(cles.length)
  })

  it('chaque entrée de la liste porte une date réelle et un lecteur prévu', () => {
    for (const [cle, ligne] of Object.entries(SANS_LECTEUR)) {
      const m = ENTREE_DATEE.exec(ligne)
      expect(m, `${cle} : « AAAA-MM-JJ : lecteur prévu » attendu, lu « ${ligne} »`).not.toBeNull()
      const [, a, mo, j] = m as RegExpExecArray
      const date = new Date(Date.UTC(Number(a), Number(mo) - 1, Number(j)))
      const relue = `${a}-${mo}-${j}`
      expect(date.toISOString().slice(0, 10), `${cle} : date inexistante`).toBe(relue)
    }
  })

  it('chaque clé de la liste est encore une clé du schéma', () => {
    const sorties = Object.keys(SANS_LECTEUR).filter((c) => !cles.includes(c))
    expect(sorties, 'ces clés ont quitté le schéma : les retirer de la liste').toEqual([])
  })

  it('une clé de la liste n’a toujours aucun lecteur (la liste ne fait que rétrécir)', () => {
    const lues = Object.keys(SANS_LECTEUR)
      .map((c) => ({ cle: c, lecteurs: lecteursDe(c) }))
      .filter((x) => x.lecteurs.length > 0)
    expect(lues, 'ces clés ont un lecteur : les retirer de la liste').toEqual([])
  })

  it('toute clé du schéma a un lecteur de production ou figure sur la liste datée', () => {
    const orphelines = cles.filter((c) => !(c in SANS_LECTEUR) && lecteursDe(c).length === 0)
    expect(orphelines, 'clés servies sans lecteur : un lecteur, ou une ligne datée dans SANS_LECTEUR').toEqual([])
  })
})
