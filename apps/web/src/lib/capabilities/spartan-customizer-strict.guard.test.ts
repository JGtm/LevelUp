/**
 * Garde-rail (V72-29) — la capability `spartan_customizer` se gate en FAIL-CLOSED.
 *
 * `spartan_customizer` déclenche des VISUELS et affordances propres à un titre (emblème /
 * nameplate recolorisés, modale de personnalisation Halo 5). Le fail-open de
 * {@link useCapability} (true quand `availableTitles` est vide / titre non résolu) rend ces
 * visuels pendant la fenêtre transitoire de re-bootstrap au switch de titre → synthèse
 * d'assets Halo 5 sur Halo Infinite (fuite cross-titre observée V72-29). SEUL
 * {@link useCapabilityStrict} (fail-CLOSED) est autorisé pour cette capability ; de même,
 * le prédicat hors-hook fail-open `hasCapabilityIn(..., 'spartan_customizer')` est interdit.
 *
 * Ce test échoue si un appel fail-open réapparaît. Pour allowlister un cas légitime
 * (aucun connu à ce jour) : ajouter son chemin exact à `ALLOWLIST` AVEC un commentaire
 * justifiant pourquoi le fail-open n'y expose aucun visuel/affordance d'un autre titre.
 */
import { describe, it, expect } from 'vitest'

// import.meta.glob (Vite) charge chaque source comme chaîne brute (cf. sortable-th.guard).
const sources = import.meta.glob('/src/**/*.{ts,tsx}', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>

const CAPABILITY = 'spartan_customizer'

/** Appels fail-open recherchés (le suffixe `Strict` ne matche pas : nom exact + `(`). */
const FAIL_OPEN_CALLS = ['useCapability(', 'hasCapabilityIn(']

/** Un identifiant ne peut pas être précédé de ces caractères sans changer de nom. */
const IDENT_CHAR = /[A-Za-z0-9_$.]/

/**
 * True si un appel à `callee` prend `CAPABILITY` en argument, quelle que soit la
 * FORME des arguments.
 *
 * Pourquoi un scan à parenthèses équilibrées plutôt qu'une regex (constat contre-revue
 * V72) : l'ancien motif `hasCapabilityIn\([^)]*['"]spartan_customizer['"]` s'arrêtait à
 * la PREMIÈRE parenthèse fermante — `hasCapabilityIn(useTitleCapabilities(), 'spartan_customizer')`
 * (un appel imbriqué en 1er argument, la forme la plus naturelle) le contournait donc
 * silencieusement. Élargir la classe de caractères ne marche pas non plus : le code du
 * repo n'utilise pas de point-virgule, un `[^;]{0,N}` déborderait sur les lignes
 * suivantes et signalerait à tort un `useCapabilityStrict` voisin. On suit donc la
 * profondeur de parenthèses jusqu'à la fermeture de l'appel, et on cherche le littéral
 * DANS ses arguments.
 */
function callPassesCapability(code: string, callee: string): boolean {
  let from = 0
  for (;;) {
    const at = code.indexOf(callee, from)
    if (at === -1) return false
    from = at + callee.length
    // Rejette les noms dont `callee` n'est qu'un suffixe (ex. `xuseCapability(`).
    if (at > 0 && IDENT_CHAR.test(code[at - 1])) continue

    let depth = 1
    let i = from
    while (i < code.length && depth > 0) {
      const c = code[i]
      if (c === '(') depth++
      else if (c === ')') depth--
      i++
    }
    const args = code.slice(from, depth === 0 ? i - 1 : code.length)
    if (args.includes(`'${CAPABILITY}'`) || args.includes(`"${CAPABILITY}"`)) return true
  }
}

/** Chemins autorisés à déroger (vide — aucun cas légitime connu). */
const ALLOWLIST = new Set<string>([])

describe('garde-rail spartan_customizer gate fail-closed (anti-fuite V72-29)', () => {
  it('aucun useCapability/hasCapabilityIn fail-open sur spartan_customizer (useCapabilityStrict seul)', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([path]) => !ALLOWLIST.has(path))
      .filter(([, code]) => FAIL_OPEN_CALLS.some((fn) => callPassesCapability(code, fn)))
      .map(([path]) => path)
    expect(
      offenders,
      `Gate fail-open sur spartan_customizer (utiliser useCapabilityStrict) : ${offenders.join(', ')}`,
    ).toEqual([])
  })

  // Le garde-rail est lui-même testé : un filet qui ne mord plus est pire qu'aucun.
  it('détecte les formes de contournement (appel imbriqué en argument, guillemets doubles)', () => {
    const cases = [
      `hasCapabilityIn(useTitleCapabilities(), '${CAPABILITY}')`,
      `hasCapabilityIn(caps(a, b(c)), "${CAPABILITY}")`,
      `useCapability('${CAPABILITY}')`,
      `const on = hasCapabilityIn(\n  resolveCaps(title),\n  '${CAPABILITY}',\n)`,
    ]
    for (const code of cases) {
      expect(
        FAIL_OPEN_CALLS.some((fn) => callPassesCapability(code, fn)),
        `non détecté : ${code}`,
      ).toBe(true)
    }
  })

  it('ne signale PAS les usages légitimes (Strict, autre capability, appel voisin)', () => {
    const cases = [
      `useCapabilityStrict('${CAPABILITY}')`,
      `hasCapabilityIn(caps, 'weapon_analysis')`,
      // Sans point-virgule : un `useCapabilityStrict` sur la ligne suivante ne doit
      // pas être imputé à l'appel fail-open qui le précède (faux positif d'une
      // approche « jusqu'au prochain ; »).
      `const a = hasCapabilityIn(caps, 'weapon_analysis')\nconst b = useCapabilityStrict('${CAPABILITY}')`,
      // Suffixe d'identifiant : nom différent, pas un appel au prédicat gardé.
      `myHasCapabilityIn(caps, '${CAPABILITY}')`,
    ]
    for (const code of cases) {
      expect(
        FAIL_OPEN_CALLS.some((fn) => callPassesCapability(code, fn)),
        `faux positif : ${code}`,
      ).toBe(false)
    }
  })
})
