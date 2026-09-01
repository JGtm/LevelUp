/**
 * bombCountdown.ts — LE COMPTE À REBOURS DE LA BOMBE D'ASSAUT. Logique pure, pas de React.
 *
 * D'OÙ VIENT LA DONNÉE. `doc.bombArmings` (schéma 29) : l'anneau du marqueur `ti=12 i14` est
 * la jauge d'armement — protocole du 2026-09-01 avec tirage nul (13/13 Neutral Bomb CV 0,016,
 * 4/4 Husky Raid, 0/1000 tirages nuls aussi bien). Chaque entrée porte l'instant ARMÉ (`t`) et
 * la MÈCHE (`fuseMs`, 4 930 ms côté Go) : la fenêtre du compte à rebours est [t, t + fuseMs],
 * rien d'autre n'est deviné.
 *
 * AUCUNE GARDE DE MODE ICI, et c'est délibéré (même doctrine que `useReplayBombBlast`) : la
 * garde EST la donnée. `bombArmings` n'est publié que par un match d'Assaut couvert (jamais
 * One Bomb — le canal y est réfuté, CV 0,725), et un film d'un autre mode rend une liste vide.
 *
 * LA GARDE D'HORLOGE EST CELLE DE LA DÉFLAGRATION (`filmClockTrusted`) : `t` n'est une frame
 * du document que si l'origine du film est résolue. Sinon l'écart est INCONNU (3,6 à 50,8 s
 * selon le match) et le compte à rebours partirait avant que la bombe ne soit armée. Muet vaut
 * mieux que faux.
 *
 * PAS DE POSITION, ET C'EST UNE PROPRIÉTÉ DE LA MESURE : le navpoint est un marqueur d'écran,
 * pas un acteur — le canal ne dit ni qui arme ni où. Le compte à rebours est donc un BANDEAU
 * (overlay non spatial), jamais un point sur la carte : poser un lieu que la mesure ne donne
 * pas serait l'erreur que la déflagration évite déjà.
 */
import { filmClockTrusted } from '@/lib/replay/scoreTimeline'

import { frameToMs, msToFrames } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'
import { soundEvent, type ReplaySoundEvent } from './replaySoundVariants'
import { ZONE_SOUND_STEMS } from './zoneSound'

/** L'état du compte à rebours à une image donnée : ce qu'il reste de mèche, et la course. */
export interface BombCountdownState {
  /** Millisecondes restantes avant l'explosion attendue (bornées à zéro). */
  remainingMs: number
  /** Course écoulée de la mèche, 0 (armée) -> 1 (explosion) — pour la barre. */
  progress: number
}

/**
 * activeBombCountdown — le compte à rebours actif à l'image lue, ou null.
 *
 * DÉRIVÉ DE LA POSITION DE LECTURE, PAS D'UN ÉTAT (doctrine de `ReplayRoundBreakOverlay`) :
 * visible tant que l'image tombe dans [t, t + fuseMs], il se rejoue si l'on repasse dessus et
 * se ferme dès qu'on le quitte. Deux armements ne peuvent pas se recouvrir (le hold puis la
 * mèche les séparent d'au moins ~6 s) — le dernier commencé gagne par construction.
 */
export function activeBombCountdown(
  doc: ReplayDocumentReady,
  frame: number,
): BombCountdownState | null {
  if (doc.bombArmings.length === 0) return null
  if (!filmClockTrusted(doc)) return null
  for (const a of doc.bombArmings) {
    const fuseFrames = Math.max(1, Math.round(msToFrames(a.fuseMs, doc)))
    if (frame < a.t || frame > a.t + fuseFrames) continue
    const elapsedMs = frameToMs(frame, doc) - frameToMs(a.t, doc)
    return {
      remainingMs: Math.max(0, a.fuseMs - elapsedMs),
      progress: Math.min(1, elapsedMs / a.fuseMs),
    }
  }
  return null
}

/**
 * bombArmingSoundEvents — le son de `bomb_armed`, un événement par armement.
 *
 * LE STEM EST CELUI DE LA NOUVELLE COLLINE (`ZONE_SOUND_STEMS.newZone`, désigné à l'oreille
 * par l'utilisateur le 2026-08-27) : décision utilisateur du portage — un son accompagne le
 * début du compte à rebours, et la banque d'Assaut extraite n'a pas de geste « bombe armée »
 * propre désigné. Emprunté par RÉFÉRENCE, jamais recopié : si la désignation change, elle
 * change pour les deux d'un seul coup. Pas de camp — le canal ne dit pas qui arme, et ce
 * son-là n'affirme rien (même propriété que la nouvelle colline).
 *
 * L'HORLOGE NE DEMANDE AUCUN RECALAGE : `t` est déjà l'index de frame du document (la
 * soustraction d'origine est faite côté Go), même conversion `frameToMs` que partout. La
 * garde d'horloge vit chez l'appelant (`replaySound.ts` ne bâtit ses événements que sur un
 * document dont il lit les frames) — et `bombArmings` vide rend une liste vide.
 */
export function bombArmingSoundEvents(doc: ReplayDocumentReady): ReplaySoundEvent[] {
  if (!filmClockTrusted(doc)) return []
  return doc.bombArmings.map((a) => soundEvent(frameToMs(a.t, doc), ZONE_SOUND_STEMS.newZone))
}
