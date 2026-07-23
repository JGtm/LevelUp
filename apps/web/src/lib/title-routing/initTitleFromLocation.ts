import { setApiTitleSlug, setApiLocale } from '@/lib/api/client'
import { parseRouteSegments } from './parseRouteSegments'

/**
 * initTitleFromLocation — câblage SYNCHRONE au boot (D-9, trou n°2). Appelé au
 * point d'entrée de l'app AVANT la première requête : pose le titre (et la langue)
 * sur le client API depuis le segment d'URL, quand présent. Ainsi X-LevelUp-Title
 * est affirmé dès le premier fetch pour les pages title-scoped (routes déplacées
 * en Phase 2), fermant la fenêtre de course « titre implicite ».
 *
 * Aucun test sur la VALEUR du slug (D-2, verbatim ; le backend valide). No-op
 * runtime tant qu'aucune route ne porte de segment (Phase 1) — câblage prêt.
 */
export function initTitleFromLocation(pathname: string): void {
  const { lang, titleSlug } = parseRouteSegments(pathname)
  if (titleSlug) setApiTitleSlug(titleSlug)
  if (lang) setApiLocale(lang)
}
