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
 * # CE QUE LE DOSSIER PORTE
 *
 *  - `manifest.json` — la version de schéma du producteur et la liste des documents. C'est la
 *    partie LÉGÈRE, celle que `testDoc.ts` lit pour ne plus écrire de numéro à la main ;
 *  - `replay_schema_<N>.json.gz` — le document complet, cuit depuis le film de référence. Il
 *    est compressé parce qu'il pèse 3,3 Mio en clair et 0,4 Mio compressé ; le dépôt versionne
 *    déjà un fixture binaire de cette famille (`testdata/inputs_000d5950.bin.gz`).
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne normalise rien et ne juge rien : il rend le document DE TRANSPORT, tel que l'API
 * l'enverrait. C'est à l'appelant de le faire passer par `normalizeReplayDocument` — sans quoi
 * le test ne prouverait pas que la frontière tient.
 */
import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gunzipSync } from 'node:zlib'

import type { ReplayDocument } from '@/lib/api/types'

import { featureRoot } from './featureFiles'

/** Le manifeste écrit par Go : la version du producteur et les documents publiés. */
export interface GoFixtureManifest {
  schemaVersion: number
  files: string[]
}

/** Un document produit par Go, avec le nom du fichier dont il vient. */
export interface GoFixture {
  file: string
  doc: ReplayDocument
}

/** Le dossier partagé par les deux applications. */
export function goFixturesDir(): string {
  return join(featureRoot(), 'test', 'fixtures', 'go')
}

/**
 * Le manifeste. Lu à chaque appel — il pèse quelques dizaines d'octets, et un cache de module
 * ferait mentir un test qui régénérerait les fixtures en cours de route.
 */
export function goFixtureManifest(): GoFixtureManifest {
  const raw = readFileSync(join(goFixturesDir(), 'manifest.json'), 'utf8')
  const m = JSON.parse(raw) as GoFixtureManifest
  if (!Number.isInteger(m.schemaVersion) || m.schemaVersion <= 0 || !Array.isArray(m.files)) {
    throw new Error(`goFixtures: manifeste illisible (${raw})`)
  }
  return m
}

/**
 * LA VERSION DE SCHÉMA DU PRODUCTEUR, seule source pour tout test web qui en a besoin.
 *
 * C'est ce qui remplace les littéraux : le ratchet `testDoc.guard.test.ts` interdit désormais
 * d'écrire un numéro de schéma en dur dans un test de la feature.
 */
export function goFixtureSchemaVersion(): number {
  return goFixtureManifest().schemaVersion
}

/** Un document, décompressé et désérialisé. */
export function loadGoFixture(file: string): GoFixture {
  const blob = readFileSync(join(goFixturesDir(), file))
  const json = file.endsWith('.gz') ? gunzipSync(blob).toString('utf8') : blob.toString('utf8')
  return { file, doc: JSON.parse(json) as ReplayDocument }
}

/**
 * Tous les documents publiés par Go.
 *
 * PARESSEUX PAR CONSTRUCTION : le document pèse 3,3 Mio une fois décompressé, et seuls les
 * tests de contrat le lisent. Un import de module l'aurait chargé dans les ~250 fichiers de
 * test qui ne veulent que la version.
 */
export function loadGoFixtures(): GoFixture[] {
  return goFixtureManifest().files.map(loadGoFixture)
}
