/**
 * useReplayPlayback — LA LECTURE du rejeu : l'état lu / en pause, la boucle d'animation, le
 * curseur de la frise, et ce qui se passe quand le film ARRIVE AU BOUT.
 *
 * EXTRAIT DE `ReplayCanvas.tsx` LE 2026-08-25 (huitième extraction imposée par le seuil de
 * taille, `max-lines` eslint, R5) : le lot y ajoutait la politique de fin de rejeu, et
 * la règle du dépôt est d'extraire plutôt que de relever le plafond. La découpe tombe sur une
 * frontière nette — ce fichier porte le TEMPS de la lecture, le canvas porte le DESSIN. Il ne
 * dessine rien lui-même : il appelle le tracé que l'appelant lui passe.
 *
 * # LE REJEU S'ARRÊTE SUR SON ÉTAT FINAL
 *
 * Demande utilisateur du 2026-08-25 : « quand le rejeu arrive au bout, rester sur l'état
 * final ». La boucle REBOUCLAIT à zéro (`if (next >= doc.frameCount - 1) next = 0`) — la
 * dernière image du match n'était donc jamais celle qu'on avait sous les yeux : elle était
 * remplacée, dans la même image d'animation, par la toute première. Un match se terminait
 * visuellement sur son coup d'envoi.
 *
 * Elle s'arrête maintenant SUR la dernière image : le curseur y reste, la scène finale reste
 * peinte, et la lecture passe en pause. Trois choses tiennent ensemble et aucune n'est
 * décorative — le curseur à `endFrame` (sans quoi la frise dirait un autre instant que le
 * terrain), le dernier `draw()` AVANT l'arrêt (sans quoi la scène resterait à l'avant-dernière
 * image), et le `soundTick` de cette même image (le curseur du son suit toujours la lecture,
 * cf. useReplaySound — le laisser en arrière ferait repartir un son enjambé au prochain clic).
 *
 * L'ARRIVÉE EST AUSSI UN ÉVÉNEMENT (`onEnded`, lot C du 2026-08-27) : c'est d'ici que part le
 * son de fin de partie, parce que c'est le seul endroit qui sait distinguer une lecture qui
 * FRANCHIT la borne d'une frise qu'on a tirée jusqu'au bout. La condition tient en trois mots —
 * l'image d'AVANT le pas était en deçà de la borne.
 *
 * RELANCER RESTE À UN CLIC, et c'est la convention des lecteurs vidéo : « Lecture » sur un
 * rejeu terminé repart du début plutôt que de rester bloqué sur la dernière image, où la boucle
 * se rendormirait aussitôt. « Recommencer » ne change pas — il ramène au début à tout instant.
 *
 * # « LE DÉBUT » ET « LA FIN » SONT CEUX DU MATCH, PAS CEUX DU FILM
 *
 * Depuis le 2026-08-26, les deux bornes viennent de la FENÊTRE DE GAMEPLAY (`replayWindow.ts`) :
 * la lecture démarre au coup d'envoi et s'arrête à la fin déclarée, sans le countdown
 * d'avant-match ni la queue de 5-6 s que le film garde après. La mécanique ne change pas d'un
 * pas — seules les deux valeurs changent. Sans fenêtre (artefact ancien, en-tête sans durée
 * jouable), les bornes redeviennent celles du film : image zéro et dernière image.
 *
 * LE CURSEUR SE POSE AU DÉBUT QUAND LA FENÊTRE SE CONNAÎT, et pas avant : elle vient de la
 * Match View, qui arrive APRÈS le document du rejeu. Le repositionnement ne va donc que vers
 * l'AVANT (jamais un retour en arrière sous les doigts de qui a déjà déplacé la frise).
 *
 * # UNE SECONDE DE PRÉAMBULE AVANT LE COUP D'ENVOI (D3, user 2026-09-02)
 *
 * Depuis le T0 mesuré dans le film, la lecture ne se pose plus SUR le coup d'envoi mais une
 * seconde avant (`playWindow.leadInFrame`, cf. `replayWindow.ts`) : le temps de poser les yeux
 * sur la scène avant que l'action parte. TROIS chemins l'utilisent, et eux seuls — la pose
 * initiale, « Recommencer » (`rewind`) et la borne BASSE de `seekTo`.
 *
 * CE QUI NE BOUGE PAS, et c'est le cœur de la décision : le CADRAGE. `startFrame` reste le coup
 * d'envoi — c'est lui que la frise publie en borne basse (`useReplayTimeline.minFrame`), lui que
 * l'horloge prend pour origine (`useReplayClock` via `displayClockMs`) et lui que l'export
 * propose par défaut (`replayExportPlan.defaultExportBounds`). Le préambule est donc UNIQUEMENT
 * une position de départ : pendant cette seconde l'horloge lit 0:00 et la frise reste à 0 %.
 *
 * # SE DÉPLACER SANS VISER : LES SAUTS, ET LE REMPLISSAGE DE LA FRISE
 *
 * Deux commandes s'ajoutent le 2026-08-28 avec la barre de lecture (planche 2a) : `seekBy`,
 * un saut de ±N SECONDES converti en images par la cadence du document, et `stepFrames`,
 * l'image par image. Les deux passent par le même `seekTo` borné à la fenêtre de gameplay —
 * une seule définition de « où le curseur a le droit d'aller ».
 *
 * # LA LECTURE AUTOMATIQUE EST UN RÉGLAGE, ET IL SE LIT UNE SEULE FOIS
 *
 * Le rejeu partait en lecture au montage, sans que personne ne puisse le demander autrement
 * (demande utilisateur du 2026-08-29, point 22). C'est désormais une préférence persistée
 * (`AUTOPLAY_KEY`), ÉTEINTE par défaut à la demande de l'utilisateur : le rejeu s'ouvre en
 * PAUSE, cadré au coup d'envoi, et attend « Lecture ». Elle se lit dans l'INITIALISEUR de
 * l'état : une seule fois, au montage.
 *
 * LE CADRAGE, LUI, S'APPLIQUE QUAND MÊME (effet de bord qu'il faut nommer) : l'effet qui pose
 * le curseur au coup d'envoi ne regarde pas `playing`. Sans cela, un rejeu ouvert en pause
 * resterait sur l'image zéro du FILM — c'est-à-dire sur le countdown d'avant-match, joueurs
 * figés à leur apparition — et la première image du match ne serait jamais celle qu'on voit.
 *
 * PAS D'ABONNEMENT ICI, ET C'EST LA DIFFÉRENCE AVEC TOUS LES AUTRES RÉGLAGES DU TIROIR. Un
 * calque qu'on éteint s'éteint tout de suite ; « lecture automatique » ne décrit pas l'état
 * courant du lecteur, il décrit son état de DÉPART. Le suivre en direct ferait partir — ou
 * arrêter — la lecture sous les doigts de qui vient d'ouvrir le tiroir pour régler la fois
 * suivante. Mettre en marche et mettre en pause restent le travail de la barre de lecture.
 *
 * `writeCursor` EST LE SEUL ENDROIT QUI DÉPLACE LE CURSEUR, et il en écrit TROIS choses : la
 * valeur du champ (ce que le navigateur dessine), la variable CSS `--played` (ce que la frise
 * habillée remplit derrière lui) et, depuis le 2026-09-06, `--played-r` — la MÊME position,
 * mais en ratio brut, que la géométrie de piste sait consommer. Les séparer les ferait
 * diverger au premier chemin oublié — c'est pourquoi la boucle, les sauts, le rembobinage, le
 * glissé manuel et la pose initiale l'appellent tous, sans exception.
 */
