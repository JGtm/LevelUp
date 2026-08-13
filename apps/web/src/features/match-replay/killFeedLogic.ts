/**
 * killFeedLogic.ts — LE FIL DES ÉLIMINATIONS SUR L'HORLOGE DU REJEU.
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
 * LE FIL EST PERMANENT (verdict utilisateur 2026-08-13, aligné POC) : il garde TOUT
 * depuis le début du match, le plus récent en tête, et défile pour remonter aux frags
 * anciens. Aucune fenêtre, aucune rémanence — l'éphémère précédent faisait perdre le fil.
 *
 * LES MÉDAILLES viennent des mêmes highlight events (event_type `medal`), identité
 * résolue côté backend (label/description locale-aware + visuel). Une médaille se
 * rattache au kill du MÊME acteur à moins de 500 ms — mesure sur le témoin : 42 des 44
 * médailles s'y rattachent, les 2 restantes s'affichent seules plutôt que d'être forcées
 * (même règle que le POC).
 *
 * Pas de React : logique pure, testée (killFeedLogic.test.ts).
 */
import type { KillEvent } from '@/features/match-view/_momentum'
import type { MatchHighlightEvent } from '@/lib/api/types'

/**
 * Tolérance de rattachement d'une médaille au kill du même acteur, en ms. Mesuré sur
 * 000d5950 : 42/44 à ≤ 500 ms, et AUCUNE de plus ne se rattache avant 5 000 ms — le seuil
 * est dans un plateau, pas sur une pente.
 */
export const MEDAL_ATTACH_MS = 500

/** Un kill placé sur l'axe de temps du rejeu. */
export interface ReplayKill extends KillEvent {
  /** Instant du kill dans le repère du film, en millisecondes depuis le début du match. */
  replayMs: number
  /** Les médailles décrochées par le tueur sur ce kill (±500 ms), dans l'ordre du fil. */
  medals: MedalEvent[]
}

/** Une médaille du fil, identité résolue côté backend. */
export interface MedalEvent {
  /** Instant sur l'horloge gameplay (event_time_ms). */
  tMs: number
  xuid: string
  gamertag: string
  teamID: number | null
  /** Nom ANGLAIS mesuré (film). Toujours présent — c'est la clé du référentiel. */
  name: string
  /** Libellé locale-aware. Vide = non résolue : le nom brut s'affiche en toutes lettres. */
  label: string
  description: string
  imageUrl: string
}

/**
 * collectMedalEvents extrait les events `medal` exploitables : un acteur, un instant, un
 * nom. Un event medal sans nom (raw_json illisible côté backend) est écarté — il n'y a
 * rien à en montrer, pas même un texte.
 */
export function collectMedalEvents(
  events: MatchHighlightEvent[] | null | undefined,
): MedalEvent[] {
  const out: MedalEvent[] = []
  for (const e of events ?? []) {
    if ((e.event_type ?? '').toLowerCase() !== 'medal') continue
    if (!e.actor_xuid || e.event_time_ms == null || !e.medal_name) continue
    out.push({
      tMs: e.event_time_ms,
      xuid: e.actor_xuid,
      gamertag: e.actor_gamertag ?? '',
      teamID: e.actor_team_id ?? null,
      name: e.medal_name,
      label: e.medal_label ?? '',
      description: e.medal_description ?? '',
      imageUrl: e.medal_image_url ?? '',
    })
  }
  return out
}

/**
 * Une ligne du fil : un kill (avec ses médailles rattachées), OU une médaille seule —
 * jamais les deux. La médaille seule existe parce que forcer un rattachement lointain
 * accrocherait la médaille au mauvais kill.
 */
export interface ReplayFeedEntry {
  /** Clé stable de rendu. */
  key: string
  /** Instant de la ligne dans le repère du film. */
  replayMs: number
  kill: ReplayKill | null
  medal: MedalEvent | null
}

/**
 * toReplayKills recale les kills sur l'horloge du film et les trie chronologiquement.
 *
 * `t0Ms` vaut 0 quand le countdown est inconnu : la correction T0 n'a alors pas eu lieu
 * non plus côté events, et ne rien ajouter est exactement juste.
 */
export function toReplayKills(kills: KillEvent[], t0Ms: number): ReplayKill[] {
  const offset = replayOffset(t0Ms)
  return kills
    .map((k) => ({ ...k, replayMs: k.tMs + offset, medals: [] as MedalEvent[] }))
    .sort((a, b) => a.replayMs - b.replayMs)
}

function replayOffset(t0Ms: number): number {
  return Number.isFinite(t0Ms) && t0Ms > 0 ? t0Ms : 0
}

/**
 * buildFeedEntries assemble le fil : kills recalés, médailles rattachées au kill du même
 * acteur le plus proche (±MEDAL_ATTACH_MS, sur l'horloge gameplay commune aux deux
 * sources), médailles orphelines en lignes seules. Trié chronologiquement.
 */
export function buildFeedEntries(
  kills: KillEvent[],
  medals: MedalEvent[],
  t0Ms: number,
): ReplayFeedEntry[] {
  const rk = toReplayKills(kills, t0Ms)
  const offset = replayOffset(t0Ms)
  const alone: MedalEvent[] = []
  for (const m of medals) {
    let best: ReplayKill | null = null
    let bestDt = MEDAL_ATTACH_MS + 1
    for (const k of rk) {
      if (k.xuid !== m.xuid) continue
      const dt = Math.abs(k.tMs - m.tMs)
      if (dt < bestDt) {
        best = k
        bestDt = dt
      }
    }
    if (best && bestDt <= MEDAL_ATTACH_MS) best.medals.push(m)
    else alone.push(m)
  }
  const entries: ReplayFeedEntry[] = rk.map((k, i) => ({
    key: `k-${k.xuid}-${k.replayMs}-${i}`,
    replayMs: k.replayMs,
    kill: k,
    medal: null,
  }))
  alone.forEach((m, i) => {
    entries.push({
      key: `m-${m.xuid}-${m.tMs}-${i}`,
      replayMs: m.tMs + offset,
      kill: null,
      medal: m,
    })
  })
  return entries.sort((a, b) => a.replayMs - b.replayMs)
}

/**
 * feedAt rend les lignes déjà survenues à l'instant `nowMs` du rejeu, de la plus récente
 * à la plus ancienne — le fil COMPLET, sans fenêtre : les lignes passées restent, le
 * défilement fait le reste.
 *
 * `entries` DOIT être trié (sortie de `buildFeedEntries`) : la coupe s'arrête au premier
 * élément postérieur à l'instant courant.
 */
export function feedAt(entries: ReplayFeedEntry[], nowMs: number): ReplayFeedEntry[] {
  const out: ReplayFeedEntry[] = []
  for (let i = entries.length - 1; i >= 0; i--) {
    const e = entries[i]
    if (e.replayMs > nowMs) continue
    out.push(e)
  }
  return out
}
