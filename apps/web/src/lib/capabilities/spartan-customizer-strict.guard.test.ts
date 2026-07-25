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

// useCapability('spartan_customizer') NON-strict — `useCapabilityStrict(` ne matche pas
// (le token `Strict` s'intercale avant la parenthèse).
const FAIL_OPEN_HOOK = /useCapability\(\s*['"]spartan_customizer['"]/
// hasCapabilityIn(<caps>, 'spartan_customizer') — variante prédicat fail-open.
const FAIL_OPEN_PREDICATE = /hasCapabilityIn\([^)]*['"]spartan_customizer['"]/

/** Chemins autorisés à déroger (vide — aucun cas légitime connu). */
const ALLOWLIST = new Set<string>([])

describe('garde-rail spartan_customizer gate fail-closed (anti-fuite V72-29)', () => {
  it('aucun useCapability/hasCapabilityIn fail-open sur spartan_customizer (useCapabilityStrict seul)', () => {
    const offenders = Object.entries(sources)
      .filter(([path]) => !/\.test\.tsx?$/.test(path))
      .filter(([path]) => !ALLOWLIST.has(path))
      .filter(([, code]) => FAIL_OPEN_HOOK.test(code) || FAIL_OPEN_PREDICATE.test(code))
      .map(([path]) => path)
    expect(
      offenders,
      `Gate fail-open sur spartan_customizer (utiliser useCapabilityStrict) : ${offenders.join(', ')}`,
    ).toEqual([])
  })
})