import { useCallback, useEffect, useRef, useState, type ChangeEvent, type RefObject } from 'react'

import { frameToMs } from '../../../lib/replay/replayLogic'
import type { ReplayDocumentReady } from '../../../lib/replay/replayNormalize'
import { readStoredFlag } from '../settings/replayPreferences'
import { AUTOPLAY_DEFAULT, AUTOPLAY_KEY } from '../settings/useReplaySettings'
import type { ReplayWindowBounds } from '../model/replayWindow'
import { usePlaybackCommands } from './useReplayPlaybackCommands'
import { useInitialCursor, useLatestSeek, useOpenAtFrame, usePlaybackLoop, useSeekRequests } from './useReplayPlaybackEffects'

/**
 * LA PART PARCOURUE, EN POURCENTAGE — le remplissage du dégradé de la piste, tel qu'il existe
 * depuis l'habillage du curseur. Elle porte son unité parce que c'est un `linear-gradient` qui
 * la consomme, et qu'un dégradé veut une longueur.
 */
export const CURSOR_PLAYED_VAR = '--played'

/**
 * LA MÊME POSITION, EN RATIO NU [0..1] (2026-09-06, lot du trait de lecture).
 *
 * POURQUOI DEUX VARIABLES POUR UNE SEULE POSITION, et ce n'est pas un doublon. Le dégradé de
 * la piste veut un POURCENTAGE de la largeur du champ ; les marques, elles, vivent dans la
 * géométrie du CURSEUR — `calc(8px + (100% - 16px) * r)`, cf. `trackLeft` — parce qu'un curseur
 * natif réserve sa demi-largeur à chaque bout. Les deux ne se rejoignent qu'au milieu de la
 * frise : un objet posé à `left: var(--played)` dérive jusqu'à 8 px de la marque qu'il désigne
 * aux extrémités. Un ratio nu est la seule forme que les DEUX géométries savent consommer
 * (`* var(--played-r)` d'un côté, `calc(var(--played-r) * 100%)` de l'autre si besoin) —
 * d'où celle-ci, écrite au même endroit unique et dans le même geste.
 */
