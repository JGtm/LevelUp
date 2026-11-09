/**
 * Le focus d'une cible vient-il du clavier ? Vrai quand la cible porte `:focus-visible`, que le navigateur
 * pose pour une navigation au clavier (Tab, flèches) et refuse à un clic de souris ou à un focus rendu par
 * programme après une fermeture. Une infobulle ouverte au focus ne doit s'afficher que dans ce cas :
 * sinon un volet qui rend le focus à son déclencheur fait apparaître une infobulle que personne n'a demandée.
 *
 * Repli `true` quand le navigateur ne connaît pas le sélecteur (il lève) ou que la cible n'est pas un
 * élément : l'infobulle reste alors disponible au focus, comme avant ce filtre.
 */
export function isKeyboardFocus(target: EventTarget | null): boolean {
  if (!(target instanceof Element)) return true
  try {
    return target.matches(':focus-visible')
  } catch {
    // Sélecteur inconnu : repli documenté ci-dessus, aucune erreur à remonter.
    return true
  }
}
