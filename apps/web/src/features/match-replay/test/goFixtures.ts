/// <reference types="node" />
/**
 * goFixtures.ts — LES DOCUMENTS DE REJEU PRODUITS PAR GO, VUS DU WEB.
 *
 * # POURQUOI CE MODULE EXISTE (2026-09-13, lot 0.B du chantier décodeur de film)
 *
 * Les preuves du rejeu étaient ASYMÉTRIQUES : côté Go des goldens sur un film réel, côté web
 * une fixture écrite à la main qui déclarait `schemaVersion: 1` — un document que le serveur
 * n'envoie jamais. Aucune preuve ne traversait la frontière : un champ renommé à la cuisson
 * laissait le web vert jusqu'à la production.
 *
 * Le dossier `fixtures/go/` est désormais ÉCRIT PAR GO
 * (`internal/games/halo_infinite/film/replay/contract_fixtures_test.go`) et LU ICI. Il ne
 * s'édite jamais à la main : sa seule porte d'écriture est la régénération, décrite en tête du
 * test Go.
 *
 * # CE QUE LE DOSSIER PORTE (lot 0.B.7 : UN DOCUMENT PAR BUILD DU JEU)
 *
 *  - `manifest.json` — la version de schéma du producteur et, pour chaque document, son
 *    fichier, son film, son BUILD, sa version et sa TAILLE. C'est la partie LÉGÈRE : elle se
 *    lit sans ouvrir un seul document, et c'est elle que lisent `testDoc.ts` (pour ne plus
 *    écrire de numéro à la main) et la matrice de compatibilité (pour statuer sur chaque
 *    fixture) ;
 *  - `replay_schema_<N>_<film>.json.gz` — le document d'un build, cuit depuis les entrées
 *    figées de ce film. Ils sont compressés parce qu'ils pèsent de 2 à 10 Mio en clair ; le
 *    dépôt versionne déjà des fixtures binaires de cette famille (`testdata/inputs_*.bin.gz`).
 *    Les sept fixtures par build ne gardent qu'UN POINT DE PISTE SUR CINQ (`pointsStride`,
 *    déclaré par entrée) : le document plein pèserait 5,86 Mio pour le jeu, contre un plafond
 *    de 3 Mio, et ces fixtures servent le contrat de FORME, pas la reproduction du document
 *    servi. `000d5950` reste intacte.
 *
 * LE DOSSIER NE PORTE QU'UN SEUL JEU, celui de la version courante : la régénération Go
 * supprime celui de la version précédente (l'historique git garde le reste).
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne normalise rien et ne juge rien : il rend le document DE TRANSPORT, tel que l'API
 * l'enverrait. C'est à l'appelant de le faire passer par `normalizeReplayDocument` — sans quoi
 * le test ne prouverait pas que la frontière tient.
 *
 * ET IL NE CHARGE JAMAIS LE JEU ENTIER D'UN COUP. Les huit documents pèsent ensemble plus de
 * 40 Mio une fois décompressés : un `loadGoFixtures()` qui les rendrait tous les tiendrait tous
 * en mémoire en même temps. L'appelant itère `goFixtureEntries()` (léger) et charge le document
 * dont il a besoin, quand il en a besoin.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gunzipSync } from 'node:zlib'

import type { ReplayDocument } from '@/lib/api/types'

import { featureRoot } from './featureFiles'

/** Ce que le manifeste dit d'UN document publié. */
export interface GoFixtureEntry {
  file: string
  film: string
  build: string
  schemaVersion: number
  bytes: number
  /**
   * UN POINT DE PISTE SUR COMBIEN ce document porte (1 = intact).
   *
   * POURQUOI C'EST DÉCLARÉ ET NON DEVINÉ. Les sept fixtures par build sont amincies à un point
   * sur cinq — mesure du lot 0.B.7 : les documents pleins pèsent 5,86 Mio pour un plafond de
   * 3 Mio, et `tracks[].points` en fait 85 à 89 %. La FORME des pistes est intacte (leur
   * nombre, leurs champs, leurs bornes), leur ÉCHANTILLONNAGE ne l'est pas : un test qui
   * compterait des points, mesurerait une vitesse ou une distance doit lire ce taux, pas le
   * supposer. `000d5950` reste à 1, document intact.
   */
  pointsStride: number
}