export const CURSOR_RATIO_VAR = '--played-r'

/**
 * L'ATTRIBUT QUI DÉSIGNE L'HÔTE DES DEUX VARIABLES — la racine de la frise habillée
 * (`ReplayTimelineTracks`), et elle seule.
 *
 * POURQUOI UN ATTRIBUT PLUTÔT QUE `el.parentElement`. Jusqu'au 2026-09-06 les variables se
 * posaient sur le parent du champ, c'est-à-dire sur le conteneur de la seule RANGÉE du curseur :
 * les pistes, qui vivent dans les rangées du dessus, ne les voyaient pas (les propriétés
 * personnalisées héritent vers le BAS, jamais vers le côté). Le trait de lecture, lui, doit les
 * lire depuis la première piste. Les poser un cran plus haut suffit — le champ continue de les
 * recevoir par héritage, son dégradé ne change pas d'un pixel.
 *
 * NOMMÉ ET PAS DEVINÉ : `closest` sur cet attribut vise CE conteneur, quel que soit le nombre de
 * niveaux que la frise s'ajoutera (L3 lui ajoute un menu, L4 des glyphes). Compter les
 * `parentElement` aurait re-cassé au premier remaniement, en silence et sans test rouge — le
 * remplissage se serait simplement posé sur le mauvais élément. Le repli sur le parent puis sur
 * le champ garde la pose fonctionnelle hors de la frise (tests de hook, champ détaché).
 */
export const CURSOR_HOST_ATTR = 'data-replay-cursor-host'

