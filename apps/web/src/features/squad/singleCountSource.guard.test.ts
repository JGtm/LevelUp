/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (ADR 0033, chantier A2 — CLAUDE.md n°6 « ≤ 2 copies ») : SOURCE
 * UNIQUE du compte de matchs d'une session escouade.
 *
 * Créé 2026-09-09. Défaut mesuré : la pastille du rail L2 et son « · N matchs »
 * venaient de `/filters/resolve` (population du JOUEUR PRINCIPAL, aveugle à la
 * composition sélectionnée) alors que le SessionMultiSelect venait de
 * `composition_sessions` (population RÉELLE de la page) — 7 côté rail, 4 côté
 * page sur une même session. `squadSessionCounts.ts` (squadSessionCount /
 * squadSessionShownCount) est désormais la SEULE porte d'entrée : ce test
 * échoue si un AUTRE fichier de `features/squad/` lit `total_matches_after_filters`
 * ou `session_options` pour en tirer un compte de matchs — la 2e occurrence du
 * même défaut (le sélecteur avait déjà été corrigé sans le rail, `3862ff083`)
 * ne doit plus être possible qu'en dupliquant explicitement ce garde-rail.
 *
 * Allowlist VIDE et datée (2026-09-09) — un seul fichier exclu du scan, pas par
 * exception mais par CONCEPTION : `squadSessionCounts.ts` est le module
 * canonique lui-même, celui qui a le droit (et le devoir) de savoir comment le
 * repli `/filters/resolve` est construit. Aucun autre fichier ne doit
 * apparaître ici — l'ajouter exigerait une nouvelle justification datée.
 */
import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

// Module canonique — seul autorisé à connaître la forme de session_options /
// total_matches_after_filters pour en tirer un compte de matchs escouade.
// Le garde-rail lui-même est aussi exclu du scan : il NOMME ces littéraux en
// prose (docblock + message d'échec), ce qui n'est pas une lecture du champ.
const CANONICAL_MODULE = 'squadSessionCounts.ts'
const SELF = 'singleCountSource.guard.test.ts'

// Littéraux interdits : lire ces champs pour EN TIRER UN COMPTE DE MATCHS
// escouade réintroduit la double source (rail vs page).
const FORBIDDEN_PATTERNS = [/total_matches_after_filters/, /session_options/]

function walk(dir: string): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name)
    if (entry.isDirectory()) {
      out.push(...walk(full)) // features/squad/ a des sous-dossiers (charts/, components/, v2/)
    } else if (/\.(ts|tsx)$/.test(entry.name) && entry.name !== CANONICAL_MODULE && entry.name !== SELF) {
      out.push(full)
    }
  }
  return out
}

describe('garde-rail source unique du compte de session escouade (ADR 0033)', () => {
  const squadRoot = resolve(process.cwd(), 'src', 'features', 'squad')
  const files = walk(squadRoot)

  it('aucun fichier hors squadSessionCounts.ts ne lit total_matches_after_filters ni session_options', () => {
    const offenders = files
      .filter((f) => FORBIDDEN_PATTERNS.some((re) => re.test(readFileSync(f, 'utf8'))))
      .map((f) => f.replace(squadRoot, 'src/features/squad'))
    expect(
      offenders,
      `compte de matchs escouade : passer par squadSessionCount()/squadSessionShownCount() ` +
        `(./squadSessionCounts) plutôt que de relire /filters/resolve directement : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
