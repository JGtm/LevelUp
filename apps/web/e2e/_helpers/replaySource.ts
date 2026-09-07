/**
 * Helper E2E — RETROUVER UN MODULE DU REJEU PAR SON NOM.
 *
 * POURQUOI (2026-09-06, lot v2 D.11). Les deux specs de rasterisation transpilent des modules
 * de `features/match-replay/` pour les exécuter dans la page, et elles les désignaient par un
 * chemin ÉCRIT EN DUR (`.../match-replay/explosionFx.ts`). L'arborescence par responsabilité
 * (`layers/`, `ui/`, `model/`, `hooks/`, `sound/`, `export/`, `settings/`, `i18n/`) a déplacé
 * ces modules : le témoin visuel — le seul gate qui dise « aucun pixel n'a bougé » — tombait
 * sur un ENOENT à chaque déplacement.
 *
 * La spec nomme donc un FICHIER, pas un chemin, et ce module le cherche dans toute la feature.
 * Le nom d'un module de dessin est unique dans le rejeu ; l'ambiguïté lève plutôt que de
 * transpiler silencieusement le mauvais fichier.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const ICI = dirname(fileURLToPath(import.meta.url))

/** La racine du dépôt, depuis `apps/web/e2e/_helpers/`. */
export const REPO = resolve(ICI, '..', '..', '..', '..')

/** La feature du rejeu. */
export const FEATURE = resolve(REPO, 'apps/web/src/features/match-replay')

/**
 * Le modèle PARTAGÉ du rejeu (2026-09-06, v2 D.13) : le document, sa normalisation, sa logique
 * de lecture et le roster vivent dans `lib/replay/` depuis que la Match View les lit aussi.
 * Un module du rejeu peut donc vivre des deux côtés, et le témoin visuel doit les voir tous.
 */
export const PARTAGE = resolve(REPO, 'apps/web/src/lib/replay')

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) out.push(...walk(full))
    else if (/\.(ts|tsx)$/.test(entry.name)) out.push(full)
  }
  return out
}

/** Le chemin absolu du module du rejeu qui porte ce nom, où qu'il vive dans la feature. */
export function moduleDuRejeu(nom: string): string {
  const trouves = [...walk(FEATURE), ...walk(PARTAGE)].filter(
    (f) => f.split(/[\\/]/).pop() === nom,
  )
  if (trouves.length !== 1) {
    throw new Error(`replaySource: ${trouves.length} fichier(s) nommé(s) ${nom} dans le rejeu`)
  }
  return trouves[0]
}

/** Le contenu du module du rejeu qui porte ce nom. */
export function sourceDuRejeu(nom: string): string {
  return readFileSync(moduleDuRejeu(nom), 'utf8')
}
