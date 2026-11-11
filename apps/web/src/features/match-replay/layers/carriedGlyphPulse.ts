/**
 * carriedGlyphPulse — LA PULSATION D'UN GLYPHE PORTÉ, écrite une fois.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, K4). Trois calques portent un objet SUR son
 * porteur — la couronne du VIP, le crâne d'Oddball, la bombe d'Assaut — et les trois
 * réécrivaient à l'identique la même respiration : quatre constantes et une sinusoïde, en md5
 * identique.
 *
 * CE QUE LA DUPLICATION RISQUAIT, exactement. Les trois glyphes relèvent de trois modes de jeu
 * distincts et ne co-occurrent jamais à l'écran : le danger n'était donc PAS une
 * désynchronisation visible, mais la DÉRIVE — un réglage d'oeil sur l'un des trois, jamais
 * reporté sur les deux autres, et trois respirations différentes pour un seul geste de
 * lecture, que personne ne peut comparer puisqu'on ne les voit jamais ensemble.
 *
 * CE QUE LA PULSATION DIT. Un port OUVERT (le joueur porte encore) respire ; un port FERMÉ —
 * un fait daté l'a clos, mort du porteur ou passation — est PLEIN. C'est la seule chose que
 * l'opacité raconte, et c'est pourquoi elle ne dépend que de `closed` et de l'image.
 *
 * MOUVEMENT RÉDUIT : la respiration s'arrête sur sa valeur MOYENNE, pas sur son minimum ni sur
 * son maximum — le glyphe reste lisible sans clignoter (`prefers-reduced-motion`).
 */

/** Opacité pleine : un fait daté ferme le port (mort du porteur, ou passation). */
export const CARRIED_ALPHA_SOLID = 0.95

/** Bornes de la pulsation d'un glyphe « ouvert » (`closed` faux), et sa période en millisecondes. */
export const CARRIED_PULSE_MIN = 0.42
export const CARRIED_PULSE_MAX = 0.78
export const CARRIED_PULSE_PERIOD_MS = 2600

/**
 * carriedGlyphAlpha rend l'opacité d'un glyphe porté à l'image donnée.
 *
 * La période est une DURÉE (`CARRIED_PULSE_PERIOD_MS`), convertie en images avec la durée réelle
 * d'une image du document (`frameMs`) : elle ne change pas avec la cadence du rejeu. `frame` reste
 * l'index de lecture : à vitesse double, la respiration double avec la scène, ce qui est le
 * comportement attendu d'un effet posé sur un porteur qui bouge lui aussi deux fois plus vite.
 */
export function carriedGlyphAlpha(
  closed: boolean,
  frame: number,
  reducedMotion: boolean,
  frameMs: number,
): number {
  if (closed) return CARRIED_ALPHA_SOLID
  if (reducedMotion) return (CARRIED_PULSE_MIN + CARRIED_PULSE_MAX) / 2
  const phase = (2 * Math.PI * frame) / (CARRIED_PULSE_PERIOD_MS / frameMs)
  return CARRIED_PULSE_MIN + (CARRIED_PULSE_MAX - CARRIED_PULSE_MIN) * (0.5 + 0.5 * Math.sin(phase))
}