/** Ce dont la lecture a besoin (objet unique : la règle des 5 paramètres du dépôt). */
export interface ReplayPlaybackOptions {
  doc: ReplayDocumentReady
  /**
   * La fenêtre de gameplay du match (cf. `replayWindow.ts`). `null` = pas de cadrage établi :
   * la lecture reprend les bornes du film entier, exactement comme avant ce lot.
   */
  playWindow: ReplayWindowBounds | null
  /** Cadence « 1× » du document, en images par seconde (cf. useReplayTiming). */
  baseFps: number
  /** Multiplicateur de vitesse choisi dans la barre de lecture. */
  speed: number
  /** Largeur de dessin : 0 = le canvas n'est pas encore mesuré, rien ne tourne. */
  renderWidth: number
  /** L'image courante, PARTAGÉE avec le dessin et les calques survolables. */
  frameRef: RefObject<number>
  /** Le tracé d'une image, tel que le canvas le publie. */
  draw: () => void
  /** Le battement du son : l'instant courant du rejeu, en ms (cf. useReplaySound.tick). */
  soundTick: (ms: number) => void
  /**
   * LE DÉPLACEMENT du son : pose le curseur sonore à cet instant SANS rien jouer
   * (cf. `useReplaySound.seek`). Servi par le GLISSÉ de frise, et par lui seul — cf. `onScrub`.
   */
  soundSeek: (ms: number) => void
  /**
   * L'ARRIVÉE EN FIN DE MATCH : appelé quand la lecture FRANCHIT la borne de fin, jamais quand
   * elle y était déjà (cf. l'en-tête). Le son de fin de partie s'y branche.
   */
  onEnded: () => void
  /**
   * UN GESTE DE TRANSPORT VIENT D'AVOIR LIEU (« Lecture »/« Pause », « Recommencer »). Le son
   * s'en sert pour reprendre vie après un rechargement de page : un AudioContext ne naît que
   * dans un geste utilisateur, et ces deux boutons en sont (cf. `useReplaySound.wake`).
   */
  onTransportGesture: () => void
  /**
   * L'ÉTAT LU/PAUSE VIENT DE CHANGER (lot moteurs de véhicules, 2026-09-04). Les sons
   * d'ÉVÉNEMENT n'en ont pas besoin — sans battement, rien ne part — mais une BOUCLE moteur
   * en vol ne s'éteint pas toute seule : la pause doit la couper (cf.
   * `useReplaySound.setTransportPlaying`). Optionnel : les autres appelants n'ont rien à dire.
   */
  onPlayingChange?: (playing: boolean) => void
  /**
   * LA LECTURE PART DU PRÉAMBULE (item 7, décision D-8, 2026-09-27) : « Lecture » ou
   * « Recommencer » avec le curseur au préambule ou en deçà, APRÈS l'éventuel rembobinage d'un
   * rejeu terminé, et après `onTransportGesture` (le lecteur audio doit exister). Jamais sur une
   * reprise, un glissé, un saut, un lien tactique ni une lecture automatique — aucun de ces
   * chemins ne passe par ici. La musique d'intro s'y branche (`useReplaySound.intro`).
   */
  onStarted?: () => void
  /**
   * OUVRIR À UNE FRAME PRÉCISE (lot M1b, 2026-09-08) : posé une seule fois par la route depuis
   * un lien tactique déjà converti (cf. `ReplayCanvasProps.openAtFrame`). Gagne sur le cadrage
   * au coup d'envoi (`leadInFrame`) même si celui-ci arrive APRÈS (la fenêtre de gameplay vient
   * de la Match View, potentiellement plus tard que ce document) — cf. l'effet plus bas et son
   * garde `appliedOpenAtFrameRef`.
   */
  openAtFrame?: number | null
  /**
   * LE CANAL DE SAUT DE LA PAGE (`playbackStore.onSeekRequest`, plan « maquettes du rejeu », D-5) :
   * inscription de l'EXÉCUTEUR, désinscrit au démontage. Une demande met la lecture en PAUSE puis
   * la pose sur l'image, par le même `seekTo` borné que tout saut — et ne compte PAS comme un
   * geste de l'utilisateur (`onUserMove` ne part pas) : le clic qui demande ce saut pose lui-même
   * la mise en valeur qu'un geste manuel effacerait. Absent = aucun canal.
   */
  onSeekRequest?: (executer: (frame: number) => void) => () => void
  /**
   * L'UTILISATEUR VIENT DE DÉPLACER OU RELANCER LA LECTURE À LA MAIN : « Lecture » (la reprise,
   * pas la pause), « Recommencer », glissé de frise, sauts de ±N secondes, pas d'image, saut vers
   * un repère — donc aussi leurs raccourcis clavier, qui passent par ces commandes. La mise en
   * valeur d'une élimination s'y efface (`playbackStore.clearFocus`). Ni le saut d'un lien
   * tactique, ni une demande de la page, ni la fin du film n'en font partie.
   */
  onUserMove?: () => void
}

