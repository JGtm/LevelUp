/**
 * _logger.ts — Logger namespacé pour la feature media (lecteur HLS CoverFlowModal).
 *
 * Pattern aligné avec features/feedback-drawer/_logger.ts et features/squad/_logger.ts :
 *   - chaque clé warn/error n'est logguée qu'une fois par session (pas de spam
 *     si un clip rejoue en boucle),
 *   - debug actif uniquement en dev,
 *   - jamais dans un hot path de rendu (uniquement sur événement hls.js ou
 *     interaction utilisateur).
 *
 * Les warn/error transitent par console.* → captés par le buffer global
 * (lib/global-capture) et donc joignables aux issues feedback.
 *
 * Cibles observables (clés stables) :
 *   - `hls:unsupported` — navigateur sans MSE ni lecture HLS native → clip illisible
 *   - `hls:fatal`       — erreur fatale du flux HLS (hls.js Events.ERROR)
 *   - `video:decode_error` — erreur de lecture d'un fichier vidéo natif (mp4/webm/mov)
 *   - `audio:tracks`    — pistes audio reçues (debug : layout détecté + noms)
 *   - `audio:toggle`    — bascule d'un interrupteur Jeu/Voix (debug)
 */

const PREFIX = '[media]'

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
