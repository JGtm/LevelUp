/**
 * Module title-routing (D-10) — source UNIQUE de l'interprétation et de la
 * construction du titre (et de la langue) porté par l'URL, sous le namespace `/t/`.
 *
 * Rôle (PLAN_TITLE_SLUG_URL) : le titre actif devient un état EXPLICITE de l'URL ;
 * le store et le client API SUIVENT le segment (inversion de contrôle). Toute
 * lecture/construction d'un segment titre passe par ce module — verrouillé par le
 * garde-rail ratchet `no-title-literals.ratchet.test.ts`.
 *
 *  - parseRouteSegments / resolveTitleGate / buildLegacyRedirect : PURES, testées
 *    en TDD (D-11).
 *  - applyActiveTitle : UNIQUE fonction effectful (extraite de switchTitle, D-6).
 *  - initTitleFromLocation : câblage synchrone au boot (D-9).
 */
export { parseRouteSegments, type RouteSegments } from './parseRouteSegments'
export { resolveTitleGate, type TitleGate } from './resolveTitleGate'
export { buildLegacyRedirect, type LegacyRedirect } from './buildLegacyRedirect'
export { applyActiveTitle } from './applyActiveTitle'
export { initTitleFromLocation } from './initTitleFromLocation'
export { KNOWN_LOCALES, isKnownLocale, type Locale } from './locales'
