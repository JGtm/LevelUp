/// <reference types="node" />
/**
 * artefactFile.ts — LIRE UN ARTEFACT DE REJEU DU PARC LOCAL, compressé ou non.
 *
 * Le serveur range ses artefacts en `{short8}.json.gz` (gzip) ; un parc ancien porte encore des
 * `{short8}.json` bruts. Les tests de mesure qui lisent `data/cache/replays/{titre}` passent par
 * ce module : la forme est détectée par les octets magiques (0x1f 0x8b), jamais par le nom, et
 * quand un match a les deux fichiers le compressé gagne — la même règle que la porte Go
 * (`replayartifact.ReadBytes`).
 */
import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { gunzipSync } from 'node:zlib'

const EXT_COMPRESSE = '.json.gz'
const EXT_BRUTE = '.json'
const NOM_ARTEFACT = /^([0-9a-f]{8})\.json(\.gz)?$/

/** Le texte JSON d'un fichier d'artefact, décompressé si ses octets sont du gzip. */
export function lireArtefactTexte(chemin: string): string {
  const octets = readFileSync(chemin)
  const gzip = octets.length >= 2 && octets[0] === 0x1f && octets[1] === 0x8b
  return (gzip ? gunzipSync(octets) : octets).toString('utf8')
}

/** L'artefact d'un fichier, désérialisé. */
export function lireArtefact<T>(chemin: string): T {
  return JSON.parse(lireArtefactTexte(chemin)) as T
}

/** Le chemin du fichier qui porte l'artefact `court` (8 hex) du dossier, ou null s'il n'y en a aucun. */
export function cheminArtefact(dossier: string, court: string): string | null {
  for (const ext of [EXT_COMPRESSE, EXT_BRUTE]) {
    const chemin = join(dossier, `${court}${ext}`)
    if (existsSync(chemin)) return chemin
  }
  return null
}

/** Les artefacts d'un dossier, un par match, triés par forme courte (ni marque de dérivation, ni en-tête). */
export function listerArtefacts(dossier: string): { court: string; chemin: string }[] {
  const parCourt = new Map<string, string>()
  for (const nom of readdirSync(dossier)) {
    const m = NOM_ARTEFACT.exec(nom)
    if (!m) continue
    if (parCourt.has(m[1]) && !m[2]) continue // le compressé gagne
    parCourt.set(m[1], join(dossier, nom))
  }
  return [...parCourt.entries()]
    .map(([court, chemin]) => ({ court, chemin }))
    .sort((a, b) => a.court.localeCompare(b.court))
}
