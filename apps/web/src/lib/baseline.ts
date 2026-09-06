/**
 * lib/baseline — L'ÉCART À L'HABITUEL, et la seule condition qui le rend muet.
 *
 * Ces deux helpers vont ensemble : l'un met en forme un écart de TAUX en points
 * signés, l'autre dit quand cet écart est nul par construction et ne doit donc pas
 * s'afficher. Les séparer, c'est afficher « ±0 pts » sur tout l'historique.
 *
 * POURQUOI ILS VIVENT DANS `lib/` ET PLUS DANS `features/explorer/` (2026-09-06,
 * phase 3 du plan tactique). Le bandeau de briefing de l'Explorateur les a écrits ;
 * la page Escouade en a besoin pour son KPI d'échange (« +7 pts vs habituel ») et
 * pour son « cap du moment ». Un import `squad -> explorer` de plus aurait dépassé
 * le plafond du ratchet `tools/lint-cross-feature-imports.mjs` (7, atteint), et une
 * COPIE aurait donné deux définitions du même écart — exactement ce que la règle des
 * ≤ 2 copies interdit. Ils sont donc DÉPLACÉS, et l'Explorateur pointe ici.
 *
 * `signOf` et `deltaToken` restent dans `ExplorerBriefing.logic` : ils n'ont qu'un
 * consommateur, et `deltaToken` porte son propre garde-rail
 * (`explorerDeltaToken.guard.test.ts`).
 */

/**
 * Formate un delta de TAUX (ratio 0..1) en points de pourcentage signés
 * (ex. +0.30 → "+30 pts"). Unité « pts » pour distinguer d'un pourcentage absolu.
 *
 * NE PAS CONFONDRE avec `formatSignedFixed` (@/lib/formatters), qui formate un delta
 * à décimales fixes : celui-ci arrondit à l'entier et suffixe l'unité.
 */
export function formatSignedPoints(v: number | null | undefined): string {
  if (v == null || !Number.isFinite(v)) return ''
  const pts = Math.round(v * 100)
  if (pts === 0) return '±0 pts'
  return pts > 0 ? `+${pts} pts` : `−${Math.abs(pts)} pts`
}

/**
 * Vrai quand le périmètre affiché couvre TOUT l'historique (aucun filtre narrowing).
 *
 * Le périmètre est toujours un SOUS-ENSEMBLE de la référence (un filtre ne peut que
 * rétrécir) : des cardinalités égales impliquent donc des ensembles identiques, d'où
 * des deltas « vs habituel » nuls par construction — à masquer. Faux si la référence
 * est absente (`undefined !== number`) : sans référence il n'y a de toute façon aucun
 * delta à afficher.
 */
export function isFullHistoryScope(
  scopeMatches: number | null | undefined,
  baselineMatches: number | null | undefined,
): boolean {
  return scopeMatches != null && baselineMatches === scopeMatches
}
