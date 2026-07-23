import { isKnownLocale, type Locale } from '@/lib/i18n/locale'

export interface RouteSegments {
  /** Langue forcée par l'URL (segment `/{lang}/t/…`). Absente = locale session. */
  lang?: Locale
  /** Slug de titre porté par l'URL (`/t/{slug}/…`), VERBATIM (D-2, non validé ici). */
  titleSlug?: string
}

/**
 * parseRouteSegments — extrait langue + slug de titre d'un pathname, sous le
 * namespace `/t/`. Fonction PURE (D-10).
 *
 * Règles (D-1..D-3) :
 *  - `lang` n'est capturé QUE si le premier segment ∈ locales connues ET est
 *    immédiatement suivi d'un segment `t` — la langue ne vit que devant un scope
 *    de titre (D-3 : `/en/settings` ne capture PAS de langue).
 *  - `titleSlug` est capturé si un segment `t` (position 0, ou 1 après lang) est
 *    suivi d'un segment non vide, pris VERBATIM (D-2 : le backend/gate valide).
 *  - Trailing slash, segments vides et double slash tolérés (segments vides
 *    ignorés). Sans `t/{slug}` valide → objet vide (y compris `/en/t` sans slug :
 *    la langue seule, sans titre, n'est pas un scope valide).
 */
export function parseRouteSegments(pathname: string): RouteSegments {
  const segs = pathname.split('/').filter(Boolean)
  // Forme longue : /{lang}/t/{slug}/… — langue devant un scope titre uniquement.
  if (segs.length >= 3 && isKnownLocale(segs[0]) && segs[1] === 't') {
    return { lang: segs[0], titleSlug: segs[2] }
  }
  // Forme courte : /t/{slug}/… — langue implicite (session serveur).
  if (segs.length >= 2 && segs[0] === 't') {
    return { titleSlug: segs[1] }
  }
  return {}
}
