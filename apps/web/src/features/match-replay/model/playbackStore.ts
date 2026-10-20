/**
 * playbackStore — LA POSITION DE LECTURE PUBLIÉE, HORS DE L'ARBRE REACT.
 *
 * POURQUOI CE MODULE EXISTE (registre 2026-09-05, W1). La même grandeur — « quelle image le
 * rejeu montre-t-il ? » — existait en TROIS exemplaires écrits séparément :
 *
 *   1. une cellule `frameRef` que la boucle de dessin avance à la cadence de l'écran, lue par
 *      douze modules (calques survolables, chaleur, capture, export…) ;
 *   2. une COPIE dans un état React de la route, poussée par un `onFrameChange` bridé à
 *      150 ms, d'où vivaient les fiches, le bandeau de score et les écrans de fin ;
 *   3. le curseur de la frise et le texte de l'horloge, écrits directement dans le DOM.
 *
 * La deuxième était la seule à être un DOUBLON : une copie React d'une grandeur que personne
 * ne modifiait depuis React, poussée par une prop de rappel que la route devait fournir au
 * canvas pour recevoir en retour ce que le canvas savait déjà. C'est elle que ce module
 * remplace.
 *
 * # UNE SOURCE, UNE PUBLICATION — ET POURQUOI CE N'EST PAS DEUX VÉRITÉS
 *
 * La SOURCE est la cellule de dessin (`frameRef` du canvas) : la boucle l'avance à la cadence
 * de l'écran et tout ce qui se recalcule DANS un tracé la lit directement, sans rendu.
 *
 * La PUBLICATION est ce magasin : la dernière image que le canvas a DÉCIDÉ de faire savoir au
 * DOM qui l'entoure. Elle ne peut pas être la lecture directe de la cellule, et ce n'est pas
 * un choix de confort : `useSyncExternalStore` exige un instantané STABLE entre deux
 * notifications — rendre la cellule vivante ferait lire à React une valeur différente à chaque
 * appel, ce qu'il signale comme une boucle. Le magasin RETIENT donc la valeur publiée, et
 * c'est exactement ce qui distingue « où en est le tracé » de « ce que la page affiche ».
 *
 * # QUI ÉCRIT, QUI LIT
 *
 * Le CANVAS écrit, tout le monde lit. `publish` n'a qu'un appelant de production — la fin du
 * tracé, via `useReplayClock` — et c'est ce qui rend la position lisible partout sans qu'aucun
 * lecteur ait à se demander qui d'autre aurait pu la bouger.
 *
 * # POURQUOI LA PUBLICATION EST BRIDÉE, ET QUI LA BRIDE
 *
 * Le canvas se redessine à la cadence de l'écran ; les fiches sont du DOM. Les re-rendre 60
 * fois par seconde coûterait tout le budget d'animation pour un contenu qui change à peine.
 * Le bridage vit chez celui qui peint (`useReplayClock` : 150 ms, avec le rattrapage sans
 * délai de la dernière image) — le magasin, lui, ne décide pas du rythme.
 *
 * # PAS DE SINGLETON DE MODULE
 *
 * Le magasin naît avec la page (`usePlaybackStore`) et meurt avec elle. Un singleton de module
 * ferait démarrer un second rejeu à l'image du premier, et rendrait deux montages simultanés
 * (un test, un rendu concurrent) indiscernables.
 */
import { useState, useSyncExternalStore } from 'react'

export interface PlaybackStore {
  /**
   * La dernière image PUBLIÉE par le canvas — un instantané stable, qui ne change qu'entre
   * deux notifications. Ce n'est PAS l'image du tracé en cours (cf. l'en-tête).
   */
  published(): number
  /**
   * Publie une image. Appelée par le CANVAS à la fin du tracé, à la cadence qu'il décide, et
   * par lui seul.
   */
  publish(frame: number): void
  /** Abonnement React (`useSyncExternalStore`) : rend le désabonnement. */
  subscribe(listener: () => void): () => void
}

/**
 * createPlaybackStore fabrique le magasin d'une page de rejeu.
 *
 * PUBLIER LA MÊME IMAGE NE RÉVEILLE PERSONNE : la boucle de lecture est souvent plus lente que
 * l'écran, donc le canvas repeint et republie la même image plusieurs fois de suite.
 */
export function createPlaybackStore(): PlaybackStore {
  let frame = 0
  const listeners = new Set<() => void>()
  return {
    published: () => frame,
    publish(next) {
      if (next === frame) return
      frame = next
      for (const l of listeners) l()
    },
    subscribe(listener) {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
  }
}

/**
 * usePlaybackStore crée le magasin de CETTE page, une seule fois.
 *
 * L'initialisation est paresseuse — la fonction est passée, pas son résultat : `useState(f)`
 * n'appelle `f` qu'au premier rendu, là où `useState(f())` fabriquerait un magasin neuf à
 * chaque rendu pour le jeter aussitôt. L'ÉTAT NE PORTE QUE L'OBJET, jamais la position : le
 * magasin ne se remplace jamais, donc ce `useState` ne déclenche aucun rendu.
 */
export function usePlaybackStore(): PlaybackStore {
  const [store] = useState(createPlaybackStore)
  return store
}

/**
 * usePlaybackFrame abonne un composant à la position publiée.
 *
 * C'est la lecture du DOM hors canvas : fiches joueur, bandeau de score, écran de fin,
 * message inter-manche. Elle suit la cadence de publication, pas celle de l'écran — qui a
 * besoin de l'image exacte du tracé en cours ne passe pas par React.
 */
export function usePlaybackFrame(store: PlaybackStore): number {
  return useSyncExternalStore(store.subscribe, store.published, store.published)
}
