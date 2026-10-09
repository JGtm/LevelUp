/**
 * piedDePageAffiche — la coquille pose-t-elle le pied de page global sous la route courante ?
 *
 * Une route le refuse par sa donnée statique `piedDePage: false` (cf. `StaticDataRouteOption`,
 * `app/router/index.ts`) : c'est le cas des vues « cockpit » qui tiennent dans l'écran (onglet
 * Tactique), où le pied de page obligerait à défiler sous une vue faite pour ne pas défiler. Une
 * seule route de la chaîne suffit à le retirer.
 */
export function piedDePageAffiche(matches: ReadonlyArray<{ staticData?: { piedDePage?: boolean } }>): boolean {
  return !matches.some((m) => m.staticData?.piedDePage === false)
}
