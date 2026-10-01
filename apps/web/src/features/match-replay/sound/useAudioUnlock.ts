/**
 * useAudioUnlock — LE LECTEUR S'OUVRE AU PREMIER GESTE SUR LA PAGE, QUEL QU'IL SOIT, quand la
 * préférence est à « activé » (item 12 du plan backlog du 2026-09-26, décision D-10).
 *
 * LE DÉFAUT MESURÉ (A2.0, Chromium et Firefox, politique d'autoplay par défaut). Seuls
 * « Lecture », « Recommencer » et le bouton du son ouvraient le lecteur. Avec la lecture
 * automatique, la lecture démarre sans aucun de ces gestes : le rejeu restait muet, clic
 * ailleurs sur la page ou pas, jusqu'à ce qu'on touche au bouton du son. En passant d'un rejeu
 * à un autre sans rechargement, même silence, alors que le document avait déjà reçu un geste.
 *
 * DEUX RÈGLES, et la politique d'autoplay des navigateurs n'est jamais contournée :
 *  - le document a DÉJÀ reçu un geste (`navigator.userActivation.hasBeenActive`, cas du passage
 *    d'un rejeu à un autre) : le lecteur s'ouvre dès l'affichage. Chromium et Firefox autorisent
 *    un contexte audio sur une activation acquise ;
 *  - sinon, un écouteur unique attend le premier geste sur le document, ouvre le lecteur, et se
 *    retire. Jamais de son sans geste préalable.
 *
 * `click` ET `keyup`, PAS `pointerdown` NI `keydown`, ET C'EST UNE CONDITION DU CORRECTIF DU
 * 2026-08-27. Le premier clic sur le bouton du son doit ACTIVER. Or `pointerdown` précède le
 * `click` du bouton, et la touche M agit sur `keydown`, écoutée par la fenêtre APRÈS le document.
 * Un écouteur sur ces deux événements ouvrirait donc le lecteur avant la bascule, qui le
 * trouverait vivant et COUPERAIT. Sur `click` et `keyup`, le document passe après le bouton et
 * après le raccourci. L'activation, elle, est acquise dès `pointerdown` ou `keydown` : le
 * contexte né dans `click` ou `keyup` démarre en marche (mesuré au même A2.0).
 *
 * `open` est `wake` : sans effet si la préférence est « coupé » ou si le lecteur vit déjà.
 */
import { useEffect, useRef } from 'react'

/** Les gestes qui ouvrent le lecteur : voir l'en-tête pour le choix des événements. */
const UNLOCK_EVENTS = ['click', 'keyup'] as const

export function useAudioUnlock(wanted: boolean, open: () => void): void {
  // `open` LU PAR UNE REF : il change d'identité avec les réglages du lecteur, et l'écouteur ne
  // doit pas être retiré puis reposé à chaque rendu pour autant.
  const openRef = useRef(open)
  useEffect(() => { openRef.current = open }, [open])
  useEffect(() => {
    if (!wanted) return
    if (navigator.userActivation?.hasBeenActive) {
      openRef.current()
      return
    }
    const onGesture = () => {
      for (const type of UNLOCK_EVENTS) document.removeEventListener(type, onGesture)
      openRef.current()
    }
    for (const type of UNLOCK_EVENTS) document.addEventListener(type, onGesture)
    return () => {
      for (const type of UNLOCK_EVENTS) document.removeEventListener(type, onGesture)
    }
  }, [wanted])
}
