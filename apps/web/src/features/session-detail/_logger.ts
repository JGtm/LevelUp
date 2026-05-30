/**
 * _logger.ts — Logger namespacé pour la feature Session (détail).
 *
 * Pattern aligné avec features/squad/_logger.ts :
 *  - chaque clé d'avertissement n'est logguée qu'une fois par session (dédup),
 *  - jamais de log dans un hot path de rendu : on logue depuis les `useMemo` de
 *    transformation des données (1 fois par changement de session), pas par frame.
 *
 * Cible les signaux de DÉGRADATION GRACIEUSE de la page : un graphe rendu vide
 * l'est presque toujours parce qu'une donnée sous-jacente manque (session social
 * sans MMR, vieux matchs sans dégâts, PSA non synchronisés…). Logguer la raison
 * rend ces cas OBSERVABLES (donnée absente) au lieu d'un « graphe cassé ».
 *
 * Clés émises (cf. usages) :
 *  - `mmr_missing:{session}`        — aucun match de la session n'a de MMR (dumbbell vide).
 *  - `damage_missing:{session}`     — aucun match n'a de dégâts (barre dégâts / OC-DR vides).
 *  - `placement_missing:{session}`  — aucun rang de placement (breakdown vide).
 *  - `lobby_size_fallback:{session}`— taille de lobby absente → axe = max(rang observé).
 *  - `participation_empty:{session}`— profil de participation absent de l'entry.
 *  - `offdef_missing:{session}`     — OC/DR absents (KPI Rendement/Résistance à "—").
 */

const PREFIX = '[session-detail]'

const _warned = new Set<string>()

export const log = {
  /** Avertissement dédupliqué (une fois par clé et par session). */
  warn(key: string, msg: string, ...args: unknown[]): void {
    if (_warned.has(key)) return
    _warned.add(key)
    console.warn(`${PREFIX} ${msg}`, ...args)
  },

  /** Réinitialise la déduplication (usage test uniquement). */
  _resetForTests(): void {
    _warned.clear()
  },
}
