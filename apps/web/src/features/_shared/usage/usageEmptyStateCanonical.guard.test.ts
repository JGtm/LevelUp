/// <reference types="node" />
// @vitest-environment node
/**
 * Garde-rail (2026-09-22) : UN ÉTAT VIDE DES BLOCS D'USAGE SE DESSINE PAR `EmptyStateNotice`,
 * jamais par un cadre maison.
 *
 * Pourquoi : `UsageEmptyNotice` rendait un simple `<p className="text-sm
 * text-muted-foreground">`, et `FormesRetenuesSection` son propre `rounded-lg border-dashed
 * py-6`. Sur une rangée dont la carte voisine portait le gabarit canonique
 * (`components/ui/empty-state.tsx` : titre `text-sm font-semibold text-foreground` + phrase
 * `mt-1 text-sm text-muted-foreground` dans un cadre `rounded-xl border-dashed border-border
 * bg-muted/80 px-4 py-5 text-center`), la MÊME absence se lisait de trois façons — constat
 * utilisateur du 2026-09-22 sur « Contrôle des armes spéciales ».
 *
 * Ce que ça protège : la typographie de l'état vide se décide en UN endroit. Une carte qui
 * réintroduit son propre cadre pointillé ou sa propre ligne grise « juste ici » fait rougir
 * ce test — c'est exactement la re-divergence que CLAUDE.md n°6 décrit.
 *
 * Preuve de mordant (deux mutations, 2026-09-22), chacune sur UNE assertion distincte :
 *   - le `<p className="text-sm text-muted-foreground" data-usage-empty=...>` d'avant, remis
 *     dans `UsageEmptyNotice.tsx`, fait rougir « aucun cadre ni ligne maison » ;
 *   - le même `<p>` SANS classe (donc invisible pour la première) fait rougir « passe par
 *     EmptyStateNotice », qui exige l'IMPORT du gabarit, pas sa mention.
 * Le rendu par `EmptyStateNotice` repasse les deux au vert. Mutations non committées — seul
 * le garde-rail l'est.
 */
import { readdirSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'

import { describe, expect, it } from 'vitest'

const FEATURES_ROOT = resolve(process.cwd(), 'src', 'features')

/**
 * LE PÉRIMÈTRE : les composants qui rendent un état vide d'un bloc d'usage. Un fichier n'y
 * entre que s'il rend REELLEMENT un état vide — inutile d'exiger l'import canonique d'une
 * grille ou d'un modèle qui n'affiche jamais « rien à montrer ».
 */
const SCOPE: { dir: string; match: (name: string) => boolean }[] = [
  { dir: join(FEATURES_ROOT, '_shared', 'usage'), match: (n) => /\.tsx$/.test(n) },
  { dir: join(FEATURES_ROOT, 'squad'), match: (n) => /Usages.*\.tsx$/.test(n) },
  { dir: join(FEATURES_ROOT, 'squad', 'formes'), match: (n) => /\.tsx$/.test(n) },
]

/**
 * LES MARQUEURS D'UN ÉTAT VIDE RENDU SUR PLACE. `EmptyStateNotice` est le seul autorisé à
 * les porter : dans le périmètre, ils signent un cadre ou une ligne recopiés.
 *
 *  - un cadre pointillé posé en bloc (`border-dashed` + un `rounded-*`) ;
 *  - un `data-testid` ou `data-*` d'état vide sur un élément qui porte aussi sa typographie
 *    (`text-muted-foreground` sur la MÊME balise).
 */
const HOME_MADE_FRAME = /className="[^"]*\brounded-(?:lg|xl|md)\b[^"]*\bborder-dashed\b[^"]*"/
const HOME_MADE_LINE =
  /className="[^"]*\btext-muted-foreground\b[^"]*"[^>]*\bdata-(?:testid="[a-z-]*empty|usage-empty|formes-empty)/

/** Un fichier compte comme « rendant un état vide » s'il en nomme un. */
const RENDERS_EMPTY = /data-usage-empty|data-formes-empty|data-testid="[a-z-]*empty|EmptyStateNotice/

/** L'import du gabarit canonique, et celui du relais du bloc usage. */
const IMPORTS_CANONICAL = /import\s*\{[^}]*\bEmptyStateNotice\b[^}]*\}\s*from\s*'@\/components\/ui\/empty-state'/
const IMPORTS_DELEGATE = /import\s*\{[^}]*\bUsageEmptyNotice\b[^}]*\}\s*from\s*'[^']*UsageEmptyNotice'/

function filesOf(dir: string, match: (name: string) => boolean): string[] {
  const out: string[] = []
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    if (entry.isFile() && match(entry.name) && !/\.test\.tsx?$/.test(entry.name)) {
      out.push(join(dir, entry.name))
    }
  }
  return out
}

const CANDIDATES = SCOPE.flatMap((s) => filesOf(s.dir, s.match))

describe("garde-rail état vide canonique des blocs d'usage", () => {
  it('le périmètre est non vide (le garde-rail scanne bien quelque chose)', () => {
    expect(CANDIDATES.length).toBeGreaterThan(0)
  })

  it("aucun bloc d'usage ne dessine son propre cadre ou sa propre ligne d'état vide", () => {
    const offenders: string[] = []
    for (const f of CANDIDATES) {
      const text = readFileSync(f, 'utf8')
      if (HOME_MADE_FRAME.test(text) || HOME_MADE_LINE.test(text)) {
        offenders.push(f.replace(FEATURES_ROOT, 'src/features'))
      }
    }
    expect(
      offenders,
      "état vide dessiné sur place — passer par `EmptyStateNotice` (@/components/ui/empty-state) " +
        `plutôt qu'un cadre ou une ligne locale : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  it("tout fichier du périmètre qui rend un état vide passe par EmptyStateNotice", () => {
    const offenders: string[] = []
    for (const f of CANDIDATES) {
      const text = readFileSync(f, 'utf8')
      if (!RENDERS_EMPTY.test(text)) continue
      // L'IMPORT, pas la mention : `UsageEmptyNotice.tsx` nomme `UsageEmptyNotice` en
      // déclarant sa propre fonction — s'en contenter laisserait passer précisément le
      // fichier que ce garde-rail doit tenir. Rendre l'état vide en DÉLÉGUANT à
      // `UsageEmptyNotice` reste le comportement voulu : c'est lui qui appelle
      // `EmptyStateNotice` pour tout le bloc.
      if (IMPORTS_CANONICAL.test(text) || IMPORTS_DELEGATE.test(text)) continue
      offenders.push(f.replace(FEATURES_ROOT, 'src/features'))
    }
    expect(
      offenders,
      "état vide rendu hors du gabarit canonique — importer `EmptyStateNotice` " +
        `(ou déléguer à \`UsageEmptyNotice\`) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
