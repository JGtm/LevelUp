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
 * # LE CANAL DE SAUT : LA PAGE DEMANDE, LE CANVAS EXÉCUTE (plan « maquettes du rejeu », D-5)
 *
 * « Le canvas écrit, tout le monde lit » reste vrai pour le TEMPS : la page ne l'écrit jamais.
 * Quand elle veut déplacer la lecture (un clic sur une ligne de frag du fil), elle DEMANDE un saut
 * (`requestSeek`) et le lecteur du canvas (`useReplayPlayback`) l'exécute — pause, puis pose sur
 * l'image —, par le même `seekTo` borné à la fenêtre de gameplay que la frise. Une demande faite
 * avant que le canvas ne soit monté l'attend (la dernière seule), et lui est remise une fois.
 *
 * # LA MISE EN VALEUR D'UNE ÉLIMINATION : UN ÉTAT DE PAGE, TENU ICI PARCE QUE DEUX LECTEURS LA
 * # PARTAGENT
 *
 * Le fil la POSE (clic) et affiche la ligne « sélectionnée » ; le canvas la LIT pour peindre les
 * équerres et les étiquettes, et l'EFFACE quand la lecture repart ou qu'on déplace la frise. Les
 * règles (jeton par clic, remplacement, effacement) sont pures — `killFocusLogic`. Elle n'est
 * jamais mémorisée : le magasin naît avec la page.
 *
 * # PAS DE SINGLETON DE MODULE
 *
 * Le magasin naît avec la page (`usePlaybackStore`) et meurt avec elle. Un singleton de module
 * ferait démarrer un second rejeu à l'image du premier, et rendrait deux montages simultanés
 * (un test, un rendu concurrent) indiscernables.
 */
import { useState, useSyncExternalStore } from 'react'

import {
  NO_KILL_FOCUS,
  reduceKillFocus,
  type KillFocusRequest,
  type KillFocusTarget,
} from './killFocusLogic'

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
  /**
   * DEMANDE de mettre la lecture en pause sur cette image (cf. « le canal de saut » en tête). La
   * page l'appelle ; ce magasin ne déplace rien lui-même, et `published()` ne bouge pas.
   */
  requestSeek(frame: number): void
  /**
   * L'EXÉCUTEUR du canal de saut : le lecteur du canvas, et lui seul. Une demande en attente lui
   * est remise à l'inscription. Rend le désabonnement.
   */
  onSeekRequest(executer: (frame: number) => void): () => void
  /** La mise en valeur courante, `null` hors clic — un instantané stable entre deux changements. */
  focus(): KillFocusRequest | null
  /** Un CLIC sur une ligne de frag : pose la mise en valeur (jeton neuf, même sur la même ligne). */
  focusKill(target: KillFocusTarget): void
  /** Efface la mise en valeur : la Lecture et tout saut manuel. Sans effet si elle est déjà vide. */
  clearFocus(): void
  /** Abonnement à la mise en valeur (`useSyncExternalStore`) : rend le désabonnement. */
  subscribeFocus(listener: () => void): () => void
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
  const executers = new Set<(frame: number) => void>()
  // La demande faite quand personne n'exécutait encore : remise une fois au premier exécuteur.
  let pendingSeek: number | null = null
  let focusState = NO_KILL_FOCUS
  const focusListeners = new Set<() => void>()
  const setFocus = (next: typeof focusState) => {
    if (next === focusState) return
    focusState = next
    for (const l of focusListeners) l()
  }
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
    requestSeek(target) {
      if (executers.size === 0) {
        pendingSeek = target
        return
      }
      for (const run of executers) run(target)
    },
    onSeekRequest(executer) {
      executers.add(executer)
      if (pendingSeek !== null) {
        const waiting = pendingSeek
        pendingSeek = null
        executer(waiting)
      }
      return () => executers.delete(executer)
    },
    focus: () => focusState.current,
    focusKill: (target) => setFocus(reduceKillFocus(focusState, { type: 'select', target })),
    clearFocus: () => setFocus(reduceKillFocus(focusState, { type: 'clear' })),
    subscribeFocus(listener) {
      focusListeners.add(listener)
      return () => focusListeners.delete(listener)
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

const NEVER_SUBSCRIBE = () => () => {}
const NO_FOCUS = (): KillFocusRequest | null => null

/**
 * usePlaybackFocus abonne un composant à la mise en valeur courante : la ligne « sélectionnée »
 * du fil, et le canvas qui la peint. Sans magasin (un fil monté hors de la page de rejeu), il ne
 * lit rien et ne s'abonne à rien.
 */
export function usePlaybackFocus(store: PlaybackStore | undefined): KillFocusRequest | null {
  return useSyncExternalStore(
    store?.subscribeFocus ?? NEVER_SUBSCRIBE,
    store?.focus ?? NO_FOCUS,
    store?.focus ?? NO_FOCUS,
  )
}
