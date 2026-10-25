import { isKnownLocale, type Locale } from '@/lib/i18n/locale'
import { parseRouteSegments } from '@/lib/title-routing/parseRouteSegments'

/**
 * Langue de l'écran d'attente, décidée AVANT la réponse du /bootstrap (seule source de la
 * locale de session). Sans elle, « Chargement LevelUp… » et « Démarrage du serveur… »
 * s'affichaient toujours en français, y compris sur une instance anglaise (démo).
 *
 * Priorité : segment de langue de l'URL > dernière locale de session vue par ce navigateur
 * > langue du navigateur > français. Le bootstrap reste autoritaire : il ré-écrit la locale
 * dès qu'il répond.
 */
const STORAGE_KEY = 'levelup-last-locale'

/** Choix PUR de la langue d'attente. `languages` : `navigator.languages`, par ordre de préférence. */
export function resolveBootLocale(
  pathname: string,
  remembered: string | null,
  languages: readonly string[],
): Locale {
  const { lang } = parseRouteSegments(pathname)
  if (lang) return lang
  if (remembered && isKnownLocale(remembered)) return remembered
  for (const tag of languages) {
    const primary = tag.split('-')[0].toLowerCase()
    if (isKnownLocale(primary)) return primary
  }
  return 'fr'
}

/** Dernière locale de session mémorisée ; null si rien n'est lisible. */
export function readRememberedLocale(): string | null {
  try {
    return localStorage.getItem(STORAGE_KEY)
  } catch {
    return null
  }
}

/** Mémorise la locale de session pour le prochain écran d'attente ; un stockage indisponible n'interrompt rien. */
export function rememberLocale(locale: Locale): void {
  try {
    localStorage.setItem(STORAGE_KEY, locale)
  } catch {
    /* stockage indisponible (navigation privée, quota) : l'attente retombe sur la langue du navigateur */
  }
}
