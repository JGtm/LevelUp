/**
 * killFeedLogic.ts — LE FIL DES ÉLIMINATIONS SUR L'HORLOGE DU REJEU.
 *
 * DEUX HORLOGES, ET C'EST TOUT LE SUJET. Les events servis par la Match View sont recalés
 * sur le début du GAMEPLAY (`correctMatchViewEventsT0`) : leur ajouter `t0Ms` — le T0 RÉEL
 * du match, tiré de `match_registry.real_start_time` — reconstruit exactement l'horloge du
 * film. L'artefact de rejeu, lui, cale sa frame 0 sur le PREMIER PAQUET DE POSITION du
 * film, un instant qui varie de 3,6 s à 39,8 s selon le match (chargement et mise en
 * place). Il reste donc UN décalage entre les deux, et un seul.
 *
 * CE DÉCALAGE EST DÉSORMAIS PUBLIÉ PAR L'ARTEFACT (`doc.originMs`, schéma v4) : c'est
 * l'instant de la frame 0 sur l'horloge du fil, mesuré hors ligne comme la différence de
 * deux en-têtes de paquet du même film (cf. Go `internal/analysis/replay/origin.go`). Le
 * recalage nominal est donc une SOUSTRACTION, `alignFeedByOrigin` :
 *
 *     replayMs = event_time_ms + t0Ms − originMs
 *
 * CE QUE ÇA REMPLACE, ET POURQUOI. Le chemin précédent (`alignFeedToTracks`, 2026-08-14)
 * MESURAIT ce décalage dans le navigateur, en appariant les kills aux fins de vie de leurs
 * victimes : médiane +3 678 ms sur 000d5950, +10 589 ms sur 64e8adfa, +39 856 ms sur
 * e94163af. Les mêmes valeurs sortent de l'artefact à 20-70 ms près — mais l'appariement
 * EXIGE des victimes nommées, et il n'y en a aucune sur les matchs que `killer_victim_pairs`
 * ne couvre pas (témoin 606d9844 : 35 kills, 0 victime nommée, aucun recalage possible).
 * L'origine publiée, elle, ne dépend d'aucune couverture.
 *
 * IL RESTE COMME REPLI, et seulement là : artefact antérieur au schéma v4, ou origine que
 * le producteur a refusé d'établir (chunk illisible, témoin contradictoire — il ne publie
 * alors rien plutôt qu'une origine douteuse).
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
 * CE QUE LE CLIENT NE PEUT PAS DÉDUIRE DE CES MORTS, c'est DE QUOI le joueur est mort :
 * cela se lit dans le film, hors ligne. Le document le publie (`doc.neutralDeaths`, schéma
 * v5) et `attachDeathKinds` le pose sur la ligne, sur la MÊME horloge que le reste du fil.
 * Une mort dont le type n'est pas établi garde son repère neutre — jamais l'icône d'une
 * autre mort.
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
  /** Instant du kill sur l'AXE DU REJEU, en millisecondes depuis la frame 0 du document. */
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
  /** Instant de la ligne sur l'axe du rejeu. */
  replayMs: number
  kill: ReplayKill | null
  medal: MedalEvent | null
  death: ReplayDeath | null
}

/**
 * toReplayKills recale les kills sur l'horloge BRUTE du film et les trie
 * chronologiquement. C'est le dernier repli, quand aucun document n'est disponible — le
 * chemin nominal est `alignFeedByOrigin` (cf. en-tête).
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
  /**
   * Le TYPE de mort, quand le document l'établit : `environment` (chute, hors-limites) ou
   * `suicide` (le joueur s'est tué avec sa propre source de dégât). Vide = NON ÉTABLI, et la
   * ligne garde alors son repère neutre — jamais l'icône d'une autre mort.
   */
  kind: string
  /** Pictogramme du type et masque à teindre. Vides en même temps que `kind`. */
  img: string
  tinted: boolean
}

