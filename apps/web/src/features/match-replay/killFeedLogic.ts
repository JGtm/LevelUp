/**
 * killFeedLogic.ts — LE FIL DES ÉLIMINATIONS SUR L'HORLOGE DU REJEU.
 *
 * DEUX HORLOGES, ET C'EST TOUT LE SUJET. Les events servis par la Match View sont recalés
 * sur le début du GAMEPLAY (`correctMatchViewEventsT0`) ; l'artefact de rejeu, lui, cale
 * sa frame 0 sur le PREMIER PAQUET DE POSITION du film (`build.go` : origin =
 * sorted[0].TimestampUS) — un instant qui varie d'un match à l'autre. Additionner
 * `event_time_ms + t0Ms` reconstruit l'horloge brute du film, PAS celle de l'artefact.
 *
 * LA MESURE QUI TRANCHE (2026-08-14, appariement ordonné kill↔fin de vie par victime,
 * artefacts v3 + Match View locale) : l'écart `event_time_ms + t0Ms − fin de vie` est
 * un décalage SYSTÉMATIQUE PAR MATCH — médiane +3 678 ms sur 000d5950 (87/91 paires à
 * ±150 ms), +10 589 ms sur 64e8adfa, +39 856 ms sur e94163af. Aucune constante ne peut
 * donc le corriger. Le référentiel JUSTE est celui que le client tient déjà : les fins
 * de vie des pistes (`doc.tracks`), qui datent aussi le flash des fiches. Le recalage
 * (`alignFeedToTracks`) pose chaque kill SUR la fin de vie de sa victime : le fil et la
 * fiche partent alors du même instant par construction.
 *
 * LE FIL EST PERMANENT (verdict utilisateur 2026-08-13, aligné POC) : il garde TOUT
 * depuis le début du match, le plus récent en tête, et défile pour remonter aux frags
 * anciens. Aucune fenêtre, aucune rémanence — l'éphémère précédent faisait perdre le fil.
 *
 * LES MORTS SANS TUEUR (suicides, chutes, sorties) font leur propre ligne NEUTRE
 * (décision produit 2026-08-13) : une fin de vie close qu'aucun kill ne revendique est
 * une mort que le fil doit dire — mesure témoin 64e8adfa : 123 morts au scoreboard pour
 * 121 kills, et exactement 2 fins de vie orphelines. L'appariement par victime garantit
 * qu'une mort déjà servie par une ligne de kill n'est JAMAIS doublée.
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

import { frameToMs, msToFrames, trackWindow } from './replayLogic'
import type { ReplayDocumentReady } from './replayNormalize'

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
 * Une ligne du fil : un kill (avec ses médailles rattachées), OU une médaille seule, OU
 * une mort neutre — jamais deux à la fois. La médaille seule existe parce que forcer un
 * rattachement lointain accrocherait la médaille au mauvais kill ; la mort neutre parce
 * qu'une mort sans tueur crédité reste une mort que le fil doit dire.
 */
export interface ReplayFeedEntry {
  /** Clé stable de rendu. */
  key: string
  /** Instant de la ligne dans le repère du film. */
  replayMs: number
  kill: ReplayKill | null
  medal: MedalEvent | null
  death: ReplayDeath | null
}

/**
 * toReplayKills recale les kills sur l'horloge BRUTE du film et les trie
 * chronologiquement. C'est le repli quand aucun document n'est disponible — le chemin
 * nominal est `alignFeedToTracks`, qui corrige en plus le décalage d'origine de
 * l'artefact (cf. en-tête).
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
 * Tolérance d'appariement d'un kill à une fin de vie de sa victime, en ms, APRÈS
 * correction du décalage médian mesuré. Les paires justes se groupent à ±150 ms de la
 * médiane (87/91 sur 000d5950) et deux vies d'un même joueur sont séparées d'au moins un
 * délai de réapparition (~8 s mesurés) : 2 s ne peut ni rater une vraie paire ni en
 * confondre deux.
 */
export const KILL_MATCH_TOL_MS = 2_000

/**
 * Fenêtre de veto d'une ligne de mort neutre autour d'un kill SANS victime nommée : tant
 * que la couverture killer_victim_pairs est partielle, un kill non attribué peut être
 * celui de cette mort — on ne double jamais une mort qui a peut-être déjà sa ligne.
 */