/** Ce que la barre de lecture reçoit — l'état à afficher et les commandes. */
export interface ReplayPlayback {
  playing: boolean
  /**
   * LA PREMIÈRE IMAGE DU GAMEPLAY : la borne basse de la FRISE, et l'origine du remplissage.
   * Ce n'est PAS l'endroit où la lecture se pose — celui-là est une seconde en deçà
   * (`playWindow.leadInFrame`, cf. l'en-tête § du préambule). Sans fenêtre : l'image zéro.
   */
  startFrame: number
  /**
   * LA DERNIÈRE IMAGE DU GAMEPLAY : la borne haute de la frise ET l'instant où la lecture
   * s'arrête. Une seule valeur pour les deux, sinon le curseur pourrait buter avant (ou
   * après) l'arrêt. Sans fenêtre : la dernière image du document.
   */
  endFrame: number
  /** Le curseur de la frise : piloté par la boucle, jamais contrôlé par React. */
  sliderRef: RefObject<HTMLInputElement | null>
  togglePlay: () => void
  restart: () => void
  onScrub: (e: ChangeEvent<HTMLInputElement>) => void
  /** Saut de ±N SECONDES sur l'axe réel (converti en images par `baseFps`). */
  seekBy: (seconds: number) => void
  /**
   * Image par image, bornée à la fenêtre de gameplay. ELLE MET LA LECTURE EN PAUSE : un pas
   * d'image sous la boucle d'animation serait écrasé en 16 ms, et avancer d'une image est un
   * geste d'arrêt sur image par nature.
   */
  stepFrames: (frames: number) => void
  /**
   * ALLER À UNE IMAGE PRÉCISE, bornée comme les deux commandes ci-dessus (c'est le même
   * `seekTo`). Exposée le 2026-09-07 pour les repères d'entrée/sortie de la frise, qui SONT des
   * boutons (décision 2 du plan « frise, point de vue ») : cliquer la porte d'un remplaçant
   * emmène le curseur à son arrivée.
   *
   * ELLE NE MET PAS EN PAUSE, contrairement à `stepFrames` : un saut vers un instant nommé est
   * un déplacement, pas un arrêt sur image — la lecture reprend de là, et c'est ce qu'on attend
   * en cliquant sur un repère pendant qu'on regarde.
   */
  seekToFrame: (frame: number) => void
}

/** Les bornes de la lecture : le coup d'envoi, la fin, et le préambule où la lecture se pose (cf. l'en-tête). */
function playbackBounds(
  doc: ReplayDocumentReady,
  playWindow: ReplayWindowBounds | null,
): { startFrame: number; endFrame: number; leadInFrame: number } {
  const lastFrame = Math.max(doc.frameCount - 1, 0)
  return {
    startFrame: playWindow?.startFrame ?? 0,
    endFrame: playWindow?.endFrame ?? lastFrame,
    // OÙ LA LECTURE SE POSE (cf. l'en-tête, § du préambule) : une seconde en deçà du coup
    // d'envoi. Sans fenêtre, c'est l'image zéro — exactement le comportement d'avant ce lot.
    leadInFrame: playWindow?.leadInFrame ?? 0,
  }
}