/** Le fil recalé : kills alignés, morts neutres, décalage appliqué. */
export interface AlignedFeed {
  kills: ReplayKill[]
  deaths: ReplayDeath[]
  /**
   * Décalage retranché à l'horloge du fil pour l'amener sur l'axe du rejeu, en ms :
   * l'ORIGINE publiée par l'artefact sur le chemin nominal, le décalage médian MESURÉ
   * (`event_time_ms + t0Ms − fin de vie`) sur le repli. Null quand rien n'est corrigible —
   * ni origine publiée, ni aucune paire (victime, fin de vie) à mesurer.
   *
   * Les lignes qui ne sont pas des kills (médailles seules) s'en servent pour rester sur la
   * même horloge que leurs voisines.
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
 * alignFeed pose le fil sur l'axe du rejeu. C'est le POINT D'ENTRÉE unique du recalage :
 * l'origine publiée quand l'artefact la porte, l'appariement statistique sinon.
 *
 * Le repli n'est pas un ancien chemin qu'on aurait oublié de retirer : un artefact
 * antérieur au schéma v4 n'a pas d'origine, et le producteur REFUSE d'en publier une
 * lorsqu'il ne peut pas l'établir (cf. origin.go). Dans ces deux cas, la mesure par
 * appariement reste la moins fausse — quand le match a des victimes nommées.
 */
export function alignFeed(
  kills: KillEvent[],
  t0Ms: number,
  doc: ReplayDocumentReady,
): AlignedFeed {
  const origin = doc.originMs
  const feed =
    typeof origin === 'number' && Number.isFinite(origin)
      ? alignFeedByOrigin(kills, t0Ms, doc, origin)
      : alignFeedToTracks(kills, t0Ms, doc)
  // LE TYPE DE MORT SE POSE ICI, une seule fois pour les deux recalages : il dépend du
  // décalage, et le décalage n'est connu qu'une fois le recalage choisi. Le poser dans
  // chacune des deux branches en ferait deux copies qui divergeraient.
  attachDeathKinds(feed.deaths, doc, feed.offsetMs)
  return feed
}

/**
 * alignFeedByOrigin — LE CHEMIN NOMINAL : une soustraction, pas un appariement.
 *
 * `replayMs = event_time_ms + t0Ms − originMs`, appliqué à TOUS les kills, qu'ils portent
 * une victime nommée ou non. Aucune couverture n'est requise, aucun kill n'est laissé sur
 * une autre horloge que ses voisins.
 *
 * LES MORTS NEUTRES restent déduites des pistes, parce que rien d'autre ne les porte : une
 * fin de vie close qu'aucun kill ne revendique est une mort que le fil doit dire. Deux
 * gardes, identiques à celles du repli : un kill de la MÊME victime à moins de
 * `KILL_MATCH_TOL_MS` consomme la fin de vie ; un kill SANS victime nommée à moins de
 * `NEUTRAL_DEATH_VETO_MS` lui met un veto — tant que `killer_victim_pairs` est partielle,
 * ce kill peut être celui de cette mort, et on ne double jamais une mort déjà servie.
 */
export function alignFeedByOrigin(
  kills: KillEvent[],
  t0Ms: number,
  doc: ReplayDocumentReady,
  originMs: number,
): AlignedFeed {
  const offset = replayOffset(t0Ms)
  const aligned: ReplayKill[] = kills
    .map((k) => ({ ...k, replayMs: k.tMs + offset - originMs, medals: [] as MedalEvent[] }))
    .sort((a, b) => a.replayMs - b.replayMs)

  const ends = lifeEndsOf(doc)
  for (const k of aligned) {
    if (!k.victimXuid) continue
    let best: LifeEnd | null = null
    let bestScore = Number.POSITIVE_INFINITY
    for (const e of ends) {
      if (e.used || e.xuid !== k.victimXuid) continue
      const score = Math.abs(k.replayMs - e.endMs)
      if (score < bestScore) {
        bestScore = score
        best = e
      }
    }
    if (best && bestScore <= KILL_MATCH_TOL_MS) best.used = true
  }
  const unattributed = aligned.filter((k) => !k.victimXuid)
  const deaths: ReplayDeath[] = []
  for (const e of ends) {
    if (!e.closed || e.used) continue
    if (unattributed.some((k) => Math.abs(k.replayMs - e.endMs) <= NEUTRAL_DEATH_VETO_MS)) continue
    deaths.push(neutralDeath(e.endMs, e.xuid))
  }
  deaths.sort((a, b) => a.replayMs - b.replayMs)
  return { kills: aligned, deaths, offsetMs: originMs }
}

/**
 * neutralDeath fabrique une ligne de mort neutre SANS type. Le type est posé après coup par
 * `attachDeathKinds`, une fois le décalage du fil connu — lui seul permet de rapprocher
 * l'horloge du document (où le type est daté) de l'axe du rejeu (où la ligne vit).
 */
function neutralDeath(replayMs: number, xuid: string): ReplayDeath {
  return { replayMs, xuid, kind: '', img: '', tinted: false }
}

/**
 * attachDeathKinds pose sur chaque mort neutre le TYPE que le document lui donne — chute /
 * hors-limites, ou sa propre source de dégât.
 *
 * CE QUE LE CLIENT NE PEUT PAS DÉDUIRE, ET POURQUOI CETTE JOINTURE EXISTE. Le fil sait déjà
 * QUI est mort et QUAND (une fin de vie qu'aucun kill ne revendique). Ce qu'il ne peut pas
 * savoir, c'est DE QUOI : cela se lit dans le composant dead-state du film, hors ligne. Le
 * document publie donc ces morts avec leur type — et sur SON horloge, celle du fil.
 *
 * LE RECALAGE EST LE MÊME QUE POUR LE RESTE DU FIL, et il doit l'être : `feedMs − offsetMs`,
 * exactement ce que subissent les kills et les médailles seules. Une entrée posée sur une
 * autre horloge que la ligne qu'elle décore ne la rencontrerait jamais.
 *
 * L'APPARIEMENT EST CONTRAINT DES DEUX CÔTÉS : même xuid, et moins de `KILL_MATCH_TOL_MS`
 * d'écart — la même tolérance que l'appariement d'un kill à la fin de vie de sa victime, et
 * pour la même raison (deux vies d'un même joueur sont séparées d'au moins un délai de
 * réapparition). Chaque entrée ne sert qu'UNE fois. Rien ne se pose sans appariement : une
 * ligne sans type garde son repère neutre, JAMAIS l'icône d'une autre mort.
 */
export function attachDeathKinds(
  deaths: ReplayDeath[],
  doc: ReplayDocumentReady,
  offsetMs: number | null,
): void {
  if (deaths.length === 0 || doc.neutralDeaths.length === 0) return
  const shift = offsetMs ?? 0
  const used = new Set<number>()
  for (const d of deaths) {
    let bestIdx = -1
    let bestScore = KILL_MATCH_TOL_MS + 1
    doc.neutralDeaths.forEach((n, i) => {
      if (used.has(i) || n.xuid !== d.xuid) return
      const score = Math.abs(d.replayMs - (n.feedMs - shift))
      if (score < bestScore) {
        bestScore = score
        bestIdx = i
      }
    })
    if (bestIdx < 0) continue
    used.add(bestIdx)
    const n = doc.neutralDeaths[bestIdx]
    d.kind = n.kind
    d.img = n.img ?? ''
    d.tinted = n.tinted ?? false
  }
}

/**
 * lifeEndsOf relit les fins de vie des pistes et dit lesquelles sont CLOSES avant
 * l'horizon du document (une vie qui se ferme dans les dernières secondes est un survivant
 * de fin de partie, pas une mort). Partagé par les deux recalages : la définition d'une
 * fin de vie ne doit pas dépendre du chemin qui la lit.
 */
function lifeEndsOf(doc: ReplayDocumentReady): LifeEnd[] {
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
  return ends
}

/**
 * alignFeedToTracks recale les kills sur le référentiel des PISTES — celui qui date le
 * flash des fiches — et en déduit les morts neutres. C'est le REPLI (cf. `alignFeed`) :
 * il n'est employé que lorsque l'artefact ne publie pas son origine.
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
  const ends = lifeEndsOf(doc)

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
      if (!vetoed) deaths.push(neutralDeath(e.endMs, e.xuid))
    }
    deaths.sort((a, b) => a.replayMs - b.replayMs)
  }
  return { kills: aligned, deaths, offsetMs }
}

/**
 * buildFeedEntries assemble le fil : kills recalés par `alignFeed` quand le document est
 * fourni (origine publiée, appariement en repli), morts neutres en
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
      ? alignFeed(kills, t0Ms, doc)
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
