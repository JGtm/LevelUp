/**
 * _logger.ts — Logger namespacé pour le shell de navigation (NavL1 + menus mobiles).
 *
 * Pattern aligné avec features/feedback-drawer/_logger.ts et features/squad/_logger.ts :
 * - warn/error dédupliqués une fois par session (pas de spam console),
 * - debug actif uniquement en dev,
 * - jamais dans un hot path de rendu (le shell ne logue que sur événement
 *   utilisateur ou échec).
 *
 * Les warn/error transitent par console.* → captés par le buffer global
 * (lib/global-capture) et donc joignables aux issues feedback.
 *
 * Cibles observables (clés stables) :
 *  - `logout:failed` — la mutation de déconnexion (menu mobile) a échoué
 *  - `nav:menu_open` / `nav:actions_open` — ouverture des tiroirs mobiles (debug)
 *  - `nav:tool_open:{tool}` — lancement d'un outil latéral depuis le menu (debug)
 */

const PREFIX = '[shell-nav]'

const _warned = new Set<string>()
const _errored = new Set<string>()

export const log = {
  warn(key: string, msg: string, ...args: unknown[]): void {
    if (_warned.has(key)) return
    _warned.add(key)
    console.warn(`${PREFIX} ${msg}`, ...args)
  },

  error(key: string, msg: string, ...args: unknown[]): void {
    if (_errored.has(key)) return
    _errored.add(key)
    console.error(`${PREFIX} ${msg}`, ...args)
  },

  debug(msg: string, ...args: unknown[]): void {
    if (!import.meta.env.DEV) return
    console.debug(`${PREFIX} ${msg}`, ...args)
  },

  /** Réinitialise la déduplication (usage test uniquement). */
  _resetForTests(): void {
    _warned.clear()
    _errored.clear()
  },
}