export const NEUTRAL_DEATH_VETO_MS = 4_000

/**
 * Marge sous l'horizon de l'artefact au-delà de laquelle une fin de vie est une VRAIE
 * mort et non un survivant de fin de partie (mesure 000d5950 : les 3 vies des survivants
 * se ferment dans les 3 dernières secondes du document).
 */
export const SURVIVOR_GRACE_MS = 3_000

/** Une mort SANS ligne de kill (suicide, chute, sortie), lue dans les pistes du film. */
export interface ReplayDeath {
  /** Instant de la fin de vie sur l'horloge du rejeu — le même que le flash de fiche. */
  replayMs: number
  xuid: string
}

/** Le fil recalé sur les pistes : kills alignés, morts neutres, décalage mesuré. */
export interface AlignedFeed {
  kills: ReplayKill[]
  deaths: ReplayDeath[]
  /**
   * Décalage médian mesuré `event_time_ms + t0Ms − fin de vie` sur CE document, en ms.
   * Null quand aucune paire (victime, fin de vie) n'existe — rien n'est alors corrigible.
   */
  offsetMs: number | null
}

/** Une fin de vie candidate à l'appariement — interne à l'alignement. */
interface LifeEnd {
  xuid: string
  endMs: number
  /** Close AVANT l'horizon du document : une vraie mort, pas un survivant. */
  closed: boolean
  used: boolean
}

/**
 * alignFeedToTracks recale les kills sur le référentiel des PISTES — celui qui date le
 * flash des fiches — et en déduit les morts neutres.
 *
 * DEUX PASSES, TOUTES DEUX MESURÉES :
 *  1. le décalage médian par match (appariement ordonné kill↔fin de vie par victime —
 *     la médiane résiste aux vies non publiées par le film) ;
 *  2. l'appariement au plus proche une fois le décalage retranché, chaque fin de vie
 *     consommée UNE fois. Un kill apparié est posé SUR la fin de vie de sa victime
 *     (même instant que le flash) ; un kill inappariable est corrigé du décalage médian ;
 *     sans aucune mesure (aucune victime nommée), l'horloge brute reste la moins fausse.
 *
 * Les MORTS NEUTRES ne sortent que d'un document où l'appariement a eu lieu : sans
 * victimes nommées, impossible de garantir qu'une fin de vie n'a pas déjà sa ligne.
 */
