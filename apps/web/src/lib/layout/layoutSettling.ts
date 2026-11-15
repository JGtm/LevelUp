/**
 * Transition de mise en page en cours : tant que la largeur d'une colonne s'anime (volet ancré qui
 * s'ouvre ou se ferme), les graphes qu'elle contient ne se redimensionnent pas à chaque image, ce qui
 * les faisait se redessiner sans cesse. Celui qui lance la transition la déclare (`beginLayoutTransition`),
 * la fin de l'animation la clôt (`endLayoutTransition`), un délai de sécurité la clôt d'office si
 * l'événement de fin n'arrive pas (mouvement réduit, élément retiré en cours de route). Les observateurs
 * de taille consultent `isLayoutSettling` et se resynchronisent une fois à `onLayoutSettled`.
 */

/** Durée de la transition de largeur du volet ancré, en millisecondes : source unique (CSS et délai de sécurité). */
export const DOCK_TRANSITION_MS = 200

/** Marge ajoutée à la durée avant de clore d'office une transition dont la fin n'a pas été signalée. */
const SETTLE_SAFETY_MS = 100

let settling = false
let safetyTimer: ReturnType<typeof setTimeout> | null = null
const listeners = new Set<() => void>()

function clearSafetyTimer(): void {
  if (safetyTimer !== null) clearTimeout(safetyTimer)
  safetyTimer = null
}

/** Une transition de mise en page est-elle en cours ? */
export function isLayoutSettling(): boolean {
  return settling
}

/** Déclare une transition ; une seconde déclaration en cours prolonge le délai de sécurité. */
export function beginLayoutTransition(): void {
  settling = true
  clearSafetyTimer()
  safetyTimer = setTimeout(endLayoutTransition, DOCK_TRANSITION_MS + SETTLE_SAFETY_MS)
}

/** Clôt la transition et prévient les observateurs, une seule fois. Sans transition en cours : sans effet. */
export function endLayoutTransition(): void {
  if (!settling) return
  settling = false
  clearSafetyTimer()
  for (const listener of listeners) listener()
}

/** S'abonne à la fin d'une transition ; renvoie la fonction qui se désabonne. */
export function onLayoutSettled(listener: () => void): () => void {
  listeners.add(listener)
  return () => {
    listeners.delete(listener)
  }
}
