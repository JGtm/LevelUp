/**
 * featureFiles.ts — CE QUE « TOUS LES FICHIERS DU REJEU » VEUT DIRE, une fois pour toutes.
 *
 * # POURQUOI CE MODULE EXISTE (2026-09-06, lot v2 D.11)
 *
 * Onze garde-rails de cette feature cherchent un motif « partout sauf ici » : une seule
 * pulsation de glyphe porté, un seul foyer pour les camps du match, un seul accès à
 * localStorage, un seul formateur d'horloge… Tous balayaient `readdirSync(__dirname)`, ce qui
 * marchait tant que la feature était UN dossier plat de 370 fichiers.
 *
 * L'arborescence par responsabilité (`layers/`, `ui/`, `model/`, `hooks/`, `sound/`, `export/`,
 * `settings/`, `i18n/`) casse cette prémisse SANS RIEN DIRE : un garde resté sur `__dirname`
 * continue de passer au vert en ne regardant plus qu'un huitième du rejeu. C'est le pire mode
 * de défaillance d'un garde-rail — vert et inerte. Ce module rend donc la LISTE COMPLÈTE,
 * récursive, quelle que soit la profondeur d'où on l'appelle.
 *
 * # CE QU'IL NE FAIT PAS
 *
 * Il ne lit aucun motif et ne décide de rien : chaque garde garde sa propre signature de
 * défaut et sa propre liste d'exemptions nommées. Ce module ne répond qu'à « quels fichiers ».
 */
/// <reference types="node" />
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'

/** La racine de la feature : le parent de `test/`, où que vive l'appelant. */
export function featureRoot(): string {
  return resolve(dirname(__filename), '..')
}

/**
 * `apps/web/`. Plusieurs gardes lisent la feuille de style (`src/styles/globals.css`) ou la
 * route du rejeu : ils comptaient leurs `..` depuis `__dirname`, ce qu'un déplacement d'un
 * seul dossier suffit à fausser. L'ancre est ici, et elle ne se compte qu'une fois.
 */
export function racineWeb(): string {
  return resolve(featureRoot(), '..', '..', '..')
}

/** La racine du dépôt : les gardes qui lisent un fichier Go ou un manifeste TOML partent d'ici. */
export function racineDuDepot(): string {
  return resolve(racineWeb(), '..', '..')
}

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === 'node_modules') continue
      out.push(...walk(full))
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      out.push(full)
    }
  }
  return out
}

/** Tous les fichiers TypeScript de la feature, tests compris, chemins absolus. */
export function tousLesFichiers(): string[] {
  return walk(featureRoot())
}

/**
 * Les fichiers TypeScript d'un dossier QUELCONQUE, récursivement. Deux gardes du rejeu
 * étendent leur balayage à `src/routes/` — la page de rejeu vit là-bas, et un défaut qui la
 * quitterait pour la route ne serait pas moins un défaut.
 */
export function fichiersSous(dossier: string): string[] {
  return walk(dossier)
}

/** Les fichiers de SOURCE : ce que le produit exécute, hors tests. */
export function sourcesDeLaFeature(): string[] {
  return tousLesFichiers().filter((f) => !/\.test\.(ts|tsx)$/.test(f))
}

/** Les fichiers de TEST de la feature (gardes compris). */
export function testsDeLaFeature(): string[] {
  return tousLesFichiers().filter((f) => /\.test\.(ts|tsx)$/.test(f))
}

/** Le nom de fichier seul — la forme sous laquelle les gardes nomment leurs exemptions. */
export function nomDe(chemin: string): string {
  return chemin.split(/[\\/]/).pop() ?? ''
}

/**
 * Le chemin RELATIF à la racine de la feature (`layers/vipCrownLayer.ts`) : ce qu'un garde
 * affiche quand il échoue, pour que le message situe le fautif dans l'arborescence.
 */
export function cheminCourt(chemin: string): string {
  return chemin.slice(featureRoot().length + 1).split('\\').join('/')
}

/**
 * Le fichier de la feature qui porte ce NOM, où qu'il vive. Les gardes s'en servent pour leur
 * contre-test (« et le foyer canonique porte bien la formule ») : ils nomment un fichier, pas
 * un chemin, et survivent donc au prochain déplacement. Lève si le nom est introuvable ou
 * ambigu — les deux sont des erreurs de garde, pas des cas à ignorer.
 */
export function fichierNomme(nom: string): string {
  const trouves = tousLesFichiers().filter((f) => nomDe(f) === nom)
  if (trouves.length !== 1) {
    throw new Error(`featureFiles: ${trouves.length} fichier(s) nomme(s) ${nom} dans le rejeu`)
  }
  return trouves[0]
}

/** Le contenu d'un fichier, en UTF-8. */
export function lire(chemin: string): string {
  return readFileSync(chemin, 'utf8')
}