export function alignFeedToTracks(
  kills: KillEvent[],
  t0Ms: number,
  doc: ReplayDocumentReady,
): AlignedFeed {
  const offset = replayOffset(t0Ms)
  const ends: LifeEnd[] = []
  let maxEndFrame = 0
  for (const tr of doc.tracks) {
    if (!tr.xuid || tr.points.length === 0) continue
    const endFrame = trackWindow(tr).end
    if (endFrame > maxEndFrame) maxEndFrame = endFrame
    ends.push({ xuid: tr.xuid, endMs: frameToMs(endFrame, doc), closed: false, used: false })
  }
  const horizonFrame = (doc.frameCount ?? 0) > 1 ? (doc.frameCount ?? 0) - 1 : maxEndFrame
  const graceMs = frameToMs(Math.max(0, horizonFrame - msToFrames(SURVIVOR_GRACE_MS, doc)), doc)
  for (const e of ends) e.closed = e.endMs < graceMs

  // PASSE 1 — la mesure : fins de vie et kills d'une même victime, triés, appariés par
  // rang. La médiane des écarts est le décalage du document (robuste aux vies que le
  // film n'a pas publiées : elles décalent des rangs, pas la moitié des paires).
  const rawKills = kills.map((k) => ({ k, raw: k.tMs + offset }))
  const endsByXuid = new Map<string, number[]>()
  for (const e of ends) {
    const list = endsByXuid.get(e.xuid)
    if (list) list.push(e.endMs)
    else endsByXuid.set(e.xuid, [e.endMs])
  }
  const killsByVictim = new Map<string, number[]>()
  for (const { k, raw } of rawKills) {
    if (!k.victimXuid) continue
    const list = killsByVictim.get(k.victimXuid)
    if (list) list.push(raw)
    else killsByVictim.set(k.victimXuid, [raw])
  }
  const dts: number[] = []
  for (const [xuid, kts] of killsByVictim) {
    const es = (endsByXuid.get(xuid) ?? []).slice().sort((a, b) => a - b)
    kts.sort((a, b) => a - b)
    for (let i = 0; i < Math.min(kts.length, es.length); i++) dts.push(kts[i] - es[i])
  }
  dts.sort((a, b) => a - b)
  const offsetMs = dts.length > 0 ? dts[Math.floor(dts.length / 2)] : null

  // PASSE 2 — l'appariement : au plus proche une fois le décalage retranché, dans
  // l'ordre chronologique, chaque fin de vie servie une seule fois.
  const aligned: ReplayKill[] = []
  for (const { k, raw } of [...rawKills].sort((a, b) => a.raw - b.raw)) {
    let best: LifeEnd | null = null
    let bestScore = Number.POSITIVE_INFINITY
    if (k.victimXuid && offsetMs !== null) {
      for (const e of ends) {
        if (e.used || e.xuid !== k.victimXuid) continue
        const score = Math.abs(raw - e.endMs - offsetMs)
        if (score < bestScore) {
          bestScore = score
          best = e
        }
      }
    }
    if (best && bestScore <= KILL_MATCH_TOL_MS) {
      best.used = true
      aligned.push({ ...k, replayMs: best.endMs, medals: [] })
    } else {
      aligned.push({ ...k, replayMs: raw - (offsetMs ?? 0), medals: [] })
    }
  }
  aligned.sort((a, b) => a.replayMs - b.replayMs)

  // MORTS NEUTRES : les fins de vie closes qu'aucun kill n'a consommées — sous garde
  // d'appariement effectif, et sous veto d'un kill sans victime au voisinage.
  const deaths: ReplayDeath[] = []
  if (offsetMs !== null) {
    const unattributed = aligned.filter((k) => !k.victimXuid)
    for (const e of ends) {
      if (!e.closed || e.used) continue
      const vetoed = unattributed.some((k) => Math.abs(k.replayMs - e.endMs) <= NEUTRAL_DEATH_VETO_MS)
      if (!vetoed) deaths.push({ replayMs: e.endMs, xuid: e.xuid })
    }
    deaths.sort((a, b) => a.replayMs - b.replayMs)
  }
  return { kills: aligned, deaths, offsetMs }
}

/**
 * buildFeedEntries assemble le fil : kills recalés sur les PISTES quand le document est
 * fourni (`alignFeedToTracks` — même instant que le flash de fiche), morts neutres en
 * lignes propres, médailles rattachées au kill du même acteur le plus proche
 * (±MEDAL_ATTACH_MS, sur l'horloge gameplay commune aux deux sources), médailles
 * orphelines en lignes seules. Trié chronologiquement.
 *
 * Sans document (tests, appels historiques) : recalage brut `+t0Ms`, aucune mort neutre.
 */
export function buildFeedEntries(
  kills: KillEvent[],
  medals: MedalEvent[],
  t0Ms: number,
  doc?: ReplayDocumentReady | null,
): ReplayFeedEntry[] {
  const alignedFeed: AlignedFeed =
    doc && doc.tracks.length > 0
      ? alignFeedToTracks(kills, t0Ms, doc)
      : { kills: toReplayKills(kills, t0Ms), deaths: [], offsetMs: null }
  const rk = alignedFeed.kills
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
    death: null,
  }))
  alone.forEach((m, i) => {
    entries.push({
      key: `m-${m.xuid}-${m.tMs}-${i}`,
      // La médaille seule suit la même correction de décalage que les kills : une ligne
      // du fil ne vit pas sur une autre horloge que ses voisines.
      replayMs: m.tMs + offset - (alignedFeed.offsetMs ?? 0),
      kill: null,
      medal: m,
      death: null,
    })
  })
  alignedFeed.deaths.forEach((d, i) => {
    entries.push({
      key: `d-${d.xuid}-${d.replayMs}-${i}`,
      replayMs: d.replayMs,
      kill: null,
      medal: null,
      death: d,
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
