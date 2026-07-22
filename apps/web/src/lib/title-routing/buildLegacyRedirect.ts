export interface LegacyRedirect {
  href: string
}

/**
 * buildLegacyRedirect — mappe une URL legacy `/players/…` vers sa forme
 * title-préfixée `/t/{activeSlug}/players/…`. Fonction PURE (D-5).
 *
 * Deux transformations combinées EN UN SEUL saut :
 *  1. Préfixe titre : `/players/…` → `/t/{activeSlug}/players/…`.
 *  2. Remaps internes legacy déjà en place AVANT D7 (reproduits VERBATIM depuis
 *     routes/players/$playerSlug/* — cf. remapLegacySuffix).
 * Suffixe + `?search` (avec son `?`) + `#hash` (avec son `#`) re-concaténés
 * verbatim. Un pathname hors `/players` → null. `/players` sans slug → null
 * (aucune route index legacy). `/players/{slug}` sans suffixe → `.../home`.
 */
export function buildLegacyRedirect(
  pathname: string,
  searchStr: string,
  hash: string,
  activeSlug: string,
): LegacyRedirect | null {
  const segs = pathname.split('/').filter(Boolean)
  if (segs[0] !== 'players') return null
  const playerSlug = segs[1]
  if (!playerSlug) return null // bare /players : pas de route index legacy → pas de cible

  const suffix = segs.slice(2)
  const targetSuffix = suffix.length === 0 ? ['home'] : remapLegacySuffix(suffix)
  const tail = targetSuffix.length ? `/${targetSuffix.join('/')}` : ''
  const path = `/t/${activeSlug}/players/${playerSlug}${tail}`
  return { href: path + suffixSearchHash(searchStr, hash) }
}

/**
 * Remaps internes legacy, reproduits VERBATIM des redirections existantes
 * (routes/players/$playerSlug/{objectifs,palmares,compare,synthesis,citations,
 * commendations}). Appliqués sur le PREMIER segment du suffixe uniquement.
 */
function remapLegacySuffix(suffix: string[]): string[] {
  const [first, ...rest] = suffix
  switch (first) {
    case 'objectifs': // objectifs/ → ascension/objectifs
      return ['ascension', 'objectifs', ...rest]
    case 'palmares': // palmares(/*) → community(/*)
      return ['community', ...rest]
    case 'compare': // compare → community/compare
      return ['community', 'compare', ...rest]
    case 'synthesis': // synthesis → stats/synthesis
      return ['stats', 'synthesis', ...rest]
    case 'citations': // citations → career/citations
      return ['career', 'citations', ...rest]
    case 'commendations': // commendations → career/commendations
      return ['career', 'commendations', ...rest]
    default:
      return suffix
  }
}

/** Re-concatène search + hash VERBATIM, en garantissant leur marqueur (`?` / `#`). */
function suffixSearchHash(searchStr: string, hash: string): string {
  let out = ''
  if (searchStr && searchStr !== '?') {
    out += searchStr.startsWith('?') ? searchStr : `?${searchStr}`
  }
  if (hash && hash !== '#') {
    out += hash.startsWith('#') ? hash : `#${hash}`
  }
  return out
}