/**
 * placeCursor POSE LE CURSEUR : la valeur du champ, et la POSITION que toute la frise
 * habillée consomme — remplissage du dégradé, bulle de temps, trait de lecture.
 *
 * `--played` est la part parcourue, en pourcentage de la fenêtre. Le dégradé de la piste la
 * consomme depuis les classes du champ (`ReplayTimelineTracks.tsx`, variantes
 * `[&::-webkit-slider-runnable-track]` / `[&::-moz-range-track]` — même technique que le
 * volume dans `ReplaySoundControls.tsx`) : aucune feuille de style à tenir à jour, et aucun
 * rendu React pour un remplissage qui suit la lecture. `--played-r` est la MÊME position en
 * ratio nu, celle qu'attend la géométrie de curseur (cf. `CURSOR_RATIO_VAR`). Les deux
 * s'écrivent ICI et nulle part ailleurs — un chemin qui déplacerait le curseur sans elles
 * laisserait la frise figée sur la position précédente, et un second chemin d'écriture
 * les ferait diverger l'une de l'autre.
 *
 * ELLES SE POSENT SUR LA RACINE DE LA FRISE, ET PAS SUR LE CHAMP (2026-09-02, remonté d'un
 * cran le 2026-09-06 — cf. `CURSOR_HOST_ATTR`). Les propriétés personnalisées HÉRITENT : le
 * champ les reçoit donc exactement comme avant, son dégradé ne change pas d'un pixel. Ce qui
 * change, c'est que la BULLE DE TEMPS et les PISTES — qui ne sont pas des descendants du
 * champ, ni même de sa rangée — peuvent les lire elles aussi. Posées sur le champ, elles
 * seraient restées invisibles à tout ce qui n'est pas lui : un `input` n'a pas de descendants.
 */
function placeCursor(el: HTMLInputElement | null, frame: number, startFrame: number, endFrame: number): void {
  if (!el) return
  el.value = String(Math.round(frame))
  const span = endFrame - startFrame
  // Borné : la frise ne se remplit ni en deçà de son début ni au-delà de sa fin, même si
  // un appelant lui sert une image hors fenêtre (le préambule d'avant coup d'envoi est
  // précisément ce cas-là, et il doit lire zéro plutôt qu'un ratio négatif).
  const ratio = Math.min(1, Math.max(0, span > 0 ? (frame - startFrame) / span : 0))
  // L'HÔTE EST CHERCHÉ À CHAQUE POSE, et c'est volontaire : le mémoriser demanderait de
  // savoir quand la frise se remonte (repli/dépli, changement de mode). `closest` sur un
  // attribut coûte trois sauts de parent — moins qu'un rendu React, et cette fonction
  // tourne déjà soixante fois par seconde sans que rien ne s'en plaigne.
  const racine = el.closest(`[${CURSOR_HOST_ATTR}]`) as HTMLElement | null
  const host = racine ?? el.parentElement ?? el
  host.style.setProperty(CURSOR_PLAYED_VAR, `${ratio * 100}%`)
  host.style.setProperty(CURSOR_RATIO_VAR, String(ratio))
}

