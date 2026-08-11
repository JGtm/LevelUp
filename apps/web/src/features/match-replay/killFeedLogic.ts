/**
 * killFeedLogic.ts — LES KILLS SUR L'HORLOGE DU REJEU.
 *
 * DEUX HORLOGES, ET C'EST TOUT LE SUJET. Les events servis par la Match View sont recalés
 * sur le début du GAMEPLAY (`correctMatchViewEventsT0`) ; le film, lui, part du début du
 * MATCH, countdown compris. Poser les uns sur l'autre sans rien faire décale le feed de
 * 18 à 28 secondes — assez pour qu'un kill s'affiche pendant que son auteur est encore en
 * train de courir vers sa victime.
 *
 * LA MESURE QUI TRANCHE (000d5950, T0 = 18 465 ms) : en rapprochant les 91 fins de vie
 * exploitables du rejeu des 93 morts du registre, l'écart médian vaut -0,6 s à offset nul,
 * contre 3,1 s en ajoutant T0 et 4,2 s en le retranchant. Le rejeu suit donc l'horloge
 * BRUTE, et le recalage va dans ce sens : `msRejeu = event_time_ms + t0Ms`.
 *
 * Pas de React : logique pure, testée (killFeedLogic.test.ts).
 */
import type { KillEvent } from '@/features/match-view/_momentum'

/** Un kill placé sur l'axe de temps du rejeu. */
export interface ReplayKill extends KillEvent {
  /** Instant du kill dans le repère du film, en millisecondes depuis le début du match. */
  replayMs: number
}

/**
 * toReplayKills recale les kills sur l'horloge du film et les trie chronologiquement.
 *
 * `t0Ms` vaut 0 quand le countdown est inconnu : la correction T0 n'a alors pas eu lieu non
 * plus côté events, et ne rien ajouter est exactement juste.
 */
export function toReplayKills(kills: KillEvent[], t0Ms: number): ReplayKill[] {
  const offset = Number.isFinite(t0Ms) && t0Ms > 0 ? t0Ms : 0
  return kills
    .map((k) => ({ ...k, replayMs: k.tMs + offset }))
    .sort((a, b) => a.replayMs - b.replayMs)
}

/**
 * killsAt rend les kills à afficher à l'instant `nowMs` du rejeu, du plus récent au plus
 * ancien, dans la limite de `max`.
 *
 * LA FENÊTRE EST EN TEMPS DE MATCH, pas en nombre d'images : la cadence d'échantillonnage
 * du rejeu est choisie au build et peut changer sans que la lecture change. Un kill sort du
 * feed quand il a `windowMs` d'âge, exactement comme à l'écran du jeu.
 *
 * `kills` DOIT être trié (sortie de `toReplayKills`) : la recherche s'arrête au premier kill
 * postérieur à l'instant courant.
 */
export function killsAt(
  kills: ReplayKill[],
  nowMs: number,
  windowMs: number,
  max: number,
): ReplayKill[] {
  const out: ReplayKill[] = []
  const debut = nowMs - windowMs
  for (let i = kills.length - 1; i >= 0; i--) {
    const k = kills[i]
    if (k.replayMs > nowMs) continue
    if (k.replayMs < debut) break
    out.push(k)
    if (out.length >= max) break
  }
  return out
}

/**
 * freshnessOf rend l'opacité d'un kill selon son âge dans la fenêtre : franc quand il vient
 * d'arriver, atténué quand il va sortir. Même graduation que le reste du rejeu — « ancien »
 * doit se lire partout de la même façon.
 */
export function freshnessOf(kill: ReplayKill, nowMs: number, windowMs: number): number {
  if (windowMs <= 0) return 1
  const age = Math.max(0, nowMs - kill.replayMs)
  const r = Math.min(1, age / windowMs)
  return 1 - 0.6 * r
}
