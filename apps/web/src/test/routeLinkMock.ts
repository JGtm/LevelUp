/**
 * Résolution d'un template de route TanStack (`to` + `params`) en pathname concret,
 * pour les mocks de `<Link>` dans les tests de composants qui rendent des liens
 * title-scoped (`/{-$lang}/t/$titleSlug/players/$playerSlug/…`).
 *
 * Reproduit la substitution minimale du routeur : params de chemin + omission du
 * segment optionnel de langue quand `lang` n'est pas fourni. Permet d'asserter sur
 * l'URL RÉELLE rendue (avec préfixe titre) plutôt que sur le template.
 */
export function resolveRoutePath(
  to: string,
  params?: Record<string, string | undefined>,
): string {
  let href = to
  if (params) {
    href = href
      .replace('$titleSlug', params.titleSlug ?? '')
      .replace('$playerSlug', params.playerSlug ?? '')
      .replace('$matchId', params.matchId ?? '')
  }
  // Segment optionnel {-$lang} : présent → /{lang}, absent → retiré (URL /t/…).
  href = params?.lang ? href.replace('{-$lang}', params.lang) : href.replace('/{-$lang}', '')
  return href
}
