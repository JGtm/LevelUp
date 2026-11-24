/**
 * observeSize — LE SEUL POINT OÙ UN OBSERVATEUR DE TAILLE EST POSÉ pour suivre une zone du rejeu : `onChange` est
 * rappelé à chaque changement de taille de `el`, et la fonction rendue arrête l'observation.
 *
 * L'observateur est OPTIONNEL : jsdom ne le fournit pas, et un environnement sans lui garde la mesure initiale que
 * l'appelant a déjà prise ; la fonction rendue ne fait alors rien. Les mesures qui s'en servent (largeur du corps du
 * cadre, largeur utile d'une zone, taille du plan, espace libre du terrain) ne diffèrent que par CE QU'ELLES LISENT
 * dans l'élément, jamais par la façon de le suivre : `resizeObserver.guard.test.ts` interdit d'en reposer un à côté.
 */
export function observeSize(el: Element, onChange: () => void): () => void {
  if (typeof ResizeObserver === 'undefined') return () => {}
  const observer = new ResizeObserver(onChange)
  observer.observe(el)
  return () => observer.disconnect()
}