export function useReplayPlayback(o: ReplayPlaybackOptions): ReplayPlayback {
  const { doc, playWindow, baseFps, speed, renderWidth, frameRef, draw, openAtFrame } = o
  const { soundTick, soundSeek, onEnded, onTransportGesture, onPlayingChange, onStarted } = o
  const { onSeekRequest, onUserMove } = o
  // POSÉ VRAI DÈS QUE `openAtFrame` A ÉTÉ APPLIQUÉ (voir `useOpenAtFrame`) : empêche le
  // cadrage au coup d'envoi (`useInitialCursor`) d'écraser un lien tactique déjà positionné si
  // la fenêtre de gameplay arrive PLUS TARD (Match View asynchrone) sur un instant antérieur
  // au coup d'envoi.
  const appliedOpenAtFrameRef = useRef(false)
  const sliderRef = useRef<HTMLInputElement>(null)
  // LA LECTURE AUTOMATIQUE EST UN RÉGLAGE, LU UNE FOIS (cf. l'en-tête, § du même nom).
  const [playing, setPlaying] = useState(() => readStoredFlag(AUTOPLAY_KEY, AUTOPLAY_DEFAULT))
  // Le changement d'état lu/pause se PUBLIE (moteurs de véhicules, cf. ReplayPlaybackOptions).
  useEffect(() => {
    onPlayingChange?.(playing)
  }, [playing, onPlayingChange])
  const { startFrame, endFrame, leadInFrame } = playbackBounds(doc, playWindow)
  const writeCursor = useCallback(
    (frame: number) => placeCursor(sliderRef.current, frame, startFrame, endFrame),
    [startFrame, endFrame],
  )
  useInitialCursor({ appliedOpenAtFrameRef, leadInFrame, frameRef, draw, writeCursor })
  usePlaybackLoop({ playing, setPlaying, doc, baseFps, speed, renderWidth, endFrame, frameRef, writeCursor, draw, soundTick, onEnded })

  /**
   * onScrub — LE GLISSÉ DE LA FRISE, et depuis le lot 5.8.7 il PRÉVIENT LE SON.
   *
   * CE QU'IL NE FAISAIT PAS, ET CE QUE ÇA COÛTAIT. Il ne touchait pas au curseur sonore, resté à
   * l'instant d'AVANT le geste. Le battement suivant (reprise de lecture) voyait alors un écart
   * dont la seule amplitude décidait : sous la seconde, tout ce que le glissé avait enjambé
   * partait EN RAFALE ; au-delà, un recalage silencieux. Deux comportements pour un même geste,
   * dont un mur de bruit — et un glissé émet des dizaines d'événements de champ.
   *
   * IL APPELLE `soundSeek`, PAS `soundTick`, ET C'EST LA RÈGLE : un déplacement pose le curseur
   * sans rien jouer, quelle que soit son amplitude — on n'a pas écouté ce qu'on a enjambé.
   * `seekTo`, lui, garde son `soundTick` à dessein : un pas d'image ou un saut vers un repère est
   * une LECTURE qui atterrit, et entendre ce qui se joue à l'arrivée est ce qu'on attend.
   */
  const onScrub = (e: ChangeEvent<HTMLInputElement>) => {
    onUserMove?.()
    frameRef.current = Number(e.currentTarget.value)
    // LE REMPLISSAGE SUIT LE GLISSÉ : le champ porte déjà sa valeur (c'est lui qui l'émet),
    // mais `--played` ne se met à jour pour personne — sans cet appel, la piste resterait
    // remplie jusqu'à la position d'AVANT le geste.
    writeCursor(frameRef.current)
    soundSeek(frameToMs(frameRef.current, doc))
    if (!playing) draw()
  }

  const rewind = () => {
    frameRef.current = leadInFrame
    writeCursor(leadInFrame)
  }

  /**
   * seekTo — LE SEUL CHEMIN DE DÉPLACEMENT DIRECT, et il porte le bornage : la lecture ne
   * sort pas de la fenêtre de gameplay, quel que soit le geste qui l'y envoie. Il peint
   * (`draw`) et fait battre le son (`soundTick`) comme un pas de boucle : un curseur déplacé
   * sans repeindre montrerait la scène de l'instant précédent.
   *
   * SA BORNE BASSE EST LE PRÉAMBULE, pas le coup d'envoi : un retour arrière depuis la première
   * seconde de jeu doit pouvoir redonner à voir ce que la lecture montrait à son ouverture.
   */
  const seekTo = (frame: number) => {
    const next = Math.min(endFrame, Math.max(leadInFrame, frame))
    frameRef.current = next
    writeCursor(next)
    soundTick(frameToMs(next, doc))
    draw()
  }
  const seekToRef = useLatestSeek(seekTo)
  useOpenAtFrame(openAtFrame, appliedOpenAtFrameRef, seekToRef)
  useSeekRequests(onSeekRequest, setPlaying, seekToRef)

  const { togglePlay, restart, seekBy, stepFrames, seekToFrame } = usePlaybackCommands({
    playing, setPlaying, frameRef, baseFps, endFrame, leadInFrame, seekTo, rewind, onUserMove, onTransportGesture, onStarted,
  })

  return { playing, startFrame, endFrame, sliderRef, togglePlay, restart, onScrub, seekBy, stepFrames, seekToFrame }
}