/** Le manifeste écrit par Go : la version du producteur et les documents publiés. */
export interface GoFixtureManifest {
  schemaVersion: number
  fixtures: GoFixtureEntry[]
}

/** Un document produit par Go, avec l'entrée de manifeste dont il vient. */
export interface GoFixture {
  entry: GoFixtureEntry
  doc: ReplayDocument
}

/** Le dossier partagé par les deux applications. */
export function goFixturesDir(): string {
  return join(featureRoot(), 'test', 'fixtures', 'go')
}

/** Une entrée de manifeste est-elle complète ? Un champ manquant est un manifeste à régénérer. */
function entreeValide(e: GoFixtureEntry): boolean {
  return (
    typeof e?.file === 'string' &&
    e.file.length > 0 &&
    typeof e.film === 'string' &&
    typeof e.build === 'string' &&
    Number.isInteger(e.schemaVersion) &&
    e.schemaVersion > 0 &&
    Number.isInteger(e.bytes) &&
    e.bytes > 0 &&
    Number.isInteger(e.pointsStride) &&
    e.pointsStride >= 1
  )
}

/**
 * Le manifeste. Lu à chaque appel — il pèse quelques centaines d'octets, et un cache de module
 * ferait mentir un test qui régénérerait les fixtures en cours de route.
 */
export function goFixtureManifest(): GoFixtureManifest {
  const raw = readFileSync(join(goFixturesDir(), 'manifest.json'), 'utf8')
  const m = JSON.parse(raw) as GoFixtureManifest
  const forme =
    Number.isInteger(m.schemaVersion) &&
    m.schemaVersion > 0 &&
    Array.isArray(m.fixtures) &&
    m.fixtures.length > 0 &&
    m.fixtures.every(entreeValide)
  if (!forme) {
    throw new Error(`goFixtures: manifeste illisible (${raw})`)
  }
  return m
}

/**
 * LES DOCUMENTS PUBLIÉS, sans en ouvrir aucun : un par build du jeu.
 *
 * C'est la forme que tout test doit préférer pour ITÉRER : elle porte la version et le build
 * de chaque document, donc de quoi statuer (matrice de compatibilité) et de quoi NOMMER le
 * fautif, pour le prix d'un `readFileSync` de quelques centaines d'octets.
 */
export function goFixtureEntries(): GoFixtureEntry[] {
  return goFixtureManifest().fixtures
}

/**
 * LA VERSION DE SCHÉMA DU PRODUCTEUR, seule source pour tout test web qui en a besoin.
 *
 * C'est ce qui remplace les littéraux : le ratchet `testDoc.guard.test.ts` interdit désormais
 * d'écrire un numéro de schéma en dur dans un test du rejeu.
 */
export function goFixtureSchemaVersion(): number {
  return goFixtureManifest().schemaVersion
}

/**
 * UN document, décompressé et désérialisé.
 *
 * PARESSEUX PAR CONSTRUCTION : un document pèse de 2 à 10 Mio décompressé, et seuls les tests
 * de contrat les lisent. Un import de module les aurait chargés dans les ~250 fichiers de test
 * qui ne veulent que la version.
 */
export function loadGoFixture(entry: GoFixtureEntry): GoFixture {
  const blob = readFileSync(join(goFixturesDir(), entry.file))
  const json = entry.file.endsWith('.gz')
    ? gunzipSync(blob).toString('utf8')
    : blob.toString('utf8')
  return { entry, doc: JSON.parse(json) as ReplayDocument }
}
