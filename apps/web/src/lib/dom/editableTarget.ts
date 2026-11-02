/**
 * Une cible d'événement clavier où l'on saisit du texte (champ, zone de texte, liste déroulante, zone
 * éditable) : un raccourci global ne doit pas s'y déclencher. Source unique du test, partagée par le
 * rejeu et la barre latérale ; la cible d'un événement clavier peut être `window` ou `document`, sans
 * nom de balise ni propriété d'édition.
 */
export function isEditableTarget(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  if (!el) return false
  const tag = el.tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
  return el.isContentEditable === true
}
