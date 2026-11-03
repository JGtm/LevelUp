/**
 * Gabarits de texte : remplacement des `{nom}` d'un libellé par leurs valeurs. SEULE définition du motif
 * `{nom}` du front (garde-rail `template.guard.test.ts` : aucune autre copie du motif hors de `lib/i18n/`).
 */

const PLACEHOLDER = /\{(\w+)\}/g

/**
 * Remplace chaque `{nom}` de `template` par ce que rend `resolve(nom)`. `resolve` rend `null` ou `undefined`
 * pour laisser `{nom}` tel quel ; toute autre chaîne (la chaîne vide comprise) remplace le motif. Sert aux
 * gabarits qui ont une règle propre (le `{plural}` d'Ascension) ; les autres passent par [fillTemplate].
 */
export function substitute(template: string, resolve: (name: string) => string | null | undefined): string {
  return template.replace(PLACEHOLDER, (whole, name: string) => resolve(name) ?? whole)
}

/**
 * Remplace les `{nom}` d'un texte par les valeurs de `params` (nombres et chaînes affichés tels quels). Un nom
 * absent de `params`, ou porté par `null` / `undefined`, reste tel quel.
 */
export function fillTemplate(template: string, params?: Record<string, unknown>): string {
  if (!params) return template
  return substitute(template, (name) => {
    const v = params[name]
    return v == null ? undefined : String(v)
  })
}
